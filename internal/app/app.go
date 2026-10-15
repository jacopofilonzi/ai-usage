// Package app wires providers, poller, wake lock, tray and windows into the Wails application.
//
// Lifecycle: Run sets up logging, config and the Wails app (single instance: a second launch
// opens the settings window, then exits), creates the hidden settings window (and, on macOS/Linux,
// the tray popover) and returns when the app quits. Everything that needs the running app (tray,
// screens, widget windows, polling) starts on ApplicationStarted. The app has no main window: it
// lives in the tray, and closing a window only hides it.
//
// State: App.mu guards the config, the detected/active providers and the language. Calls into
// Wails (tray, windows, events) are made without holding it, because Wails setters block on the UI
// thread and UI-thread callbacks may need the lock.
//
// Every change ends in the same fan-out: events for the frontend (usage snapshot, wake-lock
// status, settings) and refreshUI for the tray menu/icon and the widget context menu.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/config"
	"github.com/jacopofilonzi/ai-usage/internal/i18n"
	"github.com/jacopofilonzi/ai-usage/internal/logging"
	"github.com/jacopofilonzi/ai-usage/internal/poller"
	"github.com/jacopofilonzi/ai-usage/internal/provider"
	"github.com/jacopofilonzi/ai-usage/internal/provider/claude"
	"github.com/jacopofilonzi/ai-usage/internal/provider/codex"
	"github.com/jacopofilonzi/ai-usage/internal/tray"
	"github.com/jacopofilonzi/ai-usage/internal/wakelock"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Version is set at build time with -ldflags "-X github.com/jacopofilonzi/ai-usage/internal/app.Version=x.y.z".
var Version = "dev"

const (
	appName  = "ai-usage"
	uniqueID = "com.jacopofilonzi.ai-usage"

	// detectEvery re-checks which agents are installed, so one installed while the app runs shows up.
	detectEvery = 5 * time.Minute
	// refreshEvery re-renders the tray while a timed wake lock counts down.
	refreshEvery = 30 * time.Second
)

var errNoWakeLock = errors.New("wake lock unavailable on this system")

// App holds the running state shared by the bound services.
type App struct {
	wails    *application.App
	cfgPath  string
	all      []provider.Provider
	poller   *poller.Poller
	wake     *wakelock.Service // nil when the OS inhibitor could not be created
	widgets  *widgetManager
	settings *application.WebviewWindow
	popover  *application.WebviewWindow // nil on Windows
	tray     *tray.Tray                 // set once, before running turns true
	running  atomic.Bool                // ApplicationStarted done: events and native UI exist
	cancel   context.CancelFunc

	mu       sync.Mutex
	cfg      config.Config
	detected []provider.Provider
	active   provider.Provider // nil when no agent is installed
	msg      *i18n.Messages
}

// Run builds the application around the embedded frontend assets and blocks until it quits.
func Run(assets fs.FS) error {
	logFile, err := logging.Setup()
	if err != nil {
		slog.Error("set up logging", "err", err)
	}
	defer closeQuietly(logFile)
	slog.Info("starting", "version", Version)

	a := &App{all: []provider.Provider{claude.New(), codex.New()}}
	a.widgets = newWidgetManager(a)

	opts := application.Options{
		Name:        appName,
		Description: "Usage limits of your AI coding agent",
		Logger:      slog.Default(),
		Services: []application.Service{
			application.NewService(&UsageService{a}),
			application.NewService(&SettingsService{a}),
			application.NewService(&WakeLockService{a}),
			application.NewService(&WidgetService{a}),
		},
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID:               uniqueID,
			OnSecondInstanceLaunch: func(application.SecondInstanceData) { go a.openSettings() },
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory, // no Dock icon
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Windows:    application.WindowsOptions{DisableQuitOnLastWindowClosed: true},
		Linux:      application.LinuxOptions{DisableQuitOnLastWindowClosed: true, ProgramName: appName},
		OnShutdown: a.shutdown,
	}
	a.platformOptions(&opts)
	// A second instance exits inside New after notifying the first one.
	a.wails = application.New(opts)

	a.cfgPath, err = config.DefaultPath()
	if err != nil {
		slog.Error("locate config", "err", err)
	}
	a.cfg = config.Default()
	if a.cfgPath != "" {
		if a.cfg, err = config.Load(a.cfgPath); err != nil {
			slog.Error("load config, using defaults", "err", err)
		}
	}
	a.msg = i18n.Select(a.cfg.Language)

	if a.wake, err = wakelock.New(a.cfg.KeepUnlocked); err != nil {
		slog.Error("create wake lock", "err", err)
		a.wake = nil
	}
	a.poller = poller.New(a.onSnapshot, nil)

	a.settings = a.newSettingsWindow()
	a.popover = a.newPopoverWindow()

	a.wails.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		a.started()
	})
	if err := a.wails.Run(); err != nil {
		return fmt.Errorf("run app: %w", err)
	}
	return nil
}

