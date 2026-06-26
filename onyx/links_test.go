package main

import (
	"path"
	"sort"
	"strings"
	"testing"
)

// testVault builds an in-memory Vault and indexes it exactly the way loadVault
// does (see vault.go: PageRel/URL assignment and the ByPath/ByBase/asset
// indexes), without touching the filesystem. It lets the resolver and inline
// tests assert behavior directly against the pure functions.
func testVault(cfg Config, noteRels []string, assetRels []string) *Vault {
	v := &Vault{
		Config:       cfg,
		ByPath:       map[string]*Page{},
		ByBase:       map[string][]*Page{},
		AssetsByPath: map[string]string{},
		AssetsByBase: map[string][]string{},
	}
	for _, sr := range noteRels {
		p := &Page{SourceRel: sr}
		p.PageRel = strings.TrimSuffix(sr, path.Ext(sr))
		v.Notes = append(v.Notes, p)
	}
	sort.Slice(v.Notes, func(i, j int) bool {
		return strings.ToLower(v.Notes[i].SourceRel) < strings.ToLower(v.Notes[j].SourceRel)
	})
	for _, p := range v.Notes {
		v.ByPath[strings.ToLower(p.PageRel)] = p
		base := strings.ToLower(path.Base(p.PageRel))
		v.ByBase[base] = append(v.ByBase[base], p)
		if p.PageRel == "index" {
			p.IsHome = true
			p.URL = ""
		} else {
			p.URL = "public/" + escapeURLPath(p.PageRel) + "/"
		}
		if p.SourceRel != "" {
			p.SourceURL = escapeURLPath(path.Join(cfg.sourcePrefix(), p.SourceRel))
		}
	}
	for _, list := range v.ByBase {
		sort.Slice(list, func(i, j int) bool {
			return strings.ToLower(list[i].SourceRel) < strings.ToLower(list[j].SourceRel)
		})
	}
	for _, a := range assetRels {
		v.addAsset(a)
	}
	return v
}

// testRenderer returns a renderer positioned at currentRel (a note source path)
// plus a pointer to its collected warnings.
func testRenderer(v *Vault, currentRel string) (*MarkdownRenderer, *[]string) {
	var warnings []string
	cur := v.ByPath[strings.ToLower(strings.TrimSuffix(currentRel, path.Ext(currentRel)))]
	r := &MarkdownRenderer{
		vault:      v,
		current:    cur,
		warnings:   &warnings,
		headingIDs: map[string]int{},
		outgoing:   map[string]bool{},
	}
	return r, &warnings
}

