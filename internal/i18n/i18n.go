// Package i18n holds the strings the Go side shows (tray menu, tooltip, widget context menu), in
// English (default) and Italian. The frontend has its own dictionaries.
//
// Strings are fields of Messages so a typo in a key is a compile error; i18n_test checks that no
// language leaves a field empty. config.Language picks the language: "system" asks the OS (see
// locale_*.go), "en"/"it" force one.
package i18n

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

type Messages struct {
	// Wake lock.
	StatusOff        string
	StatusUntil      string // %s = clock time
	StatusIndefinite string
	RemainingSuffix  string // %s = duration
	KeepAwake        string // submenu
	Minutes          string // %d
	OneHour          string
	Hours            string // %d
	Indefinitely     string
	Stop             string
	KeepUnlocked     string

	// App.
	Settings string
	Quit     string

	// Usage, for the tooltip.
	Loading     string
	NotLoggedIn string
	Stale       string
	Error       string
	NoProvider  string
	Session     string // limit label when the window length is unknown
	Weekly      string
	DayUnit     string // suffix for days in window labels: "7d"
}

var en = Messages{
	StatusOff:        "Sleep allowed",
	StatusUntil:      "Awake until %s",
	StatusIndefinite: "Awake indefinitely",
	RemainingSuffix:  " (%s left)",
	KeepAwake:        "Keep awake",
	Minutes:          "%d minutes",
	OneHour:          "1 hour",
	Hours:            "%d hours",
	Indefinitely:     "Indefinitely",
	Stop:             "Stop",
	KeepUnlocked:     "Keep unlocked (screen on)",
	Settings:         "Settings…",
	Quit:             "Quit",
	Loading:          "loading…",
	NotLoggedIn:      "not logged in",
	Stale:            "not updated",
	Error:            "error",
	NoProvider:       "No supported agent found",
	Session:          "Session",
	Weekly:           "Week",
	DayUnit:          "d",
}

var it = Messages{
	StatusOff:        "Sospensione consentita",
	StatusUntil:      "Sveglio fino alle %s",
	StatusIndefinite: "Sveglio a tempo indeterminato",
	RemainingSuffix:  " (mancano %s)",
	KeepAwake:        "Mantieni sveglio",
	Minutes:          "%d minuti",
	OneHour:          "1 ora",
	Hours:            "%d ore",
	Indefinitely:     "Indefinitamente",
	Stop:             "Disattiva",
	KeepUnlocked:     "Mantieni sbloccato (schermo acceso)",
	Settings:         "Impostazioni…",
	Quit:             "Esci",
	Loading:          "caricamento…",
	NotLoggedIn:      "accesso non effettuato",
	Stale:            "non aggiornato",
	Error:            "errore",
	NoProvider:       "Nessun agente supportato trovato",
	Session:          "Sessione",
	Weekly:           "Settimana",
	DayUnit:          "g",
}

// For returns the messages for a language tag such as "it-IT" or "en_US.UTF-8".
func For(lang string) *Messages {
	if strings.HasPrefix(strings.ToLower(lang), "it") {
		return &it
	}
	return &en
}

// Select returns the messages for a config.Language value: "system" (or "") follows the OS.
func Select(language string) *Messages {
	if language == "" || language == "system" {
		return For(systemLanguage())
	}
	return For(language)
}

// Duration formats a preset duration for the menu ("5 minutes", "1 hour", "2 hours").
func (m *Messages) Duration(d time.Duration) string {
	switch {
	case d == time.Hour:
		return m.OneHour
	case d%time.Hour == 0:
		return fmt.Sprintf(m.Hours, int(d/time.Hour))
	default:
		return fmt.Sprintf(m.Minutes, int(d/time.Minute))
	}
}

// WakeStatus describes the wake lock in one line, with the time left when it has a deadline.
func (m *Messages) WakeStatus(active, indefinite bool, until, now time.Time) string {
	switch {
	case !active:
		return m.StatusOff
	case indefinite:
		return m.StatusIndefinite
	}
	return fmt.Sprintf(m.StatusUntil, clockTime(until, now)) +
		fmt.Sprintf(m.RemainingSuffix, Remaining(until.Sub(now)))
}

func clockTime(t, now time.Time) string {
	t = t.Local()
	if y, mo, d := now.Local().Date(); t.Year() == y && t.Month() == mo && t.Day() == d {
		return t.Format("15:04")
	}
	return t.Format("2006-01-02 15:04")
}

// Remaining formats a time span compactly: "45s", "12m", "1h05m". Minutes round up so a session
// never reads "0m" while still active.
func Remaining(d time.Duration) string {
	d = max(d, 0).Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d/time.Second))
	}
	mins := int((d + time.Minute - 1) / time.Minute)
	if mins < 60 {
		return fmt.Sprintf("%dm", mins)
	}
	return fmt.Sprintf("%dh%02dm", mins/60, mins%60)
}

// LimitLabel is the short name of a usage window: "5h", "7d", "Opus 7d". The window length is
// preferred over the kind because providers report it and it is what users recognise.
func (m *Messages) LimitLabel(l provider.Limit) string {
	var s string
	switch mins := l.WindowMinutes; {
	case mins > 0 && mins%(24*60) == 0:
		s = fmt.Sprintf("%d%s", mins/(24*60), m.DayUnit)
	case mins > 0 && mins%60 == 0:
		s = fmt.Sprintf("%dh", mins/60)
	case mins > 0:
		s = fmt.Sprintf("%dm", mins)
	case l.Kind == provider.KindSession:
		s = m.Session
	default:
		s = m.Weekly
	}
	if l.Kind == provider.KindWeeklyModel && l.Model != "" {
		s = capitalize(l.Model) + " " + s
	}
	return s
}

// Percent formats a used percentage without decimals ("42%").
func Percent(p float64) string {
	return fmt.Sprintf("%d%%", int(math.Round(p)))
}

// Tooltip summarises the usage on one line: "Claude Code · 5h 42% · 7d 17%".
func (m *Messages) Tooltip(name string, snap provider.Snapshot) string {
	if name == "" {
		return m.NoProvider
	}
	parts := []string{name}
	for _, l := range snap.Limits {
		parts = append(parts, m.LimitLabel(l)+" "+Percent(l.UsedPct))
	}
	switch snap.Status {
	case provider.StatusLoading:
		parts = append(parts, m.Loading)
	case provider.StatusNotLoggedIn:
		parts = append(parts, m.NotLoggedIn)
	case provider.StatusStale:
		parts = append(parts, m.Stale)
	case provider.StatusError:
		parts = append(parts, m.Error)
	}
	return strings.Join(parts, " · ")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
