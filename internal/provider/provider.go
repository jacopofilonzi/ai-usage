// Package provider defines what every agent integration (Claude Code, Codex, …) reports: a list of
// usage limits, each with the percentage used and when it resets. The UI never knows which agent it
// is drawing: it renders one ring per Limit.
//
// Providers only read what the installed agent already keeps on disk (credentials, session logs).
// They never refresh, store or log a token: when the agent's token is expired or missing they
// return ErrUnauthorized / ErrNotLoggedIn and the poller keeps showing the last known values as stale.
package provider

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Kind tells the UI how to label a limit.
type Kind string

const (
	KindSession     Kind = "session"      // short rolling window (Claude/Codex: 5 hours)
	KindWeekly      Kind = "weekly"       // 7-day window
	KindWeeklyModel Kind = "weekly_model" // 7-day window for one model family (Limit.Model)
	KindOther       Kind = "other"        // anything else: labelled from WindowMinutes
)

// Limit is one usage window.
type Limit struct {
	ID            string     `json:"id"`                 // stable within a provider, e.g. "five_hour"
	Kind          Kind       `json:"kind"`               //
	Model         string     `json:"model,omitempty"`    // only for KindWeeklyModel, e.g. "opus"
	WindowMinutes int        `json:"windowMinutes"`      // 0 when unknown
	UsedPct       float64    `json:"usedPct"`            // 0–100, may exceed 100 when over the limit
	ResetsAt      *time.Time `json:"resetsAt,omitempty"` // nil when unknown
}

// Status of a Snapshot as seen by the UI. Set by the poller, not by providers.
type Status string

const (
	StatusLoading     Status = "loading"       // nothing read yet
	StatusOK          Status = "ok"            // fresh values
	StatusStale       Status = "stale"         // last known values; the agent's token expired or was rejected
	StatusNotLoggedIn Status = "not_logged_in" // the agent is installed but has no credentials
	StatusError       Status = "error"         // last read failed for another reason; Limits may be stale
)

// Snapshot is what the UI shows for the active provider.
type Snapshot struct {
	Provider  string     `json:"provider"` // Provider.ID()
	Status    Status     `json:"status"`
	Limits    []Limit    `json:"limits"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"` // when Limits were last read successfully
	Error     string     `json:"error,omitempty"`     // short message for logs/tooltips; never contains secrets
}

// Provider is one agent integration.
type Provider interface {
	ID() string   // stable, used in config: "claude", "codex"
	Name() string // display name: "Claude Code", "Codex"
	// Detect reports whether the agent is installed for this user (its data dir exists).
	Detect() bool
	// Interval is how often Poll should run.
	Interval() time.Duration
	// Poll reads the current limits. Errors may be (or wrap) ErrNotLoggedIn, ErrUnauthorized or
	// *RateLimitedError; anything else is a generic failure.
	Poll(ctx context.Context) ([]Limit, error)
}

var (
	// ErrNotLoggedIn: the agent is installed but has no usable credentials.
	ErrNotLoggedIn = errors.New("not logged in")
	// ErrUnauthorized: the token is expired or was rejected. Opening the agent refreshes it.
	ErrUnauthorized = errors.New("token expired or rejected")
)

// RateLimitedError asks the poller to wait before the next Poll.
type RateLimitedError struct {
	RetryAfter time.Duration // 0 when the server did not say
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("rate limited (retry after %s)", e.RetryAfter)
}
