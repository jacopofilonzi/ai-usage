package poller

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

// fakeClock fires timers only when the test advances it. A timer created with a deadline already
// reached fires at once, so the loop never misses an Advance that happened before it armed.
type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}

type fakeTimer struct {
	c       chan time.Time
	at      time.Time
	stopped atomic.Bool
}

func (t *fakeTimer) C() <-chan time.Time { return t.c }
func (t *fakeTimer) Stop() bool          { t.stopped.Store(true); return true }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) NewTimer(d time.Duration) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{c: make(chan time.Time, 1), at: c.now.Add(d)}
	if d <= 0 {
		t.c <- c.now
	} else {
		c.timers = append(c.timers, t)
	}
	return t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	kept := c.timers[:0]
	for _, t := range c.timers {
		if !t.stopped.Load() && !t.at.After(c.now) {
			t.c <- c.now
		} else if !t.stopped.Load() {
			kept = append(kept, t)
		}
	}
	c.timers = kept
}

type result struct {
	limits []provider.Limit
	err    error
}

// fakeProvider blocks every Poll until the test sends its result.
type fakeProvider struct {
	id       string
	interval time.Duration
	calls    chan struct{}
	results  chan result
}

func newFake(id string) *fakeProvider {
	return &fakeProvider{id: id, interval: 2 * time.Minute, calls: make(chan struct{}, 10), results: make(chan result)}
}

