package handler

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/swim233/StickerDownloader/cache"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/lib"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	db "github.com/swim233/StickerDownloader/db"
	utils "github.com/swim233/StickerDownloader/utils"
	logger "github.com/swim233/StickerDownloader/utils/logger"
)

type MessageSender struct {
}

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

// 发送按钮消息
func (m MessageSender) ButtonMessageSender(u tgbotapi.Update, sticker tgbotapi.StickerSet, allowDownloadSingleFile bool) error {
	chatID := u.Message.From.ID
	msg := tgbotapi.NewMessage(chatID,
		lib.TranslationsMap[db.GetUserLanguage(chatID)].CurrentStickerSet+" : "+sticker.Title+"\n"+
			lib.TranslationsMap[db.GetUserLanguage(chatID)].PickDownloadMethod)
	msg.ReplyToMessageID = u.Message.MessageID
	var buttons []tgbotapi.InlineKeyboardButton
	if allowDownloadSingleFile {
		button1 := tgbotapi.NewInlineKeyboardButtonData(lib.TranslationsMap[db.GetUserLanguage(chatID)].DownloadSingleSticker, "this")
		buttons = append(buttons, button1)
	}
	button2 := tgbotapi.NewInlineKeyboardButtonData(lib.TranslationsMap[db.GetUserLanguage(chatID)].DownloadStickerPack, "zip")
	buttons = append(buttons, button2)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons)
	core.Bot.Send(msg)
	return nil
}

// 单个贴纸下载
func (m MessageSender) ThisSender(format utils.Format, u tgbotapi.Update) error {
	ChatID := u.CallbackQuery.Message.Chat.ID
	UserID := u.CallbackQuery.Message.From.ID
	go func(u tgbotapi.Update) error {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("发生错误 从Panic中恢复")
				update, err := json.MarshalIndent(u, "", "  ")
				if err != nil {
					logger.Error("序列化 update 时出错: %v", err)
				} else {
					fmt.Println(string(update))
				}
				logger.Error("%s", update)
				utils.RuntimeStatus.Errors++
				//捕获错误
			}
		}()
		chatID := ChatID
		userID := UserID

		if userID != 0 {
			u.CallbackQuery.Answer(false, lib.TranslationsMap[db.GetUserLanguage(userID)].DownloadingSingleSticker)
		}

		// 早返回
		if format == utils.WebpFormat {
			msg := tgbotapi.NewDocument(chatID, tgbotapi.FileID(u.CallbackQuery.Message.ReplyToMessage.Sticker.FileID))
			msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID
			utils.RuntimeStatus.SingleDownload++
			// 这里的FileSize可能为0 如果需要精确审计可能不能使用早返回
			db.RecordUserData(u, int64(u.CallbackQuery.Message.ReplyToMessage.Sticker.FileSize), 1)
			core.Bot.Send(msg)
			u.CallbackQuery.Delete()
			return nil
		}

		downloaderPool := NewBlockingPool(core.BotConfig.MaxConcurrency)
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
			utils.RuntimeStatus.SingleDownload++
			core.Bot.Send(msg)

		} else {
			msg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Bytes: func(u tgbotapi.Update) []byte {
				webp, err := dl.DownloadFile(u)
				db.RecordUserData(u, int64(len(webp)), 1)
				switch format {
				case utils.JpegFormat:
					if err != nil {
						logger.Error("下载文件时出错 ：%s", err.Error())
					}
					fc := formatConverter{}
					jpeg, err := fc.convertWebPToJPEG(webp, core.BotConfig.WebPToJPEGQuality)
					if err != nil {
						logger.Error("下载文件时出错 ：%s", err.Error())
					}
					return jpeg
				case utils.PngFormat:
					if err != nil {
						logger.Error("下载文件时出错 ：%s", err.Error())
					}
					fc := formatConverter{}
					png, err := fc.convertWebPToPNG(webp)
					if err != nil {
						logger.Error("下载文件时出错 ：%s", err.Error())
					}
					return png
				// 在上面早返回已经被处理了 但是留着以防万一
				case utils.WebpFormat:
					return webp
				default:
					logger.Warn("未实现的格式: %v, 作为webp处理", format)
					return webp
				}
			}(u), Name: func(u tgbotapi.Update) string { //贴纸包名字判空
				if u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName == "" {
					return "sticker"
				} else {
					return u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName
				}
			}(u) + "." + format.String()})
			downloaderPool.Put(dl)
			msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID
			utils.RuntimeStatus.SingleDownload++
			core.Bot.Send(msg)

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
	editedMsg := tgbotapi.NewEditMessageText(chatID, editMsgID, lib.TranslationsMap[db.GetUserLanguage(userID)].PickDownloadFormat)
	WebPButton := tgbotapi.NewInlineKeyboardButtonData("WebP", "webp")
	PNGButton := tgbotapi.NewInlineKeyboardButtonData("PNG", "png")
	JPEGButton := tgbotapi.NewInlineKeyboardButtonData("JPEG", "jpeg")
	CancelButton := tgbotapi.NewInlineKeyboardButtonData(lib.TranslationsMap[db.GetUserLanguage(userID)].Cancel, "cancel")
	editButton := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{WebPButton, PNGButton, JPEGButton}, {CancelButton}}}
	editedMsg.ReplyMarkup = &editButton
	core.Bot.Send(editedMsg)
	return nil
}

