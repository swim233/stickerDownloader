package lib

import "testing"

func TestRuntimeStatusRecordError(t *testing.T) {
	var status RuntimeStatus
	status.RecordError(RuntimeErrorPanic)
	status.RecordError(RuntimeErrorRequest)
	status.RecordError(RuntimeErrorRequest)
	status.RecordError(RuntimeErrorDownload)
	status.RecordError(RuntimeErrorNotification)

	if got := status.Errors.Load(); got != 5 {
		t.Fatalf("Errors = %d, want 5", got)
	}
	if got := status.PanicErrors.Load(); got != 1 {
		t.Fatalf("PanicErrors = %d, want 1", got)
	}
	if got := status.RequestErrors.Load(); got != 2 {
		t.Fatalf("RequestErrors = %d, want 2", got)
	}
	if got := status.DownloadErrors.Load(); got != 1 {
		t.Fatalf("DownloadErrors = %d, want 1", got)
	}
	if got := status.NotificationErrors.Load(); got != 1 {
		t.Fatalf("NotificationErrors = %d, want 1", got)
	}

	sum := status.PanicErrors.Load() + status.RequestErrors.Load() + status.DownloadErrors.Load() + status.NotificationErrors.Load()
	if status.Errors.Load() != sum {
		t.Fatalf("Errors = %d, category sum = %d", status.Errors.Load(), sum)
	}
}

func TestRuntimeStatusIgnoresUnknownErrorCategory(t *testing.T) {
	var status RuntimeStatus
	status.RecordError(RuntimeErrorCategory(99))
	if got := status.Errors.Load(); got != 0 {
		t.Fatalf("Errors = %d, want 0", got)
	}
}