func (f *fakeProvider) ID() string              { return f.id }
func (f *fakeProvider) Name() string            { return f.id }
func (f *fakeProvider) Detect() bool            { return true }
func (f *fakeProvider) Interval() time.Duration { return f.interval }
func (f *fakeProvider) Poll(ctx context.Context) ([]provider.Limit, error) {
	f.calls <- struct{}{}
	select {
	case r := <-f.results:
		return r.limits, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type harness struct {
	t     *testing.T
	clock *fakeClock
	p     *Poller
	snaps chan provider.Snapshot
}

func start(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, clock: &fakeClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}, snaps: make(chan provider.Snapshot, 20)}
	h.p = New(func(s provider.Snapshot) { h.snaps <- s }, h.clock)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { h.p.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	return h
}

const wait = 2 * time.Second

func (h *harness) snap() provider.Snapshot {
	h.t.Helper()
	select {
	case s := <-h.snaps:
		return s
	case <-time.After(wait):
		h.t.Fatal("no snapshot")
		return provider.Snapshot{}
	}
}

func (h *harness) expectPoll(f *fakeProvider) {
	h.t.Helper()
	select {
	case <-f.calls:
	case <-time.After(wait):
		h.t.Fatalf("%s not polled", f.id)
	}
}

func (h *harness) expectNoPoll(f *fakeProvider) {
	h.t.Helper()
	select {
	case <-f.calls:
		h.t.Fatalf("%s polled too early", f.id)
	case <-time.After(50 * time.Millisecond):
	}
}

// pollWith waits for a poll of f, answers it and returns the resulting snapshot.
func (h *harness) pollWith(f *fakeProvider, r result) provider.Snapshot {
	h.t.Helper()
	h.expectPoll(f)
	f.results <- r
	return h.snap()
}

func limits(pct ...float64) []provider.Limit {
	var out []provider.Limit
	for i, p := range pct {
		out = append(out, provider.Limit{ID: fmt.Sprint(i), Kind: provider.KindSession, UsedPct: p})
	}
	return out
}

func TestFirstPollAndInterval(t *testing.T) {
	h := start(t)
	f := newFake("a")
	h.p.SetProvider(f)
	if s := h.snap(); s.Status != provider.StatusLoading || s.Provider != "a" || s.Limits == nil {
		t.Fatalf("initial snapshot = %+v", s)
	}
	s := h.pollWith(f, result{limits: limits(42)})
	if s.Status != provider.StatusOK || len(s.Limits) != 1 || s.UpdatedAt == nil || !s.UpdatedAt.Equal(h.clock.Now()) {
		t.Fatalf("ok snapshot = %+v", s)
	}
	h.clock.Advance(f.interval - time.Second)
	h.expectNoPoll(f)
	h.clock.Advance(time.Second)
	h.pollWith(f, result{limits: limits(43)})
}

func TestErrorMapping(t *testing.T) {
	h := start(t)
	f := newFake("a")
	h.p.SetProvider(f)
	h.snap()
	h.pollWith(f, result{limits: limits(10, 20)})

	h.clock.Advance(f.interval)
	s := h.pollWith(f, result{err: fmt.Errorf("read token: %w", provider.ErrUnauthorized)})
	if s.Status != provider.StatusStale || len(s.Limits) != 2 || s.Error == "" {
		t.Fatalf("unauthorized = %+v", s)
	}

	h.clock.Advance(f.interval)
	s = h.pollWith(f, result{err: errors.New("boom")})
	if s.Status != provider.StatusError || len(s.Limits) != 2 || s.Error != "boom" {
		t.Fatalf("generic error = %+v", s)
	}

	h.clock.Advance(f.interval)
	s = h.pollWith(f, result{err: provider.ErrNotLoggedIn})
	if s.Status != provider.StatusNotLoggedIn || len(s.Limits) != 0 || s.Limits == nil {
		t.Fatalf("not logged in = %+v", s)
	}

	h.clock.Advance(f.interval)
	s = h.pollWith(f, result{limits: limits(1)})
	if s.Status != provider.StatusOK || s.Error != "" {
		t.Fatalf("recovered = %+v", s)
	}
}

func TestRateLimitBackoff(t *testing.T) {
	h := start(t)
	f := newFake("a")
	h.p.SetProvider(f)
	h.snap()
	h.pollWith(f, result{limits: limits(50)})

	// Server-provided Retry-After wins over the interval.
	h.clock.Advance(f.interval)
	s := h.pollWith(f, result{err: &provider.RateLimitedError{RetryAfter: 5 * time.Minute}})
	if s.Status != provider.StatusOK || len(s.Limits) != 1 {
		t.Fatalf("rate limited must keep status and limits: %+v", s)
	}
	h.clock.Advance(f.interval)
	h.expectNoPoll(f)
	h.p.Refresh() // a manual refresh does not bypass the rate limit
	h.expectNoPoll(f)
	h.clock.Advance(3 * time.Minute)

	// Without Retry-After: interval doubled per consecutive hit (2nd hit → 4×).
	h.pollWith(f, result{err: &provider.RateLimitedError{}})
	h.clock.Advance(4*f.interval - time.Second)
	h.expectNoPoll(f)
	h.clock.Advance(time.Second)
	h.pollWith(f, result{limits: limits(51)})

	// Success resets the backoff.
	h.clock.Advance(f.interval)
	h.expectPoll(f)
	f.results <- result{limits: limits(52)}
	h.snap()
}

func TestRateLimitFirstPollIsError(t *testing.T) {
	h := start(t)
	f := newFake("a")
	h.p.SetProvider(f)
	h.snap()
	if s := h.pollWith(f, result{err: &provider.RateLimitedError{}}); s.Status != provider.StatusError {
		t.Fatalf("rate limited before any data = %+v", s)
	}
}

func TestBackoff(t *testing.T) {
	i := 2 * time.Minute
	cases := []struct {
		retry time.Duration
		hits  int
		want  time.Duration
	}{
		{0, 1, 4 * time.Minute},
		{0, 2, 8 * time.Minute},
		{0, 3, MaxBackoff},
		{0, 10, MaxBackoff},
		{10 * time.Second, 1, i},   // never faster than the interval
		{time.Hour, 1, MaxBackoff}, // capped
		{5 * time.Minute, 1, 5 * time.Minute},
	}
	for _, c := range cases {
		if got := Backoff(i, c.retry, c.hits); got != c.want {
			t.Errorf("Backoff(%s, %s, %d) = %s, want %s", i, c.retry, c.hits, got, c.want)
		}
	}
	if got := Backoff(time.Hour, 0, 3); got != time.Hour {
		t.Errorf("long interval must not be shortened by the cap: %s", got)
	}
}

func TestSwitchShowsCacheThenPolls(t *testing.T) {
	h := start(t)
	a, b := newFake("a"), newFake("b")
	h.p.SetProvider(a)
	h.snap()
	h.pollWith(a, result{limits: limits(70)})

	h.p.SetProvider(b)
	if s := h.snap(); s.Provider != "b" || s.Status != provider.StatusLoading {
		t.Fatalf("switch to b = %+v", s)
	}
	h.pollWith(b, result{limits: limits(5)})

	h.p.SetProvider(a)
	if s := h.snap(); s.Provider != "a" || s.Status != provider.StatusOK || s.Limits[0].UsedPct != 70 {
		t.Fatalf("switch back must show the cache instantly: %+v", s)
	}
	if s := h.pollWith(a, result{limits: limits(71)}); s.Limits[0].UsedPct != 71 {
		t.Fatalf("refreshed a = %+v", s)
	}
	if got := h.p.Snapshot(); got.Provider != "a" || got.Limits[0].UsedPct != 71 {
		t.Fatalf("Snapshot() = %+v", got)
	}
}

func TestSwitchCancelsInFlightPoll(t *testing.T) {
	h := start(t)
	a, b := newFake("a"), newFake("b")
	h.p.SetProvider(a)
	h.snap()
	h.expectPoll(a) // left hanging
	h.p.SetProvider(b)
	if s := h.snap(); s.Provider != "b" {
		t.Fatalf("switch = %+v", s)
	}
	if s := h.pollWith(b, result{limits: limits(1)}); s.Provider != "b" || s.Status != provider.StatusOK {
		t.Fatalf("b = %+v", s)
	}
	// a's aborted poll left it due: switching back polls it at once.
	h.p.SetProvider(a)
	if s := h.snap(); s.Status != provider.StatusLoading {
		t.Fatalf("a after aborted poll = %+v", s)
	}
	h.expectPoll(a)
}

func TestRefreshAndOverride(t *testing.T) {
	h := start(t)
	f := newFake("a")
	h.p.SetProvider(f)
	h.snap()
	h.pollWith(f, result{limits: limits(1)})

	h.p.Refresh()
	h.pollWith(f, result{limits: limits(2)})

	h.p.SetInterval(5 * time.Minute)
	h.clock.Advance(f.interval)
	h.expectNoPoll(f)
	h.clock.Advance(3 * time.Minute)
	h.pollWith(f, result{limits: limits(3)})

	// Shortening the interval re-arms from the last attempt.
	h.p.SetInterval(time.Minute)
	h.clock.Advance(time.Minute)
	h.pollWith(f, result{limits: limits(4)})
}
