package task

import (
	"context"

	"github.com/google/uuid"
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/utils"
)

var TaskChan = make(chan lib.Task, 32)

func TaskManager() {
	for {
		task := <-TaskChan
		go TaskHandler(task, context.Background())
	}
}

func NewTask(update tgbotapi.Update, taskType lib.TaskType, taskFormat lib.TaskFileFormat) {
	var chatID int64
	var messageID int
	if update.CallbackQuery != nil {
		chatID = update.CallbackQuery.Message.Chat.ID
		// Reply to the user's original sticker message, not the bot's button message
		if update.CallbackQuery.Message.ReplyToMessage != nil {
			messageID = update.CallbackQuery.Message.ReplyToMessage.MessageID
		} else {
			messageID = update.CallbackQuery.Message.MessageID
		}
	} else if update.Message != nil {
		chatID = update.Message.Chat.ID
		messageID = update.Message.MessageID
	}

	converter := utils.FormatConverter{}
	downloader := FileDownloader{}
	task := lib.Task{
		TaskID:        uuid.NewString(),
		TaskChatID:    chatID,
		TaskMessageID: messageID,
		TaskType:      taskType,
		TaskUpdate:    update,
		TargetFormat:  taskFormat,
		Converter:     converter,
		Downloader:    &downloader,
		StickerSet:    tgbotapi.StickerSet{},
	}
	TaskChan <- task
}