func TestSanitizeHref(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"plain https", "https://example.com", "https://example.com"},
		{"absolute path", "/foo/bar", "/foo/bar"},
		{"mailto", "mailto:a@b.com", "mailto:a@b.com"},
		{"relative", "../notes/x", "../notes/x"},
		{"javascript", "javascript:alert(1)", "#"},
		{"javascript mixed case", "JavaScript:alert(1)", "#"},
		{"javascript leading whitespace", "   javascript:alert(1)", "#"},
		{"data uri", "data:text/html,<script>", "#"},
		{"data uri upper", "DATA:text/html;base64,xxx", "#"},
		// Known limitation: the guard only strips a literal leading
		// "javascript:"/"data:" scheme. An internal-whitespace bypass is NOT
		// caught today. This pins current behavior so step 3's resolver
		// consolidation cannot change it silently — revisit when hardening
		// the HTML-safety model.
		{"internal tab not caught", "java\tscript:alert(1)", "java\tscript:alert(1)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeHref(tc.in); got != tc.want {
				t.Errorf("sanitizeHref(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestResolveNote(t *testing.T) {
	v := testVault(Config{}, []string{
		"index.md",
		"Guide.md",
		"sub/Deep.md",
		"sub/Nested.md",
		"a/Twin.md",
		"b/Twin.md",
		"a/Solo.md",
		"a/sub/Solo.md",
		"a/Here.md",
	}, nil)

	t.Run("empty target returns current page", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveNote(""); got != r.current {
			t.Errorf("empty target = %v, want current page", got)
		}
	})

	t.Run("as-is exact match", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveNote("Guide"); got == nil || got.PageRel != "Guide" {
			t.Errorf("resolveNote(Guide) = %v, want Guide", got)
		}
	})

	t.Run("strips .md suffix", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveNote("Guide.md"); got == nil || got.PageRel != "Guide" {
			t.Errorf("resolveNote(Guide.md) = %v, want Guide", got)
		}
	})

	t.Run("current-directory relative wins", func(t *testing.T) {
		r, _ := testRenderer(v, "sub/Deep.md")
		if got := r.resolveNote("Nested"); got == nil || got.PageRel != "sub/Nested" {
			t.Errorf("resolveNote(Nested) from sub/Deep = %v, want sub/Nested", got)
		}
	})

	t.Run("unique basename fallback", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveNote("Deep"); got == nil || got.PageRel != "sub/Deep" {
			t.Errorf("resolveNote(Deep) = %v, want sub/Deep", got)
		}
	})

	t.Run("slash path with no match returns nil (no basename fallback)", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveNote("nope/Deep"); got != nil {
			t.Errorf("resolveNote(nope/Deep) = %v, want nil", got)
		}
	})

	t.Run("ambiguous equidistant basename returns nil and warns", func(t *testing.T) {
		r, warnings := testRenderer(v, "index.md")
		if got := r.resolveNote("Twin"); got != nil {
			t.Errorf("resolveNote(Twin) = %v, want nil (ambiguous)", got)
		}
		if len(*warnings) != 1 || !strings.Contains((*warnings)[0], "ambiguous wikilink [[Twin]]") {
			t.Errorf("warnings = %v, want one ambiguous-wikilink warning", *warnings)
		}
	})

	t.Run("ambiguous basename resolved by nearest folder", func(t *testing.T) {
		r, warnings := testRenderer(v, "a/Here.md")
		if got := r.resolveNote("Solo"); got == nil || got.PageRel != "a/Solo" {
			t.Errorf("resolveNote(Solo) from a/Here = %v, want a/Solo (nearest)", got)
		}
		if len(*warnings) != 0 {
			t.Errorf("unexpected warnings: %v", *warnings)
		}
	})
}

func TestResolveAsset(t *testing.T) {
	v := testVault(Config{}, []string{
		"index.md",
		"a/Note.md",
		"z/Note.md",
	}, []string{
		"img/logo.png",
		"a/pic.png",
		"b/pic.png",
		"a/icon.png",
		"a/sub/icon.png",
	})

	t.Run("as-is path match", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		rel, ok := r.resolveAsset("img/logo.png")
		if !ok || rel != "img/logo.png" {
			t.Errorf("resolveAsset(img/logo.png) = %q,%v want img/logo.png,true", rel, ok)
		}
	})

	t.Run("current-directory relative match", func(t *testing.T) {
		r, _ := testRenderer(v, "a/Note.md")
		rel, ok := r.resolveAsset("pic.png")
		if !ok || rel != "a/pic.png" {
			t.Errorf("resolveAsset(pic.png) from a/Note = %q,%v want a/pic.png,true", rel, ok)
		}
	})

	t.Run("unique basename fallback", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		rel, ok := r.resolveAsset("logo.png")
		if !ok || rel != "img/logo.png" {
			t.Errorf("resolveAsset(logo.png) = %q,%v want img/logo.png,true", rel, ok)
		}
	})

	t.Run("ambiguous equidistant basename fails and warns", func(t *testing.T) {
		r, warnings := testRenderer(v, "index.md")
		rel, ok := r.resolveAsset("pic.png")
		if ok || rel != "" {
			t.Errorf("resolveAsset(pic.png) from index = %q,%v want \"\",false", rel, ok)
		}
		if len(*warnings) != 1 || !strings.Contains((*warnings)[0], "ambiguous asset [[pic.png]]") {
			t.Errorf("warnings = %v, want one ambiguous-asset warning", *warnings)
		}
	})

	t.Run("ambiguous basename resolved by nearest folder", func(t *testing.T) {
		r, warnings := testRenderer(v, "z/Note.md")
		rel, ok := r.resolveAsset("icon.png")
		if !ok || rel != "a/icon.png" {
			t.Errorf("resolveAsset(icon.png) from z/Note = %q,%v want a/icon.png,true", rel, ok)
		}
		if len(*warnings) != 0 {
			t.Errorf("unexpected warnings: %v", *warnings)
		}
	})

	t.Run("slash path with no match fails", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		rel, ok := r.resolveAsset("no/such.png")
		if ok || rel != "" {
			t.Errorf("resolveAsset(no/such.png) = %q,%v want \"\",false", rel, ok)
		}
	})
}

