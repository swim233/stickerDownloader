package cache

import (
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/swim233/StickerDownloader/utils"
)

var Cacher = cache.New(time.Duration(utils.BotConfig.CacheExpirationTime)*time.Minute, 10*time.Minute)

// 添加缓存
func AddCache(name string, inputData []byte) {

	Cacher.Set(name, inputData, cache.DefaultExpiration)
}

// 获取缓存
func GetCache(name string) (data []byte, found bool) {
	value, found := Cacher.Get(name)
	if found {
		data, _ = value.([]byte)
	}
	return data, found
}
