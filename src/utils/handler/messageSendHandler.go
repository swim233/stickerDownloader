package handler

import (
	"strconv"
	"sync"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	utils "github.com/swim233/StickerDownloader/utils"
	"github.com/swim233/StickerDownloader/utils/logger"
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
	Single     int
	Pack       int
	HTTPSingle int
	HTTPPack   int
	Error      int
}

var downloadCounter DownloadCounter

// 计数器
func (m MessageSender) CountSender(u tgbotapi.Update) error {
	chatID := u.Message.From.ID
	msg := tgbotapi.NewMessage(chatID, "本次运行已下载贴纸 : "+strconv.Itoa(downloadCounter.Single)+"\n"+"本次运行已下载贴纸包 : "+strconv.Itoa(downloadCounter.Pack)+"\n"+"本次运行HTTP服务器已下载贴纸 : "+strconv.Itoa(downloadCounter.HTTPSingle)+"\n"+"本次运行HTTP服务器已下载贴纸包 : "+strconv.Itoa(downloadCounter.HTTPPack)+"\n"+"本次运行发生错误 : "+strconv.Itoa(downloadCounter.Error))

	utils.Bot.Send(msg)
	return nil
}

// 发送按钮消息
func (m MessageSender) ButtonMessageSender(u tgbotapi.Update) error {
	chatID := u.Message.From.ID
	stickerSet, err := utils.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: func(u tgbotapi.Update) string {
		return u.Message.Sticker.SetName
	}(u)})
	msg := tgbotapi.NewMessage(chatID, "当前贴纸包 : "+stickerSet.Title+"\n请选择要下载的方式")
	msg.ReplyToMessageID = u.Message.MessageID
	button1 := tgbotapi.NewInlineKeyboardButtonData("下载单个图片", "this")
	button2 := tgbotapi.NewInlineKeyboardButtonData("下载贴纸包", "zip")
	msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{button1}, {button2}}}
	utils.Bot.Send(msg)
	return err
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
				return data
			}(u), Name: u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName + ".webm"})
			msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID
			downloadCounter.Single++
			utils.Bot.Send(msg)

		} else {
			msg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Bytes: func(u tgbotapi.Update) []byte {
				if fmt == "webp" {
					data, _ := dl.DownloadFile(u)
					return data
				} else if fmt == "jpeg" {
					webp, err := dl.DownloadFile(u)
					if err != nil {
						logger.Error(err.Error())
					}
					fc := formatConverter{} //转换格式
					jpeg, err := fc.convertWebPToJPEG(webp, utils.BotConfig.WebPToJPEGQuality)
					if err != nil {
						logger.Error(err.Error())
					}
					return jpeg
				} else {
					webp, err := dl.DownloadFile(u)
					if err != nil {
						logger.Error(err.Error())
					}
					fc := formatConverter{} //转换格式
					png, err := fc.convertWebPToPNG(webp)
					if err != nil {
						logger.Error(err.Error())
					}
					return png

				}
			}(u), Name: u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName + "." + fmt})
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
		dl := downloaderPool.Get().(*StickerDownloader)
		data, stickerSetName, _ := dl.DownloadStickerSet(fmt, u)

		//贴纸包判空
		if len(data) == 0 {

			msg := tgbotapi.NewMessage(chatID, "贴纸包为空！")
			msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID
			utils.Bot.Send(msg)
			u.CallbackQuery.Delete()
			return nil

		}
		msg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Name: stickerSetName + ".zip", Bytes: data})
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

		logger.Error(err.Error())
		return err
	}
	return err
}
