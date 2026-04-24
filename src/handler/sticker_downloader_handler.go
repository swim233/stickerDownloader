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
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/logger"
	"github.com/swim233/StickerDownloader/utils"
)

// StickerDownloader handles downloading sticker files from Telegram.
type StickerDownloader struct {
	ID int
}

// DownloadFile downloads a single sticker file from a callback update.
func (s StickerDownloader) DownloadFile(u tgbotapi.Update) ([]byte, error) {
	fileID := u.CallbackQuery.Message.ReplyToMessage.Sticker.FileID
	return downloadByFileID(fileID)
}

// DownloadSetFile downloads a single sticker from a sticker set.
func (s StickerDownloader) DownloadSetFile(sticker tgbotapi.Sticker) ([]byte, error) {
	return downloadByFileID(sticker.FileID)
}

// downloadByFileID is the shared implementation for downloading a file by its ID.
func downloadByFileID(fileID string) ([]byte, error) {
	var lastErr error
	for i := 0; i < config.MaxRetry; i++ {
		file, err := core.Bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
		if err != nil {
			lastErr = fmt.Errorf("获取文件失败: %w", err)
			logger.Warn("获取文件失败 (重试 %d/%d): %s", i+1, config.MaxRetry, err)
			continue
		}
		fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", core.Bot.Token, file.FilePath)

		resp, err := http.Get(fileURL)
		if err != nil {
			lastErr = fmt.Errorf("下载文件失败: %w", err)
			logger.Warn("下载文件失败 (重试 %d/%d): %s", i+1, config.MaxRetry, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("下载文件返回非 200 状态码: %d", resp.StatusCode)
			logger.Warn("下载返回状态码 %d (重试 %d/%d)", resp.StatusCode, i+1, config.MaxRetry)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("读取响应失败: %w", err)
			continue
		}
		return data, nil
	}
	utils.RuntimeStatus.Errors.Add(1)
	return nil, lastErr
}

// downloadAndPackStickers downloads all stickers in a set, converts them to the given format, and returns a zip.
// onProgress is called every 3s with (completed, total) counts. Pass nil to disable progress reporting.
func downloadAndPackStickers(format string, stickerSet tgbotapi.StickerSet, onProgress func(completed, total int)) (zipData []byte, stickerNum int, err error) {
	stickerNum = len(stickerSet.Stickers)
	logger.Info("下载贴纸包: %s (%d 个贴纸)", stickerSet.Name, stickerNum)

	tempDir, err := os.MkdirTemp(".", "sticker")
	if err != nil {
		return nil, 0, fmt.Errorf("创建临时目录失败: %w", err)
	}

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		dlError   error
		completed atomic.Int64
	)

	// Progress ticker
	var tickerDone chan struct{}
	if onProgress != nil {
		tickerDone = make(chan struct{})
		go func() {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					onProgress(int(completed.Load()), stickerNum)
				case <-tickerDone:
					return
				}
			}
		}()
	}

	dl := StickerDownloader{}
	wg.Add(stickerNum)
	for index, sticker := range stickerSet.Stickers {
		go func(index int, sticker tgbotapi.Sticker) {
			defer wg.Done()

			data, err := dl.DownloadSetFile(sticker)
			if err != nil {
				mu.Lock()
				dlError = errors.Join(dlError, err)
				mu.Unlock()
				completed.Add(1)
				return
			}

			var filePath string
			if sticker.IsVideo {
				filePath = path.Join(tempDir, strconv.Itoa(index)+".webm")
			} else if sticker.IsAnimated {
				filePath = path.Join(tempDir, strconv.Itoa(index)+".tgs")
			} else {
				filePath, data = convertSticker(tempDir, index, format, data)
			}

			if err := os.WriteFile(filePath, data, 0644); err != nil {
				mu.Lock()
				dlError = errors.Join(dlError, err)
				mu.Unlock()
			}
			completed.Add(1)
		}(index, sticker)
	}
	wg.Wait()

	// Stop ticker and send final progress
	if tickerDone != nil {
		close(tickerDone)
		onProgress(stickerNum, stickerNum)
	}

	if dlError != nil {
		logger.Error("下载贴纸时发生错误: %s", dlError)
	}

	// Check if any files were actually written
	entries, _ := os.ReadDir(tempDir)
	if len(entries) == 0 {
		os.RemoveAll(tempDir)
		return nil, 0, fmt.Errorf("所有贴纸下载失败: %w", dlError)
	}

	zipData, err = compressFiles(tempDir)
	if err != nil {
		return nil, 0, err
	}
	return zipData, stickerNum, nil
}

// convertSticker converts sticker data to the target format and returns the file path and converted data.
func convertSticker(dir string, index int, format string, data []byte) (string, []byte) {
	fc := utils.FormatConverter{}
	switch format {
	case "png":
		converted, err := fc.WebpToPNG(data)
		if err == nil {
			data = converted
		}
		return path.Join(dir, strconv.Itoa(index)+".png"), data
	case "jpeg":
		converted, err := fc.WebpToJPEG(data, config.JpegQuality)
		if err == nil {
			data = converted
		}
		return path.Join(dir, strconv.Itoa(index)+".jpeg"), data
	default:
		return path.Join(dir, strconv.Itoa(index)+".webp"), data
	}
}

// DownloadStickerSet downloads and zips a full sticker set (bot handler version).
func (s StickerDownloader) DownloadStickerSet(format lib.TaskFileFormat, stickerSet tgbotapi.StickerSet, u tgbotapi.Update, onProgress func(completed, total int)) ([]byte, string, int, error) {
	zipData, stickerNum, err := downloadAndPackStickers(string(format), stickerSet, onProgress)
	if err != nil {
		return nil, "", 0, err
	}
	// Track stats
	utils.RuntimeStatus.SingleDownload.Add(int64(stickerNum))
	return zipData, stickerSet.Title, stickerNum, nil
}

// HTTPDownloadStickerSet downloads and zips a sticker set via HTTP API.
func (s StickerDownloader) HTTPDownloadStickerSet(format string, setName string) ([]byte, error) {
	if format != "webp" && format != "png" && format != "jpeg" {
		utils.RuntimeStatus.Errors.Add(1)
		return nil, errors.New("invalid format: must be webp, png, or jpeg")
	}

	stickerSet, err := core.HTTPBot.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: setName})
	if err != nil {
		return nil, err
	}

	zipData, _, err := downloadAndPackStickers(format, stickerSet, nil)
	if err != nil {
		utils.RuntimeStatus.Errors.Add(1)
		return nil, err
	}

	utils.RuntimeStatus.HTTPPackDownload.Add(1)
	return zipData, nil
}

// GetStickerSetName extracts the sticker set name from an update (callback query).
func GetStickerSetName(u tgbotapi.Update) string {
	var stickerLinkRegex = regexp.MustCompile(`https://t.me/addstickers/([a-zA-Z0-9_]+)`)

	if u.CallbackQuery == nil || u.CallbackQuery.Message == nil || u.CallbackQuery.Message.ReplyToMessage == nil {
		return ""
	}
	reply := u.CallbackQuery.Message.ReplyToMessage

	if reply.Sticker != nil {
		return reply.Sticker.SetName
	}
	if reply.Text != "" {
		matches := stickerLinkRegex.FindStringSubmatch(reply.Text)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// compressFiles zips all files in a directory and removes the directory afterward.
func compressFiles(dir string) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	walkErr := filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(dir, filePath)
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	closeErr := zipWriter.Close()
	os.RemoveAll(dir)

	if walkErr != nil {
		return nil, walkErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return buf.Bytes(), nil
}
