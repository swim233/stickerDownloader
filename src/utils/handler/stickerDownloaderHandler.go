package handler

import (
	"archive/zip"
	"bytes"
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
	"github.com/swim233/StickerDownloader/utils/logger"
)

type StickerDownloader struct {
}

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

func (s StickerDownloader) DownloadStickerSet(u tgbotapi.Update) ([]byte, string, error) {
	stickerSet, err := utils.Bot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: s.getStickerSet(u)})
	var wg sync.WaitGroup
	var name string
	var mu sync.Mutex
	var downloadErrorArray []error
	if err != nil {
		return nil, "", err
	}
	name, err = os.MkdirTemp(".", "sticker")
	wg.Add(len(stickerSet.Stickers))
	for index, sticker := range stickerSet.Stickers {
		go func() {
			addErr := func(err error) {
				mu.Lock()
				downloadErrorArray = append(downloadErrorArray, err)
				err = nil
				mu.Unlock()
			}
			data, err := s.DownloadSetFile(sticker)
			if err != nil {
				addErr(err)
			}
			var filePath string
			if sticker.IsVideo {

				filePath = path.Join(name, strconv.Itoa(index)+".webm")
			} else {
				filePath = path.Join(name, strconv.Itoa(index)+".webp")
			}
			file, err := os.Create(filePath)
			if err != nil {
				addErr(err)
			}
			_, err = file.Write(data)
			if err != nil {
				addErr(err)
			}
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
		logger.Error(combinedError)
	} else {
		zipfile, err := compressFiles(name)
		return zipfile, stickerSet.Title, err
	}
	return nil, "", err
}

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
		return "", err
	}
	return FileURL, nil
}
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
		return "", err
	}
	return FileURL, nil
}

func (s StickerDownloader) getFileID(u tgbotapi.Update) string {
	fileID := u.CallbackQuery.Message.ReplyToMessage.Sticker.FileID
	return fileID
}
func (s StickerDownloader) getStickerSet(u tgbotapi.Update) string {
	stickerSetName := u.CallbackQuery.Message.ReplyToMessage.Sticker.SetName
	return stickerSetName
}

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
