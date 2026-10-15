// Package poller polls the active provider and turns its results into provider.Snapshots.
//
// Only the active provider is polled. Every provider keeps its own entry (last snapshot, next due
// time, rate-limit backoff), so switching back to a provider shows its cached snapshot at once and
// a rate limit hit by one provider still holds after switching away and back.
//
// Error mapping (the UI never sees raw provider errors, only Status + a short message):
//
//	nil                 → ok, Limits replaced, UpdatedAt = now
//	ErrNotLoggedIn      → not_logged_in, Limits cleared (they belong to an account that is gone)
//	ErrUnauthorized     → stale, last Limits kept (the agent refreshes its token when used)
//	*RateLimitedError   → status and Limits kept, next poll backed off: RetryAfter, or the
//	                      interval doubled per consecutive hit; capped at MaxBackoff, never below
//	                      the interval
//	anything else       → error, last Limits kept
//
// The loop is a single goroutine; a poll in flight is cancelled when the active provider changes.
// Time comes from a Clock so tests drive it without sleeping.
package poller

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

// MaxBackoff caps the wait after rate limits.
const MaxBackoff = 15 * time.Minute

// PollTimeout bounds one Poll call.
const PollTimeout = 30 * time.Second

// Clock abstracts time for tests.
type Clock interface {
	Now() time.Time
	NewTimer(d time.Duration) Timer
}

