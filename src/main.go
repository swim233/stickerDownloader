package main

import (
	"os"
	"os/exec"
	"strings"
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
	"github.com/swim233/StickerDownloader/task"
	"github.com/swim233/StickerDownloader/utils"
)

var (
	version    string
	commitHash string
	buildTime  string
)

func main() {
	// 1. Initialize logger
	logger.InitLogger()
	logBuildInfo()

	// 2. Load configuration
	config.InitConfig()
	logger.SetLogLevel(config.LogLevel)

	// 3. Initialize database
	db.InitDB()

	// 4. Initialize bot
	core.InitBot()
	b := core.Bot.AddHandle()

	// 5. Initialize downloader pool
	handler.InitDownloaderPool()

	// 6. Start HTTP server
	go api.StartHTTPServer()

	// 7. Load translations
	if err := handler.LoadTranslations(); err != nil {
		logger.Error("加载 i18n 文件出错: %s", err)
		os.Exit(1)
	}

	// 8. Start task manager
	go task.TaskManager()
	logger.Info("任务管理器启动成功")

	// 9. Register handlers
	ms := handler.MessageSender{}
	registerHandlers(b, ms)

	// 10. Start bot
	utils.RuntimeStatus.StartTime = time.Now()
	b.Run()
}

func logBuildInfo() {
	// Fallback: read from git if ldflags were not set
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

func registerHandlers(b *tgbotapi.Bot, ms handler.MessageSender) {
	// Message processor: generate task message for incoming stickers
	b.NewProcessor(func(u tgbotapi.Update) bool {
		if u.Message == nil || u.Message.From == nil {
			return false
		}
		message.GenerateNewTaskMessage(u.Message.From.ID, u.Message.Chat.ID, u.Message.MessageID)
		return false
	}, nil)

	// Private commands
	b.NewPrivateCommandProcessor("count", utils.SendRuntimeStatusInfo)
	b.NewPrivateCommandProcessor("help", ms.HelpMessage)
	b.NewPrivateCommandProcessor("start", ms.StartMessage)
	b.NewPrivateCommandProcessor("lang", ms.LanguageChose)

	// Single sticker download → show format chooser first
	b.NewCallBackProcessor(lib.SingleDownload.String(), ms.ThisFormatChose)

	// Format choosers
	b.NewCallBackProcessor("this", ms.ThisFormatChose)
	b.NewCallBackProcessor("zip", ms.ZipFormatChose)
	b.NewCallBackProcessor(lib.SetDownload.String(), ms.ZipFormatChose) // "set" button from GenerateNewTaskMessage
	b.NewCallBackProcessor("cancel", ms.CancelDownload)

	// Single sticker format callbacks
	formats := map[string]lib.TaskFileFormat{
		"webp": lib.WebpFormat,
		"png":  lib.PngFormat,
		"jpeg": lib.JpegFormat,
	}
	for key, fmt := range formats {
		fmt := fmt // capture
		b.NewCallBackProcessor(key, func(u tgbotapi.Update) error {
			return ms.ThisSender(fmt, u)
		})
		b.NewCallBackProcessor("zip_"+key, func(u tgbotapi.Update) error {
			return ms.ZipSender(fmt, u)
		})
	}

	// Language callbacks
	languages := []string{"zh", "en", "jp"}
	for _, lang := range languages {
		lang := lang // capture
		b.NewCallBackProcessor("lang_"+lang, func(u tgbotapi.Update) error {
			return ms.ChangeUserLanguage(u, lang)
		})
	}
}
