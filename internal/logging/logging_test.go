package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRotatesOversizedLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, logName)
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), maxSize+1), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if fi, err := os.Stat(path); err != nil || fi.Size() != 0 {
		t.Fatalf("new log not empty: %v %v", fi, err)
	}
	if fi, err := os.Stat(path + ".1"); err != nil || fi.Size() != maxSize+1 {
		t.Fatalf("old log not kept: %v %v", fi, err)
	}
}

func TestOpenAppendsSmallLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, logName)
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("new\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "old\nnew\n" {
		t.Fatalf("log = %q %v", b, err)
	}
	if _, err := os.Stat(path + ".1"); !os.IsNotExist(err) {
		t.Fatalf("small log rotated: %v", err)
	}
}
