package supervisor

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/swim233/StickerDownloader/config"
)

type Policy struct {
	settings config.RestartSettings
	crashes  []time.Time
	streak   int
}

func NewPolicy(settings config.RestartSettings) *Policy {
	return &Policy{settings: settings}
}

func (p *Policy) RecordCrash(startedAt, now time.Time) (delay time.Duration, cooldown bool) {
	if now.Sub(startedAt) >= p.settings.StableAfter {
		p.streak = 0
		p.crashes = nil
	}
	p.streak++
	cutoff := now.Add(-p.settings.Window)
	kept := p.crashes[:0]
	for _, crash := range p.crashes {
		if !crash.Before(cutoff) {
			kept = append(kept, crash)
		}
	}
	p.crashes = append(kept, now)
	if len(p.crashes) >= p.settings.MaxRestarts {
		p.crashes = nil
		return p.settings.Cooldown, true
	}

	power := math.Pow(p.settings.Multiplier, float64(p.streak-1))
	delay = time.Duration(float64(p.settings.InitialDelay) * power)
	if delay > p.settings.MaxDelay {
		delay = p.settings.MaxDelay
	}
	if p.settings.Jitter > 0 {
		factor := 1 + ((rand.Float64()*2)-1)*p.settings.Jitter
		delay = time.Duration(float64(delay) * factor)
	}
	return delay, false
}

func ShouldRestart(exitCode int) bool {
	return exitCode != ExitOK && exitCode != ExitUsage && exitCode != ExitConfig
}
