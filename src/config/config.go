package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

const DefaultTelegramAPIEndpoint = "https://api.telegram.org/bot%s/%s"

var (
	// Telegram Bot
	BotToken            string
	HTTPToken           string
	OwnerChatID         int64
	TelegramAPIEndpoint string
	DebugFlag           bool

	// Log
	LogLevel    string
	ApiLogLevel string

	// Image conversion
	JpegQuality int

	// HTTP Server
	EnableHTTPServer bool
	HTTPServerPort   string

	// Concurrency
	MaxConcurrency int

	// Retry
	MaxRetry int

	// Operational notifications
	NotificationRequestTimeout time.Duration
	PanicDedupWindow           time.Duration
	MaxStackBytes              int

	// Process supervision
	SupervisorShutdownTimeout time.Duration
	RestartInitialDelay       time.Duration
	RestartMaxDelay           time.Duration
	RestartMultiplier         float64
	RestartJitter             float64
	RestartStableAfter        time.Duration
	RestartMaxRestarts        int
	RestartWindow             time.Duration
	RestartCooldown           time.Duration
)

type Settings struct {
	BotToken            string
	HTTPToken           string
	OwnerChatID         int64
	TelegramAPIEndpoint string
	DebugFlag           bool
	LogLevel            string
	APILevel            string
	JPEGQuality         int
	EnableHTTPServer    bool
	HTTPServerPort      string
	MaxConcurrency      int
	MaxRetry            int
	Notification        NotificationSettings
	Supervisor          SupervisorSettings
}

type NotificationSettings struct {
	RequestTimeout time.Duration
	PanicDedup     time.Duration
	MaxStackBytes  int
}

type SupervisorSettings struct {
	ShutdownTimeout time.Duration
	Restart         RestartSettings
}

type RestartSettings struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       float64
	StableAfter  time.Duration
	MaxRestarts  int
	Window       time.Duration
	Cooldown     time.Duration
}

// Load reads and validates a configuration without mutating package globals.
func Load(configPath string) (Settings, string, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	setDefaults(v)

	if configPath != "" {
		absolutePath, err := filepath.Abs(configPath)
		if err != nil {
			return Settings{}, "", fmt.Errorf("解析配置路径: %w", err)
		}
		v.SetConfigFile(absolutePath)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath("../config")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		return Settings{}, "", fmt.Errorf("读取配置文件: %w", err)
	}

	usedPath, err := filepath.Abs(v.ConfigFileUsed())
	if err != nil {
		return Settings{}, "", fmt.Errorf("解析配置文件路径: %w", err)
	}

	settings := Settings{
		BotToken:            v.GetString("telegram.token"),
		HTTPToken:           v.GetString("telegram.http_token"),
		OwnerChatID:         v.GetInt64("telegram.owner_chat_id"),
		TelegramAPIEndpoint: v.GetString("telegram.api_endpoint"),
		DebugFlag:           v.GetBool("telegram.debug"),
		LogLevel:            v.GetString("log.level"),
		APILevel:            v.GetString("log.api_level"),
		JPEGQuality:         v.GetInt("image.jpeg_quality"),
		EnableHTTPServer:    v.GetBool("server.enabled"),
		HTTPServerPort:      v.GetString("server.port"),
		MaxConcurrency:      v.GetInt("download.max_concurrency"),
		MaxRetry:            v.GetInt("download.max_retry"),
		Notification: NotificationSettings{
			RequestTimeout: v.GetDuration("notification.request_timeout"),
			PanicDedup:     v.GetDuration("notification.panic_dedup_window"),
			MaxStackBytes:  v.GetInt("notification.max_stack_bytes"),
		},
		Supervisor: SupervisorSettings{
			ShutdownTimeout: v.GetDuration("supervisor.shutdown_timeout"),
			Restart: RestartSettings{
				InitialDelay: v.GetDuration("supervisor.restart.initial_delay"),
				MaxDelay:     v.GetDuration("supervisor.restart.max_delay"),
				Multiplier:   v.GetFloat64("supervisor.restart.multiplier"),
				Jitter:       v.GetFloat64("supervisor.restart.jitter"),
				StableAfter:  v.GetDuration("supervisor.restart.stable_after"),
				MaxRestarts:  v.GetInt("supervisor.restart.max_restarts"),
				Window:       v.GetDuration("supervisor.restart.window"),
				Cooldown:     v.GetDuration("supervisor.restart.cooldown"),
			},
		},
	}
	if err := validate(settings); err != nil {
		return Settings{}, usedPath, err
	}
	return settings, usedPath, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("telegram.api_endpoint", DefaultTelegramAPIEndpoint)
	v.SetDefault("telegram.debug", false)
	v.SetDefault("log.level", "INFO")
	v.SetDefault("log.api_level", "INFO")
	v.SetDefault("image.jpeg_quality", 100)
	v.SetDefault("server.enabled", false)
	v.SetDefault("server.port", ":8070")
	v.SetDefault("download.max_concurrency", 30)
	v.SetDefault("download.max_retry", 3)
	v.SetDefault("notification.request_timeout", "8s")
	v.SetDefault("notification.panic_dedup_window", "5m")
	v.SetDefault("notification.max_stack_bytes", 8192)
	v.SetDefault("supervisor.shutdown_timeout", "10s")
	v.SetDefault("supervisor.restart.initial_delay", "1s")
	v.SetDefault("supervisor.restart.max_delay", "1m")
	v.SetDefault("supervisor.restart.multiplier", 2.0)
	v.SetDefault("supervisor.restart.jitter", 0.2)
	v.SetDefault("supervisor.restart.stable_after", "10m")
	v.SetDefault("supervisor.restart.max_restarts", 5)
	v.SetDefault("supervisor.restart.window", "5m")
	v.SetDefault("supervisor.restart.cooldown", "15m")
}

