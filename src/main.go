package main

import (
	"github.com/swim233/stickerDownloader/config"
	"github.com/swim233/stickerDownloader/core"
	"github.com/swim233/stickerDownloader/handler"
	"github.com/swim233/stickerDownloader/logger"
)

func main() {
	logger.InitLogger()
	config.InitConfig()
	core.InitBot()
	b := core.Bot.AddHandle()
	handler.MessageWithStickerHandler(b)
	b.Run()
}
