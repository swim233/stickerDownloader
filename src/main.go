package main

import (
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/utils"
	"github.com/swim233/StickerDownloader/utils/handler"
	httpserver "github.com/swim233/StickerDownloader/utils/httpServer"
	"time"
)

func main() {
	utils.InitBot()
	utils.Bot.Debug = true
	b := utils.Bot.AddHandle()
	messageSender := handler.MessageSender{}
	b.NewProcessor(func(u tgbotapi.Update) bool {
		if u.Message != nil {
			return u.Message.Sticker != nil
		}
		return false
	}, messageSender.ButtonMessageSender)
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
