package claude

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

// desktopNewest is the newest sample's time in testdata/plan_usage_history.json (deliberately not
// the last one in the file).
var desktopNewest = time.UnixMilli(1794996450000)

func desktopFixture(t *testing.T) func() ([]byte, error) {
	t.Helper()
	raw := readFixture(t, "plan_usage_history.json")
	return func() ([]byte, error) { return raw, nil }
}

func TestParseDesktopHistoryNewestSample(t *testing.T) {
	limits, err := parseDesktopHistory(readFixture(t, "plan_usage_history.json"), desktopNewest.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(limits) != 2 {
		t.Fatalf("got %d limits, want 2: %+v", len(limits), limits)
	}
	s, w := limits[0], limits[1]
	if s.ID != keySession || s.Kind != provider.KindSession || s.WindowMinutes != sessionMinutes || s.UsedPct != 28 || s.ResetsAt != nil {
		t.Errorf("session = %+v", s)
	}
	if w.ID != keyWeekly || w.Kind != provider.KindWeekly || w.WindowMinutes != weeklyMinutes || w.UsedPct != 93 || w.ResetsAt != nil {
		t.Errorf("weekly = %+v", w)
	}
}

func TestParseDesktopHistoryRejectsOldOrEmpty(t *testing.T) {
	raw := readFixture(t, "plan_usage_history.json")
	cases := map[string]struct {
		raw []byte
		now time.Time
	}{
		"too old":     {raw, desktopNewest.Add(maxDesktopAge + time.Second)},
		"future":      {raw, desktopNewest.Add(-maxDesktopSkew - time.Second)},
		"no samples":  {[]byte(`{"version":2,"samples":[]}`), desktopNewest},
		"no figures":  {[]byte(`{"samples":[{"t":1794996450000,"u":{}}]}`), desktopNewest},
		"null fields": {[]byte(`{"samples":[{"t":1794996450000,"u":{"fh":null,"sd":null}}]}`), desktopNewest},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseDesktopHistory(tc.raw, tc.now); !errors.Is(err, errNoDesktopUsage) {
				t.Fatalf("err = %v, want errNoDesktopUsage", err)
			}
		})
	}
	if _, err := parseDesktopHistory([]byte(`{`), desktopNewest); err == nil || errors.Is(err, errNoDesktopUsage) {
		t.Fatalf("malformed JSON: err = %v, want decode error", err)
	}
}

func TestPollFallsBackToDesktop(t *testing.T) {
	unauthorized := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer unauthorized.Close()

	cases := map[string]struct {
		creds []byte
		srv   *httptest.Server
		now   time.Time
	}{
		"no credentials": {nil, nil, desktopNewest},
		"no token":       {[]byte(`{"claudeAiOauth":{"accessToken":""}}`), nil, desktopNewest},
		"expired token":  {readFixture(t, "credentials.json"), nil, desktopNewest}, // fixture token expired in 2026-10
		"rejected token": {readFixture(t, "credentials.json"), unauthorized, fixtureNow},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := newTestProvider(t, tc.creds, tc.srv)
			if tc.srv == nil {
				p.baseURL = "http://127.0.0.1:0" // must never be reached
			}
			p.now = func() time.Time { return tc.now }
			raw := readFixture(t, "plan_usage_history.json")
			if name == "rejected token" {
				// Shift the samples so they are recent at fixtureNow, before the token's expiry.
				raw = []byte(fmt.Sprintf(`{"samples":[{"t":%d,"u":{"fh":28,"sd":93}}]}`, fixtureNow.Add(-time.Minute).UnixMilli()))
			}
			p.readDesktopHistory = func() ([]byte, error) { return raw, nil }
			limits, err := p.Poll(context.Background())
			if err != nil {
				t.Fatalf("err = %v, want desktop limits", err)
			}
			if len(limits) != 2 || limits[0].UsedPct != 28 || limits[1].UsedPct != 93 {
				t.Fatalf("limits = %+v", limits)
			}
		})
	}
}

func TestPollKeepsTokenErrorWithoutDesktop(t *testing.T) {
	cases := map[string]func() ([]byte, error){
		"disabled": nil,
		"missing":  func() ([]byte, error) { return nil, fmt.Errorf("x: %w", fs.ErrNotExist) },
		"old":      desktopFixture(t),
		"broken":   func() ([]byte, error) { return []byte(`{`), nil },
	}
	for name, read := range cases {
		t.Run(name, func(t *testing.T) {
			p := newTestProvider(t, nil, nil)
			p.baseURL = "http://127.0.0.1:0"
			p.now = func() time.Time { return desktopNewest.Add(time.Hour) }
			p.readDesktopHistory = read
			if _, err := p.Poll(context.Background()); !errors.Is(err, provider.ErrNotLoggedIn) {
				t.Fatalf("err = %v, want ErrNotLoggedIn", err)
			}
		})
	}
}

func TestPollPrefersAPIOverDesktop(t *testing.T) {
	body := readFixture(t, "usage_real.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(body) }))
	defer srv.Close()
	p := newTestProvider(t, readFixture(t, "credentials.json"), srv)
	p.readDesktopHistory = func() ([]byte, error) {
		t.Error("desktop history read although the API answered")
		return nil, fs.ErrNotExist
	}
	limits, err := p.Poll(context.Background())
	if err != nil || len(limits) != 2 || limits[0].UsedPct != 15 {
		t.Fatalf("limits = %+v, err = %v", limits, err)
	}
}

func TestPollGenericErrorSkipsDesktop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	p := newTestProvider(t, readFixture(t, "credentials.json"), srv)
	p.readDesktopHistory = func() ([]byte, error) {
		t.Error("desktop history read on a server error")
		return nil, fs.ErrNotExist
	}
	if _, err := p.Poll(context.Background()); err == nil {
		t.Fatal("want error")
	}
}

func TestNewestFile(t *testing.T) {
	dir := t.TempDir()
	older, newer := filepath.Join(dir, "a.json"), filepath.Join(dir, "b.json")
	for path, body := range map[string]string{older: "old", newer: "new"} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(older, past, past); err != nil {
		t.Fatal(err)
	}
	got, err := newestFile([]string{filepath.Join(dir, "missing.json"), older, newer})
	if err != nil || string(got) != "new" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := newestFile([]string{filepath.Join(dir, "missing.json")}); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want ErrNotExist", err)
	}
}
