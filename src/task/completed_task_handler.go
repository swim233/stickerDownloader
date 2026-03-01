package task

import (
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/logger"
)

func CompletedTaskHandler(cTask *lib.CompletedTask) {
	msg := tgbotapi.NewDocument(cTask.TaskChatID, tgbotapi.FileBytes{
		Name:  cTask.StickerSet.Title,
		Bytes: cTask.TaskData,
	})
	msg.ReplyToMessageID = cTask.TaskMessageID
	_, err := core.Bot.Send(msg)
	if err != nil {
		logger.Warn("error when sending msg: %s", err.Error())
		return
	}
}
