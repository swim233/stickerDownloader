package runtimeguard

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/swim233/StickerDownloader/logger"
	"github.com/swim233/StickerDownloader/notify"
	"github.com/swim233/StickerDownloader/utils"
)

type Severity int

const (
	Task Severity = iota
	Critical
)

type Guard struct {
	Notifier      *notify.Telegram
	Fatal         chan<- error
	RunID         string
	Generation    string
	StartTime     time.Time
	NotifyTimeout time.Duration
	WaitGroup     *sync.WaitGroup
}

type PanicError struct {
	Component string
	Recovered any
	Stack     []byte
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("critical component %s panic: %v", e.Component, e.Recovered)
}

var defaultGuard atomic.Pointer[Guard]

func SetDefault(guard *Guard) {
	defaultGuard.Store(guard)
}

func Go(name string, severity Severity, fn func()) {
	guard := defaultGuard.Load()
	if guard == nil {
		guard = &Guard{StartTime: time.Now()}
	}
	guard.Go(name, severity, fn)
}

func (g *Guard) Go(name string, severity Severity, fn func()) {
	if g != nil && g.WaitGroup != nil {
		g.WaitGroup.Add(1)
	}
	go func() {
		if g != nil && g.WaitGroup != nil {
			defer g.WaitGroup.Done()
		}
		defer g.recover(name, severity)
		fn()
	}()
}

func (g *Guard) Wrap(name string, fn func() error) func() error {
	return func() (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				g.handle(name, Task, recovered, debug.Stack())
				err = fmt.Errorf("%s panic: %v", name, recovered)
			}
		}()
		return fn()
	}
}

func (g *Guard) recover(name string, severity Severity) {
	if recovered := recover(); recovered != nil {
		g.handle(name, severity, recovered, debug.Stack())
	}
}

func (g *Guard) handle(name string, severity Severity, recovered any, stack []byte) {
	utils.RuntimeStatus.Errors.Add(1)
	func() {
		defer func() {
			if recover() != nil {
				log.Printf("组件 %s panic: %v\n%s", name, recovered, stack)
			}
		}()
		logger.Error("组件 %s panic: %v\n%s", name, recovered, stack)
	}()

	if g != nil && g.Notifier != nil {
		metadata := fmt.Sprintf("Run: %s\nGeneration: %s\nUptime: %s\nSeverity: %d", g.RunID, g.Generation, time.Since(g.StartTime).Round(time.Second), severity)
		timeout := g.NotifyTimeout
		if timeout <= 0 {
			timeout = 8 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		_ = g.Notifier.SendPanic(ctx, name, recovered, stack, metadata)
		cancel()
	}
	if severity == Critical && g != nil && g.Fatal != nil {
		select {
		case g.Fatal <- &PanicError{Component: name, Recovered: recovered, Stack: stack}:
		default:
		}
	}
}
