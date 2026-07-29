package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/lib"
	logger "github.com/swim233/StickerDownloader/logger"
	"github.com/swim233/StickerDownloader/utils"
)

// applyKind says what has to happen for a setting to take effect.
type applyKind string

const (
	applyHot     applyKind = "hot"     // takes effect immediately
	applyRestart applyKind = "restart" // needs a worker restart (one click)
	applyManual  applyKind = "manual"  // needs the supervisor process restarted by hand
)

// fieldType drives which editor the WebUI renders.
type fieldType string

const (
	fieldText     fieldType = "text"
	fieldSecret   fieldType = "secret"
	fieldInt      fieldType = "int"
	fieldFloat    fieldType = "float"
	fieldBool     fieldType = "bool"
	fieldDuration fieldType = "duration"
	fieldSelect   fieldType = "select"
)

type settingSpec struct {
	Key      string    `json:"key"`
	Section  string    `json:"section"`
	Label    string    `json:"label"`
	Help     string    `json:"help,omitempty"`
	Type     fieldType `json:"type"`
	Apply    applyKind `json:"apply"`
	Danger   bool      `json:"danger,omitempty"`
	Warning  string    `json:"warning,omitempty"`
	Options  []string  `json:"options,omitempty"`
	Min      *float64  `json:"min,omitempty"`
	Max      *float64  `json:"max,omitempty"`
	current  func() string
	hotApply func(string)
}

func f64(v float64) *float64 { return &v }

