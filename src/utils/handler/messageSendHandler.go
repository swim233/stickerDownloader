package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/swim233/StickerDownloader/utils/cache"
	"github.com/swim233/StickerDownloader/utils/hashCalculator"

	"time"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	utils "github.com/swim233/StickerDownloader/utils"
	"github.com/swim233/StickerDownloader/utils/db"
	logger "github.com/swim233/StickerDownloader/utils/logger"
)

type MessageSender struct {
}

type Translations struct {
	CurrentStickerSet        string `json:"CurrentStickerSet"`
	PickDownloadMethod       string `json:"PickDownloadMethod"`
	DownloadSingleSticker    string `json:"DownloadSingleSticker"`
	DownloadStickerPack      string `json:"DownloadStickerPack"`
	DownloadingSingleSticker string `json:"DownloadingSingleSticker"`
	PickDownloadFormat       string `json:"PickDownloadFormat"`
	DownloadingStickerSet    string `json:"DownloadingStickerSet"`
	StickerSetIsNull         string `json:"StickerSetIsNull"`
	Help                     string `json:"Help"`
	Cancel                   string `json:"Cancel"`
	SuccessChangeLanguage    string `json:"SuccessChangeLanguage"`
}

var translations map[string]Translations

// 新建对象池
type BlockingPool struct {
	pool chan *StickerDownloader
}

func NewBlockingPool(size int) *BlockingPool {
	p := &BlockingPool{
		pool: make(chan *StickerDownloader, size),
	}
	for i := 0; i < size; i++ {
		p.pool <- &StickerDownloader{ID: i}
	}
	return p
}

// 从池中拿一个对象，如果没有就阻塞等待
func (p *BlockingPool) Get() *StickerDownloader {
	return <-p.pool
}

// 归还对象到池中，如果池满了也会阻塞等待
func (p *BlockingPool) Put(d *StickerDownloader) {
	p.pool <- d
}

type DownloadCounter struct {
	Single        int
	Pack          int
	HTTPSingle    int
	HTTPPack      int
	Error         int
	CacheHit      int
	HitPercentage float64
}

var StartTime time.Time

var downloadCounter DownloadCounter

// 计数器
func (m MessageSender) CountSender(u tgbotapi.Update) error {
	chatID := u.Message.From.ID
	if downloadCounter.CacheHit != 0 {
		downloadCounter.HitPercentage = float64(downloadCounter.CacheHit) / (float64(downloadCounter.Pack) + float64(downloadCounter.HTTPPack)) * 100
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
			"缓存生效次数 : "+strconv.Itoa(downloadCounter.CacheHit)+"\n"+
			"缓存命中率 : "+strconv.FormatFloat(downloadCounter.HitPercentage, 'f', 1, 64)+"%\n"+
			"发生错误数 : "+strconv.Itoa(downloadCounter.Error))
	utils.Bot.Send(msg)
	return nil
}

// 发送按钮消息
func (m MessageSender) ButtonMessageSender(u tgbotapi.Update, sticker tgbotapi.StickerSet, allowDownloadSingleFile bool) error {
	chatID := u.Message.From.ID
	msg := tgbotapi.NewMessage(chatID,
		translations[db.GetUserLanguage(chatID)].CurrentStickerSet+" : "+sticker.Title+"\n"+
			translations[db.GetUserLanguage(chatID)].PickDownloadMethod)
	msg.ReplyToMessageID = u.Message.MessageID
	var buttons []tgbotapi.InlineKeyboardButton
	if allowDownloadSingleFile {
		button1 := tgbotapi.NewInlineKeyboardButtonData(translations[db.GetUserLanguage(chatID)].DownloadSingleSticker, "this")
		buttons = append(buttons, button1)
	}
	button2 := tgbotapi.NewInlineKeyboardButtonData(translations[db.GetUserLanguage(chatID)].DownloadStickerPack, "zip")
	buttons = append(buttons, button2)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons)
	utils.Bot.Send(msg)
	return nil
}

