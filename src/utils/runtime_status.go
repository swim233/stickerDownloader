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

// RuntimeStatusReport is the JSON shape served by the WebUI status API.
type RuntimeStatusReport struct {
	StartTime          time.Time `json:"start_time"`
	UptimeSeconds      int64     `json:"uptime_seconds"`
	SingleDownload     int64     `json:"single_download"`
	PackDownload       int64     `json:"pack_download"`
	HTTPSingleDownload int64     `json:"http_single_download"`
	HTTPPackDownload   int64     `json:"http_pack_download"`
	CacheHits          int64     `json:"cache_hits"`
	CacheHitRate       float64   `json:"cache_hit_rate"`
	TotalErrors        int64     `json:"total_errors"`
	PanicErrors        int64     `json:"panic_errors"`
	RequestErrors      int64     `json:"request_errors"`
	DownloadErrors     int64     `json:"download_errors"`
	NotificationErrors int64     `json:"notification_errors"`
}

// CurrentRuntimeStatusReport snapshots the runtime counters for the WebUI.
func CurrentRuntimeStatusReport(now time.Time) RuntimeStatusReport {
	status := currentRuntimeStatus()
	var hitRate float64
	if totalPacks := status.packDownload + status.httpPackDownload; totalPacks > 0 {
		hitRate = float64(status.cacheHits) / float64(totalPacks) * 100
	}
	return RuntimeStatusReport{
		StartTime:          status.startTime,
		UptimeSeconds:      int64(now.Sub(status.startTime).Seconds()),
		SingleDownload:     status.singleDownload,
		PackDownload:       status.packDownload,
		HTTPSingleDownload: status.httpSingleDownload,
		HTTPPackDownload:   status.httpPackDownload,
		CacheHits:          status.cacheHits,
		CacheHitRate:       hitRate,
		TotalErrors:        status.panicErrors + status.requestErrors + status.downloadErrors + status.notificationErrors,
		PanicErrors:        status.panicErrors,
		RequestErrors:      status.requestErrors,
		DownloadErrors:     status.downloadErrors,
		NotificationErrors: status.notificationErrors,
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
