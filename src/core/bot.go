package core

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/logger"
)

var (
	Bot     *tgbotapi.BotAPI
	HTTPBot *tgbotapi.BotAPI
)

// InitBot creates and configures the Telegram bot instances.
func InitBot() error {
	var err error
	newBot := func(token string) (*tgbotapi.BotAPI, error) {
		return tgbotapi.NewBotAPIWithAPIEndpoint(token, config.TelegramAPIEndpoint)
	}

	// Initialize HTTP bot (always needed)
	HTTPBot, err = newBot(config.HTTPToken)
	if err != nil {
		return fmt.Errorf("实例化 HTTP BotAPI: %w", err)
	}
	HTTPBot.Debug = config.DebugFlag

	// If tokens are the same, share the instance; otherwise create a separate bot
	if config.HTTPToken == config.BotToken {
		Bot = HTTPBot
	} else {
		Bot, err = newBot(config.BotToken)
		if err != nil {
			return fmt.Errorf("实例化 BotAPI: %w", err)
		}
		Bot.Debug = config.DebugFlag
	}

	// Configure logger for bot API
	adapter := logger.NewBotAPILoggerAdapter(config.ApiLogLevel)
	tgbotapi.SetLogger(adapter)

	// Configure proxy if available
	if proxy := fetchProxy(); proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return fmt.Errorf("解析代理 URL: %w", err)
		}
		Bot.Client = &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		}
		logger.Info("使用代理")
	}
	return nil
}

func fetchProxy() string {
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		if proxy := os.Getenv(key); proxy != "" {
			return proxy
		}
	}
	return ""
}
