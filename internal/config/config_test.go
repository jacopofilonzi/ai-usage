package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileGivesDefaults(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if c != Default() {
		t.Fatalf("got %+v, want defaults", c)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	want := Config{Provider: "codex", Corner: TopLeft, Monitor: MonitorAll, Language: "it", PollSeconds: 120, KeepUnlocked: true}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadToleratesInvalidAndPartialFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := `{"corner":"middle","language":"fr","poll_seconds":5,"keep_unlocked":true,"future_field":1}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.PollSeconds = 30
	want.KeepUnlocked = true
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadCorruptFileGivesDefaultsAndError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err == nil {
		t.Fatal("want error")
	}
	if got != Default() {
		t.Fatalf("got %+v, want defaults", got)
	}
}
