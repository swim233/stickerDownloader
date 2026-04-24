package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

var (
	// Telegram Bot
	BotToken  string
	HTTPToken string
	DebugFlag bool

	// Log
	LogLevel    string
	ApiLogLevel string

	// Image conversion
	JpegQuality int

	// HTTP Server
	EnableHTTPServer bool
	HTTPServerPort   string

	// Concurrency
	MaxConcurrency int

	// Retry
	MaxRetry int
)

// InitConfig loads configuration from config.yaml and environment variables.
func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../config")  // ../config/ (从 src/ 运行时)
	viper.AddConfigPath(".")          // 当前目录 (从项目根目录运行时)

	// Defaults
	viper.SetDefault("telegram.debug", false)
	viper.SetDefault("log.level", "INFO")
	viper.SetDefault("log.api_level", "INFO")
	viper.SetDefault("image.jpeg_quality", 100)
	viper.SetDefault("server.enabled", false)
	viper.SetDefault("server.port", ":8070")
	viper.SetDefault("download.max_concurrency", 30)
	viper.SetDefault("download.max_retry", 3)

	// Allow environment variable overrides
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			createDefaultYAMLConfig()
			fmt.Println("config/config.yaml 已创建，请填写配置后重新启动。")
			fmt.Println("config/config.yaml has been created. Please fill in your settings and restart.")
			os.Exit(1)
		}
		fmt.Printf("读取配置文件失败: %v\n", err)
		os.Exit(1)
	}

	BotToken = viper.GetString("telegram.token")
	HTTPToken = viper.GetString("telegram.http_token")
	DebugFlag = viper.GetBool("telegram.debug")
	LogLevel = viper.GetString("log.level")
	ApiLogLevel = viper.GetString("log.api_level")
	JpegQuality = viper.GetInt("image.jpeg_quality")
	EnableHTTPServer = viper.GetBool("server.enabled")
	HTTPServerPort = viper.GetString("server.port")
	MaxConcurrency = viper.GetInt("download.max_concurrency")
	MaxRetry = viper.GetInt("download.max_retry")
}

func createDefaultYAMLConfig() {
	content := `# Telegram Bot 配置 / Telegram Bot Configuration
telegram:
  # Bot Token (从 @BotFather 获取 / Get from @BotFather)
  token: "YOUR_TOKEN_HERE"
  # HTTP 服务使用的 Token，可与上面相同 / Token for HTTP service, can be the same
  http_token: "YOUR_TOKEN_HERE"
  # 是否开启 BotAPI debug 输出 / Enable BotAPI debug output
  debug: false

# 日志配置 / Log Configuration
log:
  # 日志等级 / Log level (DEBUG, INFO, WARN, ERROR)
  level: "INFO"
  # Bot API 日志等级 / Bot API log level (DEBUG, INFO, WARN, ERROR)
  api_level: "INFO"

# 图片转换配置 / Image Conversion
image:
  # WebP 转 JPEG 的质量 / WebP to JPEG quality (0-100)
  jpeg_quality: 100

# HTTP 服务器配置 / HTTP Server Configuration
server:
  # 是否启用 HTTP 服务器 / Enable HTTP server
  enabled: false
  # 监听端口 / Listen port
  port: ":8070"

# 下载配置 / Download Configuration
download:
  # 最大并发下载数 / Max concurrent downloads
  max_concurrency: 30
  # 最大重试次数 / Max retry count
  max_retry: 3
`
	// 尝试多个可能的路径写入
	for _, dir := range []string{"../config", "."} {
		os.MkdirAll(dir, 0755)
		path := dir + "/config.yaml"
		if err := os.WriteFile(path, []byte(content), 0644); err == nil {
			fmt.Printf("配置文件已写入: %s\n", path)
			return
		}
	}
}
