package task

import (
	"context"

	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/logger"
)

func TaskHandler(task lib.Task, ctx context.Context) {
	switch task.TaskType {
	case lib.SingleDownload:
		{
			data, err := task.Downloader.DownloadSingleSticker(task.TaskUpdate)
			if err != nil {
				logger.Warn("error in downloading single sticker: %s", err.Error())
			}
			switch task.TargetFormat {
			case lib.PngFormat:
				{
					pngData, err := task.Converter.WebpToPNG(data)
					if err != nil {
						logger.Warn("error in converting webp to png: %s", err.Error())
						return
					}
					CompletedTaskHandler(&lib.CompletedTask{
						Task:     task,
						TaskData: pngData,
					})
					return
				}
			case lib.JpegFormat:
				{
					jpegData, err := task.Converter.WebpToJPEG(data, config.JpegQuality)
					if err != nil {
						logger.Warn("error when converting webp to jpeg: %s", err.Error())
						return
					}
					CompletedTaskHandler(&lib.CompletedTask{
						Task:     task,
						TaskData: jpegData,
					})
					return
				}
			default:
				{
					CompletedTaskHandler(&lib.CompletedTask{
						Task:     task,
						TaskData: data,
					})
				}
			}
		}
	}
}