// 打包格式选择
func (m MessageSender) ZipFormatChose(u tgbotapi.Update) error {
	editMsgID := u.CallbackQuery.Message.MessageID
	chatID := u.CallbackQuery.Message.Chat.ID
	userID := u.CallbackQuery.Message.ReplyToMessage.From.ID
	editedMsg := tgbotapi.NewEditMessageText(chatID, editMsgID, lib.TranslationsMap[db.GetUserLanguage(userID)].PickDownloadFormat)
	WebPButton := tgbotapi.NewInlineKeyboardButtonData("WebP", "zip_webp")
	PNGButton := tgbotapi.NewInlineKeyboardButtonData("PNG", "zip_png")
	JPEGButton := tgbotapi.NewInlineKeyboardButtonData("JPEG", "zip_jpeg")
	CancelButton := tgbotapi.NewInlineKeyboardButtonData(lib.TranslationsMap[db.GetUserLanguage(userID)].Cancel, "cancel")
	editButton := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{WebPButton, PNGButton, JPEGButton}, {CancelButton}}}
	editedMsg.ReplyMarkup = &editButton
	core.Bot.Send(editedMsg)
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
	core.Bot.Send(msg)
	return nil
}

// 语言修改
func (m MessageSender) ChangeUserLanguage(u tgbotapi.Update, lang string) error {
	userID := u.CallbackQuery.Message.ReplyToMessage.From.ID
	err := db.ChangeUserLanguage(userID, lang)
	if err != nil {
		logger.Error("修改语言时出错 ：%s", err)
		return err
	}
	editMsg := tgbotapi.NewEditMessageText(u.CallbackQuery.Message.ReplyToMessage.From.ID, u.CallbackQuery.Message.MessageID, lib.TranslationsMap[db.GetUserLanguage(userID)].SuccessChangeLanguage)
	core.Bot.Send(editMsg)
	return nil
}

