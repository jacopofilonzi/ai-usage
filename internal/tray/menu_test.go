package tray

import (
	"testing"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/i18n"
	"github.com/jacopofilonzi/ai-usage/internal/wakelock"
)

type recorder struct{ calls []string }

func (r *recorder) actions() Actions {
	add := func(s string) { r.calls = append(r.calls, s) }
	return Actions{
		SetProvider:     func(id string) { add("provider " + id) },
		StartWake:       func(m int) { add("wake " + time.Duration(m*int(time.Minute)).String()) },
		StopWake:        func() { add("stop") },
		SetKeepUnlocked: func(on bool) { add("keep " + map[bool]string{true: "on", false: "off"}[on]) },
		OpenSettings:    func() { add("settings") },
		Quit:            func() { add("quit") },
	}
}

func find(items []item, label string) *item {
	for i := range items {
		if items[i].label == label {
			return &items[i]
		}
		if it := find(items[i].sub, label); it != nil {
			return it
		}
	}
	return nil
}

func TestSpec(t *testing.T) {
	m := i18n.For("en")
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.Local)
	until := now.Add(25 * time.Minute)
	s := State{
		Msg:       m,
		Providers: []Provider{{ID: "claude", Name: "Claude Code", Active: true}, {ID: "codex", Name: "Codex"}},
		Wake:      wakelock.Status{Active: true, Until: &until, RemainingS: 1500},
	}
	r := &recorder{}
	items := spec(s, now, r.actions())

	if items[0].kind != kindDisabled || items[0].label != "Awake until 10:25 (25m left)" {
		t.Errorf("status line = %+v", items[0])
	}
	if it := find(items, "Codex"); it == nil || it.kind != kindRadio || it.checked {
		t.Fatalf("codex item = %+v", it)
	} else {
		it.do()
	}
	if it := find(items, "Claude Code"); it == nil || !it.checked {
		t.Errorf("active provider not checked: %+v", it)
	}
	for _, label := range []string{"5 minutes", "1 hour", "2 hours", "Indefinitely", "Stop", "Keep unlocked (screen on)", "Settings…", "Quit"} {
		it := find(items, label)
		if it == nil {
			t.Fatalf("missing %q", label)
		}
		it.do()
	}
	want := []string{"provider codex", "wake 5m0s", "wake 1h0m0s", "wake 2h0m0s", "wake 0s", "stop", "keep on", "settings", "quit"}
	if len(r.calls) != len(want) {
		t.Fatalf("calls = %v", r.calls)
	}
	for i := range want {
		if r.calls[i] != want[i] {
			t.Errorf("call %d = %q, want %q", i, r.calls[i], want[i])
		}
	}
}

func TestSpecInactiveAndKey(t *testing.T) {
	m := i18n.For("it")
	now := time.Now()
	s := State{Msg: m}
	items := spec(s, now, (&recorder{}).actions())
	if find(items, m.NoProvider) == nil {
		t.Error("no-provider line missing")
	}
	if it := find(items, m.Stop); it == nil || it.kind != kindDisabled {
		t.Errorf("Stop must be disabled while inactive: %+v", it)
	}
	k := key(items)
	if key(spec(s, now.Add(time.Hour), (&recorder{}).actions())) != k {
		t.Error("key changed although nothing visible changed")
	}
	s.Wake.KeepUnlocked = true
	if key(spec(s, now, (&recorder{}).actions())) == k {
		t.Error("key ignored the checkbox")
	}
}
