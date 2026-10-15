package wakelock

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"
)

// Inhibitor prevents the OS from sleeping. Acquire may be called while already held: the new
// setting replaces the old one. internal/inhibit provides the real one.
type Inhibitor interface {
	Acquire(keepDisplayOn bool) error
	Release() error
}

// state is a snapshot of the session.
type state struct {
	Active       bool
	Indefinite   bool
	Until        time.Time // zero unless Active && !Indefinite
	KeepUnlocked bool
}

// remaining returns the time left before the session ends, 0 when inactive or indefinite.
func (s state) remaining(now time.Time) time.Duration {
	if !s.Active || s.Indefinite {
		return 0
	}
	return max(s.Until.Sub(now), 0)
}

// Clock abstracts time so tests can drive expiry deterministically.
type Clock interface {
	Now() time.Time
	AfterFunc(d time.Duration, f func()) (stop func() bool)
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (realClock) AfterFunc(d time.Duration, f func()) func() bool {
	return time.AfterFunc(d, f).Stop
}

// session is the state machine. Safe for concurrent use.
type session struct {
	inh   Inhibitor
	clock Clock

	mu        sync.Mutex
	state     state
	gen       uint64
	stopTimer func() bool
	listeners []func(state)
}

// newSession returns an inactive session. A nil clock means the real one.
func newSession(inh Inhibitor, keepUnlocked bool, clock Clock) *session {
	if clock == nil {
		clock = realClock{}
	}
	return &session{inh: inh, clock: clock, state: state{KeepUnlocked: keepUnlocked}}
}

// subscribe registers f to be called, outside the session lock, after every state change.
func (s *session) subscribe(f func(state)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, f)
}

func (s *session) snapshot() state {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// start activates the session for d, or indefinitely when d is 0. It replaces any running session.
func (s *session) start(d time.Duration) error {
	if d < 0 {
		return fmt.Errorf("negative duration %s", d)
	}
	s.mu.Lock()
	if err := s.inh.Acquire(s.state.KeepUnlocked); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("acquire inhibitor: %w", err)
	}
	s.cancelTimerLocked()
	s.state.Active = true
	s.state.Indefinite = d == 0
	s.state.Until = time.Time{}
	if d > 0 {
		s.state.Until = s.clock.Now().Add(d)
		gen := s.gen
		s.stopTimer = s.clock.AfterFunc(d, func() { s.expire(gen) })
	}
	s.unlockAndNotify()
	return nil
}

// stop deactivates the session. Stopping an inactive session is a no-op.
func (s *session) stop() error {
	s.mu.Lock()
	if !s.state.Active {
		s.mu.Unlock()
		return nil
	}
	return s.stopLocked()
}

// setKeepUnlocked changes whether the display is kept on, re-acquiring the inhibitor if active.
func (s *session) setKeepUnlocked(on bool) error {
	s.mu.Lock()
	if s.state.KeepUnlocked == on {
		s.mu.Unlock()
		return nil
	}
	if s.state.Active {
		if err := s.inh.Acquire(on); err != nil {
			s.mu.Unlock()
			return fmt.Errorf("acquire inhibitor: %w", err)
		}
	}
	s.state.KeepUnlocked = on
	s.unlockAndNotify()
	return nil
}

func (s *session) expire(gen uint64) {
	s.mu.Lock()
	if gen != s.gen || !s.state.Active {
		s.mu.Unlock()
		return
	}
	if err := s.stopLocked(); err != nil {
		// A timer has no caller to return to: log instead.
		slog.Error("end expired wake lock", "err", err)
	}
}

// stopLocked releases the inhibitor and notifies. It must be called with mu held and unlocks it.
// The state goes inactive even if Release fails: the OS request is gone or unrecoverable either
// way, and reporting "active" would be a lie the user cannot fix.
func (s *session) stopLocked() error {
	s.cancelTimerLocked()
	err := s.inh.Release()
	s.state.Active = false
	s.state.Indefinite = false
	s.state.Until = time.Time{}
	s.unlockAndNotify()
	if err != nil {
		return fmt.Errorf("release inhibitor: %w", err)
	}
	return nil
}

// cancelTimerLocked bumps the generation so a timer that already fired (and is waiting on mu) is
// ignored, then stops the pending one.
func (s *session) cancelTimerLocked() {
	s.gen++
	if s.stopTimer != nil {
		s.stopTimer()
		s.stopTimer = nil
	}
}

// unlockAndNotify releases mu and then calls the listeners with the new snapshot, so a listener
// may read the state without deadlocking.
func (s *session) unlockAndNotify() {
	st := s.state
	ls := slices.Clone(s.listeners)
	s.mu.Unlock()
	for _, f := range ls {
		f(st)
	}
}
