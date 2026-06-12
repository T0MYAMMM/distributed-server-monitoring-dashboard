package logtail

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLine(t *testing.T) {
	// log-geulis formatted line.
	l := parseLine("2026-06-12T10:00:00Z | error | db | connection refused | x=1", "/var/log/app.log")
	if l.Ts != "2026-06-12T10:00:00Z" || l.Level != "ERROR" || l.Module != "db" {
		t.Errorf("structured parse wrong: %+v", l)
	}
	if l.Message != "connection refused | x=1" {
		t.Errorf("message should keep the remainder: %q", l.Message)
	}

	// Non-formatted line falls back to INFO + file base name.
	r := parseLine("just some text", "/var/log/nginx/access.log")
	if r.Level != "INFO" || r.Module != "access.log" || r.Message != "just some text" {
		t.Errorf("fallback parse wrong: %+v", r)
	}
}

func TestReadNewTracksOffsetAndRotation(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "app.log")
	if err := os.WriteFile(p, []byte("2026-06-12T10:00:00Z | INFO | app | one\nplain line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := New("http://hub:5000", "web-1", []string{p})

	got := tl.readNew(p)
	if len(got) != 2 || got[0].Message != "one" || got[1].Message != "plain line" {
		t.Fatalf("first read = %+v", got)
	}

	// Append: only the new line is returned.
	f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString("more\n")
	f.Close()
	if got2 := tl.readNew(p); len(got2) != 1 || got2[0].Message != "more" {
		t.Fatalf("append read = %+v", got2)
	}

	// No change: nothing returned.
	if got3 := tl.readNew(p); len(got3) != 0 {
		t.Fatalf("no-change read = %+v", got3)
	}

	// Rotation/truncate: file shrinks, re-read from the start.
	if err := os.WriteFile(p, []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got4 := tl.readNew(p); len(got4) != 1 || got4[0].Message != "after" {
		t.Fatalf("rotation read = %+v", got4)
	}
}