// settingSpecs is the single source of truth for the settings page: what is
// editable, how risky it is, and how it takes effect.
var settingSpecs = []settingSpec{
	{
		Key: "telegram.token", Section: "Telegram", Label: "Bot Token", Type: fieldSecret,
		Apply: applyRestart, Danger: true,
		Warning: "改错会导致机器人完全失联，只能通过 SSH 修复配置文件。",
		current: func() string { return config.BotToken },
	},
	{
		Key: "telegram.http_token", Section: "Telegram", Label: "HTTP 服务 Token", Type: fieldSecret,
		Apply: applyRestart, Danger: true,
		Warning: "供 /stickerpack 使用的 Bot Token，改错会导致该接口不可用。",
		current: func() string { return config.HTTPToken },
	},
	{
		Key: "telegram.owner_chat_id", Section: "Telegram", Label: "所有者 Chat ID", Type: fieldInt,
		Apply: applyRestart, Danger: true,
		Warning: "改错会失去 /ban 等管理命令的权限，运维通知也会发给错误的人。",
		current: func() string { return strconv.FormatInt(config.OwnerChatID, 10) },
	},
	{
		Key: "telegram.api_endpoint", Section: "Telegram", Label: "API 端点", Type: fieldText,
		Apply: applyRestart, Danger: true,
		Warning: "指向非官方端点会导致无法连接 Telegram。",
		Help:    "默认 " + config.DefaultTelegramAPIEndpoint,
		current: func() string { return config.TelegramAPIEndpoint },
	},
	{
		Key: "telegram.debug", Section: "Telegram", Label: "Bot API 调试输出", Type: fieldBool,
		Apply:   applyRestart,
		current: func() string { return strconv.FormatBool(config.DebugFlag) },
	},

	{
		Key: "log.level", Section: "日志", Label: "日志等级", Type: fieldSelect,
		Options: []string{"DEBUG", "INFO", "WARN", "ERROR"}, Apply: applyHot,
		current:  func() string { return config.LogLevel },
		hotApply: func(v string) { config.LogLevel = v; logger.SetLogLevel(v) },
	},
	{
		Key: "log.api_level", Section: "日志", Label: "Bot API 日志等级", Type: fieldSelect,
		Options: []string{"DEBUG", "INFO", "WARN", "ERROR"}, Apply: applyRestart,
		current: func() string { return config.ApiLogLevel },
	},

	{
		Key: "image.jpeg_quality", Section: "图片", Label: "JPEG 质量", Type: fieldInt,
		Min: f64(1), Max: f64(100), Apply: applyHot,
		current:  func() string { return strconv.Itoa(config.JpegQuality) },
		hotApply: func(v string) { config.JpegQuality, _ = strconv.Atoi(v) },
	},

	{
		Key: "server.enabled", Section: "WebUI / HTTP", Label: "启用 HTTP 服务器", Type: fieldBool,
		Apply: applyRestart, Danger: true,
		Warning: "关闭后 WebUI 会立刻无法访问，只能通过 SSH 改回配置。",
		current: func() string { return strconv.FormatBool(config.EnableHTTPServer) },
	},
	{
		Key: "server.port", Section: "WebUI / HTTP", Label: "监听地址", Type: fieldText,
		Apply: applyRestart, Danger: true,
		Warning: "改错会导致 WebUI 无法访问。反向代理部署建议保持 127.0.0.1:8070。",
		Help:    "形如 127.0.0.1:8070（仅本机）或 :8070（所有网卡）",
		current: func() string { return config.HTTPServerPort },
	},
	{
		Key: "server.history_size", Section: "WebUI / HTTP", Label: "保留下载记录条数", Type: fieldInt,
		Min: f64(1), Max: f64(1000), Apply: applyHot,
		current: func() string { return strconv.Itoa(config.DownloadHistorySize) },
		hotApply: func(v string) {
			size, _ := strconv.Atoi(v)
			config.DownloadHistorySize = size
			utils.DownloadHistory.SetCapacity(size)
		},
	},
	{
		Key: "server.password", Section: "WebUI / HTTP", Label: "WebUI 密码", Type: fieldSecret,
		Apply: applyHot, Danger: true,
		Warning:  "保存后当前会话仍然有效，但下次登录必须使用新密码。留空会让所有数据接口拒绝访问。",
		current:  func() string { return config.WebUIPassword },
		hotApply: func(v string) { config.WebUIPassword = v },
	},
	{
		Key: "server.api_token", Section: "WebUI / HTTP", Label: "/stickerpack 访问令牌", Type: fieldSecret,
		Apply:    applyHot,
		Help:     "留空则该下载接口对所有人开放",
		current:  func() string { return config.APIToken },
		hotApply: func(v string) { config.APIToken = v },
	},
	{
		Key: "server.behind_proxy", Section: "WebUI / HTTP", Label: "位于 HTTPS 反向代理之后", Type: fieldBool,
		Apply:    applyHot,
		Help:     "开启后信任 X-Forwarded-Proto，为会话 Cookie 加 Secure 标记并启用 HSTS",
		current:  func() string { return strconv.FormatBool(config.BehindProxy) },
		hotApply: func(v string) { config.BehindProxy = v == "true" },
	},

	{
		Key: "download.max_concurrency", Section: "下载", Label: "最大并发下载数", Type: fieldInt,
		Min: f64(1), Max: f64(200), Apply: applyRestart,
		current: func() string { return strconv.Itoa(config.MaxConcurrency) },
	},
	{
		Key: "download.max_retry", Section: "下载", Label: "最大重试次数", Type: fieldInt,
		Min: f64(1), Max: f64(20), Apply: applyHot,
		current:  func() string { return strconv.Itoa(config.MaxRetry) },
		hotApply: func(v string) { config.MaxRetry, _ = strconv.Atoi(v) },
	},

	{
		Key: "notification.request_timeout", Section: "通知", Label: "请求超时", Type: fieldDuration,
		Apply:   applyRestart,
		current: func() string { return config.NotificationRequestTimeout.String() },
	},
	{
		Key: "notification.panic_dedup_window", Section: "通知", Label: "Panic 去重窗口", Type: fieldDuration,
		Apply:   applyRestart,
		current: func() string { return config.PanicDedupWindow.String() },
	},
	{
		Key: "notification.max_stack_bytes", Section: "通知", Label: "堆栈最大字节数", Type: fieldInt,
		Min: f64(256), Max: f64(65536), Apply: applyRestart,
		current: func() string { return strconv.Itoa(config.MaxStackBytes) },
	},

	{
		Key: "supervisor.shutdown_timeout", Section: "进程守护", Label: "停机超时", Type: fieldDuration,
		Apply:   applyManual,
		current: func() string { return config.SupervisorShutdownTimeout.String() },
	},
	{
		Key: "supervisor.restart.initial_delay", Section: "进程守护", Label: "首次重启延迟", Type: fieldDuration,
		Apply:   applyManual,
		current: func() string { return config.RestartInitialDelay.String() },
	},
	{
		Key: "supervisor.restart.max_delay", Section: "进程守护", Label: "最大重启延迟", Type: fieldDuration,
		Apply:   applyManual,
		current: func() string { return config.RestartMaxDelay.String() },
	},
	{
		Key: "supervisor.restart.multiplier", Section: "进程守护", Label: "退避倍数", Type: fieldFloat,
		Min: f64(1), Max: f64(10), Apply: applyManual,
		current: func() string { return strconv.FormatFloat(config.RestartMultiplier, 'g', -1, 64) },
	},
	{
		Key: "supervisor.restart.jitter", Section: "进程守护", Label: "抖动系数", Type: fieldFloat,
		Min: f64(0), Max: f64(1), Apply: applyManual,
		current: func() string { return strconv.FormatFloat(config.RestartJitter, 'g', -1, 64) },
	},
	{
		Key: "supervisor.restart.stable_after", Section: "进程守护", Label: "稳定运行判定时长", Type: fieldDuration,
		Apply:   applyManual,
		current: func() string { return config.RestartStableAfter.String() },
	},
	{
		Key: "supervisor.restart.max_restarts", Section: "进程守护", Label: "窗口内最大重启次数", Type: fieldInt,
		Min: f64(1), Max: f64(100), Apply: applyManual,
		current: func() string { return strconv.Itoa(config.RestartMaxRestarts) },
	},
	{
		Key: "supervisor.restart.window", Section: "进程守护", Label: "重启计数窗口", Type: fieldDuration,
		Apply:   applyManual,
		current: func() string { return config.RestartWindow.String() },
	},
	{
		Key: "supervisor.restart.cooldown", Section: "进程守护", Label: "崩溃循环冷却时长", Type: fieldDuration,
		Apply:   applyManual,
		current: func() string { return config.RestartCooldown.String() },
	},
}

