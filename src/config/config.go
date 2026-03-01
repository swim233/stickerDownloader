package config

import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/swim233/stickerDownloader/logger"
)

func InitConfig() {
	viper.AddConfigPath("/config/")
	viper.AddConfigPath("/")
	viper.AddConfigPath(".")
	viper.SetConfigName("config.yaml")
	viper.SetConfigType("yaml")

	viper.SetDefault("bot_login_max_retry", 1)

	err := viper.ReadInConfig()
	if err != nil {
		logger.Error("can not load config: %s", err.Error())
	} else {
		logger.Info("config load successful: %s", viper.ConfigFileUsed())
		logger.Debugln(viper.AllKeys())
	}
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		logger.Info("detaching config changed: %s", in.Name)
		logger.Debug("config file event operation: %s", in.Op)
	})
}
