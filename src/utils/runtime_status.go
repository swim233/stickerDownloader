package utils

import (
	"fmt"
	"strconv"
	"time"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/lib"
)

var RuntimeStatus lib.RuntimeStatus

// 计数器
func SendRuntimeStatusInfo(u tgbotapi.Update) error {
	if u.Message.From != nil {
		return nil
	}
	chatID := u.Message.From.ID

	if RuntimeStatus.CacheHits != 0 {
		RuntimeStatus.HitPercentage = float64(RuntimeStatus.CacheHits) / (float64(RuntimeStatus.PackDownload) + float64(RuntimeStatus.HTTPPackDownload)) * 100
	}

	//运行时间计算
	timeString := func(duration time.Duration) string {
		var timeString string
		days := duration / (24 * time.Hour)
		if days > 0 {
			timeString += fmt.Sprintf("%d天", days)
		}
		hours := (duration - days*24*time.Hour) / time.Hour
		if days > 0 || hours > 0 {
			timeString += fmt.Sprintf("%d时", hours)
		}
		minutes := (duration - days*24*time.Hour - hours*time.Hour) / time.Minute
		if days > 0 || hours > 0 || minutes > 0 {
			timeString += fmt.Sprintf("%d分", minutes)
		}
		seconds := (duration - days*24*time.Hour - hours*time.Hour - minutes*time.Minute) / time.Second
		if days > 0 || hours > 0 || minutes > 0 || seconds > 0 {
			timeString += fmt.Sprintf("%d秒", seconds)
		}
		return timeString
	}(time.Since(RuntimeStatus.StartTime))

	msg := tgbotapi.NewMessage(chatID,
		"启动时间 : "+RuntimeStatus.StartTime.Format("2006-01-02 15:04:05")+"\n"+
			"本次运行时间 : "+timeString+"\n"+
			"机器人已下载贴纸总数 : "+strconv.Itoa(RuntimeStatus.SingleDownload)+"\n"+
			"机器人已下载贴纸包数 : "+strconv.Itoa(RuntimeStatus.PackDownload)+"\n"+
			"HTTP服务器已下载贴纸总数 : "+strconv.Itoa(RuntimeStatus.HTTPSingleDownload)+"\n"+
			"HTTP服务器已下载贴纸包数 : "+strconv.Itoa(RuntimeStatus.HTTPPackDownload)+"\n"+
			"缓存生效次数 : "+strconv.Itoa(RuntimeStatus.CacheHits)+"\n"+
			"缓存命中率 : "+strconv.FormatFloat(RuntimeStatus.HitPercentage, 'f', 1, 64)+"%\n"+
			"发生错误数 : "+strconv.Itoa(RuntimeStatus.Errors))
	core.Bot.Send(msg)
	return nil
}