// started runs once the Wails event loop is up (on its own goroutine).
func (a *App) started() {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	a.detect()
	a.tray = tray.New(a.wails, a.trayActions(), a.popoverWindow(), ContextMenuWidget)
	a.running.Store(true)
	if a.wake != nil {
		// The callback may run while the wake lock holds its own lock: never call back into it inline.
		a.wake.OnChange(func() { go a.onWakeChange() })
	}

	a.mu.Lock()
	cfg, active, none := a.cfg, a.active, len(a.detected) == 0
	a.mu.Unlock()
	a.poller.SetInterval(time.Duration(cfg.PollSeconds) * time.Second)
	a.setPollerProvider(active)
	go a.poller.Run(ctx)
	go a.housekeeping(ctx)

	a.widgets.apply(cfg)
	a.refreshUI()
	if none {
		// First run without any agent: the settings window explains what is missing.
		a.openSettings()
	}
	slog.Info("started", "providers", a.providerIDs(), "active", idOf(active))
}

func (a *App) shutdown() {
	if a.cancel != nil {
		a.cancel()
	}
	a.widgets.close()
	if a.wake != nil {
		if err := a.wake.Close(); err != nil {
			slog.Error("release wake lock", "err", err)
		}
	}
	slog.Info("stopped")
}

func (a *App) housekeeping(ctx context.Context) {
	refresh := time.NewTicker(refreshEvery)
	defer refresh.Stop()
	detect := time.NewTicker(detectEvery)
	defer detect.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-refresh.C:
			if a.wake != nil {
				if st := a.wake.Status(); st.Active && !st.Indefinite {
					a.tray.Refresh()
				}
			}
		case <-detect.C:
			a.detect()
		}
	}
}

// --- fan-out ---

func (a *App) onSnapshot(s provider.Snapshot) {
	if !a.isRunning() {
		return
	}
	a.wails.Event.Emit(EventUsageSnapshot, s)
	a.refreshUI()
}

func (a *App) onWakeChange() {
	a.wails.Event.Emit(EventWakeLockStatus, a.wakeStatus())
	a.refreshUI()
}

func (a *App) emitSettings() {
	if a.isRunning() {
		a.wails.Event.Emit(EventSettings, a.settingsSnapshot())
	}
}

func (a *App) isRunning() bool { return a.running.Load() }

// refreshUI pushes the current state to the tray and the widget context menu.
func (a *App) refreshUI() {
	if !a.isRunning() {
		return
	}
	a.mu.Lock()
	st := tray.State{Msg: a.msg}
	for _, p := range a.detected {
		st.Providers = append(st.Providers, tray.Provider{ID: p.ID(), Name: p.Name(), Active: p == a.active})
	}
	a.mu.Unlock()
	st.Snapshot = a.snapshot()
	st.Wake = a.wakeStatus()
	a.tray.Update(st)
}

func (a *App) trayActions() tray.Actions {
	logErr := func(what string, err error) {
		if err != nil {
			slog.Error(what, "err", err)
		}
	}
	return tray.Actions{
		SetProvider:     func(id string) { logErr("switch provider", a.setProvider(id)) },
		StartWake:       func(m int) { logErr("start wake lock", a.startWake(m)) },
		StopWake:        func() { logErr("stop wake lock", a.stopWake()) },
		SetKeepUnlocked: func(on bool) { logErr("set keep unlocked", a.setKeepUnlocked(on)) },
		OpenSettings:    a.openSettings,
		Quit:            a.wails.Quit,
	}
}

// --- providers ---

// detect refreshes the installed providers and keeps a valid active one. Returns whether the list
// changed.
func (a *App) detect() bool {
	var found []provider.Provider
	for _, p := range a.all {
		if p.Detect() {
			found = append(found, p)
		}
	}
	a.mu.Lock()
	if slices.Equal(found, a.detected) {
		a.mu.Unlock()
		return false
	}
	a.detected = found
	prev := a.active
	if prev == nil || !slices.Contains(found, prev) {
		a.active = pickActive(found, a.cfg.Provider)
	}
	active := a.active
	a.mu.Unlock()

	slog.Info("providers detected", "providers", a.providerIDs(), "active", idOf(active))
	if a.isRunning() {
		if active != prev {
			a.setPollerProvider(active)
		}
		a.emitSettings()
		a.refreshUI()
	}
	return true
}

// pickActive returns the configured provider when installed, else the first installed one.
func pickActive(detected []provider.Provider, preferred string) provider.Provider {
	for _, p := range detected {
		if p.ID() == preferred {
			return p
		}
	}
	if len(detected) > 0 {
		return detected[0]
	}
	return nil
}

// setPollerProvider switches polling; with no provider it tells the UI there is nothing to show.
func (a *App) setPollerProvider(p provider.Provider) {
	a.poller.SetProvider(p)
	if p == nil {
		a.onSnapshot(noProviderSnapshot())
	}
}

