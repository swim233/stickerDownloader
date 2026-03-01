package core

import (
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/spf13/viper"
	"github.com/swim233/stickerDownloader/logger"
)

var Bot *tgbotapi.BotAPI

func InitBot() {
	botToken := viper.GetString("bot_token")

	if botToken == "" {
		logger.Panic("fail to get bot token, please check config")
	}

	for i := 0; i < viper.GetInt("bot_login_max_retry")+1; i++ {
		if i >= viper.GetInt("bot_login_max_retry") {
			logger.Panic("fail to login bot, tried: %d times, please check network and bot token", i)
		}
		bot, err := tgbotapi.NewBotAPI(botToken)
		if err != nil {
			logger.Warn("fail to login: %s, retrying, current retry count: %d", err.Error(), i+1)
			continue
		}
		Bot = bot
		break
	}
	logger.Info("success login, bot name: %s", Bot.Self.FullName())
}
