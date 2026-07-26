package utils

import (
	"fmt"
	"time"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/logger"
)

var RuntimeStatus lib.RuntimeStatus

type runtimeStatusSnapshot struct {
	startTime          time.Time
	singleDownload     int64
	packDownload       int64
	httpSingleDownload int64
	httpPackDownload   int64
	cacheHits          int64
	panicErrors        int64
	requestErrors      int64
	downloadErrors     int64
	notificationErrors int64
}

func currentRuntimeStatus() runtimeStatusSnapshot {
	return runtimeStatusSnapshot{
		startTime:          RuntimeStatus.StartTime,
		singleDownload:     RuntimeStatus.SingleDownload.Load(),
		packDownload:       RuntimeStatus.PackDownload.Load(),
		httpSingleDownload: RuntimeStatus.HTTPSingleDownload.Load(),
		httpPackDownload:   RuntimeStatus.HTTPPackDownload.Load(),
		cacheHits:          RuntimeStatus.CacheHits.Load(),
		panicErrors:        RuntimeStatus.PanicErrors.Load(),
		requestErrors:      RuntimeStatus.RequestErrors.Load(),
		downloadErrors:     RuntimeStatus.DownloadErrors.Load(),
		notificationErrors: RuntimeStatus.NotificationErrors.Load(),
	}
}

func formatRuntimeStatus(status runtimeStatusSnapshot, now time.Time) string {
	var hitPercentage float64
	totalPacks := status.packDownload + status.httpPackDownload
	if totalPacks > 0 {
		hitPercentage = float64(status.cacheHits) / float64(totalPacks) * 100
	}

	totalErrors := status.panicErrors + status.requestErrors + status.downloadErrors + status.notificationErrors
	return fmt.Sprintf(
		"启动时间 : %s\n"+
			"本次运行时间 : %s\n"+
			"机器人已下载贴纸总数 : %d\n"+
			"机器人已下载贴纸包数 : %d\n"+
			"HTTP服务器已下载贴纸总数 : %d\n"+
			"HTTP服务器已下载贴纸包数 : %d\n"+
			"缓存生效次数 : %d\n"+
			"缓存命中率 : %.1f%%\n"+
			"当前 Worker 错误总数 : %d\n"+
			"  Panic : %d\n"+
			"  过期/请求 : %d\n"+
			"  下载 : %d\n"+
			"  通知失败 : %d",
		status.startTime.Format("2006-01-02 15:04:05"),
		formatDuration(now.Sub(status.startTime)),
		status.singleDownload, status.packDownload, status.httpSingleDownload, status.httpPackDownload,
		status.cacheHits, hitPercentage, totalErrors,
		status.panicErrors, status.requestErrors, status.downloadErrors, status.notificationErrors,
	)
}

// SendRuntimeStatusInfo sends runtime statistics to the requesting user.
func SendRuntimeStatusInfo(u tgbotapi.Update) error {
	if u.Message == nil || u.Message.From == nil {
		return nil
	}

	msg := tgbotapi.NewMessage(u.Message.From.ID, formatRuntimeStatus(currentRuntimeStatus(), time.Now()))
	if _, err := core.Bot.Send(msg); err != nil {
		RuntimeStatus.RecordError(lib.RuntimeErrorNotification)
		logger.Warn("发送运行状态消息失败: %s", err)
		return fmt.Errorf("发送运行状态消息: %w", err)
	}
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
