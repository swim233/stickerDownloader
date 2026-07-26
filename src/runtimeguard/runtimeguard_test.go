package runtimeguard

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/swim233/StickerDownloader/notify"
	"github.com/swim233/StickerDownloader/utils"
)

func preserveErrorCounters(t *testing.T) {
	t.Helper()
	errors := utils.RuntimeStatus.Errors.Load()
	panics := utils.RuntimeStatus.PanicErrors.Load()
	notifications := utils.RuntimeStatus.NotificationErrors.Load()
	t.Cleanup(func() {
		utils.RuntimeStatus.Errors.Store(errors)
		utils.RuntimeStatus.PanicErrors.Store(panics)
		utils.RuntimeStatus.NotificationErrors.Store(notifications)
	})
}

func TestTaskPanicDoesNotSignalFatal(t *testing.T) {
	preserveErrorCounters(t)
	beforeErrors := utils.RuntimeStatus.Errors.Load()
	beforePanics := utils.RuntimeStatus.PanicErrors.Load()
	fatal := make(chan error, 1)
	var wg sync.WaitGroup
	guard := &Guard{Fatal: fatal, WaitGroup: &wg, StartTime: time.Now()}
	guard.Go("task", Task, func() { panic("boom") })
	wg.Wait()
	select {
	case err := <-fatal:
		t.Fatalf("unexpected fatal error: %v", err)
	default:
	}
	if got := utils.RuntimeStatus.Errors.Load(); got != beforeErrors+1 {
		t.Fatalf("Errors = %d, want %d", got, beforeErrors+1)
	}
	if got := utils.RuntimeStatus.PanicErrors.Load(); got != beforePanics+1 {
		t.Fatalf("PanicErrors = %d, want %d", got, beforePanics+1)
	}
}

func TestCriticalPanicSignalsFatal(t *testing.T) {
	preserveErrorCounters(t)
	beforePanics := utils.RuntimeStatus.PanicErrors.Load()
	fatal := make(chan error, 1)
	var wg sync.WaitGroup
	guard := &Guard{Fatal: fatal, WaitGroup: &wg, StartTime: time.Now()}
	guard.Go("critical", Critical, func() { panic("boom") })
	wg.Wait()
	select {
	case <-fatal:
	case <-time.After(time.Second):
		t.Fatal("missing fatal signal")
	}
	if got := utils.RuntimeStatus.PanicErrors.Load(); got != beforePanics+1 {
		t.Fatalf("PanicErrors = %d, want %d", got, beforePanics+1)
	}
}

func TestPanicNotificationSuccessDoesNotRecordFailure(t *testing.T) {
	preserveErrorCounters(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer server.Close()

	notifier := notify.New(notify.Config{
		Token:       "fake-token",
		OwnerChatID: 1,
		APIEndpoint: server.URL + "/bot%s/%s",
	})
	beforePanics := utils.RuntimeStatus.PanicErrors.Load()
	beforeNotifications := utils.RuntimeStatus.NotificationErrors.Load()
	guard := &Guard{Notifier: notifier, StartTime: time.Now()}

	guard.handle("task", Task, "boom", []byte("stack"))

	if got := utils.RuntimeStatus.PanicErrors.Load(); got != beforePanics+1 {
		t.Fatalf("PanicErrors = %d, want %d", got, beforePanics+1)
	}
	if got := utils.RuntimeStatus.NotificationErrors.Load(); got != beforeNotifications {
		t.Fatalf("NotificationErrors changed: %d -> %d", beforeNotifications, got)
	}
}

func TestPanicNotificationFailureIsRecorded(t *testing.T) {
	preserveErrorCounters(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"ok":false,"description":"unavailable"}`))
	}))
	defer server.Close()

	notifier := notify.New(notify.Config{
		Token:       "fake-token",
		OwnerChatID: 1,
		APIEndpoint: server.URL + "/bot%s/%s",
		Timeout:     time.Second,
	})
	beforeErrors := utils.RuntimeStatus.Errors.Load()
	beforePanics := utils.RuntimeStatus.PanicErrors.Load()
	beforeNotifications := utils.RuntimeStatus.NotificationErrors.Load()
	fatal := make(chan error, 1)
	guard := &Guard{
		Notifier:   notifier,
		Fatal:      fatal,
		StartTime:  time.Now(),
		Generation: "1",
	}

	guard.handle("critical", Critical, "boom", []byte("stack"))

	if got := utils.RuntimeStatus.Errors.Load(); got != beforeErrors+2 {
		t.Fatalf("Errors = %d, want %d", got, beforeErrors+2)
	}
	if got := utils.RuntimeStatus.PanicErrors.Load(); got != beforePanics+1 {
		t.Fatalf("PanicErrors = %d, want %d", got, beforePanics+1)
	}
	if got := utils.RuntimeStatus.NotificationErrors.Load(); got != beforeNotifications+1 {
		t.Fatalf("NotificationErrors = %d, want %d", got, beforeNotifications+1)
	}
	select {
	case <-fatal:
	case <-time.After(time.Second):
		t.Fatal("notification failure blocked fatal signal")
	}
}

func TestWrapPanicReportsOnce(t *testing.T) {
	preserveErrorCounters(t)
	beforePanics := utils.RuntimeStatus.PanicErrors.Load()
	guard := &Guard{StartTime: time.Now()}
	err := guard.Wrap("callback", func() error { panic("boom") })()
	if err == nil {
		t.Fatal("expected wrapped panic error")
	}
	if got := utils.RuntimeStatus.PanicErrors.Load(); got != beforePanics+1 {
		t.Fatalf("PanicErrors = %d, want %d", got, beforePanics+1)
	}
}
