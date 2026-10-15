// Package claude reads Claude Code's usage limits.
//
// Design:
//
//   - Credentials: Claude Code keeps its OAuth token in <config dir>/.credentials.json (config dir is
//     $CLAUDE_CONFIG_DIR, else ~/.claude); on macOS it keeps the same JSON in the Keychain (generic
//     password, service "Claude Code-credentials"), see credentials_darwin.go. The file shape is
//     {"claudeAiOauth":{"accessToken","refreshToken","expiresAt" (ms epoch),"scopes",...}}.
//     The token is re-read on every Poll because Claude Code rewrites it whenever it refreshes; this
//     package never refreshes, stores, caches or logs it. An expired token is reported as
//     provider.ErrUnauthorized without calling the API at all.
//   - Endpoint: GET https://api.anthropic.com/api/oauth/usage (undocumented, the one behind Claude
//     Code's /usage) with the bearer token and "anthropic-beta: oauth-2025-04-20".
//   - Response (observed 2026-10, see testdata/usage_real.json): a flat object whose keys are usage
//     windows ("five_hour", "seven_day", "seven_day_<model>", plus codenamed ones) mapping to either
//     null or {"utilization": 0–100, "resets_at": ISO-8601|null, ...}, mixed with non-window keys
//     ("extra_usage", "limits", "spend", ...). Every key whose value is an object with a non-null
//     numeric "utilization" is treated as a window; everything else is ignored, so new keys never
//     break parsing.
package claude

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

const (
	defaultBaseURL = "https://api.anthropic.com"
	usagePath      = "/api/oauth/usage"
	betaHeader     = "oauth-2025-04-20"
	userAgent      = "ai-usage (usage limits viewer)"
	requestTimeout = 15 * time.Second
	// The real response is a few KiB; the cap only guards against a misbehaving server.
	maxBodyBytes = 1 << 20
)

// Provider reads Claude Code's usage limits. Unexported fields are injection points for tests.
type Provider struct {
	client    *http.Client
	baseURL   string
	now       func() time.Time
	configDir func() (string, error)
	// readCredentials returns the raw credentials JSON, or an error wrapping os.ErrNotExist when
	// there are none.
	readCredentials func(dir string) ([]byte, error)
}

// New returns a Provider that talks to the real API and reads the real config dir.
func New() *Provider {
	return &Provider{
		client:          &http.Client{Timeout: requestTimeout},
		baseURL:         defaultBaseURL,
		now:             time.Now,
		configDir:       defaultConfigDir,
		readCredentials: readCredentials,
	}
}

func (p *Provider) ID() string              { return "claude" }
func (p *Provider) Name() string            { return "Claude Code" }
func (p *Provider) Interval() time.Duration { return 2 * time.Minute }

// Detect reports whether Claude Code's config dir exists.
func (p *Provider) Detect() bool {
	dir, err := p.configDir()
	if err != nil {
		return false
	}
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// Poll reads the token from disk and asks the usage endpoint for the current limits.
func (p *Provider) Poll(ctx context.Context) ([]provider.Limit, error) {
	dir, err := p.configDir()
	if err != nil {
		return nil, fmt.Errorf("claude: locate config dir: %w", err)
	}
	creds, err := p.loadCredentials(dir)
	if err != nil {
		return nil, err
	}
	if creds.expired(p.now()) {
		return nil, fmt.Errorf("claude: token expired: %w", provider.ErrUnauthorized)
	}
	body, err := p.fetchUsage(ctx, creds.AccessToken)
	if err != nil {
		return nil, err
	}
	limits, err := parseUsage(body)
	if err != nil {
		return nil, fmt.Errorf("claude: parse usage response: %w", err)
	}
	return limits, nil
}

// defaultConfigDir mirrors Claude Code's own lookup.
func defaultConfigDir() (string, error) {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

// fetchUsage performs the request. Errors never include the token or the request headers.
func (p *Provider) fetchUsage(ctx context.Context, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+usagePath, nil)
	if err != nil {
		return nil, fmt.Errorf("claude: build usage request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", betaHeader)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		// *url.Error carries only method and URL, never headers, so it is safe to wrap.
		return nil, fmt.Errorf("claude: usage request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("claude: usage request returned %d: %w", resp.StatusCode, provider.ErrUnauthorized)
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, &provider.RateLimitedError{RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), p.now())}
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		// The body is not included: it is opaque and could echo request details.
		return nil, fmt.Errorf("claude: usage request returned unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("claude: read usage response: %w", err)
	}
	if len(body) > maxBodyBytes {
		return nil, errors.New("claude: usage response too large")
	}
	return body, nil
}

// parseRetryAfter accepts both forms allowed by RFC 9110: delay-seconds and an HTTP date.
func parseRetryAfter(v string, now time.Time) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := t.Sub(now); d > 0 {
			return d
		}
	}
	return 0
}
