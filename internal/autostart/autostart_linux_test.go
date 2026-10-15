package autostart

import (
	"strings"
	"testing"
)

func TestQuoteExec(t *testing.T) {
	cases := map[string]string{
		"/usr/bin/ai-usage": `"/usr/bin/ai-usage"`,
		"/opt/my app/bin":   `"/opt/my app/bin"`,
		`a"b`:               `"a\\"b"`,
		`a\b`:               `"a\\\\b"`,
		"50%":               `"50%%"`,
		"$HOME":             `"\\$HOME"`,
	}
	for in, want := range cases {
		if got := quoteExec(in); got != want {
			t.Errorf("quoteExec(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestRenderDesktopEntry(t *testing.T) {
	got := render([]string{"/opt/AI Usage/ai-usage", "--hidden"})
	for _, want := range []string{
		"Name=AI Usage\n",
		`Exec="/opt/AI Usage/ai-usage" "--hidden"` + "\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("entry lacks %q:\n%s", want, got)
		}
	}
}

func TestPathUsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	p, err := path()
	if err != nil || p != "/tmp/xdg/autostart/ai-usage.desktop" {
		t.Fatalf("path = %q %v", p, err)
	}
}
