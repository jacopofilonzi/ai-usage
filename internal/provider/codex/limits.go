package codex

import (
	"sort"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

const (
	sessionMinutes = 5 * 60
	weeklyMinutes  = 7 * 24 * 60
)

// snapshot is one rate_limits object together with the timestamp of the line it came from.
type snapshot struct {
	at  time.Time
	raw rateLimitsRaw
}

// limits converts the snapshot, ordered session, weekly, other (by window length).
func (s *snapshot) limits(now time.Time) []provider.Limit {
	out := make([]provider.Limit, 0, 2)
	for _, w := range []struct {
		id  string
		raw *windowRaw
	}{{"primary", s.raw.Primary}, {"secondary", s.raw.Secondary}} {
		if w.raw == nil {
			continue
		}
		out = append(out, s.limit(w.id, w.raw, now))
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := kindRank(out[i].Kind), kindRank(out[j].Kind)
		if ri != rj {
			return ri < rj
		}
		return out[i].WindowMinutes < out[j].WindowMinutes
	})
	return out
}

func (s *snapshot) limit(id string, w *windowRaw, now time.Time) provider.Limit {
	l := provider.Limit{ID: id, Kind: provider.KindOther, UsedPct: w.UsedPercent}
	if w.WindowMinutes != nil && *w.WindowMinutes > 0 {
		l.WindowMinutes = int(*w.WindowMinutes)
		l.Kind = kindFor(l.WindowMinutes)
	}
	var reset *time.Time
	switch {
	case w.ResetsAt != nil:
		t := time.Unix(*w.ResetsAt, 0).UTC()
		reset = &t
	case w.ResetsInSeconds != nil && !s.at.IsZero():
		t := s.at.Add(time.Duration(*w.ResetsInSeconds) * time.Second).UTC()
		reset = &t
	}
	if reset != nil && !reset.After(now) {
		// The window rolled over after Codex last wrote: nothing is used in the new one yet.
		l.UsedPct = 0
		reset = nil
	}
	l.ResetsAt = reset
	return l
}

// kindFor matches window lengths with some tolerance: older Codex versions reported 299 minutes
// for the 5-hour window.
func kindFor(minutes int) provider.Kind {
	switch {
	case abs(minutes-sessionMinutes) <= 5:
		return provider.KindSession
	case abs(minutes-weeklyMinutes) <= 60:
		return provider.KindWeekly
	default:
		return provider.KindOther
	}
}

func kindRank(k provider.Kind) int {
	switch k {
	case provider.KindSession:
		return 0
	case provider.KindWeekly:
		return 1
	default:
		return 2
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