// 单个贴纸下载
func (m MessageSender) ThisSender(fmt string, u tgbotapi.Update) error {
	go func(u tgbotapi.Update) error {
		chatID := u.CallbackQuery.Message.Chat.ID
		userID := u.CallbackQuery.Message.From.ID

		u.CallbackQuery.Answer(false, translations[db.GetUserLanguage(userID)].DownloadingSingleSticker)

		downloaderPool := NewBlockingPool(utils.BotConfig.MaxConcurrency)
		dl := downloaderPool.Get()

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
	chatID := u.CallbackQuery.Message.Chat.ID
	userID := u.CallbackQuery.Message.ReplyToMessage.From.ID
	editedMsg := tgbotapi.NewEditMessageText(chatID, editMsgID, translations[db.GetUserLanguage(userID)].PickDownloadFormat)
	WebPButton := tgbotapi.NewInlineKeyboardButtonData("WebP", "webp")
	PNGButton := tgbotapi.NewInlineKeyboardButtonData("PNG", "png")
	JPEGButton := tgbotapi.NewInlineKeyboardButtonData("JPEG", "jpeg")
	CancelButton := tgbotapi.NewInlineKeyboardButtonData(translations[db.GetUserLanguage(userID)].Cancel, "cancel")
	editButton := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{WebPButton, PNGButton, JPEGButton}, {CancelButton}}}
	editedMsg.ReplyMarkup = &editButton
	utils.Bot.Send(editedMsg)
	return nil
}

// 打包格式选择
func (m MessageSender) ZipFormatChose(u tgbotapi.Update) error {
	editMsgID := u.CallbackQuery.Message.MessageID
	chatID := u.CallbackQuery.Message.Chat.ID
	userID := u.CallbackQuery.Message.ReplyToMessage.From.ID
	editedMsg := tgbotapi.NewEditMessageText(chatID, editMsgID, translations[db.GetUserLanguage(userID)].PickDownloadFormat)
	WebPButton := tgbotapi.NewInlineKeyboardButtonData("WebP", "zip_webp")
	PNGButton := tgbotapi.NewInlineKeyboardButtonData("PNG", "zip_png")
	JPEGButton := tgbotapi.NewInlineKeyboardButtonData("JPEG", "zip_jpeg")
	CancelButton := tgbotapi.NewInlineKeyboardButtonData(translations[db.GetUserLanguage(userID)].Cancel, "cancel")
	editButton := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{WebPButton, PNGButton, JPEGButton}, {CancelButton}}}
	editedMsg.ReplyMarkup = &editButton
	utils.Bot.Send(editedMsg)
	return nil
}

// 语言选择
func (m MessageSender) LanguageChose(u tgbotapi.Update) error {
	ChatID := u.Message.Chat.ID
	CNButton := tgbotapi.NewInlineKeyboardButtonData("🇨🇳 中文", "lang_zh")
	ENButton := tgbotapi.NewInlineKeyboardButtonData("🇺🇸 English", "lang_en")
	JPButton := tgbotapi.NewInlineKeyboardButtonData("🇯🇵 Japanese", "lang_jp")
	CancelButton := tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel")
	msgButton := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{CNButton, ENButton, JPButton}, {CancelButton}}}
	msg := tgbotapi.NewMessage(ChatID, "请选择语言 \nPlease select Language")
	msg.ReplyMarkup = msgButton
	msg.ReplyToMessageID = u.Message.MessageID
	utils.Bot.Send(msg)
	return nil
}

// 语言修改
func (m MessageSender) ChangeUserLanguage(u tgbotapi.Update, lang string) error {
	userID := u.CallbackQuery.Message.ReplyToMessage.From.ID
	err := db.ChangeUserLanguage(userID, lang)
	if err != nil {
		logger.Error("%s", err)
		return err
	}
	editMsg := tgbotapi.NewEditMessageText(u.CallbackQuery.Message.ReplyToMessage.From.ID, u.CallbackQuery.Message.MessageID, translations[db.GetUserLanguage(userID)].SuccessChangeLanguage)
	utils.Bot.Send(editMsg)
	return nil
}

