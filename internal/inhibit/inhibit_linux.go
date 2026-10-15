// Package inhibit keeps the OS from sleeping. Each platform implements the same Inhibitor type:
// Acquire(keepDisplayOn) replaces any previous request, Release drops it. If the process dies the
// OS drops the request too, so a crash never leaves the machine awake forever.
// Ported from github.com/jacopofilonzi/wake-lock (internal/inhibit).
//
// Linux:
//   - sleep: a systemd-logind "block" inhibitor (org.freedesktop.login1.Manager.Inhibit), held
//     for as long as we keep the returned file descriptor open.
//   - display (keep unlocked): additionally an "idle" logind inhibitor plus
//     org.freedesktop.ScreenSaver.Inhibit on the session bus, which desktop environments use to
//     skip blanking and locking. That inhibit is tied to our bus connection, so we keep a
//     private connection for the life of the process.
//
// The screensaver interface is optional (not every desktop provides it): failing to reach it is
// logged, not returned, so the sleep inhibitor still works.
package inhibit

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/godbus/dbus/v5"
)

const (
	// Shown by `systemd-inhibit --list` and desktop power UIs.
	appName = "ai-usage"
	reason  = "Sleep prevented by AI Usage"
)

type Inhibitor struct {
	mu      sync.Mutex
	system  *dbus.Conn
	session *dbus.Conn // nil when there is no session bus
	fd      *os.File   // logind inhibitor lock
	cookie  uint32     // screensaver cookie, 0 = none
}

func New() (*Inhibitor, error) {
	system, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("connect to system bus: %w", err)
	}
	in := &Inhibitor{system: system}
	if session, err := dbus.ConnectSessionBus(); err != nil {
		slog.Warn("no session bus: keep unlocked will only use logind", "err", err)
	} else {
		in.session = session
	}
	return in, nil
}

func (in *Inhibitor) Acquire(keepDisplayOn bool) error {
	in.mu.Lock()
	defer in.mu.Unlock()
	what := "sleep"
	if keepDisplayOn {
		what = "sleep:idle"
	}
	// Take the new lock before dropping the old one so there is no gap.
	var fd dbus.UnixFD
	err := in.system.Object("org.freedesktop.login1", "/org/freedesktop/login1").
		Call("org.freedesktop.login1.Manager.Inhibit", 0, what, appName, reason, "block").
		Store(&fd)
	if err != nil {
		return fmt.Errorf("logind inhibit %q: %w", what, err)
	}
	old := in.fd
	in.fd = os.NewFile(uintptr(fd), "logind-inhibit")
	if old != nil {
		if err := old.Close(); err != nil {
			slog.Warn("close previous logind inhibitor", "err", err)
		}
	}

	if keepDisplayOn && in.cookie == 0 {
		in.inhibitScreenSaverLocked()
	} else if !keepDisplayOn {
		in.uninhibitScreenSaverLocked()
	}
	return nil
}

func (in *Inhibitor) Release() error {
	in.mu.Lock()
	defer in.mu.Unlock()
	in.uninhibitScreenSaverLocked()
	if in.fd == nil {
		return nil
	}
	fd := in.fd
	in.fd = nil
	if err := fd.Close(); err != nil {
		return fmt.Errorf("close logind inhibitor: %w", err)
	}
	return nil
}

func (in *Inhibitor) screenSaver() dbus.BusObject {
	return in.session.Object("org.freedesktop.ScreenSaver", "/org/freedesktop/ScreenSaver")
}

func (in *Inhibitor) inhibitScreenSaverLocked() {
	if in.session == nil {
		return
	}
	var cookie uint32
	err := in.screenSaver().Call("org.freedesktop.ScreenSaver.Inhibit", 0, appName, reason).Store(&cookie)
	if err != nil {
		slog.Warn("screensaver inhibit unavailable", "err", err)
		return
	}
	in.cookie = cookie
}

func (in *Inhibitor) uninhibitScreenSaverLocked() {
	if in.session == nil || in.cookie == 0 {
		return
	}
	cookie := in.cookie
	in.cookie = 0
	if err := in.screenSaver().Call("org.freedesktop.ScreenSaver.UnInhibit", 0, cookie).Err; err != nil {
		slog.Warn("screensaver uninhibit", "err", err)
	}
}
