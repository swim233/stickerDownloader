package handler

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	utils "github.com/swim233/StickerDownloader/utils"
	"github.com/swim233/StickerDownloader/utils/db"
	logger "github.com/swim233/StickerDownloader/utils/logger"
)

type MessageSender struct {
}

// 新建线程池
var downloaderPool = sync.Pool{
	New: func() any {
		return &StickerDownloader{}
	},
}

type DownloadCounter struct {
	Single        int
	Pack          int
	HTTPSingle    int
	HTTPPack      int
	Error         int
	Cache         int
	HitPercentage float64
}

var StartTime time.Time

var downloadCounter DownloadCounter

// 计数器
func (m MessageSender) CountSender(u tgbotapi.Update) error {
	chatID := u.Message.From.ID
	if downloadCounter.Cache != 0 {
		downloadCounter.HitPercentage = float64(downloadCounter.Cache) / (float64(downloadCounter.Pack) + float64(downloadCounter.HTTPPack)) * 100
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
	}(time.Since(StartTime))

	msg := tgbotapi.NewMessage(chatID,
		"启动时间 : "+StartTime.Format("2006-01-02 15:04:05")+"\n"+
			"本次运行时间 : "+timeString+"\n"+
			"机器人已下载贴纸总数 : "+strconv.Itoa(downloadCounter.Single)+"\n"+
			"机器人已下载贴纸包数 : "+strconv.Itoa(downloadCounter.Pack)+"\n"+
			"HTTP服务器已下载贴纸总数 : "+strconv.Itoa(downloadCounter.HTTPSingle)+"\n"+
			"HTTP服务器已下载贴纸包数 : "+strconv.Itoa(downloadCounter.HTTPPack)+"\n"+
			"缓存生效次数 : "+strconv.Itoa(downloadCounter.Cache)+"\n"+
			"缓存命中率 : "+strconv.FormatFloat(downloadCounter.HitPercentage, 'f', 1, 64)+"%\n"+
			"发生错误数 : "+strconv.Itoa(downloadCounter.Error))
	utils.Bot.Send(msg)
	return nil
}

// 发送按钮消息
func (m MessageSender) ButtonMessageSender(u tgbotapi.Update, sticker tgbotapi.StickerSet, allowDownloadSingleFile bool) error {
	chatID := u.Message.From.ID
	msg := tgbotapi.NewMessage(chatID,
		"当前贴纸包 : "+sticker.Title+"\n"+
			"请选择要下载的方式")
	msg.ReplyToMessageID = u.Message.MessageID
	var buttons []tgbotapi.InlineKeyboardButton
	if allowDownloadSingleFile {
		button1 := tgbotapi.NewInlineKeyboardButtonData("下载单个图片", "this")
		buttons = append(buttons, button1)
	}
	button2 := tgbotapi.NewInlineKeyboardButtonData("下载贴纸包", "zip")
	buttons = append(buttons, button2)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons)
	utils.Bot.Send(msg)
	return nil
}

// 单个贴纸下载
func (m MessageSender) ThisSender(fmt string, u tgbotapi.Update) error {
	go func(u tgbotapi.Update) error {
		chatID := u.CallbackQuery.Message.Chat.ID
		u.CallbackQuery.Answer(false, "正在下载单个图片")
		dl := downloaderPool.Get().(*StickerDownloader)

		if u.CallbackQuery.Message.ReplyToMessage.Sticker.IsVideo { //判断是否webm贴纸
			msg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Bytes: func(u tgbotapi.Update) []byte {
				data, _ := dl.DownloadFile(u)
				db.RecordUserData(u, int64(len(data)), 1)
				return data
			}(u), Name: func(u tgbotapi.Update) string { //贴纸包名字判空
				if u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName == "" {
					return "sticker"
				} else {
					return u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName
				}
			}(u) + ".webm"})
			msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID
			downloadCounter.Single++
			utils.Bot.Send(msg)

		} else {
			msg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Bytes: func(u tgbotapi.Update) []byte {
				webp, err := dl.DownloadFile(u)
				db.RecordUserData(u, int64(len(webp)), 1)
				if fmt == "webp" {
					return webp
				} else if fmt == "jpeg" {
					if err != nil {
						logger.Error("%s", err.Error())
					}
					fc := formatConverter{} //转换格式
					jpeg, err := fc.convertWebPToJPEG(webp, utils.BotConfig.WebPToJPEGQuality)
					if err != nil {
						logger.Error("%s", err.Error())
					}
					return jpeg
				} else {
					if err != nil {
						logger.Error("%s", err.Error())
					}
					fc := formatConverter{} //转换格式
					png, err := fc.convertWebPToPNG(webp)
					if err != nil {
						logger.Error("%s", err.Error())
					}
					return png

				}
			}(u), Name: func(u tgbotapi.Update) string { //贴纸包名字判空
				if u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName == "" {
					return "sticker"
				} else {
					return u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName
				}
			}(u) + "." + fmt})
			downloaderPool.Put(dl)
			msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID
			downloadCounter.Single++
			utils.Bot.Send(msg)

		}
		//删除回调消息
		u.CallbackQuery.Delete()
		return nil
	}(u)
	return nil
}

