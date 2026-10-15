package autostart

// System exposes this package's functions as a value, for callers that take an interface (so
// their tests can swap in a fake instead of touching the real registration).
type System struct{}

func (System) Enabled() (bool, error)     { return Enabled() }
func (System) Enable(args []string) error { return Enable(args) }
func (System) Disable() error             { return Disable() }
