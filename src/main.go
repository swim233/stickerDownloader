package main

import (
	"github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/utils"
	"github.com/swim233/StickerDownloader/utils/handler"
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
	b.NewCallBackProcessor("this", messageSender.FormatChose)
	b.NewCallBackProcessor("zip", messageSender.ZipSender)
	b.NewCallBackProcessor("webp", func(u tgbotapi.Update) error {
		err := messageSender.ThisSender("webp", u)
		return err
	})
	b.NewCallBackProcessor("png", func(u tgbotapi.Update) error {
		err := messageSender.ThisSender("png", u)
		return err
	})
	b.Run()
}
