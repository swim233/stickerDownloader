package utils

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/swim233/StickerDownloader/utils/logger"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	godotenv "github.com/joho/godotenv"
)

var Bot *tgbotapi.BotAPI

type Config struct {
	Token             string
	DebugFlag         bool
	ApiLogLevel       int
	WebPToJPEGQuality int
}

var BotConfig Config

func InitBot() {

	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		// 如果 .env 文件不存在，创建并写入默认值
		logger.Info(".env 文件不存在，正在创建...")

		// 创建并打开 .env 文件
		file, err := os.Create(".env")
		if err != nil {
			logger.Error("创建 .env 文件失败: %v", err)
		}
		defer file.Close()

		// 写入默认的环境变量内容
		defaultEnv := `Token=YOUR_TOKEN_ID
LogLevel=DEBUG/INFO/WARN/ERROR
ApiLogLevel=DEBUG/INFO/WARN/ERROR
WebPToJPEGQuality=100
`
		if _, err := file.WriteString(defaultEnv); err != nil {
			logger.Error("写入 .env 文件失败: %v", err)
		}
		logger.Info(".env 文件已创建，并写入默认内容.")
	}
	err := godotenv.Load()
	if err != nil {
		logger.Error("%s", err)
	}

	//读取环境变量
	loglevel := logger.ParseLogLevel(os.Getenv("LogLevel"))
	BotConfig.ApiLogLevel = logger.ParseLogLevel(os.Getenv("ApiLogLevel"))
	logger.SetLogLevel(loglevel)
	BotConfig.Token = os.Getenv("Token")
	adapter := &logger.TelegramBotApiLoggerAdapter{}
	adapter.SetLogger(logger.GetInstance())
	adapter.SetLogLevel(BotConfig.ApiLogLevel)
	tgbotapi.SetLogger(adapter)
	bot, err := tgbotapi.NewBotAPI(BotConfig.Token)
	if err != nil {
		logger.Error("%s", err)
	}
	BotConfig.WebPToJPEGQuality, err = strconv.Atoi(os.Getenv("WebPToJPEGQuality"))
	Bot = bot
	if err != nil {
		logger.Error("%s", BotConfig.Token)
		logger.Error("%s", err)
	}

	proxy := FetchProxy()
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			logger.Error("Failed to parse proxy url: %s", proxy)
			return
		}
		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
		Bot.Client = client
		logger.Info("Using proxy: %s", proxy)
	}
}

func FetchProxy() string {
	proxy := os.Getenv("HTTP_PROXY")
	if proxy == "" {
		proxy = os.Getenv("HTTPS_PROXY")
	}
	if proxy == "" {
		proxy = os.Getenv("http_proxy")
	}
	if proxy == "" {
		proxy = os.Getenv("https_proxy")
	}
	return proxy
}
func UpdateEnvValue(key, newValue string) error {
	// 读取现有内容
	content, err := os.ReadFile(".env")
	if err != nil {
		return fmt.Errorf("读取文件失败: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	found := false

	// 更新值
	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			lines[i] = key + "=" + newValue
			found = true
			break
		}
	}

	// 如果键不存在，添加新行
	if !found {
		lines = append(lines, key+"="+newValue)
	}

	// 写回文件
	output := strings.Join(lines, "\n")
	err = os.WriteFile(".env", []byte(output), 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}
	return nil
}
