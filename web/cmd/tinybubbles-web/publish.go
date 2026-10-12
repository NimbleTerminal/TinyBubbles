package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"nimbleterminal.dev/tinybubbles/web/internal/scaffold"
)

const assetManifest = ".tinybubbles-assets.json"

// Track only harness-owned output. App binaries and custom assets are not pruned.
func publishAssets(stage, target string, force bool) error {
	var previous []string
	data, err := os.ReadFile(filepath.Join(target, assetManifest))
	if err == nil {
		if err := json.Unmarshal(data, &previous); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, name := range previous {
		if !filepath.IsLocal(name) || name == "." {
			return fmt.Errorf("invalid asset path %q", name)
		}
	}
	current := make(map[string]bool)
	var names []string
	err = filepath.WalkDir(stage, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := scaffold.WriteFile(filepath.Join(target, name), data, force); err != nil {
			return err
		}
		names = append(names, name)
		current[name] = true
		return nil
	})
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(target)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, name := range previous {
		if current[name] {
			continue
		}
		if force {
			if err := root.Remove(name); err != nil && !os.IsNotExist(err) {
				return err
			}
		} else {
			names = append(names, name)
			current[name] = true
		}
	}
	data, err = json.Marshal(names)
	if err != nil {
		return err
	}
	return scaffold.WriteFile(filepath.Join(target, assetManifest), data, true)
}
