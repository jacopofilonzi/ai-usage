package claude

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

// fixtureNow is before the fixture token's expiresAt (1792098093224 ms = 2026-10-15T15:01:33Z).
var fixtureNow = time.Date(2026, 10, 15, 14, 0, 0, 0, time.UTC)

const fixtureToken = "sk-ant-oat01-FIXTURE"

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

// newTestProvider returns a provider whose config dir holds the given credentials (nil = none) and
// that talks to srv.
func newTestProvider(t *testing.T, creds []byte, srv *httptest.Server) *Provider {
	t.Helper()
	dir := t.TempDir()
	if creds != nil {
		if err := os.WriteFile(filepath.Join(dir, credentialsFile), creds, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	p := New()
	p.configDir = func() (string, error) { return dir, nil }
	p.readCredentials = readCredentialsFile
	p.now = func() time.Time { return fixtureNow }
	if srv != nil {
		p.baseURL = srv.URL
		p.client = srv.Client()
	}
	return p
}

func TestParseUsageReal(t *testing.T) {
	limits, err := parseUsage(readFixture(t, "usage_real.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(limits) != 3 {
		t.Fatalf("got %d limits, want 3: %+v", len(limits), limits)
	}
	s := limits[0]
	if s.ID != "five_hour" || s.Kind != provider.KindSession || s.WindowMinutes != 300 || s.UsedPct != 15 {
		t.Errorf("session = %+v", s)
	}
	wantReset := time.Date(2026, 10, 15, 17, 59, 59, 979213000, time.UTC)
	if s.ResetsAt == nil || !s.ResetsAt.Equal(wantReset) {
		t.Errorf("session resetsAt = %v, want %v", s.ResetsAt, wantReset)
	}
	w := limits[1]
	if w.ID != "seven_day" || w.Kind != provider.KindWeekly || w.WindowMinutes != 10080 || w.UsedPct != 58 {
		t.Errorf("weekly = %+v", w)
	}
	o := limits[2]
	if o.ID != "iguana_necktie" || o.Kind != provider.KindOther || o.WindowMinutes != 0 || o.ResetsAt == nil {
		t.Errorf("other = %+v", o)
	}
}

func TestParseUsageModelsAndOddities(t *testing.T) {
	limits, err := parseUsage(readFixture(t, "usage_models.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, l := range limits {
		ids = append(ids, l.ID)
	}
	want := "five_hour,seven_day,seven_day_opus,seven_day_sonnet,brand_new_window"
	if got := strings.Join(ids, ","); got != want {
		t.Fatalf("order = %s, want %s", got, want)
	}
	if limits[0].UsedPct != 104.5 {
		t.Errorf("over-limit utilization not preserved: %v", limits[0].UsedPct)
	}
	if limits[1].ResetsAt != nil {
		t.Errorf("null resets_at should be nil, got %v", limits[1].ResetsAt)
	}
	opus := limits[2]
	if opus.Kind != provider.KindWeeklyModel || opus.Model != "opus" || opus.WindowMinutes != 10080 || opus.UsedPct != 40.25 {
		t.Errorf("opus = %+v", opus)
	}
	if limits[4].Kind != provider.KindOther || limits[4].ResetsAt != nil {
		t.Errorf("unknown window with bad date = %+v", limits[4])
	}
}

func TestParseUsageInvalid(t *testing.T) {
	if _, err := parseUsage([]byte(`[1,2]`)); err == nil {
		t.Error("expected error for non-object body")
	}
}

func TestPollSuccessSendsHeaders(t *testing.T) {
	body := readFixture(t, "usage_real.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != usagePath || r.Method != http.MethodGet {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+fixtureToken {
			t.Errorf("Authorization header not as expected")
		}
		if got := r.Header.Get("anthropic-beta"); got != betaHeader {
			t.Errorf("anthropic-beta = %q", got)
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent")
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	p := newTestProvider(t, readFixture(t, "credentials.json"), srv)
	limits, err := p.Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(limits) != 3 {
		t.Fatalf("got %d limits", len(limits))
	}
}

func TestPollNotLoggedIn(t *testing.T) {
	cases := map[string][]byte{
		"no file":     nil,
		"no oauth":    []byte(`{"other":{}}`),
		"empty token": []byte(`{"claudeAiOauth":{"accessToken":""}}`),
	}
	for name, creds := range cases {
		t.Run(name, func(t *testing.T) {
			p := newTestProvider(t, creds, nil)
			p.baseURL = "http://127.0.0.1:0" // must never be reached
			_, err := p.Poll(context.Background())
			if !errors.Is(err, provider.ErrNotLoggedIn) {
				t.Fatalf("err = %v, want ErrNotLoggedIn", err)
			}
		})
	}
}

func TestPollMalformedCredentialsDoesNotLeak(t *testing.T) {
	p := newTestProvider(t, []byte(`{"claudeAiOauth":{"accessToken":"sk-secret-value"`), nil)
	_, err := p.Poll(context.Background())
	if err == nil || errors.Is(err, provider.ErrNotLoggedIn) {
		t.Fatalf("err = %v, want generic error", err)
	}
	if strings.Contains(err.Error(), "sk-secret") {
		t.Fatalf("error leaks the token: %v", err)
	}
}

func TestPollExpiredTokenSkipsRequest(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	defer srv.Close()

	p := newTestProvider(t, readFixture(t, "credentials.json"), srv)
	p.now = func() time.Time { return time.UnixMilli(1792098093224) } // exactly at expiry
	_, err := p.Poll(context.Background())
	if !errors.Is(err, provider.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("expired token still made %d requests", calls.Load())
	}
}

func TestPollHTTPErrors(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		retryAfter string
		check      func(t *testing.T, err error)
	}{
		{"401", 401, "", func(t *testing.T, err error) {
			if !errors.Is(err, provider.ErrUnauthorized) {
				t.Errorf("err = %v", err)
			}
		}},
		{"403", 403, "", func(t *testing.T, err error) {
			if !errors.Is(err, provider.ErrUnauthorized) {
				t.Errorf("err = %v", err)
			}
		}},
		{"429 seconds", 429, "42", func(t *testing.T, err error) {
			var rl *provider.RateLimitedError
			if !errors.As(err, &rl) || rl.RetryAfter != 42*time.Second {
				t.Errorf("err = %v", err)
			}
		}},
		{"429 date", 429, fixtureNow.Add(90 * time.Second).Format(http.TimeFormat), func(t *testing.T, err error) {
			var rl *provider.RateLimitedError
			if !errors.As(err, &rl) || rl.RetryAfter != 90*time.Second {
				t.Errorf("err = %v", err)
			}
		}},
		{"429 none", 429, "", func(t *testing.T, err error) {
			var rl *provider.RateLimitedError
			if !errors.As(err, &rl) || rl.RetryAfter != 0 {
				t.Errorf("err = %v", err)
			}
		}},
		{"500", 500, "", func(t *testing.T, err error) {
			var rl *provider.RateLimitedError
			if err == nil || errors.Is(err, provider.ErrUnauthorized) || errors.As(err, &rl) {
				t.Errorf("err = %v", err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.retryAfter != "" {
					w.Header().Set("Retry-After", tc.retryAfter)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"error":"x"}`))
			}))
			defer srv.Close()
			p := newTestProvider(t, readFixture(t, "credentials.json"), srv)
			_, err := p.Poll(context.Background())
			tc.check(t, err)
			if err != nil && strings.Contains(err.Error(), fixtureToken) {
				t.Errorf("error leaks the token: %v", err)
			}
		})
	}
}

func TestDetect(t *testing.T) {
	p := New()
	dir := t.TempDir()
	p.configDir = func() (string, error) { return dir, nil }
	if !p.Detect() {
		t.Error("existing dir not detected")
	}
	p.configDir = func() (string, error) { return filepath.Join(dir, "missing"), nil }
	if p.Detect() {
		t.Error("missing dir detected")
	}
}

func TestDefaultConfigDirEnv(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join("x", "y"))
	got, err := defaultConfigDir()
	if err != nil || got != filepath.Join("x", "y") {
		t.Fatalf("got %q, %v", got, err)
	}
}
