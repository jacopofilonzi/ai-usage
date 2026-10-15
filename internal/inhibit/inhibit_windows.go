// Package inhibit keeps the OS from sleeping. Each platform implements the same Inhibitor type:
// Acquire(keepDisplayOn) replaces any previous request, Release drops it. If the process dies the
// OS drops the request too, so a crash never leaves the machine awake forever.
// Ported from github.com/jacopofilonzi/wake-lock (internal/inhibit).
//
// Windows: SetThreadExecutionState is per thread and, with ES_CONTINUOUS, lasts until the same
// thread changes it or exits. Go moves goroutines between threads, so one goroutine locked to its
// OS thread owns the state and serves every request.
package inhibit

import (
	"fmt"
	"runtime"

	"golang.org/x/sys/windows"
)

const (
	esSystemRequired  = 0x00000001
	esDisplayRequired = 0x00000002
	esContinuous      = 0x80000000
)

var procSetThreadExecutionState = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetThreadExecutionState")

type Inhibitor struct {
	reqs chan request
}

type request struct {
	flags uint32
	done  chan error
}

// New starts the goroutine that owns the execution state.
func New() (*Inhibitor, error) {
	if err := procSetThreadExecutionState.Find(); err != nil {
		return nil, fmt.Errorf("find SetThreadExecutionState: %w", err)
	}
	in := &Inhibitor{reqs: make(chan request)}
	go in.loop()
	return in, nil
}

func (in *Inhibitor) loop() {
	runtime.LockOSThread()
	// Never unlocked: the thread dies with the goroutine, which only ends with the process.
	for r := range in.reqs {
		ret, _, err := procSetThreadExecutionState.Call(uintptr(r.flags))
		if ret == 0 {
			r.done <- fmt.Errorf("SetThreadExecutionState(%#x): %w", r.flags, err)
			continue
		}
		r.done <- nil
	}
}

func (in *Inhibitor) set(flags uint32) error {
	done := make(chan error, 1)
	in.reqs <- request{flags: flags, done: done}
	return <-done
}

func (in *Inhibitor) Acquire(keepDisplayOn bool) error {
	flags := uint32(esContinuous | esSystemRequired)
	if keepDisplayOn {
		flags |= esDisplayRequired
	}
	return in.set(flags)
}

func (in *Inhibitor) Release() error {
	return in.set(esContinuous)
}
