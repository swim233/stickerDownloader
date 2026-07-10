package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/cache"
	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/core"
	db "github.com/swim233/StickerDownloader/db"
	"github.com/swim233/StickerDownloader/lib"
	logger "github.com/swim233/StickerDownloader/logger"
	utils "github.com/swim233/StickerDownloader/utils"
)

// MessageSender handles sending messages to Telegram users.
type MessageSender struct{}

// BlockingPool is a bounded pool of StickerDownloader instances.
type BlockingPool struct {
	pool chan *StickerDownloader
}

// downloaderPool is the shared global pool (Phase 5.1).
var downloaderPool *BlockingPool

// InitDownloaderPool creates the shared downloader pool. Must be called after config is loaded.
func InitDownloaderPool() {
	downloaderPool = &BlockingPool{
		pool: make(chan *StickerDownloader, config.MaxConcurrency),
	}
	for i := 0; i < config.MaxConcurrency; i++ {
		downloaderPool.pool <- &StickerDownloader{ID: i}
	}
}

func (p *BlockingPool) Get() *StickerDownloader { return <-p.pool }
func (p *BlockingPool) Put(d *StickerDownloader) { p.pool <- d }

// tr is a helper to get translated text for a user.
func tr(userID int64) lib.Translations {
	return lib.TranslationsMap[db.GetUserLanguage(userID)]
}

// ButtonMessageSender sends the initial download method choice.
func (m MessageSender) ButtonMessageSender(u tgbotapi.Update, sticker tgbotapi.StickerSet, allowDownloadSingleFile bool) error {
	chatID := u.Message.From.ID
	t := tr(chatID)
	msg := tgbotapi.NewMessage(chatID, t.CurrentStickerSet+" : "+sticker.Title+"\n"+t.PickDownloadMethod)
	msg.ReplyToMessageID = u.Message.MessageID

	var buttons []tgbotapi.InlineKeyboardButton
	if allowDownloadSingleFile {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(t.DownloadSingleSticker, "this"))
	}
	buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(t.DownloadStickerPack, "zip"))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons)
	core.Bot.Send(msg)
	return nil
}

// ThisSender downloads a single sticker in the requested format.
func (m MessageSender) ThisSender(format lib.TaskFileFormat, u tgbotapi.Update) error {
	chatID := u.CallbackQuery.Message.Chat.ID
	userID := u.CallbackQuery.From.ID

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic 恢复: %v", r)
				update, err := json.MarshalIndent(u, "", "  ")
				if err == nil {
					logger.Error("update: %s", string(update))
				}
				utils.RuntimeStatus.Errors.Add(1)
			}
		}()

		if userID != 0 {
			u.CallbackQuery.Answer(false, tr(userID).DownloadingSingleSticker)
		}

		replyMsg := u.CallbackQuery.Message.ReplyToMessage
		replyMsgID := replyMsg.MessageID

		// Fast path: webp format, just forward the file
		if format == lib.WebpFormat {
			msg := tgbotapi.NewDocument(chatID, tgbotapi.FileID(replyMsg.Sticker.FileID))
			msg.ReplyToMessageID = replyMsgID
			utils.RuntimeStatus.SingleDownload.Add(1)
			db.RecordUserData(u, int64(replyMsg.Sticker.FileSize), 1)
			core.Bot.Send(msg)
			u.CallbackQuery.Delete()
			return
		}

		dl := downloaderPool.Get()
		defer downloaderPool.Put(dl)

		stickerName := replyMsg.Sticker.SetName
		if stickerName == "" {
			stickerName = "sticker"
		}

		if replyMsg.Sticker.IsVideo {
			data, _ := dl.DownloadFile(u)
			db.RecordUserData(u, int64(len(data)), 1)
			msg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
				Bytes: data,
				Name:  stickerName + ".webm",
			})
			msg.ReplyToMessageID = replyMsgID
			utils.RuntimeStatus.SingleDownload.Add(1)
			core.Bot.Send(msg)
		} else {
			webp, err := dl.DownloadFile(u)
			if err != nil {
				logger.Error("下载文件出错: %s", err)
				return
			}
			db.RecordUserData(u, int64(len(webp)), 1)

			data := convertStickerData(webp, format)
			msg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
				Bytes: data,
				Name:  stickerName + "." + format.String(),
			})
			msg.ReplyToMessageID = replyMsgID
			utils.RuntimeStatus.SingleDownload.Add(1)
			core.Bot.Send(msg)
		}
		u.CallbackQuery.Delete()
	}()
	return nil
}

