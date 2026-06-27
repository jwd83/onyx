package main

import (
	"os"
	"path/filepath"
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
