package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	return filepath.Join(dir, "autostart", "ai-usage.desktop"), nil
}

func render(cmdline []string) string {
	quoted := make([]string, len(cmdline))
	for i, a := range cmdline {
		quoted[i] = quoteExec(a)
	}
	return "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=AI Usage\n" +
		"Comment=Show AI agents' usage limits\n" +
		"Exec=" + strings.Join(quoted, " ") + "\n" +
		"Terminal=false\n" +
		"X-GNOME-Autostart-enabled=true\n"
}

// quoteExec quotes one argument following the Desktop Entry spec for the Exec key: inside double
// quotes, the double quote, backtick, dollar sign and backslash are backslash-escaped; then every
// backslash is escaped again because the value is also a string-type key. A percent sign is
// doubled so it is not read as a field code.
func quoteExec(a string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range a {
		switch r {
		case '"', '`', '$':
			b.WriteString(`\\`)
			b.WriteRune(r)
		case '\\':
			b.WriteString(`\\\\`)
		case '%':
			b.WriteString("%%")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
