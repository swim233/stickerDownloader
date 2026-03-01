package main

import (
	"github.com/swim233/stickerDownloader/config"
	"github.com/swim233/stickerDownloader/core"
	"github.com/swim233/stickerDownloader/logger"
)

func main() {
	logger.InitLogger()
	config.InitConfig()
	core.InitBot()
	logger.Info("hello world")
}
