package handler

import (
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/stickerDownloader/download"
	"github.com/swim233/stickerDownloader/lib"
	"github.com/swim233/stickerDownloader/logger"
	"github.com/swim233/stickerDownloader/message"
)

func MessageWithStickerHandler(b *tgbotapi.Bot) {

	b.NewProcessor(func(u tgbotapi.Update) bool {

		return u.Message != nil && u.Message.Sticker != nil

	}, func(u tgbotapi.Update) error {

		download.OperateDownloadRequest(u.Message.Chat.ID, u.Message.MessageID, &lib.DownloadRequest{
			ChatID:            u.Message.Chat.ID,
			MessageID:         u.Message.MessageID,
			CallbackMessageID: 0,
			DownloadType:      "",
			DownloadFormat:    "",
		})

		message.SendDownloadMethodSelectMessage(u)
		logger.Debug("get a sticker message")
		return nil

	})
}
