package lib

type FormatConverter interface {
	WebpToPNG(src []byte) (dist []byte, err error)
	WebpToJPEG(src []byte, quality int) (dist []byte, err error)
}
type Downloader interface {
	DownloadSticker(target string) (data []byte, err error)
}
type Task struct {
	TaskID        string
	TaskChatID    int64
	TaskMessageID int64
	TaskType      TaskType
	FileFormat    FileFormat
	Converter     FormatConverter
	Downloader    Downloader
}

type TaskType string
type FileFormat string

const (
	SingleDownload TaskType = "single"
	SetDownload    TaskType = "set"
)

const (
	JpegFormat FileFormat = "jpeg"
	PngFormat  FileFormat = "png"
	WebpFormat FileFormat = "webp"
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
func (f FileFormat) String() string {
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
