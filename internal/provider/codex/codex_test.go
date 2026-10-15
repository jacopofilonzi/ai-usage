package codex

// The fixtures in testdata/ are synthesised (Codex is not installed on the dev machine) from the
// rollout format in the Codex source:
// https://github.com/openai/codex/blob/main/codex-rs/protocol/src/protocol.rs (TokenCountEvent,
// RateLimitSnapshot, RateLimitWindow) for current.jsonl, and from the 2025 format with
// "resets_in_seconds" and 299-minute windows documented by
// https://github.com/xiangz19/codex-ratelimit for legacy.jsonl.
//
// real_full.jsonl, swapped.jsonl and primary_only.jsonl reproduce the shapes seen in real rollouts
// published as test fixtures by other Codex usage tools (values and ids changed):
// full current fields (limit_id, plan_type, credits…) and a half-written last line from
// github.com/wadackel/dotfiles (home/programs/codex/fixtures/rate-limits-ok.jsonl), weekly window
// as primary and 5-hour as secondary from github.com/bbastia/quotabar
// (NativeApp/QuotaBarTests/Fixtures/codex-session.jsonl), a single window from github.com/crup/runrate.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

var testNow = time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

// writeRollout copies a fixture (or writes content) to <home>/sessions/<day>/<name> with the
// given modification time.
func writeRollout(t *testing.T, home, day, name string, content []byte, mod time.Time) string {
	t.Helper()
	dir := filepath.Join(append([]string{home, "sessions"}, strings.Split(day, "/")...)...)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
	return path
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func testProvider(home string) *Provider {
	p := New()
	p.home = func() (string, error) { return home, nil }
	p.now = func() time.Time { return testNow }
	return p
}

func TestPollCurrentFormat(t *testing.T) {
	home := t.TempDir()
	writeRollout(t, home, "2026/10/15", "rollout-2026-10-15T11-40-02-a.jsonl", fixture(t, "current.jsonl"), testNow)
	limits, err := testProvider(home).Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(limits) != 2 {
		t.Fatalf("got %+v", limits)
	}
	s, w := limits[0], limits[1]
	wantReset := time.Date(2026, 10, 15, 14, 30, 0, 0, time.UTC)
	if s.ID != "primary" || s.Kind != provider.KindSession || s.WindowMinutes != 300 || s.UsedPct != 42.5 ||
		s.ResetsAt == nil || !s.ResetsAt.Equal(wantReset) {
		t.Errorf("session = %+v", s)
	}
	if w.ID != "secondary" || w.Kind != provider.KindWeekly || w.WindowMinutes != 10080 || w.UsedPct != 18 || w.ResetsAt == nil {
		t.Errorf("weekly = %+v", w)
	}
}

func TestPollLegacyFormat(t *testing.T) {
	home := t.TempDir()
	writeRollout(t, home, "2026/10/15", "rollout-legacy.jsonl", fixture(t, "legacy.jsonl"), testNow)
	limits, err := testProvider(home).Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(limits) != 2 {
		t.Fatalf("got %+v", limits)
	}
	s, w := limits[0], limits[1]
	wantReset := time.Date(2026, 10, 15, 12, 30, 0, 0, time.UTC) // line timestamp 11:30 + 3600 s
	if s.Kind != provider.KindSession || s.UsedPct != 61 || s.ResetsAt == nil || !s.ResetsAt.Equal(wantReset) {
		t.Errorf("session = %+v", s)
	}
	// Resets at 11:30 + 1800 s = now: the window already rolled over.
	if w.Kind != provider.KindWeekly || w.UsedPct != 0 || w.ResetsAt != nil {
		t.Errorf("weekly = %+v", w)
	}
}

func TestPollRealShapes(t *testing.T) {
	cases := []struct {
		fixture string
		want    []provider.Limit // ResetsAt checked for presence only
	}{
		{"real_full.jsonl", []provider.Limit{
			{ID: "primary", Kind: provider.KindSession, WindowMinutes: 300, UsedPct: 32.4},
			{ID: "secondary", Kind: provider.KindWeekly, WindowMinutes: 10080, UsedPct: 5.6},
		}},
		// Order follows the window length, not primary/secondary.
		{"swapped.jsonl", []provider.Limit{
			{ID: "secondary", Kind: provider.KindSession, WindowMinutes: 300, UsedPct: 44.5},
			{ID: "primary", Kind: provider.KindWeekly, WindowMinutes: 10080, UsedPct: 11},
		}},
		{"primary_only.jsonl", []provider.Limit{
			{ID: "primary", Kind: provider.KindSession, WindowMinutes: 300, UsedPct: 1},
		}},
	}
	for _, c := range cases {
		t.Run(c.fixture, func(t *testing.T) {
			home := t.TempDir()
			writeRollout(t, home, "2026/10/15", "rollout-2026-10-15T11-20-00-x.jsonl", fixture(t, c.fixture), testNow)
			got, err := testProvider(home).Poll(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(c.want) {
				t.Fatalf("got %+v", got)
			}
			for i, w := range c.want {
				g := got[i]
				if g.ID != w.ID || g.Kind != w.Kind || g.WindowMinutes != w.WindowMinutes || g.UsedPct != w.UsedPct || g.ResetsAt == nil {
					t.Errorf("limit %d = %+v, want %+v with a reset time", i, g, w)
				}
			}
		})
	}
}

func TestPollNothingYet(t *testing.T) {
	t.Run("no sessions dir", func(t *testing.T) {
		limits, err := testProvider(t.TempDir()).Poll(context.Background())
		if err != nil || limits == nil || len(limits) != 0 {
			t.Fatalf("limits=%v err=%v", limits, err)
		}
	})
	t.Run("no snapshot", func(t *testing.T) {
		home := t.TempDir()
		writeRollout(t, home, "2026/10/15", "rollout-x.jsonl", fixture(t, "no_limits.jsonl"), testNow)
		limits, err := testProvider(home).Poll(context.Background())
		if err != nil || limits == nil || len(limits) != 0 {
			t.Fatalf("limits=%v err=%v", limits, err)
		}
	})
}

func TestLatestRolloutPicksNewestModified(t *testing.T) {
	home := t.TempDir()
	old := testNow.Add(-48 * time.Hour)
	writeRollout(t, home, "2026/09/30", "rollout-a.jsonl", []byte("{}\n"), old)
	// A session started yesterday and still running is newer than today's short one.
	want := writeRollout(t, home, "2026/10/14", "rollout-long.jsonl", []byte("{}\n"), testNow)
	writeRollout(t, home, "2026/10/15", "rollout-short.jsonl", []byte("{}\n"), testNow.Add(-time.Hour))
	// Compressed and unrelated files are ignored even when newer.
	writeRollout(t, home, "2026/10/15", "rollout-cold.jsonl.zst", []byte("x"), testNow.Add(time.Hour))
	writeRollout(t, home, "2026/10/15", "notes.jsonl", []byte("x"), testNow.Add(time.Hour))
	if err := os.MkdirAll(filepath.Join(home, "sessions", "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := latestRollout(context.Background(), filepath.Join(home, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestLatestRolloutOrdersNumerically(t *testing.T) {
	home := t.TempDir()
	// Lexical order would put "9" after "10"; Codex zero-pads, but numeric order is safe either way.
	want := writeRollout(t, home, "2026/10/01", "rollout-b.jsonl", []byte("{}\n"), testNow)
	for _, d := range []string{"2026/9/28", "2026/9/29", "2026/9/30"} {
		writeRollout(t, home, d, "rollout-a.jsonl", []byte("{}\n"), testNow.Add(-time.Hour))
	}
	got, err := latestRollout(context.Background(), filepath.Join(home, "sessions"))
	if err != nil || got != want {
		t.Fatalf("got %s (%v), want %s", got, err, want)
	}
}

func TestLastSnapshotExtendsTail(t *testing.T) {
	var b strings.Builder
	b.Write(fixture(t, "current.jsonl"))
	filler := `{"timestamp":"2026-10-15T11:53:00.000Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"` +
		strings.Repeat("x", 1000) + `"}]}}` + "\n"
	for b.Len() < 3*initialTail {
		b.WriteString(filler)
	}
	home := t.TempDir()
	path := writeRollout(t, home, "2026/10/15", "rollout-big.jsonl", []byte(b.String()), testNow)
	snap, err := lastSnapshot(path)
	if err != nil {
		t.Fatal(err)
	}
	if snap == nil || snap.raw.Primary == nil || snap.raw.Primary.UsedPercent != 42.5 {
		t.Fatalf("snapshot = %+v", snap)
	}
}

func TestKindFor(t *testing.T) {
	cases := map[int]provider.Kind{300: provider.KindSession, 299: provider.KindSession, 10080: provider.KindWeekly, 60: provider.KindOther, 1440: provider.KindOther}
	for m, want := range cases {
		if got := kindFor(m); got != want {
			t.Errorf("kindFor(%d) = %s, want %s", m, got, want)
		}
	}
}

func TestDetect(t *testing.T) {
	home := t.TempDir()
	if !testProvider(home).Detect() {
		t.Error("existing home not detected")
	}
	if testProvider(filepath.Join(home, "missing")).Detect() {
		t.Error("missing home detected")
	}
}

func TestDefaultHomeEnv(t *testing.T) {
	t.Setenv("CODEX_HOME", filepath.Join("a", "b"))
	got, err := defaultHome()
	if err != nil || got != filepath.Join("a", "b") {
		t.Fatalf("got %q, %v", got, err)
	}
}
