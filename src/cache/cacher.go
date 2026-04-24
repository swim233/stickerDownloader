package cache

import (
	"errors"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/db"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/utils"
)

// GetCacheFileID checks if a cached version of the sticker set exists in the given format.
// Returns the Telegram file ID, file size, sticker count, and any error.
func GetCacheFileID(set tgbotapi.StickerSet, format lib.TaskFileFormat) (fileID string, fileSize int64, stickerNum int, err error) {
	stickerData, err := db.GetStickerData(set.Name)
	if err != nil {
		return "", 0, 0, err
	}

	// Verify hash matches (sticker set hasn't changed)
	currentHash := utils.CalculateStickerSet(set)
	if currentHash != stickerData.SetHash {
		return "", 0, 0, errors.New("哈希不匹配，贴纸包已更新")
	}

	switch format {
	case lib.WebpFormat:
		return stickerData.WebpFileID, stickerData.WebpFileSize, stickerData.StickerNum, nil
	case lib.PngFormat:
		return stickerData.PNGFileID, stickerData.PNGFileSize, stickerData.StickerNum, nil
	case lib.JpegFormat:
		return stickerData.JPEGFileID, stickerData.JPEGFileSize, stickerData.StickerNum, nil
	default:
		return "", 0, 0, errors.New("不支持的格式")
	}
}
