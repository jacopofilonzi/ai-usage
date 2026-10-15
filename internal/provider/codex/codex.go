// Package codex reads Codex's usage limits from its local session logs.
//
// Design:
//
//   - Codex (openai/codex) writes one JSONL "rollout" file per session under
//     <CODEX_HOME>/sessions/YYYY/MM/DD/rollout-<timestamp>-<uuid>.jsonl, CODEX_HOME defaulting to
//     ~/.codex. Every line is {"timestamp": RFC 3339, "type": ..., "payload": ...}. After each model
//     response Codex appends a line with type "event_msg" and payload.type "token_count" whose
//     payload.rate_limits holds the account's rate-limit snapshot as sent by the backend:
//     {"primary": window|null, "secondary": window|null, "limit_id", ...}, with
//     window = {"used_percent": 0–100, "window_minutes", "resets_at" (unix seconds)}.
//     Older versions (2025) wrote "resets_in_seconds", relative to the line's timestamp, instead of
//     "resets_at", and reported 299 instead of 300 minutes; both are supported.
//     Source: https://github.com/openai/codex/blob/main/codex-rs/protocol/src/protocol.rs
//     (TokenCountEvent, RateLimitSnapshot, RateLimitWindow) and
//     https://github.com/openai/codex/blob/main/codex-rs/rollout/src/policy.rs (token_count is persisted).
//   - No network and no token: the limits are whatever Codex last saw. They are account-wide, so the
//     most recently modified rollout file is enough. A window whose reset time has passed has reset
//     since, so it is reported as 0% with no reset time.
//   - Finding that file must stay cheap with years of history: date directories are walked newest
//     first and only the files of the newest few day directories are stat'ed. Only the file's tail is
//     read, growing the tail only when it holds no snapshot. Cold rollouts that Codex compressed to
//     .jsonl.zst are skipped: the active session is never compressed.
package codex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

// Provider reads Codex's usage limits. Unexported fields are injection points for tests.
type Provider struct {
	now  func() time.Time
	home func() (string, error)
}

// New returns a Provider that reads the real Codex home.
func New() *Provider {
	return &Provider{now: time.Now, home: defaultHome}
}

func (p *Provider) ID() string              { return "codex" }
func (p *Provider) Name() string            { return "Codex" }
func (p *Provider) Interval() time.Duration { return 30 * time.Second }

// Detect reports whether Codex's home dir exists.
func (p *Provider) Detect() bool {
	dir, err := p.home()
	if err != nil {
		return false
	}
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// Poll returns the limits from the latest rate-limit snapshot, or no limits (and no error) when
// Codex has not recorded one yet.
func (p *Provider) Poll(ctx context.Context) ([]provider.Limit, error) {
	dir, err := p.home()
	if err != nil {
		return nil, fmt.Errorf("codex: locate home dir: %w", err)
	}
	path, err := latestRollout(ctx, filepath.Join(dir, "sessions"))
	if err != nil {
		return nil, fmt.Errorf("codex: find latest session: %w", err)
	}
	if path == "" {
		return []provider.Limit{}, nil
	}
	snap, err := lastSnapshot(path)
	if err != nil {
		return nil, fmt.Errorf("codex: read session %s: %w", filepath.Base(path), err)
	}
	if snap == nil {
		return []provider.Limit{}, nil
	}
	return snap.limits(p.now()), nil
}

func defaultHome() (string, error) {
	if dir := os.Getenv("CODEX_HOME"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}
	return filepath.Join(home, ".codex"), nil
}