// convertStickerData converts webp data to the target format.
func convertStickerData(webp []byte, format lib.TaskFileFormat) []byte {
	fc := utils.FormatConverter{}
	switch format {
	case lib.JpegFormat:
		data, err := fc.WebpToJPEG(webp, config.JpegQuality)
		if err != nil {
			logger.Error("转换 JPEG 出错: %s", err)
			return webp
		}
		return data
	case lib.PngFormat:
		data, err := fc.WebpToPNG(webp)
		if err != nil {
			logger.Error("转换 PNG 出错: %s", err)
			return webp
		}
		return data
	default:
		return webp
	}
}

// formatChooser is the unified implementation for ThisFormatChose and ZipFormatChose (Phase 4.3).
func (m MessageSender) formatChooser(u tgbotapi.Update, callbackPrefix string) error {
	editMsgID := u.CallbackQuery.Message.MessageID
	chatID := u.CallbackQuery.Message.Chat.ID
	userID := u.CallbackQuery.Message.ReplyToMessage.From.ID
	t := tr(userID)

	prefix := ""
	if callbackPrefix != "" {
		prefix = callbackPrefix + "_"
	}

	editedMsg := tgbotapi.NewEditMessageText(chatID, editMsgID, t.PickDownloadFormat)
	editButton := tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonData("WebP", prefix+"webp"),
				tgbotapi.NewInlineKeyboardButtonData("PNG", prefix+"png"),
				tgbotapi.NewInlineKeyboardButtonData("JPEG", prefix+"jpeg"),
			},
			{tgbotapi.NewInlineKeyboardButtonData(t.Cancel, "cancel")},
		},
	}
	editedMsg.ReplyMarkup = &editButton
	core.Bot.Send(editedMsg)
	return nil
}

func (m MessageSender) ThisFormatChose(u tgbotapi.Update) error {
	return m.formatChooser(u, "")
}

func (m MessageSender) ZipFormatChose(u tgbotapi.Update) error {
	return m.formatChooser(u, "zip")
}

// LanguageChose sends the language selection message.
func (m MessageSender) LanguageChose(u tgbotapi.Update) error {
	chatID := u.Message.Chat.ID
	msg := tgbotapi.NewMessage(chatID, "请选择语言 \nPlease select Language")
	msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonData("🇨🇳 中文", "lang_zh"),
				tgbotapi.NewInlineKeyboardButtonData("🇺🇸 English", "lang_en"),
				tgbotapi.NewInlineKeyboardButtonData("🇯🇵 Japanese", "lang_jp"),
			},
			{tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel")},
		},
	}
	msg.ReplyToMessageID = u.Message.MessageID
	core.Bot.Send(msg)
	return nil
}

// ChangeUserLanguage updates the user's language preference.
func (m MessageSender) ChangeUserLanguage(u tgbotapi.Update, lang string) error {
	userID := u.CallbackQuery.Message.ReplyToMessage.From.ID
	if err := db.ChangeUserLanguage(userID, lang); err != nil {
		logger.Error("修改语言出错: %s", err)
		return err
	}
	editMsg := tgbotapi.NewEditMessageText(
		u.CallbackQuery.Message.ReplyToMessage.From.ID,
		u.CallbackQuery.Message.MessageID,
		tr(userID).SuccessChangeLanguage,
	)
	core.Bot.Send(editMsg)
	return nil
}

