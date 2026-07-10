package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/api"
	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/db"
	"github.com/swim233/StickerDownloader/handler"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/logger"
	"github.com/swim233/StickerDownloader/message"
	"github.com/swim233/StickerDownloader/notify"
	"github.com/swim233/StickerDownloader/runtimeguard"
	"github.com/swim233/StickerDownloader/supervisor"
	"github.com/swim233/StickerDownloader/task"
	"github.com/swim233/StickerDownloader/utils"
)

var (
	version    string
	commitHash string
	buildTime  string
)

var stickerPackLinkRegex = regexp.MustCompile(`https?://t\.me/addstickers/[a-zA-Z0-9_]+`)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("stickerDownloader", flag.ContinueOnError)
	worker := flags.Bool("worker", false, "internal worker mode")
	configPath := flags.String("config", "", "配置文件路径")
	if err := flags.Parse(args); err != nil {
		return supervisor.ExitUsage
	}

	settings, usedPath, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n", err)
		return supervisor.ExitConfig
	}
	if settings.OwnerChatID == 0 {
		fmt.Fprintln(os.Stderr, "警告: telegram.owner_chat_id 为 0，运维通知已禁用")
	}

	notifier := notify.New(notify.Config{
		Token:       settings.BotToken,
		OwnerChatID: settings.OwnerChatID,
		APIEndpoint: settings.TelegramAPIEndpoint,
		Timeout:     settings.Notification.RequestTimeout,
		DedupWindow: settings.Notification.PanicDedup,
		MaxStack:    settings.Notification.MaxStackBytes,
	})
	if *worker {
		return runWorker(settings, notifier)
	}
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法确定当前二进制路径: %v\n", err)
		return supervisor.ExitUsage
	}
	return supervisor.New(executable, usedPath, settings, notifier).Run(context.Background())
}

func runWorker(settings config.Settings, notifier *notify.Telegram) (exitCode int) {
	config.Apply(settings)
	logger.InitLogger()
	logger.SetLogLevel(config.LogLevel)
	logBuildInfo()

	runID := os.Getenv("STICKERDOWNLOADER_RUN_ID")
	generation := os.Getenv("STICKERDOWNLOADER_GENERATION")
	startedAt := time.Now()
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error("worker 主流程 panic: %v", recovered)
			ctx, cancel := context.WithTimeout(context.Background(), settings.Notification.RequestTimeout)
			_ = notifier.SendPanic(ctx, "worker-main", recovered, nil, "Run: "+runID+"\nGeneration: "+generation)
			cancel()
			exitCode = supervisor.ExitCrash
		}
	}()

	if err := db.InitDB(); err != nil {
		logger.Error("数据库初始化失败: %s", err)
		return supervisor.ExitTemporary
	}
	if err := core.InitBot(); err != nil {
		logger.Error("Bot 初始化失败: %s", err)
		return supervisor.ExitTemporary
	}
	if err := handler.LoadTranslations(); err != nil {
		return supervisor.ExitConfig
	}
	handler.InitDownloaderPool()
	b := core.Bot.AddHandle()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fatal := make(chan error, 4)
	guard := &runtimeguard.Guard{
		Notifier:      notifier,
		Fatal:         fatal,
		RunID:         runID,
		Generation:    generation,
		StartTime:     startedAt,
		NotifyTimeout: settings.Notification.RequestTimeout,
	}
	runtimeguard.SetDefault(guard)
	defer runtimeguard.SetDefault(nil)

	registerHandlers(b, handler.MessageSender{}, guard)
	guard.Go("http-server", runtimeguard.Critical, func() {
		if err := api.RunHTTPServer(ctx); err != nil {
			panic(err)
		}
	})
	guard.Go("task-manager", runtimeguard.Critical, func() { task.TaskManager() })
	guard.Go("telegram-polling", runtimeguard.Critical, func() { b.Run() })

	utils.RuntimeStatus.StartTime = startedAt
	workerPID := os.Getpid()
	readyText := fmt.Sprintf("✅ StickerDownloader 已启动\nRun: %s\nGeneration: %s\nPID: %d\nVersion: %s\nCommit: %s", runID, generation, workerPID, version, commitHash)
	notifyCtx, cancelNotify := context.WithTimeout(context.Background(), settings.Notification.RequestTimeout)
	if err := notifier.SendEvent(notifyCtx, runID+":"+generation+":started", readyText); err != nil {
		logger.Warn("发送启动通知失败: %s", err)
	}
	cancelNotify()
	logger.Info("服务启动成功")

	select {
	case <-ctx.Done():
		logger.Info("收到停止信号")
		core.Bot.StopReceivingUpdates()
		return supervisor.ExitOK
	case err := <-fatal:
		logger.Error("关键组件退出: %s", err)
		return supervisor.ExitCrash
	}
}

