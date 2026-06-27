package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeINI(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "onyx.ini")
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestParseINI(t *testing.T) {
	t.Run("BOM and CRLF with section dual keys", func(t *testing.T) {
		f := writeINI(t, "\xef\xbb\xbf[Build]\r\nsearch = true\r\n")
		values, err := parseINI(f)
		if err != nil {
			t.Fatal(err)
		}
		if values["search"] != "true" {
			t.Errorf("bare key search = %q, want true", values["search"])
		}
		if values["build.search"] != "true" {
			t.Errorf("section key build.search = %q, want true", values["build.search"])
		}
	})

	t.Run("inline comment after whitespace is stripped", func(t *testing.T) {
		f := writeINI(t, "site_title = Hello World # a comment\n")
		values, err := parseINI(f)
		if err != nil {
			t.Fatal(err)
		}
		if values["site_title"] != "Hello World" {
			t.Errorf("site_title = %q, want \"Hello World\"", values["site_title"])
		}
	})

	t.Run("hash inside quotes is preserved and quotes stripped", func(t *testing.T) {
		f := writeINI(t, `color = "#ffcc00"`+"\n")
		values, err := parseINI(f)
		if err != nil {
			t.Fatal(err)
		}
		if values["color"] != "#ffcc00" {
			t.Errorf("color = %q, want \"#ffcc00\"", values["color"])
		}
	})

	t.Run("comment and blank lines are ignored", func(t *testing.T) {
		f := writeINI(t, "# top comment\n\n; semicolon comment\nkey = value\n")
		values, err := parseINI(f)
		if err != nil {
			t.Fatal(err)
		}
		if values["key"] != "value" {
			t.Errorf("key = %q, want value", values["key"])
		}
	})

	t.Run("line without equals is an error", func(t *testing.T) {
		f := writeINI(t, "site_title = ok\nnonsense line\n")
		if _, err := parseINI(f); err == nil {
			t.Fatal("expected error for line without '=', got nil")
		}
	})
}

func TestFindConfigDiscovery(t *testing.T) {
	t.Run("walks up to onyx.ini from nested directory", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "onyx.ini", "source = docs\n")
		if err := os.MkdirAll(filepath.Join(root, "docs", "deep"), 0o755); err != nil {
			t.Fatal(err)
		}

		gotRoot, gotConfig, err := findConfig(filepath.Join(root, "docs", "deep"))
		if err != nil {
			t.Fatal(err)
		}
		if gotRoot != root {
			t.Fatalf("root = %q, want %q", gotRoot, root)
		}
		wantConfig := filepath.Join(root, "onyx.ini")
		if gotConfig != wantConfig {
			t.Fatalf("config = %q, want %q", gotConfig, wantConfig)
		}
	})

	t.Run("content folder marks root without onyx.ini", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "wiki/deep/Note.md", "# Note\n")

		gotRoot, gotConfig, err := findConfig(filepath.Join(root, "wiki", "deep", "Note.md"))
		if err != nil {
			t.Fatal(err)
		}
		if gotRoot != root {
			t.Fatalf("root = %q, want %q", gotRoot, root)
		}
		wantConfig := filepath.Join(root, "onyx.ini")
		if gotConfig != wantConfig {
			t.Fatalf("config = %q, want optional path %q", gotConfig, wantConfig)
		}
	})

	t.Run("missing root errors clearly", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
			t.Fatal(err)
		}

		_, _, err := findConfig(filepath.Join(root, "nested"))
		if err == nil {
			t.Fatal("findConfig succeeded, want error")
		}
		if !strings.Contains(err.Error(), "no onyx.ini or content directory") {
			t.Fatalf("error = %q, want missing-root message", err)
		}
	})
}

