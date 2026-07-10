package runtimeguard

import (
	"sync"
	"testing"
	"time"
)

func TestTaskPanicDoesNotSignalFatal(t *testing.T) {
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
}

func TestCriticalPanicSignalsFatal(t *testing.T) {
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
}