// ZipSender downloads a full sticker pack as a zip file.
func (m MessageSender) ZipSender(format lib.TaskFileFormat, u tgbotapi.Update) error {
	if u.CallbackQuery == nil || u.CallbackQuery.Message == nil || u.CallbackQuery.Message.Chat == nil {
		logger.Warn("无法下载贴纸包：回调消息上下文不完整")
		utils.RuntimeStatus.Errors.Add(1)
		return nil
	}

	callback := u.CallbackQuery
	callbackMsg := callback.Message
	chatID := callbackMsg.Chat.ID
	userID := chatID
	if callback.From != nil {
		userID = callback.From.ID
	}
	if callbackMsg.ReplyToMessage == nil {
		logger.Warn("无法下载贴纸包：原始消息上下文已失效 (chat_id=%d, message_id=%d, user_id=%d)", chatID, callbackMsg.MessageID, userID)
		utils.RuntimeStatus.Errors.Add(1)
		if _, err := callback.Answer(true, "操作已失效，请重新发送贴纸或贴纸包链接。\nThis action has expired. Please send the sticker or sticker pack link again."); err != nil {
			logger.Warn("响应失效回调出错: %s", err)
		}
		return nil
	}

	replyMsg := callbackMsg.ReplyToMessage
	replyMsgID := replyMsg.MessageID

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("下载贴纸包 panic 恢复: %v", r)
				utils.RuntimeStatus.Errors.Add(1)
			}
		}()

		callback.Answer(false, tr(userID).DownloadingStickerSet)

		stickerSetName := GetStickerSetName(u)
		if stickerSetName == "" {
			logger.Warn("无法下载贴纸包：原始消息中没有贴纸包信息 (chat_id=%d, message_id=%d, user_id=%d)", chatID, callbackMsg.MessageID, userID)
			utils.RuntimeStatus.Errors.Add(1)
			return
		}

		stickerSet, err := core.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: stickerSetName})
		if err != nil {
			logger.Error("获取贴纸集出错: %s", err)
			return
		}

		var requestFile tgbotapi.RequestFileData
		var fileSize int64
		var stickerNum int

		// Check cache
		fileID, cachedSize, cachedNum, err := cache.GetCacheFileID(stickerSet, format)
		if err == nil && fileID != "" && cachedSize > 0 && cachedNum > 0 {
			requestFile = tgbotapi.FileID(fileID)
			fileSize = cachedSize
			stickerNum = cachedNum
			utils.RuntimeStatus.CacheHits.Add(1)
			logger.Info("缓存命中: %s", stickerSet.Name)
		} else {
			// Download
			processingMsg := tgbotapi.EditMessageTextConfig{
				Text:     "贴纸包下载中 请稍等... \nDownloading...",
				BaseEdit: tgbotapi.BaseEdit{ChatID: chatID, MessageID: u.CallbackQuery.Message.MessageID},
			}
			core.Bot.Send(processingMsg)

			dl := downloaderPool.Get()
			defer downloaderPool.Put(dl)
			// Progress callback: edit the message with a progress bar every 3s
			msgID := u.CallbackQuery.Message.MessageID
			onProgress := func(completed, total int) {
				pct := 0
				if total > 0 {
					pct = completed * 100 / total
				}
				bar := buildProgressBar(completed, total, 20)
				text := fmt.Sprintf("📦 下载贴纸包中 / Downloading...\n\n%s\n%d/%d (%d%%)", bar, completed, total, pct)
				edit := tgbotapi.EditMessageTextConfig{
					Text:     text,
					BaseEdit: tgbotapi.BaseEdit{ChatID: chatID, MessageID: msgID},
				}
				core.Bot.Send(edit)
			}
			data, title, num, err := dl.DownloadStickerSet(format, stickerSet, u, onProgress)

			if err != nil {
				logger.Error("下载贴纸出错: %s", err)
				return
			}
			fileSize = int64(len(data))
			stickerNum = num
			if fileSize == 0 {
				msg := tgbotapi.NewMessage(chatID, tr(userID).StickerSetIsNull)
				msg.ReplyToMessageID = replyMsgID
				core.Bot.Send(msg)
				u.CallbackQuery.Delete()
				return
			}
			requestFile = tgbotapi.FileBytes{Name: title + ".zip", Bytes: data}
		}

		db.RecordUserData(u, fileSize, stickerNum)

		msg := tgbotapi.NewDocument(chatID, requestFile)
		msg.ReplyToMessageID = replyMsgID
		utils.RuntimeStatus.PackDownload.Add(1)

		message, err := core.Bot.Send(msg)
		if err != nil {
			logger.Error("发送贴纸包出错: %s", err)
		} else {
			// Record cache data
			recordStickerCache(stickerSet, userID, format, message.Document.FileID, fileSize)
		}

		u.CallbackQuery.Delete()
	}()
	return nil
}

