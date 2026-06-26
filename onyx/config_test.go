package main

import (
	"os"
	"path/filepath"
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
