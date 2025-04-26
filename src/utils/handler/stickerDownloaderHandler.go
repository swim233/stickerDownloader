package handler

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/utils"
	"github.com/swim233/StickerDownloader/utils/cache"
	"github.com/swim233/StickerDownloader/utils/logger"
)

type StickerDownloader struct {
}

// 下载单个贴纸
func (s StickerDownloader) DownloadFile(u tgbotapi.Update) ([]byte, error) {
	url, _ := s.getUrl(u)
	rsps, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer rsps.Body.Close()
	data, err := io.ReadAll(rsps.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// 下载贴纸集中的单个文件
func (s StickerDownloader) DownloadSetFile(sticker tgbotapi.Sticker) ([]byte, error) {
	url, _ := s.getSetUrl(sticker)
	rsps, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer rsps.Body.Close()
	data, err := io.ReadAll(rsps.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// 下载贴纸集
func (s StickerDownloader) DownloadStickerSet(fmt string, u tgbotapi.Update) ([]byte, string, int, error) {
	stickerSet, err := utils.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: s.getStickerSet(u)})
	setName := stickerSet.Name
	stickerNum := len(stickerSet.Stickers)
	if utils.BotConfig.EnableCache {
		cacheData, found := cache.GetCache(setName + fmt) //查找缓存
		if found {
			downloadCounter.Single += stickerNum
			downloadCounter.Cache++
			return cacheData, stickerSet.Title, 0, nil
		}
	}

	var wg sync.WaitGroup
	var name string
	var mu sync.Mutex
	var downloadErrorArray []error
	if err != nil {
		return nil, "", 0, err
	}
	logger.Info("%s", stickerSet.Name)
	name, err = os.MkdirTemp(".", "sticker")
	addErr := func(err error) { //错误处理
		mu.Lock()
		downloadErrorArray = append(downloadErrorArray, err)
		err = nil
		mu.Unlock()
	}
	if err != nil {
		addErr(err)
	}
	wg.Add(len(stickerSet.Stickers))
	for index, sticker := range stickerSet.Stickers {
		go func() {

			data, err := s.DownloadSetFile(sticker)

			if err != nil {
				addErr(err)
			}
			var filePath string
			if sticker.IsVideo {

				filePath = path.Join(name, strconv.Itoa(index)+".webm")
			} else {
				if fmt == "png" {
					fc := formatConverter{}
					data, _ = fc.convertWebPToPNG(data)
					filePath = path.Join(name, strconv.Itoa(index)+".png")
				} else if fmt == "jpeg" {
					fc := formatConverter{}
					data, _ = fc.convertWebPToJPEG(data, utils.BotConfig.WebPToJPEGQuality)
					filePath = path.Join(name, strconv.Itoa(index)+".jpeg")
				} else {
					filePath = path.Join(name, strconv.Itoa(index)+".webp")
				}
			}
			file, err := os.Create(filePath)
			if err != nil {
				addErr(err)
			}
			_, err = file.Write(data)
			if err != nil {
				addErr(err)
			}
			downloadCounter.Single++
			file.Close()
			wg.Done()
		}()
	}
	wg.Wait()
	err = nil
	if len(downloadErrorArray) > 0 {
		var combinedError string
		for _, err := range downloadErrorArray {
			combinedError += err.Error() + "; "
		}
		logger.Error("%s", combinedError)
	} else {
		zipfile, err := compressFiles(name)
		cache.AddCache(stickerSet.Name+fmt, zipfile) //写入缓存
		return zipfile, stickerSet.Title, stickerNum, err
	}
	return nil, "", 0, err
}

// HTTP下载贴纸集
func (s StickerDownloader) HTTPDownloadStickerSet(fmt string, setName string) ([]byte, error) {
	if fmt != "webp" && fmt != "png" && fmt != "jpeg" {
		err := errors.New("format is error")
		downloadCounter.Error++
		return nil, err
	}
	stickerSet, err := utils.HTTPBot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: setName})
	stickerNum := len(stickerSet.Stickers)

	if utils.BotConfig.EnableCache {
		cacheData, found := cache.GetCache(setName + fmt) //查找缓存
		if found {
			downloadCounter.HTTPPack++
			downloadCounter.HTTPSingle += stickerNum
			downloadCounter.Cache++
			return cacheData, nil
		}
	}
	var wg sync.WaitGroup
	var name string
	var mu sync.Mutex
	var downloadErrorArray []error
	if err != nil {
		return nil, err
	}
	name, err = os.MkdirTemp(".", "sticker")
	addErr := func(err error) { //错误处理
		mu.Lock()
		downloadErrorArray = append(downloadErrorArray, err)
		err = nil
		mu.Unlock()
	}
	if err != nil {
		addErr(err)
	}

	wg.Add(stickerNum)
	for index, sticker := range stickerSet.Stickers {
		go func() {

			data, err := s.DownloadSetFile(sticker)

			if err != nil {
				addErr(err)
			}
			var filePath string
			if sticker.IsVideo {

				filePath = path.Join(name, strconv.Itoa(index)+".webm")
			} else {

				if fmt == "png" {
					fc := formatConverter{}
					data, _ = fc.convertWebPToPNG(data)
					filePath = path.Join(name, strconv.Itoa(index)+".png")
				} else if fmt == "jpeg" {
					fc := formatConverter{}
					data, _ = fc.convertWebPToJPEG(data, utils.BotConfig.WebPToJPEGQuality)
					filePath = path.Join(name, strconv.Itoa(index)+".jpeg")
				} else {
					filePath = path.Join(name, strconv.Itoa(index)+".webp")
				}

			}
			file, err := os.Create(filePath)
			if err != nil {
				addErr(err)
			}
			_, err = file.Write(data)
			if err != nil {
				addErr(err)
			}
			downloadCounter.HTTPSingle++
			file.Close()
			wg.Done()
		}()
	}
	wg.Wait()
	err = nil
	if len(downloadErrorArray) > 0 {
		var combinedError string
		for _, err := range downloadErrorArray {
			combinedError += err.Error() + "; "
			downloadCounter.Error++
		}
		logger.Error("%s", combinedError)
		err := errors.New(combinedError)
		return nil, err
	} else {
		zipfile, err := compressFiles(name)
		downloadCounter.HTTPPack++
		cache.AddCache(setName+fmt, zipfile) //写入缓存
		return zipfile, err
	}
}

