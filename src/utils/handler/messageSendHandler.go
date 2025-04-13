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

var Counter int
var PackCounter int

// 计数器
func (m MessageSender) CountSender(u tgbotapi.Update) error {
	chatID := u.Message.From.ID
	msg := tgbotapi.NewMessage(chatID, "本次运行已下载贴纸 : "+strconv.Itoa(Counter)+"\n"+"本次运行已下载贴纸包 : "+strconv.Itoa(PackCounter))

	utils.Bot.Send(msg)
	return nil
}

// 发送按钮消息
func (m MessageSender) ButtonMessageSender(u tgbotapi.Update) error {
	chatID := u.Message.From.ID
	msg := tgbotapi.NewMessage(chatID, "请选择要下载的方式")
	msg.ReplyToMessageID = u.Message.MessageID
	button1 := tgbotapi.NewInlineKeyboardButtonData("下载单个图片", "this")
	button2 := tgbotapi.NewInlineKeyboardButtonData("下载贴纸包", "zip")
	msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{button1}, {button2}}}
	utils.Bot.Send(msg)
	return nil
}

// 单个WebP贴纸下载
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
			Counter++
			utils.Bot.Send(msg)

		} else {
			msg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Bytes: func(u tgbotapi.Update) []byte {
				if fmt == "webp" {
					data, _ := dl.DownloadFile(u)
					return data
				} else {
					webp, err := dl.DownloadFile(u)
					if err != nil {
						logger.Error(err.Error())
					}
					fc := formatConverter{}
					png, err := fc.convertWebPToPNG(webp)
					if err != nil {
						logger.Error(err.Error())
					}
					return png
				}
			}(u), Name: u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName + "." + fmt})
			downloaderPool.Put(dl)
			msg.ReplyToMessageID = u.CallbackQuery.Message.ReplyToMessage.MessageID
			Counter++
			utils.Bot.Send(msg)

		}
		//删除回调消息
		u.CallbackQuery.Delete()
		return nil
	}(u)
	return nil
}

// 格式选择
func (m MessageSender) FormatChose(u tgbotapi.Update) error {
	editMsgID := u.CallbackQuery.Message.MessageID
	ChatID := u.CallbackQuery.Message.Chat.ID
	editedMsg := tgbotapi.NewEditMessageText(ChatID, editMsgID, "请选择要下载的格式")
	WebPButton := tgbotapi.NewInlineKeyboardButtonData("WebP", "webp")
	PNGButton := tgbotapi.NewInlineKeyboardButtonData("PNG", "png")
	editButton := tgbotapi.NewInlineKeyboardMarkup([]tgbotapi.InlineKeyboardButton{WebPButton, PNGButton})
	editedMsg.ReplyMarkup = &editButton
	utils.Bot.Send(editedMsg)
	return nil
}

// PNG格式转换
func (m MessageSender) WebPFormatConverter(webp []byte) []byte {
	fc := formatConverter{}
	data, err := fc.convertWebPToPNG(webp)
	if err != nil {
		logger.Error(err.Error())
	}
	return data
}

// 贴纸集下载
func (m MessageSender) ZipSender(u tgbotapi.Update) error {
	go func(u tgbotapi.Update) error {
		chatID := u.CallbackQuery.Message.Chat.ID
		u.CallbackQuery.Answer(false, "正在下载贴纸包")
		dl := downloaderPool.Get().(*StickerDownloader)
		data, stickerSetName, _ := dl.DownloadStickerSet(u)

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
		PackCounter++
		utils.Bot.Send(msg) //发送消息
		u.CallbackQuery.Delete()
		return nil
	}(u)
	return nil
}
