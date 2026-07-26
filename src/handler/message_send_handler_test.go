package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/core"
	db "github.com/swim233/StickerDownloader/db"
	"github.com/swim233/StickerDownloader/lib"
	"github.com/swim233/StickerDownloader/logger"
	"github.com/swim233/StickerDownloader/utils"
	"gorm.io/gorm"
)

func TestThisSenderRejectsExpiredCallback(t *testing.T) {
	logger.InitLogger()

	var (
		mu      sync.Mutex
		methods []string
		bodies  []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		methods = append(methods, method)
		bodies = append(bodies, string(body))
		mu.Unlock()

		if method == "getMe" {
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"test","username":"test_bot"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer server.Close()

	bot, err := tgbotapi.NewBotAPIWithAPIEndpoint("test-token", server.URL+"/bot%s/%s")
	if err != nil {
		t.Fatal(err)
	}
	oldBot := core.Bot
	core.Bot = bot
	defer func() { core.Bot = oldBot }()

	oldErrors := utils.RuntimeStatus.Errors.Load()
	oldRequestErrors := utils.RuntimeStatus.RequestErrors.Load()
	oldPanicErrors := utils.RuntimeStatus.PanicErrors.Load()
	oldDownloadErrors := utils.RuntimeStatus.DownloadErrors.Load()
	oldDownloads := utils.RuntimeStatus.SingleDownload.Load()
	defer func() {
		utils.RuntimeStatus.Errors.Store(oldErrors)
		utils.RuntimeStatus.RequestErrors.Store(oldRequestErrors)
		utils.RuntimeStatus.PanicErrors.Store(oldPanicErrors)
		utils.RuntimeStatus.DownloadErrors.Store(oldDownloadErrors)
		utils.RuntimeStatus.SingleDownload.Store(oldDownloads)
	}()

	baseUpdate := func() tgbotapi.Update {
		return tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "callback-id",
			From: &tgbotapi.User{ID: 42},
			Message: &tgbotapi.Message{
				MessageID: 7,
				Chat:      &tgbotapi.Chat{ID: 42},
				ReplyToMessage: &tgbotapi.Message{
					MessageID: 6,
				},
			},
		}}
	}

	tests := []struct {
		name   string
		update func() tgbotapi.Update
		alert  bool
	}{
		{name: "callback missing", update: func() tgbotapi.Update { return tgbotapi.Update{} }},
		{name: "message missing", update: func() tgbotapi.Update {
			return tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{ID: "callback-id"}}
		}, alert: true},
		{name: "chat missing", update: func() tgbotapi.Update {
			u := baseUpdate()
			u.CallbackQuery.Message.Chat = nil
			return u
		}, alert: true},
		{name: "reply missing", update: func() tgbotapi.Update {
			u := baseUpdate()
			u.CallbackQuery.Message.ReplyToMessage = nil
			return u
		}, alert: true},
		{name: "sticker missing", update: baseUpdate, alert: true},
		{name: "file id missing", update: func() tgbotapi.Update {
			u := baseUpdate()
			u.CallbackQuery.Message.ReplyToMessage.Sticker = &tgbotapi.Sticker{}
			return u
		}, alert: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeErrors := utils.RuntimeStatus.Errors.Load()
			beforeRequestErrors := utils.RuntimeStatus.RequestErrors.Load()
			beforePanicErrors := utils.RuntimeStatus.PanicErrors.Load()
			beforeDownloadErrors := utils.RuntimeStatus.DownloadErrors.Load()
			beforeDownloads := utils.RuntimeStatus.SingleDownload.Load()
			mu.Lock()
			beforeCalls := len(methods)
			mu.Unlock()

			if err := (MessageSender{}).ThisSender(lib.PngFormat, tt.update()); err != nil {
				t.Fatal(err)
			}
			if got := utils.RuntimeStatus.Errors.Load(); got != beforeErrors+1 {
				t.Fatalf("Errors = %d, want %d", got, beforeErrors+1)
			}
			if got := utils.RuntimeStatus.RequestErrors.Load(); got != beforeRequestErrors+1 {
				t.Fatalf("RequestErrors = %d, want %d", got, beforeRequestErrors+1)
			}
			if got := utils.RuntimeStatus.PanicErrors.Load(); got != beforePanicErrors {
				t.Fatalf("PanicErrors changed: %d -> %d", beforePanicErrors, got)
			}
			if got := utils.RuntimeStatus.DownloadErrors.Load(); got != beforeDownloadErrors {
				t.Fatalf("DownloadErrors changed: %d -> %d", beforeDownloadErrors, got)
			}
			if got := utils.RuntimeStatus.SingleDownload.Load(); got != beforeDownloads {
				t.Fatalf("SingleDownload changed: %d -> %d", beforeDownloads, got)
			}

			mu.Lock()
			newMethods := append([]string(nil), methods[beforeCalls:]...)
			newBodies := append([]string(nil), bodies[beforeCalls:]...)
			mu.Unlock()
			if !tt.alert {
				if len(newMethods) != 0 {
					t.Fatalf("unexpected API calls: %v", newMethods)
				}
				return
			}
			if len(newMethods) != 1 || newMethods[0] != "answerCallbackQuery" {
				t.Fatalf("API calls = %v, want one answerCallbackQuery", newMethods)
			}
			if !strings.Contains(newBodies[0], "show_alert=true") || !strings.Contains(newBodies[0], "This+action+has+expired") {
				t.Fatalf("unexpected callback answer: %s", newBodies[0])
			}
		})
	}
}