func specByKey(key string) (settingSpec, bool) {
	for _, spec := range settingSpecs {
		if spec.Key == key {
			return spec, true
		}
	}
	return settingSpec{}, false
}

// secretHint shows enough of a secret to recognise it without leaking it.
func secretHint(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "••••"
	}
	return "••••" + value[len(value)-4:]
}

type settingValue struct {
	settingSpec
	Value string `json:"value"`
	Set   bool   `json:"set"`
}

type settingsResponse struct {
	Settings   []settingValue `json:"settings"`
	ConfigPath string         `json:"config_path"`
	Writable   bool           `json:"writable"`
}

// configPath resolves the config file this worker was started with, falling
// back to the same search order config.Load uses when no flag was passed.
func configPath() string {
	args := os.Args
	for i, arg := range args {
		if arg == "--config" && i+1 < len(args) {
			return args[i+1]
		}
		if after, ok := strings.CutPrefix(arg, "--config="); ok {
			return after
		}
	}
	for _, candidate := range []string{"../config/config.yaml", "config.yaml"} {
		if _, err := os.Stat(candidate); err == nil {
			absolute, err := filepath.Abs(candidate)
			if err == nil {
				return absolute
			}
			return candidate
		}
	}
	return ""
}

func configWritable(path string) bool {
	if path == "" {
		return false
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

func handleAPISettings(w http.ResponseWriter, r *http.Request) {
	path := configPath()
	resp := settingsResponse{
		Settings:   make([]settingValue, 0, len(settingSpecs)),
		ConfigPath: path,
		Writable:   configWritable(path),
	}
	for _, spec := range settingSpecs {
		value := spec.current()
		item := settingValue{settingSpec: spec, Set: value != ""}
		if spec.Type == fieldSecret {
			item.Value = secretHint(value)
		} else {
			item.Value = value
		}
		resp.Settings = append(resp.Settings, item)
	}
	writeJSON(w, resp)
}

type settingsUpdateRequest struct {
	Updates map[string]string `json:"updates"`
}

type settingsUpdateResponse struct {
	Saved         []string `json:"saved"`
	NeedsRestart  []string `json:"needs_restart"`
	NeedsManual   []string `json:"needs_manual"`
	AppliedNow    []string `json:"applied_now"`
	RestartCalled bool     `json:"restart_called"`
}

// validateSetting checks one value against its spec and normalises it.
func validateSetting(spec settingSpec, value string) (string, error) {
	switch spec.Type {
	case fieldBool:
		if value != "true" && value != "false" {
			return "", fmt.Errorf("%s 必须是 true 或 false", spec.Label)
		}
	case fieldInt:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return "", fmt.Errorf("%s 必须是整数", spec.Label)
		}
		if spec.Min != nil && float64(n) < *spec.Min {
			return "", fmt.Errorf("%s 不能小于 %g", spec.Label, *spec.Min)
		}
		if spec.Max != nil && float64(n) > *spec.Max {
			return "", fmt.Errorf("%s 不能大于 %g", spec.Label, *spec.Max)
		}
	case fieldFloat:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return "", fmt.Errorf("%s 必须是数字", spec.Label)
		}
		if spec.Min != nil && f < *spec.Min {
			return "", fmt.Errorf("%s 不能小于 %g", spec.Label, *spec.Min)
		}
		if spec.Max != nil && f > *spec.Max {
			return "", fmt.Errorf("%s 不能大于 %g", spec.Label, *spec.Max)
		}
	case fieldDuration:
		d, err := time.ParseDuration(value)
		if err != nil {
			return "", fmt.Errorf("%s 必须是时长，例如 10s、5m", spec.Label)
		}
		if d <= 0 {
			return "", fmt.Errorf("%s 必须大于 0", spec.Label)
		}
	case fieldSelect:
		if len(spec.Options) > 0 {
			for _, option := range spec.Options {
				if option == value {
					return value, nil
				}
			}
			return "", fmt.Errorf("%s 必须是 %s 之一", spec.Label, strings.Join(spec.Options, "、"))
		}
	case fieldText, fieldSecret:
		if strings.ContainsAny(value, "\n\r") {
			return "", fmt.Errorf("%s 不能包含换行", spec.Label)
		}
	}
	return value, nil
}

func handleAPISettingsUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "仅支持 POST")
		return
	}
	var req settingsUpdateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if len(req.Updates) == 0 {
		writeJSONError(w, http.StatusBadRequest, "没有需要保存的修改")
		return
	}

	path := configPath()
	if path == "" {
		writeJSONError(w, http.StatusInternalServerError, "无法确定配置文件位置")
		return
	}

	validated := make(map[string]string, len(req.Updates))
	specs := make(map[string]settingSpec, len(req.Updates))
	for key, value := range req.Updates {
		spec, ok := specByKey(key)
		if !ok {
			writeJSONError(w, http.StatusBadRequest, "未知配置项: "+key)
			return
		}
		normalized, err := validateSetting(spec, value)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		validated[key] = normalized
		specs[key] = spec
	}

	if err := config.UpdateFile(path, validated); err != nil {
		logger.Error("保存配置失败: %s", err)
		writeJSONError(w, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}

	resp := settingsUpdateResponse{
		Saved: []string{}, NeedsRestart: []string{}, NeedsManual: []string{}, AppliedNow: []string{},
	}
	for key, value := range validated {
		spec := specs[key]
		resp.Saved = append(resp.Saved, key)
		switch spec.Apply {
		case applyHot:
			if spec.hotApply != nil {
				spec.hotApply(value)
			}
			resp.AppliedNow = append(resp.AppliedNow, key)
		case applyRestart:
			resp.NeedsRestart = append(resp.NeedsRestart, key)
		case applyManual:
			resp.NeedsManual = append(resp.NeedsManual, key)
		}
	}
	sort.Strings(resp.Saved)
	sort.Strings(resp.NeedsRestart)
	sort.Strings(resp.NeedsManual)
	sort.Strings(resp.AppliedNow)
	logger.Info("WebUI 更新了配置: %s", strings.Join(resp.Saved, ", "))
	writeJSON(w, resp)
}

func handleAPIRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "仅支持 POST")
		return
	}
	if !lib.RequestRestart("WebUI 请求重启") {
		writeJSONError(w, http.StatusConflict, "已有重启正在进行")
		return
	}
	logger.Info("WebUI 请求重启 worker")
	writeJSON(w, map[string]bool{"ok": true})
}