func logBuildInfo() {
	if version == "" {
		version = gitOutput("describe", "--tags", "--abbrev=0", "--always")
	}
	if commitHash == "" {
		commitHash = gitOutput("rev-parse", "--short", "HEAD")
	}
	if buildTime == "" {
		buildTime = time.Now().Format(time.RFC3339)
	}

	logger.Info("版本号: %s", version)
	logger.Info("提交哈希: %s", commitHash)
	if t, err := time.Parse(time.RFC3339, buildTime); err == nil {
		logger.Info("构建时间: %s", t.Format("2006-01-02 15:04:05"))
	} else {
		logger.Info("构建时间: %s", buildTime)
	}
}

func gitOutput(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func registerHandlers(b *tgbotapi.Bot, ms handler.MessageSender, guard *runtimeguard.Guard) {
	wrap := func(name string, fn func(tgbotapi.Update) error) func(tgbotapi.Update) error {
		return func(u tgbotapi.Update) error {
			return guard.Wrap(name, func() error { return fn(u) })()
		}
	}

	b.NewProcessor(func(u tgbotapi.Update) bool {
		if !isDownloadableStickerMessage(u.Message) {
			return false
		}
		message.GenerateNewTaskMessage(u.Message.From.ID, u.Message.Chat.ID, u.Message.MessageID)
		return false
	}, nil)

	b.NewPrivateCommandProcessor("count", wrap("command-count", utils.SendRuntimeStatusInfo))
	b.NewPrivateCommandProcessor("help", wrap("command-help", ms.HelpMessage))
	b.NewPrivateCommandProcessor("start", wrap("command-start", ms.StartMessage))
	b.NewPrivateCommandProcessor("lang", wrap("command-lang", ms.LanguageChose))
	b.NewCallBackProcessor(lib.SingleDownload.String(), wrap("callback-single", ms.ThisFormatChose))
	b.NewCallBackProcessor("this", wrap("callback-this", ms.ThisFormatChose))
	b.NewCallBackProcessor("zip", wrap("callback-zip", ms.ZipFormatChose))
	b.NewCallBackProcessor(lib.SetDownload.String(), wrap("callback-set", ms.ZipFormatChose))
	b.NewCallBackProcessor("cancel", wrap("callback-cancel", ms.CancelDownload))

	formats := map[string]lib.TaskFileFormat{"webp": lib.WebpFormat, "png": lib.PngFormat, "jpeg": lib.JpegFormat}
	for key, format := range formats {
		b.NewCallBackProcessor(key, wrap("callback-"+key, func(u tgbotapi.Update) error { return ms.ThisSender(format, u) }))
		b.NewCallBackProcessor("zip_"+key, wrap("callback-zip-"+key, func(u tgbotapi.Update) error { return ms.ZipSender(format, u) }))
	}
	for _, language := range []string{"zh", "en", "jp"} {
		b.NewCallBackProcessor("lang_"+language, wrap("callback-lang-"+language, func(u tgbotapi.Update) error { return ms.ChangeUserLanguage(u, language) }))
	}
}

func isDownloadableStickerMessage(msg *tgbotapi.Message) bool {
	if msg == nil || msg.From == nil {
		return false
	}
	if msg.Sticker != nil {
		return true
	}
	return stickerPackLinkRegex.MatchString(msg.Text)
}
