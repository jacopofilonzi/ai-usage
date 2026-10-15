package autostart

import (
	"slices"
	"testing"
)

// Enable itself is not tested: it would write the real registry value or login file.

func TestCommandLinePutsExecutableFirst(t *testing.T) {
	args := []string{"--hidden"}
	got := commandLine("/bin/ai-usage", args)
	if !slices.Equal(got, []string{"/bin/ai-usage", "--hidden"}) {
		t.Fatalf("command line = %v", got)
	}
	if got := commandLine("/bin/ai-usage", nil); !slices.Equal(got, []string{"/bin/ai-usage"}) {
		t.Fatalf("command line without args = %v", got)
	}
}
