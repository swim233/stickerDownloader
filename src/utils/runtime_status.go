package utils

import (
	"fmt"
	"time"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/lib"
)

var RuntimeStatus lib.RuntimeStatus

// SendRuntimeStatusInfo sends runtime statistics to the requesting user.
func SendRuntimeStatusInfo(u tgbotapi.Update) error {
	// Bug fix: was `!= nil` which returned early when user exists
	if u.Message.From == nil {
		return nil
	}
	chatID := u.Message.From.ID

	singleDL := RuntimeStatus.SingleDownload.Load()
	packDL := RuntimeStatus.PackDownload.Load()
	httpSingleDL := RuntimeStatus.HTTPSingleDownload.Load()
	httpPackDL := RuntimeStatus.HTTPPackDownload.Load()
	cacheHits := RuntimeStatus.CacheHits.Load()
	errors := RuntimeStatus.Errors.Load()

	var hitPercentage float64
	totalPacks := packDL + httpPackDL
	if totalPacks > 0 {
		hitPercentage = float64(cacheHits) / float64(totalPacks) * 100
	}

	duration := time.Since(RuntimeStatus.StartTime)
	timeStr := formatDuration(duration)

	text := fmt.Sprintf(
		"启动时间 : %s\n"+
			"本次运行时间 : %s\n"+
			"机器人已下载贴纸总数 : %d\n"+
			"机器人已下载贴纸包数 : %d\n"+
			"HTTP服务器已下载贴纸总数 : %d\n"+
			"HTTP服务器已下载贴纸包数 : %d\n"+
			"缓存生效次数 : %d\n"+
			"缓存命中率 : %.1f%%\n"+
			"发生错误数 : %d",
		RuntimeStatus.StartTime.Format("2006-01-02 15:04:05"),
		timeStr, singleDL, packDL, httpSingleDL, httpPackDL,
		cacheHits, hitPercentage, errors,
	)

	msg := tgbotapi.NewMessage(chatID, text)
	core.Bot.Send(msg)
	return nil
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	var result string
	if days > 0 {
		result += fmt.Sprintf("%d天", days)
	}
	if days > 0 || hours > 0 {
		result += fmt.Sprintf("%d时", hours)
	}
	if days > 0 || hours > 0 || minutes > 0 {
		result += fmt.Sprintf("%d分", minutes)
	}
	result += fmt.Sprintf("%d秒", seconds)
	return result
}
