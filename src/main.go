package main

import (
	"regexp"
	"time"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/utils"
	"github.com/swim233/StickerDownloader/utils/db"
	"github.com/swim233/StickerDownloader/utils/handler"
	httpserver "github.com/swim233/StickerDownloader/utils/httpServer"
)

func main() {
	db.InitDB()
	utils.InitBot()
	utils.Bot.Debug = true
	b := utils.Bot.AddHandle()
	messageSender := handler.MessageSender{}
	var stickerLinkRegex = regexp.MustCompile(`https://t.me/addstickers/([a-zA-Z0-9_]+)`)
	b.NewProcessor(func(u tgbotapi.Update) bool {
		if u.Message == nil {
			return false
		}
		if u.Message.Sticker != nil {
			// 如果是 sticker，直接传递 sticker 的 set name
			sticker, err := utils.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: func(u tgbotapi.Update) string {
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
				sticker, err := utils.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: stickerSetName})
				if err != nil {
					return false
				}
				messageSender.ButtonMessageSender(u, sticker, false)
				return true
			}
		}
		return false
	}, nil)
	b.NewCommandProcessor("count", messageSender.CountSender)
	b.NewCommandProcessor("start", messageSender.StartMessage)
	b.NewCommandProcessor("help", messageSender.StartMessage)
	b.NewCallBackProcessor("this", messageSender.ThisFormatChose)
	b.NewCallBackProcessor("zip", messageSender.ZipFormatChose)
	b.NewCallBackProcessor("cancel", messageSender.CancelDownload)
	go httpserver.StartHTTPServer()
	b.NewCallBackProcessor("webp", func(u tgbotapi.Update) error {
		err := messageSender.ThisSender("webp", u)
		return err
	})
	b.NewCallBackProcessor("png", func(u tgbotapi.Update) error {
		err := messageSender.ThisSender("png", u)
		return err
	})
	b.NewCallBackProcessor("jpeg", func(u tgbotapi.Update) error {
		err := messageSender.ThisSender("jpeg", u)
		return err
	})
	b.NewCallBackProcessor("zip_webp", func(u tgbotapi.Update) error {
		err := messageSender.ZipSender("webp", u)
		return err
	})
	b.NewCallBackProcessor("zip_png", func(u tgbotapi.Update) error {
		err := messageSender.ZipSender("png", u)
		return err
	})
	b.NewCallBackProcessor("zip_jpeg", func(u tgbotapi.Update) error {
		err := messageSender.ZipSender("jpeg", u)
		return err
	})
	handler.StartTime = time.Now()
	b.Run()
}