func noProviderSnapshot() provider.Snapshot {
	return provider.Snapshot{Status: provider.StatusError, Limits: []provider.Limit{}, Error: "no supported agent detected"}
}

func (a *App) snapshot() provider.Snapshot {
	a.mu.Lock()
	none := a.active == nil
	a.mu.Unlock()
	if none {
		return noProviderSnapshot()
	}
	return a.poller.Snapshot()
}

func (a *App) setProvider(id string) error {
	a.mu.Lock()
	var target provider.Provider
	for _, p := range a.detected {
		if p.ID() == id {
			target = p
		}
	}
	if target == nil {
		a.mu.Unlock()
		return fmt.Errorf("provider %q is not installed", id)
	}
	changed := target != a.active
	a.active = target
	a.cfg.Provider = id
	cfg := a.cfg
	a.mu.Unlock()

	a.poller.SetProvider(target)
	err := a.persist(cfg)
	if changed {
		a.emitSettings()
		a.refreshUI()
	}
	return err
}

func (a *App) cycle(step int) error {
	if step == 0 {
		return nil
	}
	a.mu.Lock()
	n := len(a.detected)
	if n == 0 {
		a.mu.Unlock()
		return nil
	}
	i := slices.Index(a.detected, a.active)
	if step > 0 {
		i = (i + 1) % n
	} else {
		i = (i - 1 + n) % n
	}
	id := a.detected[i].ID()
	a.mu.Unlock()
	return a.setProvider(id)
}

func (a *App) providerInfos() []ProviderInfo {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := []ProviderInfo{}
	for _, p := range a.detected {
		out = append(out, ProviderInfo{ID: p.ID(), Name: p.Name(), Active: p == a.active})
	}
	return out
}

func (a *App) providerIDs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	var ids []string
	for _, p := range a.detected {
		ids = append(ids, p.ID())
	}
	return ids
}

func idOf(p provider.Provider) string {
	if p == nil {
		return ""
	}
	return p.ID()
}

// --- config ---

func (a *App) persist(c config.Config) error {
	if a.cfgPath == "" {
		return errors.New("no config location on this system")
	}
	if err := config.Save(a.cfgPath, c); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}

func (a *App) config() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

// saveSettings normalises c, applies it (language, poll interval, provider, keep unlocked,
// widgets) and persists it. The new settings are applied even when writing the file fails.
func (a *App) saveSettings(c config.Config) error {
	c = c.Normalize()
	a.mu.Lock()
	old := a.cfg
	a.cfg = c
	a.msg = i18n.Select(c.Language)
	var switchTo provider.Provider
	if c.Provider != old.Provider {
		for _, p := range a.detected {
			if p.ID() == c.Provider && p != a.active {
				switchTo = p
			}
		}
		if switchTo != nil {
			a.active = switchTo
		}
	}
	a.mu.Unlock()

	err := a.persist(c)
	a.poller.SetInterval(time.Duration(c.PollSeconds) * time.Second)
	if switchTo != nil {
		a.poller.SetProvider(switchTo)
	}
	if c.KeepUnlocked != old.KeepUnlocked && a.wake != nil {
		if werr := a.wake.SetKeepUnlocked(c.KeepUnlocked); werr != nil {
			err = errors.Join(err, fmt.Errorf("apply keep unlocked: %w", werr))
		}
	}
	a.widgets.apply(c)
	a.emitSettings()
	a.refreshUI()
	return err
}

// --- wake lock ---

func (a *App) wakeStatus() wakelock.Status {
	if a.wake == nil {
		return wakelock.Status{KeepUnlocked: a.config().KeepUnlocked}
	}
	return a.wake.Status()
}

func (a *App) startWake(minutes int) error {
	if a.wake == nil {
		return errNoWakeLock
	}
	if minutes < 0 {
		return fmt.Errorf("invalid duration: %d minutes", minutes)
	}
	if err := a.wake.Start(time.Duration(minutes) * time.Minute); err != nil {
		return fmt.Errorf("start wake lock: %w", err)
	}
	return nil
}

func (a *App) stopWake() error {
	if a.wake == nil {
		return errNoWakeLock
	}
	if err := a.wake.Stop(); err != nil {
		return fmt.Errorf("stop wake lock: %w", err)
	}
	return nil
}

func (a *App) setKeepUnlocked(on bool) error {
	if a.wake == nil {
		return errNoWakeLock
	}
	if err := a.wake.SetKeepUnlocked(on); err != nil {
		return fmt.Errorf("apply keep unlocked: %w", err)
	}
	a.mu.Lock()
	a.cfg.KeepUnlocked = on
	cfg := a.cfg
	a.mu.Unlock()
	err := a.persist(cfg)
	a.emitSettings()
	return err
}

func closeQuietly(c io.Closer) {
	if c != nil {
		_ = c.Close() // the log file: nothing left to report the error to
	}
}
