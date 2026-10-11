package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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

func TestConsumerOwnsOnlyPageAndMakefile(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir, true, false); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if !reflect.DeepEqual(names, []string{"Makefile", "index.html", "style.css"}) {
		t.Fatalf("consumer files: %v", names)
	}
	out := t.TempDir()
	if err := Write(out, true, true); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.js", "keys.js", "notices.py", "bubbles.svg", "licenses/tinygo.txt"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestModuleSurvivesForceRefresh(t *testing.T) {
	dir := t.TempDir()
	custom := []byte("module example.org/app\n\nrequire example.org/library v1.0.0\n\nreplace example.org/library => ../library\n")
	if err := WriteModule(dir, custom); err != nil {
		t.Fatal(err)
	}
	if err := WriteModule(dir, []byte("replacement")); err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, true, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil || string(got) != string(custom) {
		t.Fatalf("module changed: %q, %v", got, err)
	}
}

func TestAssetsIgnoreLegacyConsumerJavaScript(t *testing.T) {
	// Run the real assets recipe with an offline stand-in for the asset tool.
	// Old scaffolds may still have JavaScript beside their custom HTML and CSS.
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	bin := filepath.Join(dir, "bin")
	for _, path := range []string{out, bin} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"index.html", "style.css", "main.js", "keys.js"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("consumer"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	stub := "#!/bin/sh\nprintf fresh > \"$HARNESS_OUT/main.js\"\nprintf fresh > \"$HARNESS_OUT/keys.js\"\n"
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte(stub), 0755); err != nil {
		t.Fatal(err)
	}
	mk, err := filepath.Abs("../../application.mk")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("make", "-f", mk, "assets", "SITE=.", "OUT="+out)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "HARNESS_OUT="+out)
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make: %v\n%s", err, result)
	}
	for _, name := range []string{"main.js", "keys.js", "index.html", "style.css"} {
		want := "fresh"
		if name == "index.html" || name == "style.css" {
			want = "consumer"
		}
		got, err := os.ReadFile(filepath.Join(out, name))
		if err != nil || string(got) != want {
			t.Fatalf("%s: %q, %v", name, got, err)
		}
	}
}