func TestReadConfigSourceResolutionContract(t *testing.T) {
	t.Run("auto-detects existing conventional sources in default order", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "wiki/Beta.md", "# Beta\n")
		writeTestFile(t, root, "docs/Alpha.md", "# Alpha\n")

		cfg, err := readConfig(root, filepath.Join(root, "onyx.ini"))
		if err != nil {
			t.Fatal(err)
		}
		assertSources(t, cfg, []string{"docs", "wiki"}, true)
	})

	t.Run("explicit single source stays unsectioned", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "docs/index.md", "# Home\n")
		writeTestFile(t, root, "onyx.ini", "source = docs\n")

		cfg, err := readConfig(root, filepath.Join(root, "onyx.ini"))
		if err != nil {
			t.Fatal(err)
		}
		assertSources(t, cfg, []string{"docs"}, false)
	})

	t.Run("explicit multi source includes non-default folder", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "docs/Alpha.md", "# Alpha\n")
		writeTestFile(t, root, "notes/Gamma.md", "# Gamma\n")
		writeTestFile(t, root, "onyx.ini", "source = docs, notes\n")

		cfg, err := readConfig(root, filepath.Join(root, "onyx.ini"))
		if err != nil {
			t.Fatal(err)
		}
		assertSources(t, cfg, []string{"docs", "notes"}, true)
	})

	t.Run("missing explicit source errors loudly", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "docs/index.md", "# Home\n")
		writeTestFile(t, root, "onyx.ini", "source = docs, missing\n")

		_, err := readConfig(root, filepath.Join(root, "onyx.ini"))
		if err == nil {
			t.Fatal("readConfig succeeded, want error")
		}
		if !strings.Contains(err.Error(), `source directory "missing"`) {
			t.Fatalf("error = %q, want missing explicit source", err)
		}
	})

	t.Run("explicit source must be a directory", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "docs", "not a directory\n")
		writeTestFile(t, root, "onyx.ini", "source = docs\n")

		_, err := readConfig(root, filepath.Join(root, "onyx.ini"))
		if err == nil {
			t.Fatal("readConfig succeeded, want error")
		}
		if !strings.Contains(err.Error(), `source "docs" is not a directory`) {
			t.Fatalf("error = %q, want non-directory explicit source", err)
		}
	})

	t.Run("no implicit sources found errors clearly", func(t *testing.T) {
		root := t.TempDir()

		_, err := readConfig(root, filepath.Join(root, "onyx.ini"))
		if err == nil {
			t.Fatal("readConfig succeeded, want error")
		}
		if !strings.Contains(err.Error(), "no content directory found") {
			t.Fatalf("error = %q, want no-content-directory message", err)
		}
	})
}

func TestReadConfigCompatibilityContract(t *testing.T) {
	t.Run("legacy toggle aliases are still honored", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "docs/index.md", "# Home\n")
		writeTestFile(t, root, "onyx.ini", strings.Join([]string{
			"source = docs",
			"build.search = false",
			"build.graph = no",
			"publish_raw_markdown = off",
			"",
		}, "\n"))

		cfg, err := readConfig(root, filepath.Join(root, "onyx.ini"))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Search {
			t.Error("Search = true, want false from legacy build.search alias")
		}
		if cfg.Graph {
			t.Error("Graph = true, want false from legacy build.graph alias")
		}
		if cfg.ShowSource {
			t.Error("ShowSource = true, want false from legacy publish_raw_markdown alias")
		}
	})

	t.Run("modern toggles win over legacy aliases", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "docs/index.md", "# Home\n")
		writeTestFile(t, root, "onyx.ini", strings.Join([]string{
			"source = docs",
			"search = true",
			"build.search = false",
			"graph = true",
			"build.graph = false",
			"show_source = true",
			"publish_raw_markdown = false",
			"",
		}, "\n"))

		cfg, err := readConfig(root, filepath.Join(root, "onyx.ini"))
		if err != nil {
			t.Fatal(err)
		}
		if !cfg.Search {
			t.Error("Search = false, want true from modern search key")
		}
		if !cfg.Graph {
			t.Error("Graph = false, want true from modern graph key")
		}
		if !cfg.ShowSource {
			t.Error("ShowSource = false, want true from modern show_source key")
		}
	})

	t.Run("unknown keys are tolerated", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "docs/index.md", "# Home\n")
		writeTestFile(t, root, "onyx.ini", strings.Join([]string{
			"source = docs",
			"base_url = /ignored",
			"future_setting = maybe",
			"",
		}, "\n"))

		cfg, err := readConfig(root, filepath.Join(root, "onyx.ini"))
		if err != nil {
			t.Fatal(err)
		}
		if len(cfg.Sources) != 1 || cfg.Sources[0] != "docs" {
			t.Fatalf("Sources = %v, want [docs]", cfg.Sources)
		}
		if !cfg.Search || !cfg.Graph || !cfg.ShowSource {
			t.Fatalf("default toggles changed: Search=%v Graph=%v ShowSource=%v", cfg.Search, cfg.Graph, cfg.ShowSource)
		}
	})
}

func assertSources(t *testing.T, cfg Config, want []string, wantMulti bool) {
	t.Helper()
	if !reflect.DeepEqual(cfg.Sources, want) {
		t.Fatalf("Sources = %v, want %v", cfg.Sources, want)
	}
	if cfg.Multi != wantMulti {
		t.Fatalf("Multi = %v, want %v", cfg.Multi, wantMulti)
	}
}
