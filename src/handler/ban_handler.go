package handler

import (
	"errors"
	"strconv"
	"strings"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/core"
	db "github.com/swim233/StickerDownloader/db"
	logger "github.com/swim233/StickerDownloader/logger"
)

// parseBanArgs splits "/ban 12345 reason..." arguments into user ID and reason.
func parseBanArgs(args string) (int64, string, error) {
	fields := strings.Fields(args)
	if len(fields) == 0 {
		return 0, "", errors.New("缺少用户 ID")
	}
	userID, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil || userID <= 0 {
		return 0, "", errors.New("用户 ID 无效")
	}
	return userID, strings.Join(fields[1:], " "), nil
}

// isOwnerCommand reports whether the message is a private command from the owner.
func isOwnerCommand(u tgbotapi.Update) bool {
	return config.OwnerChatID != 0 &&
		u.Message != nil && u.Message.From != nil &&
		u.Message.From.ID == config.OwnerChatID
}

// updateUserID extracts the acting user's ID from any update type.
func updateUserID(u tgbotapi.Update) int64 {
	switch {
	case u.Message != nil && u.Message.From != nil:
		return u.Message.From.ID
	case u.CallbackQuery != nil && u.CallbackQuery.From != nil:
		return u.CallbackQuery.From.ID
	case u.InlineQuery != nil && u.InlineQuery.From != nil:
		return u.InlineQuery.From.ID
	default:
		return 0
	}
}

func (m MessageSender) BanUser(u tgbotapi.Update) error {
	return m.banUser(u, "/ban", false)
}

func (m MessageSender) SilentBanUser(u tgbotapi.Update) error {
	return m.banUser(u, "/sban", true)
}

func (m MessageSender) banUser(u tgbotapi.Update, command string, silent bool) error {
	if !isOwnerCommand(u) {
		return nil
	}
	chatID := u.Message.Chat.ID
	userID, reason, err := parseBanArgs(u.Message.CommandArguments())
	if err != nil {
		SendFormatMessage(chatID, "%s，用法：%s <userid> [原因]", err, command)
		return nil
	}
	if userID == config.OwnerChatID {
		SendFormatMessage(chatID, "不能封禁所有者")
		return nil
	}
	if err := db.BanUser(userID, reason, silent); err != nil {
		logger.Error("封禁用户 %d 失败: %s", userID, err)
		SendFormatMessage(chatID, "封禁失败：%s", err)
		return err
	}
	mode := "已封禁"
	if silent {
		mode = "已静默封禁"
	}
	if reason == "" {
		reason = "默认（各语言的“您已被封禁”）"
	}
	SendFormatMessage(chatID, "%s用户 %d\n原因：%s", mode, userID, reason)
	return nil
}

func (m MessageSender) UnbanUser(u tgbotapi.Update) error {
	if !isOwnerCommand(u) {
		return nil
	}
	chatID := u.Message.Chat.ID
	userID, _, err := parseBanArgs(u.Message.CommandArguments())
	if err != nil {
		SendFormatMessage(chatID, "%s，用法：/unban <userid>", err)
		return nil
	}
	existed, err := db.UnbanUser(userID)
	if err != nil {
		logger.Error("解封用户 %d 失败: %s", userID, err)
		SendFormatMessage(chatID, "解封失败：%s", err)
		return err
	}
	if !existed {
		SendFormatMessage(chatID, "用户 %d 不在封禁列表中", userID)
		return nil
	}
	SendFormatMessage(chatID, "已解封用户 %d", userID)
	return nil
}

// BannedUpdateMatch reports whether an update comes from a banned user.
// Registered as the first generic processor so it swallows every update
// (messages, commands, callbacks) from banned users.
func BannedUpdateMatch(u tgbotapi.Update) bool {
	userID := updateUserID(u)
	if userID == 0 || (config.OwnerChatID != 0 && userID == config.OwnerChatID) {
		return false
	}
	_, banned := db.GetBan(userID)
	return banned
}

// BannedUserResponder tells a banned user why, unless the ban is silent.
func (m MessageSender) BannedUserResponder(u tgbotapi.Update) error {
	userID := updateUserID(u)
	ban, ok := db.GetBan(userID)
	if !ok || ban.Silent {
		return nil
	}
	text := tr(userID).YouAreBanned
	if ban.Reason != "" {
		text += "：" + ban.Reason
	}
	if u.CallbackQuery != nil {
		if _, err := u.CallbackQuery.Answer(true, text); err != nil {
			logger.Warn("响应被封禁用户回调出错: %s", err)
		}
		return nil
	}
	if u.Message != nil {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, text)
		msg.ReplyToMessageID = u.Message.MessageID
		core.Bot.Send(msg)
	}
	return nil
}
