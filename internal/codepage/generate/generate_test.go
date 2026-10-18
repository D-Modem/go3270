package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadBadLine: a CHARMAP line read cannot is an error naming it, not a
// panic; a good file's mappings are read, the non-roundtrip ones left out.
func TestReadBadLine(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.ucm")
	os.WriteFile(good, []byte("CHARMAP\n<U0041> \\xC1 |0\n<U00A0> \\x41 |1\nEND CHARMAP\n"), 0644)
	if m, err := read(good); err != nil || len(m) != 1 || m[0x41] != 0xc1 {
		t.Errorf("good: %v %v", m, err)
	}
	bad := filepath.Join(dir, "bad.ucm")
	os.WriteFile(bad, []byte("CHARMAP\nnonsense\nEND CHARMAP\n"), 0644)
	if _, err := read(bad); err == nil || !strings.Contains(err.Error(), "nonsense") {
		t.Errorf("bad: %v", err)
	}
}
