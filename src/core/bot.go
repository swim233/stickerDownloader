package core

import (
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
func InitBot() {
	var err error

	// Initialize HTTP bot (always needed)
	HTTPBot, err = tgbotapi.NewBotAPI(config.HTTPToken)
	if err != nil {
		logger.Error("实例化 HTTP BotAPI 失败: %s", err)
		os.Exit(1)
	}
	HTTPBot.Debug = config.DebugFlag

	// If tokens are the same, share the instance; otherwise create a separate bot
	if config.HTTPToken == config.BotToken {
		Bot = HTTPBot
	} else {
		Bot, err = tgbotapi.NewBotAPI(config.BotToken)
		if err != nil {
			logger.Error("实例化 BotAPI 失败: %s", err)
			os.Exit(1)
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
			logger.Error("解析代理 URL 失败: %s", proxy)
			return
		}
		Bot.Client = &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		}
		logger.Info("使用代理: %s", proxy)
	}
}

func fetchProxy() string {
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		if proxy := os.Getenv(key); proxy != "" {
			return proxy
		}
	}
	return ""
}
