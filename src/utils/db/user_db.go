package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/utils/hashCalculator"
	"gorm.io/gorm"
)

type UserData struct {
	UserID             int64  `gorm:"primaryKey,autoIncrement:false"`
	FirstName          string `gorm:"not null"`
	LastName           string
	UserName           string
	DownloadFiles      int
	DownloadFileSize   int64
	CreateTime         string `gorm:"not null,default:0"`
	RecentDownloadTime string `gorm:"not null,default:0"`
	UserLanguage       string `gorm:"not null,default:'zh'"`
}

type StickerData struct {
	StickerName        string `gorm:"primaryKey,autoIncrement:false,not null"`
	StickerTitle       string `gorm:"not null"`
	RecentDownloadTime string `gorm:"not null,default:0"`
	LastDownloadUser   int64  `gorm:"default:0"`
	DownloadCount      int
	WebpFileID         string
	PNGFileID          string
	JPEGFileID         string
	StickerNum         int
	SetHash            string
}

var DB *gorm.DB

// 初始化数据库
func InitDB() {
	os.MkdirAll("db", 0700)
	db, err := gorm.Open(sqlite.Open("db/data.db"), &gorm.Config{})
	if err != nil {
		fmt.Printf("数据库初始化失败: %v\n", err)
		os.Exit(1)
	}
	DB = db
	DB.AutoMigrate(&UserData{}, &StickerData{})
}

// 初始化用户数据
func InitUserData(u tgbotapi.Update) error {
	user := u.Message.From
	if user == nil {
		return errors.New("用户为空")
	}
	newUser := UserData{}

	err := DB.Where("user_id = ?", user.ID).First(&newUser).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		err := DB.Create(UserData{
			UserID:       user.ID,
			FirstName:    user.FirstName,
			LastName:     user.LastName,
			UserName:     user.UserName,
			UserLanguage: "zh",
			CreateTime:   time.Now().Format(time.RFC3339),
		}).Error
		if err != nil {
			DB.Logger.Error(context.Background(), err.Error())
			return err
		} else {
			DB.Logger.Info(context.Background(), "创建成功")
		}
	} else if err != nil {
		fmt.Printf("err: %v\n", err)
		return err
	}
	return err
}

// 记录用户数据
func RecordUserData(u tgbotapi.Update, fileSize int64, fileCount int) {
	user := u.CallbackQuery.From
	if user == nil {
		return
	}
	newUser := UserData{}

	err := DB.Where("user_id = ?", user.ID).First(&newUser).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		err := DB.Create(UserData{
			UserID:             user.ID,
			FirstName:          user.FirstName,
			LastName:           user.LastName,
			UserName:           user.UserName,
			DownloadFiles:      fileCount,
			DownloadFileSize:   fileSize,
			CreateTime:         time.Now().Format(time.RFC3339),
			RecentDownloadTime: time.Now().Format(time.RFC3339),
		}).Error
		if err != nil {
			DB.Logger.Error(context.Background(), err.Error())
		} else {
			DB.Logger.Info(context.Background(), "创建成功")
		}
	} else if err != nil {
		fmt.Printf("err: %v\n", err)
	} else {
		newUser.DownloadFiles += fileCount
		newUser.DownloadFileSize += fileSize
		newUser.FirstName = user.FirstName
		newUser.LastName = user.LastName
		newUser.UserName = user.UserName
		newUser.RecentDownloadTime = time.Now().Format(time.RFC3339)
		newUser.UserLanguage = "zh"
		err := DB.Where("user_id = ?", user.ID).Save(&newUser).Error
		if err != nil {
			DB.Logger.Error(context.Background(), err.Error())
		}
	}

}

// 记录贴纸数据
func RecordStickerData(setName string, title string, UserID int64, WebPFileID string, PNGFileID string, JPEGFileID string, StickerNum int, SetHash string) {
	newStickerSetData := StickerData{}

	err := DB.Where("sticker_name = ?", setName).First(&newStickerSetData).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		err := DB.Create(StickerData{
			StickerName:        setName,
			StickerTitle:       title,
			DownloadCount:      1,
			LastDownloadUser:   UserID,
			RecentDownloadTime: time.Now().Format(time.RFC3339),
			WebpFileID:         WebPFileID,
			PNGFileID:          PNGFileID,
			JPEGFileID:         JPEGFileID,
			StickerNum:         StickerNum,
			SetHash:            SetHash,
		}).Error
		if err != nil {
			DB.Logger.Error(context.Background(), err.Error())
		} else {
			DB.Logger.Info(context.Background(), "创建成功")
		}
	} else if err != nil {
		fmt.Printf("err: %v\n", err)
	} else {
		newStickerSetData.DownloadCount += 1
		newStickerSetData.StickerTitle = title
		newStickerSetData.RecentDownloadTime = time.Now().Format(time.RFC3339)
		newStickerSetData.LastDownloadUser = UserID
		if WebPFileID != "" {
			newStickerSetData.WebpFileID = WebPFileID
		}
		if PNGFileID != "" {
			newStickerSetData.PNGFileID = PNGFileID
		}
		if JPEGFileID != "" {
			newStickerSetData.JPEGFileID = JPEGFileID
		}
		newStickerSetData.SetHash = hashCalculator.CalculateHashViaSetName(setName)
		newStickerSetData.StickerNum += StickerNum
		err := DB.Where("sticker_name = ?", setName).Save(&newStickerSetData).Error
		if err != nil {
			DB.Logger.Error(context.Background(), err.Error())
		}
	}

}

// 获取贴纸包数据
func GetStickerData(SetName string) (StickerData, error) {
	var data StickerData
	err := DB.Where("sticker_name = ?", SetName).First(&data).Error
	return data, err
}

// 获取用户语言
func GetUserLanguage(UserID int64) string {
	var lang string
	err := DB.Model(&UserData{}).Select("user_language").Where("user_id = ?", UserID).Scan(&lang).Error
	if err != nil {
		DB.Logger.Error(context.Background(), err.Error())
		return "zh"
	}
	if lang == "" {
		return "zh"
	}
	return lang
}

// 修改用户语言
func ChangeUserLanguage(UserID int64, lang string) error {
	err := DB.Model(&UserData{}).Where("user_id = ?", UserID).Update("user_language", lang).Error
	if err != nil {
		DB.Logger.Error(context.Background(), err.Error())
		return err
	}
	return nil
}