// 贴纸集下载
func (m MessageSender) ZipSender(fmt utils.Format, u tgbotapi.Update) error {
	go func(u tgbotapi.Update) error {
		var requestFile tgbotapi.RequestFileData
		var fileSize int64
		chatID := u.CallbackQuery.Message.Chat.ID
		userID := u.CallbackQuery.Message.ReplyToMessage.From.ID

		u.CallbackQuery.Answer(false, lib.TranslationsMap[db.GetUserLanguage(userID)].DownloadingStickerSet) //贴纸下载中

		stickerSet, err := core.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: getStickerSet(u)}) //获取贴纸包
		if err != nil {
			logger.Error("获取贴纸集时出错 ：%s", err)
		}

		fileID, fileSize, stickerNum, err := cache.GetCacheFileID(stickerSet, fmt)
		if err == nil && fileID != "" && !(fileSize == 0 || stickerNum == 0) { //判定缓存 如果数据库中贴纸数量和大小存在问题 强制刷新
			requestFile = tgbotapi.FileID(fileID)
			utils.RuntimeStatus.PackDownload++
			utils.RuntimeStatus.CacheHits++
			db.RecordUserData(u, fileSize, stickerNum)
			logger.Info("缓存命中")
		} else {

			processingMsg := tgbotapi.EditMessageTextConfig{Text: "贴纸包下载中 请稍等... \nDownloading... ", BaseEdit: tgbotapi.BaseEdit{ChatID: chatID, MessageID: u.CallbackQuery.Message.MessageID}}
			core.Bot.Send(processingMsg)                                     //TODO 进度汇报
			downloaderPool := NewBlockingPool(core.BotConfig.MaxConcurrency) //获取下载线程
			dl := downloaderPool.Get()
			data, stickerSetTitle, stickerNum, err := dl.DownloadStickerSet(fmt, stickerSet, u) //下载贴纸数据
			fileSize = int64(len(data))
			if err != nil {
				logger.Error("下载贴纸时出错 ：%s", err)
			}
			if fileSize == 0 {
				msg := tgbotapi.NewMessage(chatID, lib.TranslationsMap[db.GetUserLanguage(userID)].StickerSetIsNull) //贴纸包为空
				msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID                              //回复消息
				core.Bot.Send(msg)
				u.CallbackQuery.Delete()
				return nil
			} //贴纸包判空
			db.RecordUserData(u, int64(len(data)), stickerNum) //记录数据库
			requestFile = tgbotapi.FileBytes{Name: stickerSetTitle + ".zip", Bytes: data}
		}

		msg := tgbotapi.NewDocument(chatID, requestFile)
		msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID
		utils.RuntimeStatus.PackDownload++
		message, err := core.Bot.Send(msg)
		if err == nil {
			switch fmt { //为数据库添加数据
			case utils.WebpFormat:
				{
					db.RecordStickerData(stickerSet, userID, message.Document.FileID, fileSize, "", 0, "", 0)
				}
			case utils.PngFormat:
				{
					db.RecordStickerData(stickerSet, userID, "", 0, message.Document.FileID, fileSize, "", 0)
				}
			case utils.JpegFormat:
				{
					db.RecordStickerData(stickerSet, userID, "", 0, "", 0, message.Document.FileID, fileSize)
				}
			default:
				//TODO 默认处理
			}

		} else {
			logger.Error("为数据库添加贴纸数据时出错 ：%s", err.Error())
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
	_, err := core.Bot.Request(deleteMsg)
	if err != nil {

		logger.Error("取消操作时出错 ： %s", err.Error())
		return err
	}
	return err
}

// 发送欢迎和帮助消息
func (m MessageSender) HelpMessage(u tgbotapi.Update) error {
	chatID := u.Message.Chat.ID
	msg := tgbotapi.NewMessage(chatID, "您好！请将您喜欢的贴纸发送给我 我可以帮您下载单个贴纸或整个贴纸包，并支持转换为多种格式！如需切换语言，请发送指令：/lang \n\n"+
		"Hi there!  Just send me the stickers you want I can download individual stickers or entire sticker packs, and convert them into various formats for you!To switch the language, just type /lang ")
	core.Bot.Send(msg)
	return nil
}
func (m MessageSender) StartMessage(u tgbotapi.Update) error {
	err := m.LanguageChose(u)
	if err != nil {
		logger.Error("发送开始消息时出错 ：%s", err)
	}
	m.HelpMessage(u)
	return db.InitUserData(u)
}

// 加载翻译
func LoadTranslations() error {
	data, err := os.ReadFile("locales.json")
	if err != nil {
		logger.Error("加载翻译文件时出错 ：%s", err.Error())
		return err
	}
	return json.Unmarshal(data, &lib.TranslationsMap)
}