// 贴纸集下载
func (m MessageSender) ZipSender(fmt string, u tgbotapi.Update) error {
	go func(u tgbotapi.Update) error {
		var requestFile tgbotapi.RequestFileData
		var fileSize int64
		chatID := u.CallbackQuery.Message.Chat.ID
		userID := u.CallbackQuery.Message.ReplyToMessage.From.ID

		u.CallbackQuery.Answer(false, translations[db.GetUserLanguage(userID)].DownloadingStickerSet) //贴纸下载中

		stickerSet, err := utils.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: getStickerSet(u)}) //获取贴纸包
		if err != nil {
			logger.Error("%s", err)
		}

		fileID, fileSize, stickerNum, err := cache.GetCacheFileID(stickerSet, fmt)
		if err == nil && fileID != "" && !(fileSize == 0 || stickerNum == 0) { //判定缓存 如果数据库中贴纸数量和大小存在问题 强制刷新
			requestFile = tgbotapi.FileID(fileID)
			downloadCounter.CacheHit++
			db.RecordUserData(u, fileSize, stickerNum)
			logger.Info("缓存命中")
			logger.Error("size:%d,num:%d", fileSize, stickerNum) //FIXME
		} else {

			processingMsg := tgbotapi.EditMessageTextConfig{Text: "贴纸包下载中 请稍等... \nDownloading... ", BaseEdit: tgbotapi.BaseEdit{ChatID: chatID, MessageID: u.CallbackQuery.Message.MessageID}}
			utils.Bot.Send(processingMsg)                                     //TODO 进度汇报
			downloaderPool := NewBlockingPool(utils.BotConfig.MaxConcurrency) //获取下载线程
			dl := downloaderPool.Get()
			data, stickerSetTitle, stickerNum, err := dl.DownloadStickerSet(fmt, stickerSet, u) //下载贴纸数据
			fileSize = int64(len(data))
			if err != nil {
				logger.Error("%s", err)
			}
			if fileSize == 0 {
				msg := tgbotapi.NewMessage(chatID, translations[db.GetUserLanguage(userID)].StickerSetIsNull) //贴纸包为空
				msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID                       //回复消息
				utils.Bot.Send(msg)
				u.CallbackQuery.Delete()
				return nil
			} //贴纸包判空
			db.RecordUserData(u, int64(len(data)), stickerNum) //记录数据库
			requestFile = tgbotapi.FileBytes{Name: stickerSetTitle + ".zip", Bytes: data}
		}

		msg := tgbotapi.NewDocument(chatID, requestFile)
		msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID
		downloadCounter.Pack++
		message, err := utils.Bot.Send(msg)
		if err == nil {
			switch fmt { //为数据库添加数据
			case "webp":
				{
					db.RecordStickerData(stickerSet, userID, message.Document.FileID, fileSize, "", 0, "", 0)
				}
			case "png":
				{
					db.RecordStickerData(stickerSet, userID, "", 0, message.Document.FileID, fileSize, "", 0)
				}
			case "jpeg":
				{
					db.RecordStickerData(stickerSet, userID, "", 0, "", 0, message.Document.FileID, fileSize)
				}
			default:
				//TODO 默认处理
			}

		} //发送消息

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
func (m MessageSender) HelpMessage(u tgbotapi.Update) error {
	chatID := u.Message.Chat.ID
	msg := tgbotapi.NewMessage(chatID, "请将贴纸发送给我 我可以下载单个贴纸和贴纸包 并转换成不同的格式 你可以使用 /lang 来切换语言\n\n"+
		"Please send me the stickers. I can download individual stickers and sticker packs, and convert them into different formats. You can use /lang to switch the language.")
	utils.Bot.Send(msg)
	return nil
}
func (m MessageSender) StartMessage(u tgbotapi.Update) error {
	err := m.LanguageChose(u)
	if err != nil {
		logger.Error("%s", err)
	}
	m.HelpMessage(u)
	return db.InitUserData(u)
}

// 加载翻译
func LoadTranslations() error {
	data, err := os.ReadFile("locales.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &translations)
}
