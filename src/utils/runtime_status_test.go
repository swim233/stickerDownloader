package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/ijnkawakaze/telegram-bot-api"
	"github.com/swim233/StickerDownloader/core"
	"github.com/swim233/StickerDownloader/logger"
)

func TestFormatRuntimeStatusIncludesErrorCategories(t *testing.T) {
	start := time.Date(2026, 7, 20, 10, 0, 0, 0, time.Local)
	text := formatRuntimeStatus(runtimeStatusSnapshot{
		startTime:          start,
		singleDownload:     10,
		packDownload:       2,
		httpPackDownload:   2,
		cacheHits:          1,
		panicErrors:        1,
		requestErrors:      2,
		downloadErrors:     3,
		notificationErrors: 4,
	}, start.Add(time.Hour+2*time.Minute+3*time.Second))

	for _, want := range []string{
		"本次运行时间 : 1时2分3秒",
		"缓存命中率 : 25.0%",
		"当前 Worker 错误总数 : 10",
		"Panic : 1",
		"过期/请求 : 2",
		"下载 : 3",
		"通知失败 : 4",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("status text missing %q:\n%s", want, text)
		}
	}
}

func TestSendRuntimeStatusInfoIgnoresMissingMessage(t *testing.T) {
	if err := SendRuntimeStatusInfo(tgbotapi.Update{}); err != nil {
		t.Fatal(err)
	}
	if err := SendRuntimeStatusInfo(tgbotapi.Update{Message: &tgbotapi.Message{}}); err != nil {
		t.Fatal(err)
	}
}

func TestSendRuntimeStatusInfoRecordsSendFailure(t *testing.T) {
	logger.InitLogger()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/getMe") {
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"test","username":"test_bot"}}`))
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"ok":false,"description":"unavailable"}`))
	}))
	defer server.Close()

	bot, err := tgbotapi.NewBotAPIWithAPIEndpoint("test-token", server.URL+"/bot%s/%s")
	if err != nil {
		t.Fatal(err)
	}
	oldBot := core.Bot
	core.Bot = bot
	defer func() { core.Bot = oldBot }()

	oldErrors := RuntimeStatus.Errors.Load()
	oldNotifications := RuntimeStatus.NotificationErrors.Load()
	oldStart := RuntimeStatus.StartTime
	RuntimeStatus.StartTime = time.Now()
	defer func() {
		RuntimeStatus.Errors.Store(oldErrors)
		RuntimeStatus.NotificationErrors.Store(oldNotifications)
		RuntimeStatus.StartTime = oldStart
	}()

	err = SendRuntimeStatusInfo(tgbotapi.Update{Message: &tgbotapi.Message{
		From: &tgbotapi.User{ID: 42},
	}})
	if err == nil {
		t.Fatal("expected send error")
	}
	if got := RuntimeStatus.Errors.Load(); got != oldErrors+1 {
		t.Fatalf("Errors = %d, want %d", got, oldErrors+1)
	}
	if got := RuntimeStatus.NotificationErrors.Load(); got != oldNotifications+1 {
		t.Fatalf("NotificationErrors = %d, want %d", got, oldNotifications+1)
	}
}
