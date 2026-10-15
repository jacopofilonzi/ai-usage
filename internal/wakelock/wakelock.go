// Package wakelock keeps the computer from sleeping for a while or indefinitely.
//
// Ported from github.com/jacopofilonzi/wake-lock (internal/session plus the useful part of
// internal/service); IPC, preference persistence and autostart stay with the app layer.
//
// A session is either inactive, active until a deadline, or active indefinitely. While active it
// holds the OS inhibitor (internal/inhibit); starting again replaces the deadline (it does not add
// to it). The "keep unlocked" flag is independent of the active state: changing it while active
// re-acquires the inhibitor with the new display setting.
//
// Timers are tagged with a generation counter so that a timer which fires after being replaced or
// stopped is ignored instead of ending the newer session. Inhibitor and Clock are interfaces so
// tests run on fakes and never inhibit the real machine.
package wakelock

import (
	"fmt"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/inhibit"
)

// Status is the view shared by the tray, the widget and the frontend.
type Status struct {
	Active       bool       `json:"active"`
	Indefinite   bool       `json:"indefinite"`
	Until        *time.Time `json:"until,omitempty"` // only when active with a deadline
	RemainingS   int64      `json:"remainingS"`      // 0 when inactive or indefinite
	KeepUnlocked bool       `json:"keepUnlocked"`    // also keep the display on
}

// Service owns the OS sleep inhibitor. Safe for concurrent use.
type Service struct {
	sess  *session
	clock Clock
}

// New creates the OS inhibitor. Nothing is inhibited until Start.
func New(keepUnlocked bool) (*Service, error) {
	inh, err := inhibit.New()
	if err != nil {
		return nil, fmt.Errorf("create sleep inhibitor: %w", err)
	}
	return newService(inh, keepUnlocked, nil), nil
}

// newService wires a service on any inhibitor; a nil clock means the real one.
func newService(inh Inhibitor, keepUnlocked bool, clock Clock) *Service {
	if clock == nil {
		clock = realClock{}
	}
	return &Service{sess: newSession(inh, keepUnlocked, clock), clock: clock}
}

// Status returns the current state. RemainingS is rounded to the nearest second so a fresh
// 10-minute session reads 600, not 599.
func (s *Service) Status() Status {
	st := s.sess.snapshot()
	out := Status{
		Active:       st.Active,
		Indefinite:   st.Indefinite,
		RemainingS:   int64(st.remaining(s.clock.Now()).Round(time.Second) / time.Second),
		KeepUnlocked: st.KeepUnlocked,
	}
	if st.Active && !st.Indefinite {
		until := st.Until
		out.Until = &until
	}
	return out
}

// Start keeps the computer awake for d; d == 0 means indefinitely. Restarting replaces the deadline.
func (s *Service) Start(d time.Duration) error { return s.sess.start(d) }

// Stop ends the session; stopping an inactive one is a no-op.
func (s *Service) Stop() error { return s.sess.stop() }

// SetKeepUnlocked applies the preference immediately (also to an active session). Persisting it is
// the caller's job.
func (s *Service) SetKeepUnlocked(on bool) error { return s.sess.setKeepUnlocked(on) }

// OnChange registers f, called after every change, including a deadline expiring. f runs outside
// the service lock (it may call Status), on the caller's goroutine or, for an expiry, on a timer
// goroutine.
func (s *Service) OnChange(f func()) { s.sess.subscribe(func(state) { f() }) }

// Close releases the inhibitor. The OS also drops it when the process exits, so this only makes
// a clean shutdown explicit.
func (s *Service) Close() error {
	if err := s.sess.stop(); err != nil {
		return fmt.Errorf("close wake lock: %w", err)
	}
	return nil
}
