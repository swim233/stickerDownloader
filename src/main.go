package main

import (
	"os"
	"regexp"
	"time"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/api"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/db"
	"github.com/swim233/StickerDownloader/handler"
	"github.com/swim233/StickerDownloader/utils"
	"github.com/swim233/StickerDownloader/utils/logger"
)

var (
	version    string
	commitHash string
	buildTime  string
)

func main() {
	logger.Info("版本号: %s", version)
	logger.Info("提交哈希: %s", commitHash)
	parse, err := time.Parse(time.RFC3339, buildTime)
	if err != nil {
		logger.Info("构建时间: %s", buildTime)
	} else {
		logger.Info("构建时间: %s", parse.Format("2006-01-02 15:04:05"))
	}
	db.InitDB()                              //初始化数据库
	core.InitBot()                           //初始化bot配置
	b := core.Bot.AddHandle()                //注册handler
	messageSender := handler.MessageSender{} //实例化handler
	go api.StartHTTPServer()                 //开启http服务器
	err = handler.LoadTranslations()         //加载i18n文件
	if err != nil {
		logger.Error("加载i18文件时出错 : %s", err.Error())
		os.Exit(1)
	}

	var stickerLinkRegex = regexp.MustCompile(`https://t.me/addstickers/([a-zA-Z0-9_]+)`)
	b.NewProcessor(func(u tgbotapi.Update) bool {
		if u.Message == nil {
			return false
		}
		if u.Message.Sticker != nil {
			// 如果是 sticker，直接传递 sticker 的 set name
			sticker, err := core.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: func(u tgbotapi.Update) string {
				return u.Message.Sticker.SetName
			}(u)})
			if err != nil {
				return false
			}
			// 支持下载单个贴纸的用true
			messageSender.ButtonMessageSender(u, sticker, true)
			return true
		}
		if u.Message.Text != "" && stickerLinkRegex.MatchString(u.Message.Text) {
			// 提取 sticker set name
			matches := stickerLinkRegex.FindStringSubmatch(u.Message.Text)
			if len(matches) > 1 {
				stickerSetName := matches[1] // 提取的 SetName
				sticker, err := core.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: stickerSetName})
				if err != nil {
					return false
				}
				messageSender.ButtonMessageSender(u, sticker, false)
				return true
			}
		}
		return false
	}, nil)
	b.NewPrivateCommandProcessor("count", messageSender.CountSender)
	b.NewPrivateCommandProcessor("help", messageSender.HelpMessage)
	b.NewPrivateCommandProcessor("start", messageSender.StartMessage)
	b.NewPrivateCommandProcessor("lang", messageSender.LanguageChose)
	b.NewCallBackProcessor("this", messageSender.ThisFormatChose)
	b.NewCallBackProcessor("zip", messageSender.ZipFormatChose)
	b.NewCallBackProcessor("cancel", messageSender.CancelDownload)
	b.NewCallBackProcessor("webp", func(u tgbotapi.Update) error {
		return messageSender.ThisSender(utils.WebpFormat, u)

	})
	b.NewCallBackProcessor("png", func(u tgbotapi.Update) error {
		return messageSender.ThisSender(utils.PngFormat, u)

	})
	b.NewCallBackProcessor("jpeg", func(u tgbotapi.Update) error {
		return messageSender.ThisSender(utils.JpegFormat, u)

	})
	b.NewCallBackProcessor("zip_webp", func(u tgbotapi.Update) error {
		return messageSender.ZipSender(utils.WebpFormat, u)

	})
	b.NewCallBackProcessor("zip_png", func(u tgbotapi.Update) error {
		return messageSender.ZipSender(utils.PngFormat, u)

	})
	b.NewCallBackProcessor("zip_jpeg", func(u tgbotapi.Update) error {
		return messageSender.ZipSender(utils.JpegFormat, u)
	})
	b.NewCallBackProcessor("lang_zh", func(u tgbotapi.Update) error {
		return messageSender.ChangeUserLanguage(u, "zh")
	})
	b.NewCallBackProcessor("lang_en", func(u tgbotapi.Update) error {
		return messageSender.ChangeUserLanguage(u, "en")
	})
	b.NewCallBackProcessor("lang_jp", func(u tgbotapi.Update) error {
		return messageSender.ChangeUserLanguage(u, "jp")
	})
	handler.StartTime = time.Now()
	b.Run()
}
