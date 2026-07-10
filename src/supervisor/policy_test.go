package supervisor

import (
	"testing"
	"time"

	"github.com/swim233/StickerDownloader/config"
)

func TestPolicyBackoffAndCooldown(t *testing.T) {
	settings := config.RestartSettings{
		InitialDelay: time.Second,
		MaxDelay:     time.Minute,
		Multiplier:   2,
		StableAfter:  10 * time.Minute,
		MaxRestarts:  3,
		Window:       5 * time.Minute,
		Cooldown:     15 * time.Minute,
	}
	policy := NewPolicy(settings)
	now := time.Now()

	delay, cooldown := policy.RecordCrash(now, now.Add(time.Second))
	if delay != time.Second || cooldown {
		t.Fatalf("first crash: delay=%s cooldown=%v", delay, cooldown)
	}
	delay, cooldown = policy.RecordCrash(now, now.Add(2*time.Second))
	if delay != 2*time.Second || cooldown {
		t.Fatalf("second crash: delay=%s cooldown=%v", delay, cooldown)
	}
	delay, cooldown = policy.RecordCrash(now, now.Add(3*time.Second))
	if delay != 15*time.Minute || !cooldown {
		t.Fatalf("third crash: delay=%s cooldown=%v", delay, cooldown)
	}
}

func TestPolicyResetsAfterStableRun(t *testing.T) {
	settings := config.RestartSettings{
		InitialDelay: time.Second,
		MaxDelay:     time.Minute,
		Multiplier:   2,
		StableAfter:  10 * time.Minute,
		MaxRestarts:  5,
		Window:       5 * time.Minute,
		Cooldown:     15 * time.Minute,
	}
	policy := NewPolicy(settings)
	now := time.Now()
	_, _ = policy.RecordCrash(now, now.Add(time.Second))
	delay, cooldown := policy.RecordCrash(now, now.Add(11*time.Minute))
	if delay != time.Second || cooldown {
		t.Fatalf("stable reset: delay=%s cooldown=%v", delay, cooldown)
	}
}

func TestShouldRestart(t *testing.T) {
	if ShouldRestart(ExitOK) || ShouldRestart(ExitUsage) || ShouldRestart(ExitConfig) {
		t.Fatal("permanent/normal exits must not restart")
	}
	if !ShouldRestart(ExitCrash) || !ShouldRestart(ExitTemporary) {
		t.Fatal("crash/temporary exits must restart")
	}
}
