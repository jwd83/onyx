package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

// defaultSources are the conventional content folders Onyx looks for when no
// explicit source is configured. Every one that exists is included in the
// build: one folder builds as-is, several build a sectioned site.
var defaultSources = []string{"doc", "docs", "plan", "plans", "wiki"}

type Config struct {
	Root       string
	ConfigPath string
	SiteTitle  string
	Sources    []string
	Multi      bool
	Theme      string
	Search     bool
	Graph      bool
	ShowSource bool
}

// sourcePrefix is the path segment prepended when turning an in-vault relative
// path into a real, root-relative one (for Markdown-source links and assets).
// With a single source every page path is relative to that folder, so the
// folder name is the prefix. With several sources page paths are already
// root-relative (they carry their own folder), so the prefix is empty.
func (c Config) sourcePrefix() string {
	if c.Multi || len(c.Sources) == 0 {
		return ""
	}
	return c.Sources[0]
}

func findConfig(start string) (string, string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() {
		abs = filepath.Dir(abs)
	}

	for {
		ini := filepath.Join(abs, "onyx.ini")
		if _, err := os.Stat(ini); err == nil {
			return abs, ini, nil
		}
		if hasDefaultSource(abs) {
			// No onyx.ini, but a conventional source folder (docs/, wiki/, ...)
			// marks the site root. The returned path is where an optional
			// onyx.ini would live; readConfig tolerates its absence.
			return abs, ini, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", "", errors.New("no onyx.ini or content directory (doc/, docs/, plan/, plans/, wiki/) found in this directory or its parents")
		}
		abs = parent
	}
}

func hasDefaultSource(dir string) bool {
	for _, name := range defaultSources {
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

func readConfig(root, configPath string) (Config, error) {
	// onyx.ini is optional: a missing file leaves every setting at its default.
	values := map[string]string{}
	if configPath != "" {
		parsed, err := parseINI(configPath)
		if err != nil && !os.IsNotExist(err) {
			return Config{}, err
		}
		if parsed != nil {
			values = parsed
		}
	}

	sources, err := resolveSources(root, values)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Root:       root,
		ConfigPath: configPath,
		// An empty title is a sentinel: loadVault later falls back to the home
		// page's title and finally to "Onyx".
		SiteTitle:  valueOr(values, "site_title", ""),
		Sources:    sources,
		Multi:      len(sources) > 1,
		Theme:      valueOr(values, "theme", "theme"),
		Search:     boolOr(values, true, "search", "build.search"),
		Graph:      boolOr(values, true, "graph", "build.graph"),
		ShowSource: boolOr(values, true, "show_source", "publish_raw_markdown"),
	}

	if filepath.IsAbs(cfg.Theme) {
		return Config{}, errors.New("theme must be relative to the site root")
	}

	return cfg, nil
}

// resolveSources determines which content folders to build. An explicit
// source = a, b, c in onyx.ini wins and every listed folder must exist;
// otherwise Onyx auto-detects which of the default folders are present and
// includes all of them.
func resolveSources(root string, values map[string]string) ([]string, error) {
	requested := defaultSources
	explicit := false
	if raw, ok := values["source"]; ok && strings.TrimSpace(raw) != "" {
		requested = splitList(raw)
		explicit = true
	}

	var sources []string
	for _, name := range requested {
		if filepath.IsAbs(name) {
			return nil, errors.New("source must be relative to the site root")
		}
		dir := filepath.Join(root, filepath.FromSlash(name))
		info, err := os.Stat(dir)
		if err != nil {
			if explicit {
				return nil, fmt.Errorf("source directory %q: %w", name, err)
			}
			continue // an auto-detected default that simply isn't present
		}
		if !info.IsDir() {
			if explicit {
				return nil, fmt.Errorf("source %q is not a directory", name)
			}
			continue
		}
		sources = append(sources, path.Clean(filepath.ToSlash(name)))
	}

	if len(sources) == 0 {
		if explicit {
			return nil, errors.New("none of the configured source directories exist")
		}
		return nil, fmt.Errorf("no content directory found (looked for %s)", strings.Join(defaultSources, ", "))
	}
	return sources, nil
}

// splitList parses a comma- or whitespace-separated value into a deduplicated,
// order-preserving list.
func splitList(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
	seen := map[string]bool{}
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}

func parseINI(filename string) (map[string]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	text := strings.ReplaceAll(string(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))), "\r\n", "\n")
	values := map[string]string{}
	section := ""

	for lineNo, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d: expected key = value", filename, lineNo+1)
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = stripQuotes(strings.TrimSpace(stripInlineComment(val)))
		if section != "" {
			values[section+"."+key] = val
		}
		values[key] = val
	}

	return values, nil
}

func stripInlineComment(s string) string {
	inQuote := rune(0)
	for i, r := range s {
		if r == '\'' || r == '"' {
			if inQuote == 0 {
				inQuote = r
			} else if inQuote == r {
				inQuote = 0
			}
			continue
		}
		if inQuote == 0 && (r == '#' || r == ';') {
			if i == 0 || unicode.IsSpace(rune(s[i-1])) {
				return strings.TrimSpace(s[:i])
			}
		}
	}
	return s
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func valueOr(values map[string]string, key, fallback string) string {
	if v, ok := values[key]; ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func boolOr(values map[string]string, fallback bool, keys ...string) bool {
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			switch strings.ToLower(strings.TrimSpace(raw)) {
			case "1", "true", "yes", "on":
				return true
			case "0", "false", "no", "off":
				return false
			}
		}
	}
	return fallback
}