func TestFormatChooserHandlesExpiredSingleAndZipTextReply(t *testing.T) {
	logger.InitLogger()

	var (
		mu      sync.Mutex
		methods []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		mu.Lock()
		methods = append(methods, method)
		mu.Unlock()
		if method == "getMe" {
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"test","username":"test_bot"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer server.Close()

	bot, err := tgbotapi.NewBotAPIWithAPIEndpoint("test-token", server.URL+"/bot%s/%s")
	if err != nil {
		t.Fatal(err)
	}
	oldBot := core.Bot
	core.Bot = bot
	defer func() { core.Bot = oldBot }()

	oldDB := db.DB
	db.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.DB.AutoMigrate(&db.UserData{}); err != nil {
		t.Fatal(err)
	}
	defer func() { db.DB = oldDB }()

	oldTranslations := lib.TranslationsMap
	lib.TranslationsMap = map[string]lib.Translations{
		"zh": {PickDownloadFormat: "选择格式", Cancel: "取消"},
	}
	defer func() { lib.TranslationsMap = oldTranslations }()

	oldErrors := utils.RuntimeStatus.Errors.Load()
	oldRequestErrors := utils.RuntimeStatus.RequestErrors.Load()
	defer func() {
		utils.RuntimeStatus.Errors.Store(oldErrors)
		utils.RuntimeStatus.RequestErrors.Store(oldRequestErrors)
	}()

	update := tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{
		ID: "callback-id",
		Message: &tgbotapi.Message{
			MessageID: 7,
			Chat:      &tgbotapi.Chat{ID: 42},
			ReplyToMessage: &tgbotapi.Message{
				MessageID: 6,
				Text:      "https://t.me/addstickers/example",
			},
		},
	}}

	if err := (MessageSender{}).ThisFormatChose(update); err != nil {
		t.Fatal(err)
	}
	if got := utils.RuntimeStatus.Errors.Load(); got != oldErrors+1 {
		t.Fatalf("Errors = %d, want %d", got, oldErrors+1)
	}
	if got := utils.RuntimeStatus.RequestErrors.Load(); got != oldRequestErrors+1 {
		t.Fatalf("RequestErrors = %d, want %d", got, oldRequestErrors+1)
	}
	if err := (MessageSender{}).ZipFormatChose(update); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	gotMethods := append([]string(nil), methods...)
	mu.Unlock()
	if len(gotMethods) != 3 || gotMethods[0] != "getMe" || gotMethods[1] != "answerCallbackQuery" || gotMethods[2] != "editMessageText" {
		t.Fatalf("API calls = %v", gotMethods)
	}
}