// 获取文件url
func (s StickerDownloader) getUrl(update tgbotapi.Update) (url string, err error) {
	fileID := s.getFileID(update)
	FileURL, err := func(bot tgbotapi.BotAPI, fileID string) (string, error) {
		file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", bot.Token, file.FilePath), nil
	}(*utils.Bot, fileID)
	if err != nil {
		downloadCounter.Error++
		return "", err
	}
	return FileURL, nil
}

// 获取贴纸集url
func (s StickerDownloader) getSetUrl(sticker tgbotapi.Sticker) (url string, err error) {
	fileID := sticker.FileID
	FileURL, err := func(bot tgbotapi.BotAPI, fileID string) (string, error) {
		file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", bot.Token, file.FilePath), nil
	}(*utils.Bot, fileID)
	if err != nil {
		downloadCounter.Error++
		return "", err
	}
	return FileURL, nil
}

// 获取文件FileID
func (s StickerDownloader) getFileID(u tgbotapi.Update) string {
	fileID := u.CallbackQuery.Message.ReplyToMessage.Sticker.FileID
	return fileID
}

// 获取贴纸集
func (s StickerDownloader) getStickerSet(u tgbotapi.Update) string {
	stickerSetName := u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName
	return stickerSetName
}

// 压缩文件
func compressFiles(dir string) (data []byte, err error) {
	// 创建一个内存缓冲区
	var buf bytes.Buffer

	zipWriter := zip.NewWriter(&buf)

	// 遍历目录并添加文件到zip
	walkErr := filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err // 处理遍历错误
		}

		// 获取相对于根目录的相对路径
		relPath, err := filepath.Rel(dir, filePath)
		if err != nil {
			return err
		}

		// 跳过目录，仅处理文件
		if info.IsDir() {
			return nil
		}

		// 创建zip文件头
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath       // 设置文件名
		header.Method = zip.Deflate // 使用Deflate压缩算法

		// 创建条目写入器
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// 打开文件并复制内容到zip条目
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	// 确保关闭zip写入器以完成压缩
	closeErr := zipWriter.Close()

	// 错误处理优先级：遍历错误 > 关闭错误
	if walkErr != nil {
		return nil, walkErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	os.RemoveAll(dir)
	return buf.Bytes(), nil
}
