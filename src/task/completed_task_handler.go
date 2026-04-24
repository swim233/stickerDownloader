package task

import (
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/logger"
)

func CompletedTaskHandler(cTask *lib.CompletedTask) {
	// Build filename: use sticker set name or fallback to "sticker"
	name := cTask.StickerSet.Title
	if name == "" {
		name = "sticker"
	}
	// Add file extension based on target format
	name += "." + cTask.TargetFormat.String()

	msg := tgbotapi.NewDocument(cTask.TaskChatID, tgbotapi.FileBytes{
		Name:  name,
		Bytes: cTask.TaskData,
	})
	msg.ReplyToMessageID = cTask.TaskMessageID
	_, err := core.Bot.Send(msg)
	if err != nil {
		logger.Warn("发送文件出错: %s", err)
	}

	// Delete the bot's format chooser message
	if cTask.TaskUpdate.CallbackQuery != nil {
		cTask.TaskUpdate.CallbackQuery.Delete()
	}
}
