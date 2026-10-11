package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestForcePreservesConsumerModule(t *testing.T) {
	target := filepath.Join(t.TempDir(), "consumer")
	harness, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	run := func(force bool) {
		t.Helper()
		args := []string{"run", ".", "--module", "example.org/consumer", "--local-harness", harness}
		if force {
			args = append(args, "--force")
		}
		cmd := exec.Command("go", append(args, target)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("scaffold: %v\n%s", err, out)
		}
	}
	run(false)
	path := filepath.Join(target, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte("\nrequire example.org/custom v1.0.0\nreplace example.org/custom => ../custom\n")...)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "style.css"), []byte("custom"), 0644); err != nil {
		t.Fatal(err)
	}
	run(true)
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(data) {
		t.Fatalf("force changed go.mod: %q, %v", got, err)
	}
	css, err := os.ReadFile(filepath.Join(target, "style.css"))
	if err != nil || string(css) == "custom" {
		t.Fatalf("force did not refresh the page: %v", err)
	}
}

func TestAssetRefreshPrunesOnlyHarnessFiles(t *testing.T) {
	stage, target := t.TempDir(), t.TempDir()
	write := func(dir, name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(stage, "obsolete.js")
	write(stage, "main.js")
	if err := publishAssets(stage, target, true); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"app.wasm", "go.mod", "custom.png", "demos.js"} {
		write(target, name)
	}
	if err := os.Remove(filepath.Join(stage, "obsolete.js")); err != nil {
		t.Fatal(err)
	}
	if err := publishAssets(stage, target, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "obsolete.js")); err != nil {
		t.Fatal("non-force removed file", err)
	}
	if err := publishAssets(stage, target, true); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"obsolete.js", "demos.js"} {
		if _, err := os.Stat(filepath.Join(target, name)); !os.IsNotExist(err) {
			t.Fatal("stale file survived", name, err)
		}
	}
	for _, name := range []string{"app.wasm", "go.mod", "custom.png", "main.js"} {
		if _, err := os.Stat(filepath.Join(target, name)); err != nil {
			t.Fatal("lost file", name, err)
		}
	}
}

func TestAssetManifestRejectsEscapes(t *testing.T) {
	stage, target := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(target, assetManifest), []byte(`["../outside"]`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := publishAssets(stage, target, true); err == nil {
		t.Fatal("accepted traversal")
	}
}
