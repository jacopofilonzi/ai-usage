package tray

import (
	"strconv"
	"strings"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/i18n"
	"github.com/jacopofilonzi/ai-usage/internal/provider"
	"github.com/jacopofilonzi/ai-usage/internal/wakelock"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Provider is one detected provider as the menu shows it.
type Provider struct {
	ID     string
	Name   string
	Active bool
}

// State is everything the tray and the widget context menu show.
type State struct {
	Msg       *i18n.Messages
	Providers []Provider // detected ones
	Snapshot  provider.Snapshot
	Wake      wakelock.Status
}

// activeName is the display name of the active provider, "" when none.
func (s State) activeName() string {
	for _, p := range s.Providers {
		if p.Active {
			return p.Name
		}
	}
	return ""
}

// Actions are what the menu entries do. All are called from Wails' menu goroutines.
type Actions struct {
	SetProvider     func(id string)
	StartWake       func(minutes int) // 0 = indefinitely
	StopWake        func()
	SetKeepUnlocked func(on bool)
	OpenSettings    func()
	Quit            func()
}

// WakePresets are the "Keep awake" durations in minutes.
var WakePresets = []int{5, 10, 30, 60, 120}

type kind int

const (
	kindAction kind = iota
	kindDisabled
	kindRadio
	kindCheckbox
	kindSeparator
	kindSubmenu
)

// item is a menu entry independent of Wails, so the menu can be compared (rebuild only on
// change) and tested without a running app.
type item struct {
	label   string
	kind    kind
	checked bool
	do      func()
	sub     []item
}

// spec builds the menu: wake-lock status line, providers, Keep awake submenu, Settings, Quit.
func spec(s State, now time.Time, act Actions) []item {
	m := s.Msg
	var until time.Time
	if s.Wake.Until != nil {
		until = *s.Wake.Until
	}
	items := []item{
		{label: m.WakeStatus(s.Wake.Active, s.Wake.Indefinite, until, now), kind: kindDisabled},
		{kind: kindSeparator},
	}
	if len(s.Providers) == 0 {
		items = append(items, item{label: m.NoProvider, kind: kindDisabled})
	}
	for _, p := range s.Providers {
		id := p.ID
		items = append(items, item{label: p.Name, kind: kindRadio, checked: p.Active, do: func() { act.SetProvider(id) }})
	}
	items = append(items, item{kind: kindSeparator})

	var wake []item
	for _, mins := range WakePresets {
		mins := mins
		wake = append(wake, item{label: m.Duration(time.Duration(mins) * time.Minute), do: func() { act.StartWake(mins) }})
	}
	keep := !s.Wake.KeepUnlocked
	stop := kindAction
	if !s.Wake.Active {
		stop = kindDisabled
	}
	wake = append(wake,
		item{label: m.Indefinitely, do: func() { act.StartWake(0) }},
		item{kind: kindSeparator},
		item{label: m.Stop, kind: stop, do: act.StopWake},
		item{kind: kindSeparator},
		item{label: m.KeepUnlocked, kind: kindCheckbox, checked: s.Wake.KeepUnlocked, do: func() { act.SetKeepUnlocked(keep) }},
	)
	items = append(items,
		item{label: m.KeepAwake, kind: kindSubmenu, sub: wake},
		item{kind: kindSeparator},
		item{label: m.Settings, do: act.OpenSettings},
		item{label: m.Quit, do: act.Quit},
	)
	return items
}

// key identifies what a menu looks like, ignoring the callbacks.
func key(items []item) string {
	var b strings.Builder
	var walk func([]item)
	walk = func(items []item) {
		for _, it := range items {
			b.WriteString(strconv.Itoa(int(it.kind)))
			b.WriteString(strconv.FormatBool(it.checked))
			b.WriteString(it.label)
			b.WriteByte(0)
			if it.sub != nil {
				b.WriteByte('[')
				walk(it.sub)
				b.WriteByte(']')
			}
		}
	}
	walk(items)
	return b.String()
}

// fill materialises items into a Wails menu.
func fill(menu *application.Menu, items []item) {
	for _, it := range items {
		var mi *application.MenuItem
		switch it.kind {
		case kindSeparator:
			menu.AddSeparator()
			continue
		case kindSubmenu:
			fill(menu.AddSubmenu(it.label), it.sub)
			continue
		case kindRadio:
			mi = menu.AddRadio(it.label, it.checked)
		case kindCheckbox:
			mi = menu.AddCheckbox(it.label, it.checked)
		case kindDisabled:
			mi = menu.Add(it.label).SetEnabled(false)
		default:
			mi = menu.Add(it.label)
		}
		if do := it.do; do != nil {
			mi.OnClick(func(*application.Context) { do() })
		}
	}
}

// release drops a replaced menu's items from Wails' global click-dispatch map, which otherwise
// keeps every menu ever built.
func release(menu *application.Menu) {
	if menu == nil {
		return
	}
	for i := 0; ; i++ {
		it := menu.ItemAt(i)
		if it == nil {
			break
		}
		release(it.GetSubmenu())
	}
	menu.Clear()
}