// 格式选择
func (m MessageSender) ThisFormatChose(u tgbotapi.Update) error {
	editMsgID := u.CallbackQuery.Message.MessageID
	ChatID := u.CallbackQuery.Message.Chat.ID
	editedMsg := tgbotapi.NewEditMessageText(ChatID, editMsgID, "请选择要下载的格式")
	WebPButton := tgbotapi.NewInlineKeyboardButtonData("WebP", "webp")
	PNGButton := tgbotapi.NewInlineKeyboardButtonData("PNG", "png")
	JPEGButton := tgbotapi.NewInlineKeyboardButtonData("JPEG", "jpeg")
	CancelButton := tgbotapi.NewInlineKeyboardButtonData("取消", "cancel")
	editButton := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{WebPButton, PNGButton, JPEGButton}, {CancelButton}}}
	editedMsg.ReplyMarkup = &editButton
	utils.Bot.Send(editedMsg)
	return nil
}

// 打包格式选择
func (m MessageSender) ZipFormatChose(u tgbotapi.Update) error {
	editMsgID := u.CallbackQuery.Message.MessageID
	ChatID := u.CallbackQuery.Message.Chat.ID
	editedMsg := tgbotapi.NewEditMessageText(ChatID, editMsgID, "请选择要下载的格式")
	WebPButton := tgbotapi.NewInlineKeyboardButtonData("WebP", "zip_webp")
	PNGButton := tgbotapi.NewInlineKeyboardButtonData("PNG", "zip_png")
	JPEGButton := tgbotapi.NewInlineKeyboardButtonData("JPEG", "zip_jpeg")
	CancelButton := tgbotapi.NewInlineKeyboardButtonData("取消", "cancel")
	editButton := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{WebPButton, PNGButton, JPEGButton}, {CancelButton}}}
	editedMsg.ReplyMarkup = &editButton
	utils.Bot.Send(editedMsg)
	return nil
}

// 贴纸集下载
func (m MessageSender) ZipSender(fmt string, u tgbotapi.Update) error {
	go func(u tgbotapi.Update) error {
		chatID := u.CallbackQuery.Message.Chat.ID
		u.CallbackQuery.Answer(false, "正在下载贴纸包")
		processingMsg := tgbotapi.EditMessageTextConfig{Text: "贴纸包下载中 请稍等... \nDownloading... ", BaseEdit: tgbotapi.BaseEdit{ChatID: chatID, MessageID: u.CallbackQuery.Message.MessageID}}
		utils.Bot.Send(processingMsg) //TODO 进度汇报
		dl := downloaderPool.Get().(*StickerDownloader)
		data, stickerSetTitle, stickerNum, _ := dl.DownloadStickerSet(fmt, u)

		//贴纸包判空
		if len(data) == 0 {

			msg := tgbotapi.NewMessage(chatID, "贴纸包为空！")
			msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID
			utils.Bot.Send(msg)
			u.CallbackQuery.Delete()
			return nil

		}
		db.RecordUserData(u, int64(len(data)), stickerNum) //记录数据库
		db.RecordStickerData(getStickerSet(u), stickerSetTitle, u.CallbackQuery.From.ID)
		msg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Name: stickerSetTitle + ".zip", Bytes: data})
		msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID
		downloadCounter.Pack++
		utils.Bot.Send(msg) //发送消息
		u.CallbackQuery.Delete()
		return nil
	}(u)
	return nil
}

// 取消
func (m MessageSender) CancelDownload(u tgbotapi.Update) error {

	chatID := u.CallbackQuery.Message.Chat.ID
	messageID := u.CallbackQuery.Message.ReplyToMessage.MessageID
	u.CallbackQuery.Delete()

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, err := utils.Bot.Request(deleteMsg)
	if err != nil {

		logger.Error("%s", err.Error())
		return err
	}
	return err
}

// 发送欢迎和帮助消息
func (m MessageSender) StartMessage(u tgbotapi.Update) error {
	chatID := u.Message.Chat.ID
	msg := tgbotapi.NewMessage(chatID, "请将贴纸发送给我 我可以下载单个贴纸和贴纸包 并转换成不同的格式")
	utils.Bot.Send(msg)
	return nil
}
