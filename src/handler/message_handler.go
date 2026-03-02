package handler

import (
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/stickerDownloader/logger"
	"github.com/swim233/stickerDownloader/message"
)

func MessageWithStickerHandler(b *tgbotapi.Bot) {

	b.NewProcessor(func(u tgbotapi.Update) bool {

		return u.Message != nil && u.Message.Sticker != nil

	}, func(u tgbotapi.Update) error {

		message.SendDownloadMethodSelectMessage(u)
		logger.Debug("get a sticker message")
		return nil

	})
}