// Timer is the part of time.Timer the poller uses.
type Timer interface {
	C() <-chan time.Time
	Stop() bool
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
func (realClock) NewTimer(d time.Duration) Timer {
	return realTimer{time.NewTimer(d)}
}

type realTimer struct{ t *time.Timer }

func (t realTimer) C() <-chan time.Time { return t.t.C }
func (t realTimer) Stop() bool          { return t.t.Stop() }

type entry struct {
	snap        provider.Snapshot
	nextAt      time.Time // zero: due now
	lastAttempt time.Time
	retryAt     time.Time // rate-limit floor for nextAt, also for Refresh and switches
	rateHits    int       // consecutive rate limits
}

// Poller is safe for concurrent use.
type Poller struct {
	clock  Clock
	onSnap func(provider.Snapshot)

	// emitMu serialises callbacks so snapshots reach the UI in the order they were produced;
	// it is taken before mu and released after the callback, and the callback must not call
	// SetProvider (it may call Snapshot).
	emitMu sync.Mutex

	mu         sync.Mutex
	active     provider.Provider
	override   time.Duration
	entries    map[string]*entry
	cancelPoll context.CancelFunc

	wake chan struct{}
}

// New creates a poller that reports every new snapshot of the active provider to onSnap.
// clock may be nil for the real clock. Nothing is polled until SetProvider and Run.
func New(onSnap func(provider.Snapshot), clock Clock) *Poller {
	if clock == nil {
		clock = realClock{}
	}
	return &Poller{
		clock:   clock,
		onSnap:  onSnap,
		entries: map[string]*entry{},
		wake:    make(chan struct{}, 1),
	}
}

func (p *Poller) signal() {
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

// entryFor returns the entry for pr, creating it in the loading state. Caller holds mu.
func (p *Poller) entryFor(pr provider.Provider) *entry {
	e, ok := p.entries[pr.ID()]
	if !ok {
		e = &entry{snap: provider.Snapshot{Provider: pr.ID(), Status: provider.StatusLoading, Limits: []provider.Limit{}}}
		p.entries[pr.ID()] = e
	}
	return e
}

// interval of pr, honouring the user override. Caller holds mu.
func (p *Poller) interval(pr provider.Provider) time.Duration {
	if p.override > 0 {
		return p.override
	}
	if d := pr.Interval(); d > 0 {
		return d
	}
	return 2 * time.Minute
}

func latest(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// SetProvider makes pr the active provider: its cached snapshot (or a loading one) is reported
// at once, then it is polled as soon as its rate limit allows. nil stops polling.
func (p *Poller) SetProvider(pr provider.Provider) {
	p.emitMu.Lock()
	defer p.emitMu.Unlock()
	p.mu.Lock()
	if pr != nil && p.active != nil && p.active.ID() == pr.ID() {
		p.mu.Unlock()
		return
	}
	if p.cancelPoll != nil {
		p.cancelPoll()
	}
	p.active = pr
	if pr == nil {
		p.mu.Unlock()
		p.signal()
		return
	}
	e := p.entryFor(pr)
	e.nextAt = latest(p.clock.Now(), e.retryAt)
	snap := cloneSnapshot(e.snap)
	p.mu.Unlock()
	p.onSnap(snap)
	p.signal()
}

// SetInterval overrides every provider's interval; 0 restores the providers' own.
func (p *Poller) SetInterval(d time.Duration) {
	p.mu.Lock()
	if d < 0 {
		d = 0
	}
	if d == p.override {
		p.mu.Unlock()
		return
	}
	p.override = d
	if pr := p.active; pr != nil {
		e := p.entryFor(pr)
		if !e.lastAttempt.IsZero() && e.rateHits == 0 {
			e.nextAt = latest(e.lastAttempt.Add(p.interval(pr)), e.retryAt)
		}
	}
	p.mu.Unlock()
	p.signal()
}

// Refresh polls the active provider now, unless it is rate limited.
func (p *Poller) Refresh() {
	p.mu.Lock()
	if pr := p.active; pr != nil {
		e := p.entryFor(pr)
		e.nextAt = latest(p.clock.Now(), e.retryAt)
	}
	p.mu.Unlock()
	p.signal()
}

// Snapshot returns the active provider's latest snapshot (loading when there is none yet).
func (p *Poller) Snapshot() provider.Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.active == nil {
		return provider.Snapshot{Status: provider.StatusLoading, Limits: []provider.Limit{}}
	}
	return cloneSnapshot(p.entryFor(p.active).snap)
}

// Run polls until ctx is done.
func (p *Poller) Run(ctx context.Context) {
	for {
		p.mu.Lock()
		pr := p.active
		var wait time.Duration
		if pr != nil {
			wait = p.entryFor(pr).nextAt.Sub(p.clock.Now())
		}
		p.mu.Unlock()

		if pr != nil && wait <= 0 {
			p.pollOnce(ctx, pr)
			if ctx.Err() != nil {
				return
			}
			continue
		}

		var timer Timer
		var fire <-chan time.Time
		if pr != nil {
			timer = p.clock.NewTimer(wait)
			fire = timer.C()
		}
		select {
		case <-ctx.Done():
		case <-p.wake:
		case <-fire:
		}
		if timer != nil {
			timer.Stop()
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (p *Poller) pollOnce(ctx context.Context, pr provider.Provider) {
	pctx, cancel := context.WithTimeout(ctx, PollTimeout)
	p.mu.Lock()
	p.cancelPoll = cancel
	p.mu.Unlock()

	limits, err := pr.Poll(pctx)
	aborted := errors.Is(pctx.Err(), context.Canceled) // before our own cancel below
	cancel()

	p.emitMu.Lock()
	defer p.emitMu.Unlock()
	p.mu.Lock()
	p.cancelPoll = nil
	// Cancelled because the app quits or the provider was switched away: not a result. The entry
	// stays due, so switching back polls at once.
	if ctx.Err() != nil || aborted {
		p.mu.Unlock()
		return
	}
	now := p.clock.Now()
	e := p.entryFor(pr)
	prev := e.snap
	apply(e, limits, err, now, p.interval(pr))
	isActive := p.active != nil && p.active.ID() == pr.ID()
	snap := cloneSnapshot(e.snap)
	p.mu.Unlock()

	if snap.Status != prev.Status || snap.Error != prev.Error {
		slog.Info("usage status changed", "provider", pr.ID(), "status", snap.Status, "error", snap.Error)
	}
	if isActive {
		p.onSnap(snap)
	}
}

// apply folds one poll result into e (see the package doc for the mapping).
func apply(e *entry, limits []provider.Limit, err error, now time.Time, interval time.Duration) {
	e.lastAttempt = now
	var rl *provider.RateLimitedError
	if !errors.As(err, &rl) {
		e.rateHits = 0
		e.retryAt = time.Time{}
		e.nextAt = now.Add(interval)
	}
	s := &e.snap
	switch {
	case err == nil:
		s.Status = provider.StatusOK
		s.Limits = limits
		t := now
		s.UpdatedAt = &t
		s.Error = ""
	case errors.Is(err, provider.ErrNotLoggedIn):
		s.Status = provider.StatusNotLoggedIn
		s.Limits = nil
		s.Error = err.Error()
	case errors.Is(err, provider.ErrUnauthorized):
		s.Status = provider.StatusStale
		s.Error = err.Error()
	case rl != nil:
		e.rateHits++
		d := Backoff(interval, rl.RetryAfter, e.rateHits)
		e.retryAt = now.Add(d)
		e.nextAt = e.retryAt
		if s.Status == provider.StatusLoading {
			s.Status = provider.StatusError
		}
		s.Error = err.Error()
	default:
		s.Status = provider.StatusError
		s.Error = err.Error()
	}
	if s.Limits == nil {
		s.Limits = []provider.Limit{}
	}
}

// Backoff is the wait after the hits-th consecutive rate limit: the server's RetryAfter when
// given, otherwise interval·2^hits; capped at MaxBackoff but never shorter than interval.
func Backoff(interval, retryAfter time.Duration, hits int) time.Duration {
	d := retryAfter
	if d <= 0 {
		d = interval
		for i := 0; i < hits && d < MaxBackoff; i++ {
			d *= 2
		}
	}
	d = min(d, MaxBackoff)
	return max(d, interval)
}

func cloneSnapshot(s provider.Snapshot) provider.Snapshot {
	s.Limits = append([]provider.Limit{}, s.Limits...)
	return s
}
