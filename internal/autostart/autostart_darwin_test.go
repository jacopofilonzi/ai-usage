package autostart

import (
	"strings"
	"testing"
)

func TestRenderPlist(t *testing.T) {
	got := render([]string{"/Applications/AI Usage.app/Contents/MacOS/ai-usage", "a&b"})
	for _, want := range []string{
		"<string>com.jacopofilonzi.ai-usage</string>",
		"<string>/Applications/AI Usage.app/Contents/MacOS/ai-usage</string>",
		"<string>a&amp;b</string>",
		"<key>RunAtLoad</key>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("plist lacks %q:\n%s", want, got)
		}
	}
}
