package lib

import tgbotapi "github.com/ijnkawakaze/telegram-bot-api"

type FormatConverter interface {
	WebpToPNG(src []byte) (dist []byte, err error)
	WebpToJPEG(src []byte, quality int) (dist []byte, err error)
}
type Downloader interface {
	DownloadSingleSticker(update tgbotapi.Update) (data []byte, err error)
	DownloadStickerSet(update tgbotapi.Update) (data []byte, err error)
}

type Task struct {
	TaskID        string
	TaskChatID    int64
	TaskMessageID int
	TaskType      TaskType
	TaskUpdate    tgbotapi.Update
	TargetFormat  TaskFileFormat
	Converter     FormatConverter
	Downloader    Downloader
	StickerSet    tgbotapi.StickerSet
}

type CompletedTask struct {
	Task
	TaskData []byte
}

type TaskType string
type TaskFileFormat string

const (
	SingleDownload TaskType = "single"
	SetDownload    TaskType = "set"
)

const (
	JpegFormat TaskFileFormat = "jpeg"
	PngFormat  TaskFileFormat = "png"
	WebpFormat TaskFileFormat = "webp"
)

func (t TaskType) String() string {
	switch t {
	case SingleDownload:
		return "single"
	case SetDownload:
		return "set"
	default:
		return string(t)
	}

}
func (f TaskFileFormat) String() string {
	switch f {
	case JpegFormat:
		return "jpeg"
	case PngFormat:
		return "png"
	case WebpFormat:
		return "webp"
	default:
		return string(f)
	}
}
