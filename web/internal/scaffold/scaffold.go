package scaffold

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed files
var Files embed.FS

func Write(target string, force, assets bool) error {
	return fs.WalkDir(Files, "files", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name := path[len("files/"):]
		// Consumers own only page markup/styles and their build entry point.
		// JavaScript and support assets always come from the installed harness.
		if !assets && name != "index.html" && name != "style.css" && name != "Makefile" {
			return nil
		}
		if assets && name == "Makefile" {
			return nil
		}
		data, err := Files.ReadFile(path)
		if err != nil {
			return err
		}
		return WriteFile(filepath.Join(target, name), data, force)
	})
}
func WriteFile(path string, data []byte, force bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, 0644)
	if os.IsExist(err) && !force {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

// WriteModule creates a consumer module once. Asset refreshes never replace it.
func WriteModule(target string, data []byte) error {
	return WriteFile(filepath.Join(target, "go.mod"), data, false)
}
