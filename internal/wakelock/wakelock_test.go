package wakelock

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func newTestService() (*Service, *fakeInhibitor, *fakeClock) {
	inh := &fakeInhibitor{}
	clk := &fakeClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	return newService(inh, false, clk), inh, clk
}

func TestStatusTimed(t *testing.T) {
	s, _, clk := newTestService()
	if err := s.Start(10 * time.Minute); err != nil {
		t.Fatal(err)
	}
	clk.advance(1500*time.Millisecond, false)
	st := s.Status()
	if !st.Active || st.Indefinite || st.KeepUnlocked {
		t.Fatalf("status = %+v", st)
	}
	// 598.5 s left rounds to 599 (half away from zero), not truncated to 598.
	if st.RemainingS != 599 {
		t.Fatalf("remainingS = %d", st.RemainingS)
	}
	want := time.Date(2026, 1, 1, 12, 10, 0, 0, time.UTC)
	if st.Until == nil || !st.Until.Equal(want) {
		t.Fatalf("until = %v, want %v", st.Until, want)
	}
}

func TestStatusIndefiniteAndInactive(t *testing.T) {
	s, _, _ := newTestService()
	if st := s.Status(); st.Active || st.Until != nil || st.RemainingS != 0 {
		t.Fatalf("inactive status = %+v", st)
	}
	_ = s.Start(0)
	st := s.Status()
	if !st.Active || !st.Indefinite || st.Until != nil || st.RemainingS != 0 {
		t.Fatalf("indefinite status = %+v", st)
	}
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "until") {
		t.Fatalf("until serialized without a deadline: %s", b)
	}
}

func TestStatusUntilIsACopy(t *testing.T) {
	s, _, _ := newTestService()
	_ = s.Start(time.Minute)
	st := s.Status()
	*st.Until = time.Time{}
	if s.Status().Until.IsZero() {
		t.Fatal("Status exposed internal state")
	}
}

func TestSetKeepUnlockedShowsInStatus(t *testing.T) {
	s, inh, _ := newTestService()
	_ = s.Start(0)
	if err := s.SetKeepUnlocked(true); err != nil {
		t.Fatal(err)
	}
	if !s.Status().KeepUnlocked || !inh.display {
		t.Fatalf("keep unlocked not applied: %+v display=%v", s.Status(), inh.display)
	}
}

func TestOnChange(t *testing.T) {
	s, _, clk := newTestService()
	var seen []Status
	s.OnChange(func() { seen = append(seen, s.Status()) })

	_ = s.Start(time.Minute)        // 1
	_ = s.SetKeepUnlocked(true)     // 2
	_ = s.SetKeepUnlocked(true)     // unchanged: no notification
	clk.advance(time.Minute, false) // 3: expiry
	_ = s.Stop()                    // inactive: no notification

	if len(seen) != 3 {
		t.Fatalf("notifications = %d, want 3", len(seen))
	}
	if !seen[0].Active || !seen[1].KeepUnlocked || seen[2].Active {
		t.Fatalf("notified states = %+v", seen)
	}
}

func TestCloseReleases(t *testing.T) {
	s, inh, _ := newTestService()
	_ = s.Start(0)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if inh.held || s.Status().Active {
		t.Fatal("close did not release the inhibitor")
	}
}
