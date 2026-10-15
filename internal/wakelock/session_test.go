package wakelock

import (
	"errors"
	"testing"
	"time"
)

type fakeInhibitor struct {
	held       bool
	display    bool
	acquires   int
	failAcq    bool
	releaseErr error
}

func (f *fakeInhibitor) Acquire(display bool) error {
	if f.failAcq {
		return errors.New("boom")
	}
	f.held, f.display = true, display
	f.acquires++
	return nil
}

func (f *fakeInhibitor) Release() error {
	f.held = false
	return f.releaseErr
}

type fakeTimer struct {
	at      time.Time
	f       func()
	stopped bool
}

type fakeClock struct {
	now    time.Time
	timers []*fakeTimer
}

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) AfterFunc(d time.Duration, f func()) func() bool {
	t := &fakeTimer{at: c.now.Add(d), f: f}
	c.timers = append(c.timers, t)
	return func() bool { t.stopped = true; return true }
}

// advance moves time forward and fires due timers, including stopped ones whose callback the
// runtime might still run: the session must ignore those.
func (c *fakeClock) advance(d time.Duration, fireStopped bool) {
	c.now = c.now.Add(d)
	for _, t := range c.timers {
		if !t.at.After(c.now) && (!t.stopped || fireStopped) && t.f != nil {
			f := t.f
			t.f = nil
			f()
		}
	}
}

func newTestSession() (*session, *fakeInhibitor, *fakeClock) {
	inh := &fakeInhibitor{}
	clk := &fakeClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	return newSession(inh, false, clk), inh, clk
}

func TestTimedSessionExpires(t *testing.T) {
	s, inh, clk := newTestSession()
	if err := s.start(10 * time.Minute); err != nil {
		t.Fatal(err)
	}
	st := s.snapshot()
	if !st.Active || st.Indefinite || !inh.held {
		t.Fatalf("not active after start: %+v held=%v", st, inh.held)
	}
	if got := st.remaining(clk.now); got != 10*time.Minute {
		t.Fatalf("remaining = %s", got)
	}
	clk.advance(9*time.Minute, false)
	if !s.snapshot().Active {
		t.Fatal("expired early")
	}
	clk.advance(time.Minute, false)
	if s.snapshot().Active || inh.held {
		t.Fatal("still active after deadline")
	}
}

func TestRestartReplacesDeadline(t *testing.T) {
	s, inh, clk := newTestSession()
	_ = s.start(5 * time.Minute)
	clk.advance(4*time.Minute, false)
	_ = s.start(30 * time.Minute)
	// The first timer fires even though it was stopped: it must not end the new session.
	clk.advance(2*time.Minute, true)
	if !s.snapshot().Active || !inh.held {
		t.Fatal("stale timer ended the replacing session")
	}
	if got := s.snapshot().remaining(clk.now); got != 28*time.Minute {
		t.Fatalf("remaining = %s", got)
	}
}

func TestIndefinite(t *testing.T) {
	s, _, clk := newTestSession()
	_ = s.start(0)
	clk.advance(1000*time.Hour, true)
	st := s.snapshot()
	if !st.Active || !st.Indefinite || !st.Until.IsZero() || st.remaining(clk.now) != 0 {
		t.Fatalf("bad indefinite state: %+v", st)
	}
}

func TestKeepUnlockedReacquiresOnlyWhenActive(t *testing.T) {
	s, inh, _ := newTestSession()
	_ = s.setKeepUnlocked(true)
	if inh.acquires != 0 {
		t.Fatal("acquired while inactive")
	}
	_ = s.start(0)
	if !inh.display {
		t.Fatal("start ignored keep-unlocked")
	}
	_ = s.setKeepUnlocked(false)
	if inh.display || inh.acquires != 2 {
		t.Fatalf("not re-acquired: display=%v acquires=%d", inh.display, inh.acquires)
	}
}

func TestAcquireFailureLeavesStateUnchanged(t *testing.T) {
	s, inh, _ := newTestSession()
	inh.failAcq = true
	if err := s.start(time.Minute); err == nil {
		t.Fatal("expected error")
	}
	if s.snapshot().Active {
		t.Fatal("active despite failed acquire")
	}
}

func TestReleaseErrorStillStops(t *testing.T) {
	s, inh, _ := newTestSession()
	_ = s.start(0)
	inh.releaseErr = errors.New("boom")
	if err := s.stop(); err == nil {
		t.Fatal("expected error")
	}
	if s.snapshot().Active {
		t.Fatal("still active")
	}
}

func TestListenersSeeChanges(t *testing.T) {
	s, _, _ := newTestSession()
	var got []bool
	s.subscribe(func(st state) {
		// Reading the state from a listener must not deadlock.
		_ = s.snapshot()
		got = append(got, st.Active)
	})
	_ = s.start(time.Minute)
	_ = s.stop()
	_ = s.stop() // no-op: no notification
	if len(got) != 2 || !got[0] || got[1] {
		t.Fatalf("notifications = %v", got)
	}
}

func TestNegativeDuration(t *testing.T) {
	s, inh, _ := newTestSession()
	if err := s.start(-time.Second); err == nil {
		t.Fatal("expected error")
	}
	if inh.acquires != 0 || s.snapshot().Active {
		t.Fatal("negative duration acquired the inhibitor")
	}
}
