package cache

import (
	"errors"

	"github.com/swim233/StickerDownloader/utils/db"
	"github.com/swim233/StickerDownloader/utils/hashCalculator"
)

func GetCacheFileID(setName string, format string) (string, error) {
	stickerData, err := db.GetStickerData(setName)
	if err != nil {
		return "", err
	}
	if format == "webp" && hashCalculator.CalculateHashViaSetName(setName) == stickerData.SetHash {
		return stickerData.WebpFileID, nil
	}
	if format == "png" && hashCalculator.CalculateHashViaSetName(setName) == stickerData.SetHash {
		return stickerData.PNGFileID, nil
	}
	if format == "jpeg" && hashCalculator.CalculateHashViaSetName(setName) == stickerData.SetHash {
		return stickerData.JPEGFileID, nil
	}
	return "", errors.New("哈希不匹配或发生其他错误")
}
