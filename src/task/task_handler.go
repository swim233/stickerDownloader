package task

import (
	"context"

	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/logger"
)

// TaskHandler processes a download task based on its type and target format.
func TaskHandler(task lib.Task, ctx context.Context) {
	switch task.TaskType {
	case lib.SingleDownload:
		data, err := task.Downloader.DownloadSingleSticker(task.TaskUpdate)
		if err != nil {
			logger.Warn("下载单个贴纸出错: %s", err)
			return
		}

		var result []byte
		switch task.TargetFormat {
		case lib.PngFormat:
			result, err = task.Converter.WebpToPNG(data)
			if err != nil {
				logger.Warn("WebP 转 PNG 出错: %s", err)
				return
			}
		case lib.JpegFormat:
			result, err = task.Converter.WebpToJPEG(data, config.JpegQuality)
			if err != nil {
				logger.Warn("WebP 转 JPEG 出错: %s", err)
				return
			}
		default:
			result = data
		}

		CompletedTaskHandler(&lib.CompletedTask{
			Task:     task,
			TaskData: result,
		})
	}
}
