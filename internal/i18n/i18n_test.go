package i18n

import (
	"reflect"
	"testing"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

func TestNoEmptyStrings(t *testing.T) {
	for name, m := range map[string]Messages{"en": en, "it": it} {
		v := reflect.ValueOf(m)
		for i := range v.NumField() {
			if v.Field(i).String() == "" {
				t.Errorf("%s: %s is empty", name, v.Type().Field(i).Name)
			}
		}
	}
}

func TestFor(t *testing.T) {
	cases := map[string]*Messages{"it-IT": &it, "it_IT.UTF-8": &it, "IT": &it, "en-US": &en, "": &en, "de": &en}
	for lang, want := range cases {
		if got := For(lang); got != want {
			t.Errorf("For(%q) picked the wrong language", lang)
		}
	}
}

func TestSelect(t *testing.T) {
	if Select("it") != &it || Select("en") != &en {
		t.Error("forced language ignored")
	}
	if got := Select("system"); got != &it && got != &en {
		t.Error("system language not resolved")
	}
}

func TestDuration(t *testing.T) {
	cases := map[time.Duration]string{5 * time.Minute: "5 minutes", time.Hour: "1 hour", 2 * time.Hour: "2 hours"}
	for d, want := range cases {
		if got := en.Duration(d); got != want {
			t.Errorf("Duration(%s) = %q, want %q", d, got, want)
		}
	}
}

func TestRemaining(t *testing.T) {
	cases := map[time.Duration]string{
		-time.Second:                          "0s",
		45 * time.Second:                      "45s",
		time.Minute:                           "1m",
		59*time.Second + 990*time.Millisecond: "1m",
		11*time.Minute + time.Second:          "12m",
		65 * time.Minute:                      "1h05m",
		2 * time.Hour:                         "2h00m",
	}
	for d, want := range cases {
		if got := Remaining(d); got != want {
			t.Errorf("Remaining(%s) = %q, want %q", d, got, want)
		}
	}
}

func TestWakeStatus(t *testing.T) {
	now := time.Date(2026, 3, 1, 14, 0, 0, 0, time.Local)
	if got := en.WakeStatus(false, false, time.Time{}, now); got != "Sleep allowed" {
		t.Errorf("off: %q", got)
	}
	if got := en.WakeStatus(true, true, time.Time{}, now); got != "Awake indefinitely" {
		t.Errorf("indefinite: %q", got)
	}
	if got := en.WakeStatus(true, false, now.Add(30*time.Minute), now); got != "Awake until 14:30 (30m left)" {
		t.Errorf("timed: %q", got)
	}
	if got := it.WakeStatus(true, false, now.Add(24*time.Hour), now); got != "Sveglio fino alle 2026-03-02 14:00 (mancano 24h00m)" {
		t.Errorf("other day: %q", got)
	}
}

func TestLimitLabel(t *testing.T) {
	cases := []struct {
		l      provider.Limit
		en, it string
	}{
		{provider.Limit{Kind: provider.KindSession, WindowMinutes: 300}, "5h", "5h"},
		{provider.Limit{Kind: provider.KindWeekly, WindowMinutes: 10080}, "7d", "7g"},
		{provider.Limit{Kind: provider.KindWeeklyModel, Model: "opus", WindowMinutes: 10080}, "Opus 7d", "Opus 7g"},
		{provider.Limit{Kind: provider.KindOther, WindowMinutes: 90}, "90m", "90m"},
		{provider.Limit{Kind: provider.KindSession}, "Session", "Sessione"},
	}
	for _, c := range cases {
		if got := en.LimitLabel(c.l); got != c.en {
			t.Errorf("en LimitLabel(%+v) = %q, want %q", c.l, got, c.en)
		}
		if got := it.LimitLabel(c.l); got != c.it {
			t.Errorf("it LimitLabel(%+v) = %q, want %q", c.l, got, c.it)
		}
	}
}

func TestTooltip(t *testing.T) {
	snap := provider.Snapshot{Status: provider.StatusOK, Limits: []provider.Limit{
		{Kind: provider.KindSession, WindowMinutes: 300, UsedPct: 42.4},
		{Kind: provider.KindWeekly, WindowMinutes: 10080, UsedPct: 16.6},
	}}
	if got := en.Tooltip("Claude Code", snap); got != "Claude Code · 5h 42% · 7d 17%" {
		t.Errorf("ok: %q", got)
	}
	snap.Status = provider.StatusStale
	if got := en.Tooltip("Claude Code", snap); got != "Claude Code · 5h 42% · 7d 17% · not updated" {
		t.Errorf("stale: %q", got)
	}
	if got := en.Tooltip("", provider.Snapshot{}); got != en.NoProvider {
		t.Errorf("none: %q", got)
	}
}
