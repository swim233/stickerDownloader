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
	converter := utils.FormatConverter{}
	downloader := FileDownloader{}
	task := lib.Task{
		TaskID:        uuid.NewString(),
		TaskChatID:    update.Message.Chat.ID,
		TaskMessageID: update.Message.MessageID,
		TaskType:      taskType,
		TaskUpdate:    update,
		TargetFormat:  taskFormat,
		Converter:     converter,
		Downloader:    &downloader,
		StickerSet:    tgbotapi.StickerSet{},
	}
	TaskChan <- task
}
