package handler

import (
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	utils "github.com/swim233/StickerDownloader/utils"
	"github.com/swim233/StickerDownloader/utils/logger"
)

type MessageSender struct {
}

func (m MessageSender) MessageSender(u tgbotapi.Update) error {
	dl := StickerDownloader{}
	chatID := u.Message.From.ID
	data, _, stickerSetName := dl.DownloadStickerSet(u)
	msg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Name: stickerSetName + ".zip", Bytes: data})
	_, err := utils.Bot.Send(msg)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	return nil
}
