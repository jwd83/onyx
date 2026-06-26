package main

import (
	"io/fs"
	"os"
	"path/filepath"
)

func writeAssets(vault *Vault) error {
	publicDir := filepath.Join(vault.Config.Root, "public")
	themeDir := filepath.Join(vault.Config.Root, filepath.FromSlash(vault.Config.Theme))

	if err := writeThemeOrDefault(filepath.Join(themeDir, "style.css"), filepath.Join(publicDir, "onyx.css"), defaultCSS); err != nil {
		return err
	}
	if err := writeThemeOrDefault(filepath.Join(themeDir, "onyx.js"), filepath.Join(publicDir, "onyx.js"), defaultJS); err != nil {
		return err
	}

	staticDir := filepath.Join(themeDir, "static")
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		return copyDir(staticDir, filepath.Join(publicDir, "theme"))
	}
	return nil
}

func writeThemeOrDefault(source, dest, fallback string) error {
	if data, err := os.ReadFile(source); err == nil {
		return os.WriteFile(dest, data, 0o644)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(dest, []byte(fallback), 0o644)
}

func copyDir(source, dest string) error {
	return filepath.WalkDir(source, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if abs == source {
			return nil
		}
		if shouldSkipName(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(source, abs)
		if err != nil {
			return err
		}
		out := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o644)
	})
}
