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
	b.NewCallBackProcessor("this", messageSender.ThisSender)
	b.NewCallBackProcessor("zip", messageSender.ZipSender)
	b.Run()
}
