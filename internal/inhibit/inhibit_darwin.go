// Package inhibit keeps the OS from sleeping. Each platform implements the same Inhibitor type:
// Acquire(keepDisplayOn) replaces any previous request, Release drops it. If the process dies the
// OS drops the request too, so a crash never leaves the machine awake forever.
// Ported from github.com/jacopofilonzi/wake-lock (internal/inhibit).
//
// macOS: a child `caffeinate` process holds the power assertion (-i idle sleep, -d display sleep).
// `-w <our pid>` makes it exit on its own if we die without releasing. Using the CLI avoids cgo
// for IOKit.
package inhibit

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
)

type Inhibitor struct {
	mu  sync.Mutex
	cmd *exec.Cmd
}

func New() (*Inhibitor, error) {
	if _, err := exec.LookPath("caffeinate"); err != nil {
		return nil, fmt.Errorf("caffeinate not found: %w", err)
	}
	return &Inhibitor{}, nil
}

func (in *Inhibitor) Acquire(keepDisplayOn bool) error {
	in.mu.Lock()
	defer in.mu.Unlock()
	if err := in.stopLocked(); err != nil {
		return err
	}
	flags := "-i"
	if keepDisplayOn {
		flags = "-id"
	}
	cmd := exec.Command("caffeinate", flags, "-w", strconv.Itoa(os.Getpid()))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start caffeinate: %w", err)
	}
	in.cmd = cmd
	return nil
}

func (in *Inhibitor) Release() error {
	in.mu.Lock()
	defer in.mu.Unlock()
	return in.stopLocked()
}

func (in *Inhibitor) stopLocked() error {
	if in.cmd == nil {
		return nil
	}
	cmd := in.cmd
	in.cmd = nil
	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("stop caffeinate: %w", err)
	}
	// The exit status of a killed process is an error by definition: only reap it.
	_ = cmd.Wait()
	return nil
}