// recordStickerCache records the file ID for caching based on format.
func recordStickerCache(set tgbotapi.StickerSet, userID int64, format lib.TaskFileFormat, fileID string, fileSize int64) {
	switch format {
	case lib.WebpFormat:
		db.RecordStickerData(set, userID, fileID, fileSize, "", 0, "", 0)
	case lib.PngFormat:
		db.RecordStickerData(set, userID, "", 0, fileID, fileSize, "", 0)
	case lib.JpegFormat:
		db.RecordStickerData(set, userID, "", 0, "", 0, fileID, fileSize)
	}
}

// CancelDownload cancels a pending download operation.
func (m MessageSender) CancelDownload(u tgbotapi.Update) error {
	chatID := u.CallbackQuery.Message.Chat.ID
	messageID := u.CallbackQuery.Message.ReplyToMessage.MessageID
	u.CallbackQuery.Delete()

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	if _, err := core.Bot.Request(deleteMsg); err != nil {
		logger.Error("取消操作出错: %s", err)
		return err
	}
	return nil
}

// HelpMessage sends the help/welcome text.
func (m MessageSender) HelpMessage(u tgbotapi.Update) error {
	chatID := u.Message.Chat.ID
	msg := tgbotapi.NewMessage(chatID,
		"您好！请将您喜欢的贴纸发送给我，我可以帮您下载单个贴纸或整个贴纸包，并支持转换为多种格式！如需切换语言，请发送指令：/lang\n\n"+
			"Hi there! Just send me the stickers you want. I can download individual stickers or entire sticker packs, and convert them into various formats for you! To switch the language, just type /lang")
	core.Bot.Send(msg)
	return nil
}

// StartMessage sends the welcome flow (language chooser + help).
func (m MessageSender) StartMessage(u tgbotapi.Update) error {
	if err := m.LanguageChose(u); err != nil {
		logger.Error("发送开始消息出错: %s", err)
	}
	m.HelpMessage(u)
	return db.InitUserData(u)
}

// LoadTranslations loads the i18n translations from locales.json.
func LoadTranslations() error {
	data, err := os.ReadFile("locales.json")
	if err != nil {
		logger.Error("加载翻译文件出错: %s", err)
		return err
	}
	return json.Unmarshal(data, &lib.TranslationsMap)
}

// SendFormatMessage creates a formatted message using fmt.Sprintf (utility).
func SendFormatMessage(chatID int64, format string, args ...interface{}) {
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(format, args...))
	core.Bot.Send(msg)
}

// buildProgressBar generates a text progress bar like [████████░░░░]
func buildProgressBar(completed, total, width int) string {
	if total <= 0 {
		return "[" + strings.Repeat("░", width) + "]"
	}
	filled := completed * width / total
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}
