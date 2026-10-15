package i18n

import (
	"os/exec"
	"strings"
)

func systemLanguage() string {
	if v := envLanguage(); v != "" {
		return v
	}
	// Apps started from Finder or launchd get no LANG: ask the user defaults.
	out, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