func TestResolveMarkdownHref(t *testing.T) {
	v := testVault(Config{}, []string{"index.md", "Guide.md", "sub/Deep.md"}, nil)

	t.Run("external passes through", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveMarkdownHref("https://go.dev"); got != "https://go.dev" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("absolute path passes through", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveMarkdownHref("/style.css"); got != "/style.css" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("anchor-only passes through", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveMarkdownHref("#section"); got != "#section" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("md link resolves and records outgoing", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveMarkdownHref("Guide.md"); got != "public/Guide/" {
			t.Errorf("got %q, want public/Guide/", got)
		}
		if !r.outgoing["Guide"] {
			t.Errorf("outgoing link to Guide not recorded: %v", r.outgoing)
		}
	})

	t.Run("md link with anchor", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveMarkdownHref("Guide.md#Setup Steps"); got != "public/Guide/#setup-steps" {
			t.Errorf("got %q, want public/Guide/#setup-steps", got)
		}
	})

	t.Run("md link from nested page is relativized", func(t *testing.T) {
		r, _ := testRenderer(v, "sub/Deep.md")
		if got := r.resolveMarkdownHref("Guide.md"); got != "../../../public/Guide/" {
			t.Errorf("got %q, want ../../../public/Guide/", got)
		}
	})

	t.Run("non-md destination delegates to asset resolution", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveMarkdownHref("diagram.png"); got != "diagram.png" {
			t.Errorf("got %q, want diagram.png", got)
		}
	})
}

func TestResolveMarkdownAsset(t *testing.T) {
	t.Run("external passes through", func(t *testing.T) {
		v := testVault(Config{}, []string{"index.md"}, nil)
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveMarkdownAsset("https://x/y.png"); got != "https://x/y.png" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("relative from home", func(t *testing.T) {
		v := testVault(Config{}, []string{"index.md"}, nil)
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveMarkdownAsset("pic.png"); got != "pic.png" {
			t.Errorf("got %q, want pic.png", got)
		}
	})

	t.Run("relative from nested page joins current dir and relativizes", func(t *testing.T) {
		v := testVault(Config{}, []string{"index.md", "sub/Deep.md"}, nil)
		r, _ := testRenderer(v, "sub/Deep.md")
		if got := r.resolveMarkdownAsset("pic.png"); got != "../../../sub/pic.png" {
			t.Errorf("got %q, want ../../../sub/pic.png", got)
		}
	})

	t.Run("single source prefix is prepended", func(t *testing.T) {
		v := testVault(Config{Sources: []string{"docs"}}, []string{"index.md"}, nil)
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveMarkdownAsset("pic.png"); got != "docs/pic.png" {
			t.Errorf("got %q, want docs/pic.png", got)
		}
	})

	t.Run("anchor is preserved", func(t *testing.T) {
		v := testVault(Config{}, []string{"index.md"}, nil)
		r, _ := testRenderer(v, "index.md")
		if got := r.resolveMarkdownAsset("doc.pdf#page=2"); got != "doc.pdf#page=2" {
			t.Errorf("got %q, want doc.pdf#page=2", got)
		}
	})
}
