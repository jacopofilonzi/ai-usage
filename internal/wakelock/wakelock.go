// Package wakelock keeps the computer from sleeping for a while or indefinitely.
//
// CONTRACT STUB (phase 0): signatures are final, the implementation is ported from
// github.com/jacopofilonzi/wake-lock (session + service, without IPC) by the wake-lock agent.
package wakelock

import (
	"time"
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
type Service struct{}

// New creates the OS inhibitor. Nothing is inhibited until Start.
func New(keepUnlocked bool) (*Service, error) { return &Service{}, nil }

func (s *Service) Status() Status { return Status{} }

// Start keeps the computer awake for d; d == 0 means indefinitely. Restarting replaces the deadline.
func (s *Service) Start(d time.Duration) error { return nil }

func (s *Service) Stop() error { return nil }

// SetKeepUnlocked applies the preference immediately (also to an active session). Persisting it is
// the caller's job.
func (s *Service) SetKeepUnlocked(on bool) error { return nil }

// OnChange registers f, called after every change, including a deadline expiring.
func (s *Service) OnChange(f func()) {}

// Close releases the inhibitor.
func (s *Service) Close() error { return nil }
