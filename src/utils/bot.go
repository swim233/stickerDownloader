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
var HTTPBot *tgbotapi.BotAPI

type Config struct {
	Token               string // Telegram Bot Token
	HTTPToken           string // HTTP服务Bot Token
	DebugFlag           bool   // 是否开启debug输出
	ApiLogLevel         int    // 日志等级
	WebPToJPEGQuality   int    // WebP转JPEG的质量 范围为0-100
	HTTPServerPort      string // HTTP服务器端口
	EnableHTTPServer    bool   // 是否开启HTTP服务器
	MaxConcurrency      int    //最大并发数
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
		defaultEnv := `# Telegram Bot Token
Token=YOUR_TOKEN_ID

# HTTP服务使用的Telegram Bot Token(可以与Telegram Bot Token一致)
HTTPToken=YOUR_TOKEN_ID

# 日志等级 (可选值: DEBUG, INFO, WARN, ERROR)
LogLevel=DEBUG/INFO/WARN/ERROR

# 是否开启BotAPI debug输出(true/false)
DebugFlag=true

# API 日志等级 (可选值: DEBUG, INFO, WARN, ERROR)
ApiLogLevel=DEBUG/INFO/WARN/ERROR

# WebP 转 JPEG 的质量 (范围: 0-100)
WebPToJPEGQuality=100

# HTTP 服务器端口 (格式: :端口号)
HTTPServerPort=:8070

# 是否启用 HTTP 服务器 (true/false)
EnableHTTPServer=false

# 缓存过期时间 (单位: 分钟)
CacheExpirationTime=120

#最大并发数
MaxConcurrency=30
`
		if _, err := file.WriteString(defaultEnv); err != nil {
			logger.Error("写入 .env 文件失败: %v", err)
		}
		logger.Info(".env 文件已创建，并写入默认内容.")
		os.Exit(1)
	}
	err := godotenv.Load()
	if err != nil {
		logger.Error("%s", err)
	}

	getEnv() //读取环境变量

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

func getEnv() {
	var err error

	BotConfig.DebugFlag = (os.Getenv("DebugFlag") == "true") //读取debug输出

	BotConfig.Token = os.Getenv("Token") //读取token

	BotConfig.HTTPToken = os.Getenv("HTTPToken") //读取token

	// 后实例化的会被赋值到全局
	httpBot, err := tgbotapi.NewBotAPI(BotConfig.HTTPToken) //实例化BotAPI
	if err != nil {
		logger.Error("实例化BotApi时: %s", err)
		os.Exit(1)
	}
	HTTPBot = httpBot
	HTTPBot.Debug = BotConfig.DebugFlag
	if BotConfig.HTTPToken == BotConfig.Token {
		Bot = HTTPBot
	} else {
		bot, err := tgbotapi.NewBotAPI(BotConfig.Token) //实例化BotAPI
		if err != nil {
			logger.Error("实例化BotApi时: %s", err)
			err = nil
			os.Exit(1)
		}
		Bot = bot
		Bot.Debug = BotConfig.DebugFlag
	}
	loglevel := logger.ParseLogLevel(os.Getenv("LogLevel")) //读取bot log level
	logger.SetLogLevel(loglevel)

	BotConfig.ApiLogLevel = logger.ParseLogLevel(os.Getenv("ApiLogLevel")) //读取bot api log level
	adapter := &logger.TelegramBotApiLoggerAdapter{}
	adapter.SetLogger(logger.GetInstance())
	adapter.SetLogLevel(BotConfig.ApiLogLevel)
	tgbotapi.SetLogger(adapter)

	BotConfig.WebPToJPEGQuality, err = strconv.Atoi(os.Getenv("WebPToJPEGQuality")) //读取WebP转JPEG质量
	if err != nil {
		logger.Error("%s", err.Error())
		err = nil
	}

	BotConfig.EnableHTTPServer = (os.Getenv("EnableHTTPServer") == "true") //读取是否开启http服务

	BotConfig.HTTPServerPort = os.Getenv("HTTPServerPort") //读取http server 端口


	BotConfig.MaxConcurrency, err = (strconv.Atoi(os.Getenv("MaxConcurrency"))) // 读取最大并发数
	if err != nil {
		logger.Error("读取配置文件时出错: %s", err.Error())
		os.Exit(1)
	}

}
