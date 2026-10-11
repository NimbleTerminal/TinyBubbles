package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreserveAndForce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.html")
	if err := os.WriteFile(path, []byte("custom"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, false, false); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "custom" {
		t.Fatal("overwrote existing file")
	}
	if _, err := os.Stat(filepath.Join(dir, "Makefile")); err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, true, true); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if string(data) == "custom" {
		t.Fatal("force did not replace file")
	}
}