func validate(s Settings) error {
	var validationErrors []error
	if s.BotToken == "" || s.BotToken == "YOUR_TOKEN_HERE" || s.BotToken == "YOUR_BOT_TOKEN" {
		validationErrors = append(validationErrors, errors.New("telegram.token 未配置"))
	}
	if s.HTTPToken == "" {
		validationErrors = append(validationErrors, errors.New("telegram.http_token 未配置"))
	}
	if s.TelegramAPIEndpoint == "" {
		validationErrors = append(validationErrors, errors.New("telegram.api_endpoint 不能为空"))
	}
	if s.JPEGQuality < 0 || s.JPEGQuality > 100 {
		validationErrors = append(validationErrors, errors.New("image.jpeg_quality 必须在 0 到 100 之间"))
	}
	if s.MaxConcurrency <= 0 {
		validationErrors = append(validationErrors, errors.New("download.max_concurrency 必须大于 0"))
	}
	if s.MaxRetry <= 0 {
		validationErrors = append(validationErrors, errors.New("download.max_retry 必须大于 0"))
	}
	if s.Notification.RequestTimeout <= 0 || s.Notification.PanicDedup <= 0 || s.Notification.MaxStackBytes <= 0 {
		validationErrors = append(validationErrors, errors.New("notification 配置必须大于 0"))
	}
	r := s.Supervisor.Restart
	if s.Supervisor.ShutdownTimeout <= 0 || r.InitialDelay <= 0 || r.MaxDelay < r.InitialDelay || r.Multiplier < 1 || r.Jitter < 0 || r.Jitter > 1 || r.StableAfter <= 0 || r.MaxRestarts <= 0 || r.Window <= 0 || r.Cooldown <= 0 {
		validationErrors = append(validationErrors, errors.New("supervisor 配置无效"))
	}
	return errors.Join(validationErrors...)
}

// Apply publishes validated settings for existing packages that use config globals.
func Apply(s Settings) {
	BotToken = s.BotToken
	HTTPToken = s.HTTPToken
	OwnerChatID = s.OwnerChatID
	TelegramAPIEndpoint = s.TelegramAPIEndpoint
	DebugFlag = s.DebugFlag
	LogLevel = s.LogLevel
	ApiLogLevel = s.APILevel
	JpegQuality = s.JPEGQuality
	EnableHTTPServer = s.EnableHTTPServer
	HTTPServerPort = s.HTTPServerPort
	MaxConcurrency = s.MaxConcurrency
	MaxRetry = s.MaxRetry
	NotificationRequestTimeout = s.Notification.RequestTimeout
	PanicDedupWindow = s.Notification.PanicDedup
	MaxStackBytes = s.Notification.MaxStackBytes
	SupervisorShutdownTimeout = s.Supervisor.ShutdownTimeout
	RestartInitialDelay = s.Supervisor.Restart.InitialDelay
	RestartMaxDelay = s.Supervisor.Restart.MaxDelay
	RestartMultiplier = s.Supervisor.Restart.Multiplier
	RestartJitter = s.Supervisor.Restart.Jitter
	RestartStableAfter = s.Supervisor.Restart.StableAfter
	RestartMaxRestarts = s.Supervisor.Restart.MaxRestarts
	RestartWindow = s.Supervisor.Restart.Window
	RestartCooldown = s.Supervisor.Restart.Cooldown
}

// InitConfig loads and applies configuration for compatibility with existing callers.
func InitConfig(configPath ...string) error {
	path := ""
	if len(configPath) > 0 {
		path = configPath[0]
	}
	settings, _, err := Load(path)
	if err != nil {
		return err
	}
	Apply(settings)
	return nil
}

// WriteExampleConfig writes a safe placeholder configuration when explicitly requested.
func WriteExampleConfig(path string) error {
	const content = `telegram:
  token: "YOUR_BOT_TOKEN"
  http_token: "YOUR_BOT_TOKEN"
  owner_chat_id: 0
  debug: false

log:
  level: "INFO"
  api_level: "INFO"

image:
  jpeg_quality: 100

server:
  enabled: false
  port: ":8070"

download:
  max_concurrency: 30
  max_retry: 3

notification:
  request_timeout: 8s
  panic_dedup_window: 5m
  max_stack_bytes: 8192

supervisor:
  shutdown_timeout: 10s
  restart:
    initial_delay: 1s
    max_delay: 1m
    multiplier: 2
    jitter: 0.2
    stable_after: 10m
    max_restarts: 5
    window: 5m
    cooldown: 15m
`
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0600)
}
