package supervisor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/swim233/StickerDownloader/config"
	"github.com/swim233/StickerDownloader/notify"
)

const (
	ExitOK        = 0
	ExitCrash     = 1
	ExitUsage     = 64
	ExitTemporary = 75
	ExitConfig    = 78
)

type Runner struct {
	Executable string
	ConfigPath string
	Settings   config.Settings
	Notifier   *notify.Telegram
	RunID      string

	mu      sync.Mutex
	current *exec.Cmd
}

func New(executable, configPath string, settings config.Settings, notifier *notify.Telegram) *Runner {
	return &Runner{
		Executable: executable,
		ConfigPath: configPath,
		Settings:   settings,
		Notifier:   notifier,
		RunID:      uuid.NewString(),
	}
}

func (r *Runner) Run(parent context.Context) int {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	policy := NewPolicy(r.Settings.Supervisor.Restart)
	generation := 0
	var stopOnce sync.Once

	for {
		if ctx.Err() != nil {
			stopOnce.Do(func() { r.notify("stopped", generation, "🛑 StickerDownloader 已正常停止") })
			return ExitOK
		}
		generation++
		startedAt := time.Now()
		cmd := exec.Command(r.Executable, "--worker", "--config", r.ConfigPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = append(os.Environ(),
			"STICKERDOWNLOADER_RUN_ID="+r.RunID,
			"STICKERDOWNLOADER_GENERATION="+strconv.Itoa(generation),
		)

		if generation > 1 {
			r.notify("restarting", generation, fmt.Sprintf("🔄 正在重启 StickerDownloader\nRun: %s\nGeneration: %d", r.RunID, generation))
		}
		if err := cmd.Start(); err != nil {
			log.Printf("启动 worker 失败: %v", err)
			r.notify("start-failed", generation, fmt.Sprintf("❌ Worker 启动失败\nGeneration: %d\n错误: %s", generation, safeError(err)))
			if !r.waitAfterCrash(ctx, policy, generation, startedAt) {
				stopOnce.Do(func() { r.notify("stopped", generation, "🛑 StickerDownloader 已正常停止") })
				return ExitOK
			}
			continue
		}
		r.setCurrent(cmd)

		waitCh := make(chan error, 1)
		go func() { waitCh <- cmd.Wait() }()
		var waitErr error
		select {
		case waitErr = <-waitCh:
			r.setCurrent(nil)
		case <-ctx.Done():
			r.forwardStop(cmd, waitCh)
			r.setCurrent(nil)
			stopOnce.Do(func() { r.notify("stopped", generation, "🛑 StickerDownloader 已正常停止") })
			return ExitOK
		}

		exitCode := commandExitCode(waitErr)
		if exitCode == ExitOK {
			stopOnce.Do(func() { r.notify("stopped", generation, "🛑 StickerDownloader 已正常停止") })
			return ExitOK
		}
		r.notify("crashed", generation, fmt.Sprintf("💥 Worker 意外退出\nRun: %s\nGeneration: %d\nPID: %d\n退出码: %d\n运行时长: %s", r.RunID, generation, cmd.Process.Pid, exitCode, time.Since(startedAt).Round(time.Second)))
		if !ShouldRestart(exitCode) {
			r.notify("permanent-failure", generation, fmt.Sprintf("⛔ Worker 遇到永久错误，不会自动重启\n退出码: %d", exitCode))
			return exitCode
		}
		if !r.waitAfterCrash(ctx, policy, generation, startedAt) {
			stopOnce.Do(func() { r.notify("stopped", generation, "🛑 StickerDownloader 已正常停止") })
			return ExitOK
		}
	}
}

func (r *Runner) waitAfterCrash(ctx context.Context, policy *Policy, generation int, startedAt time.Time) bool {
	delay, cooldown := policy.RecordCrash(startedAt, time.Now())
	if cooldown {
		r.notify("crash-loop", generation, fmt.Sprintf("⚠️ 检测到崩溃循环，暂停重启 %s", delay))
	} else {
		r.notify("restart-scheduled", generation, fmt.Sprintf("⏳ Worker 将在 %s 后重启", delay.Round(time.Millisecond)))
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func (r *Runner) forwardStop(cmd *exec.Cmd, waitCh <-chan error) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(syscall.SIGTERM)
	timer := time.NewTimer(r.Settings.Supervisor.ShutdownTimeout)
	defer timer.Stop()
	select {
	case <-waitCh:
		return
	case <-timer.C:
		_ = cmd.Process.Kill()
		<-waitCh
	}
}

func (r *Runner) setCurrent(cmd *exec.Cmd) {
	r.mu.Lock()
	r.current = cmd
	r.mu.Unlock()
}

func (r *Runner) notify(event string, generation int, text string) {
	if r.Notifier == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), r.Settings.Notification.RequestTimeout)
	defer cancel()
	key := fmt.Sprintf("%s:%d:%s", r.RunID, generation, event)
	if err := r.Notifier.SendEvent(ctx, key, text); err != nil {
		log.Printf("发送 Owner 通知失败: %v", err)
	}
}

func commandExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return ExitCrash
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%T", err)
}
