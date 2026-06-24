package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRendersHomepageWikilinksAndBacklinks(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "onyx.ini", "site_title = Test Notes\nsource = docs\nbase_url = /\n")
	writeTestFile(t, root, "docs/index.md", "---\ntitle: Home\n---\nWelcome to [[Foo|the foo note]].\n")
	writeTestFile(t, root, "docs/Foo.md", "# Foo\n\nBack home: [[index|Home]].\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{root}, &stdout, &stderr); code != 0 {
		t.Fatalf("run failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	index := readTestFile(t, root, "index.html")
	if strings.Contains(index, `href="/public/`) || strings.Contains(index, `src="/public/`) {
		t.Fatalf("homepage contains root-relative public path:\n%s", index)
	}
	if !strings.Contains(index, `href="public/Foo/"`) {
		t.Fatalf("homepage did not link to Foo:\n%s", index)
	}
	if !strings.Contains(index, `href="public/onyx.css"`) || !strings.Contains(index, `src="public/onyx.js"`) {
		t.Fatalf("homepage did not use relative assets:\n%s", index)
	}

	foo := readTestFile(t, root, "public/Foo/index.html")
	if !strings.Contains(foo, "Linked From") || !strings.Contains(foo, "Home") {
		t.Fatalf("Foo page did not include backlink to Home:\n%s", foo)
	}
	if !strings.Contains(foo, `href="../../public/onyx.css"`) || !strings.Contains(foo, `src="../../public/onyx.js"`) {
		t.Fatalf("nested page did not use relative assets:\n%s", foo)
	}
	if !strings.Contains(foo, `href="../../"`) {
		t.Fatalf("nested page did not link home relatively:\n%s", foo)
	}
	if _, err := os.Stat(filepath.Join(root, ".nojekyll")); err != nil {
		t.Fatalf(".nojekyll was not created: %v", err)
	}
}

func TestBuildResolvesSlashWikilinksRelativeToCurrentFolder(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "onyx.ini", "site_title = Test Notes\nsource = docs\nbase_url = /\n")
	writeTestFile(t, root, "docs/index.md", "# Home\n\n[[Games/Baldur's Gate]]\n")
	writeTestFile(t, root, "docs/Games/Baldur's Gate.md", "# Baldur's Gate\n\n[[Baldur's Gate 3/Astarion Build]]\n")
	writeTestFile(t, root, "docs/Games/Baldur's Gate 3/Astarion Build.md", "# Astarion Build\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{root}, &stdout, &stderr); code != 0 {
		t.Fatalf("run failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), "unresolved wikilink") {
		t.Fatalf("relative slash wikilink was not resolved:\n%s", stderr.String())
	}

	page := readTestFile(t, root, "public/Games/Baldur's Gate/index.html")
	if !strings.Contains(page, `../../../public/Games/Baldur%27s%20Gate%203/Astarion%20Build/`) {
		t.Fatalf("relative slash wikilink href missing:\n%s", page)
	}
}

func TestBuildExcludesDraftAndPublishFalse(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "onyx.ini", "site_title = Test Notes\nsource = docs\nbase_url = /\n")
	writeTestFile(t, root, "docs/index.md", "# Home\n\n[[Draft]] [[Private]] [[Public]]\n")
	writeTestFile(t, root, "docs/Draft.md", "---\ndraft: true\n---\n# Draft\n")
	writeTestFile(t, root, "docs/Private.md", "---\npublish: false\n---\n# Private\n")
	writeTestFile(t, root, "docs/Public.md", "# Public\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{root}, &stdout, &stderr); code != 0 {
		t.Fatalf("run failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	if _, err := os.Stat(filepath.Join(root, "public", "Draft", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("draft page was generated, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "public", "Private", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("private page was generated, err=%v", err)
	}
	search := readTestFile(t, root, "public/search-index.json")
	if strings.Contains(search, `"path": "Draft.md"`) || strings.Contains(search, `"path": "Private.md"`) {
		t.Fatalf("excluded notes leaked into search index:\n%s", search)
	}
}

func TestBuildRefusesUnmarkedPublicDirectory(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "onyx.ini", "site_title = Test Notes\nsource = docs\nbase_url = /\n")
	writeTestFile(t, root, "docs/index.md", "# Home\n")
	writeTestFile(t, root, "public/handmade.txt", "do not delete me\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{root}, &stdout, &stderr); code == 0 {
		t.Fatalf("run succeeded unexpectedly\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "missing .onyx-generated") {
		t.Fatalf("unexpected error:\n%s", stderr.String())
	}
}

func TestBuildRendersMathBlocksAndCompactTables(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "onyx.ini", "site_title = Test Notes\nsource = docs\nbase_url = /\n")
	writeTestFile(t, root, "docs/index.md", "# Home\n\n[[BMI]]\n")
	writeTestFile(t, root, "docs/BMI.md", strings.Join([]string{
		"Formula below.",
		"$$",
		"\\text{BMI}=703.0717*\\frac{\\text{Pounds}}{\\text{Inches}^2}",
		"$$",
		"Breakpoints:",
		"",
		"Range|Class",
		"--|--",
		"Below 18.5|Underweight",
		"18.5-25|Normal",
		"",
	}, "\n"))

	var stdout, stderr bytes.Buffer
	if code := run([]string{root}, &stdout, &stderr); code != 0 {
		t.Fatalf("run failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	page := readTestFile(t, root, "public/BMI/index.html")
	if !strings.Contains(page, `<div class="onyx-math">`) {
		t.Fatalf("math block was not rendered as a math container:\n%s", page)
	}
	// The asterisk inside math must survive verbatim, not become emphasis.
	if !strings.Contains(page, "703.0717*\\frac") {
		t.Fatalf("math content was mangled (asterisk treated as markdown):\n%s", page)
	}
	if strings.Contains(page, "<em>") {
		t.Fatalf("math asterisk leaked into an <em> tag:\n%s", page)
	}
	// The compact 2-dash table must become a real table.
	if !strings.Contains(page, "<table>") || !strings.Contains(page, "<th>Range</th>") {
		t.Fatalf("compact table was not rendered as a table:\n%s", page)
	}
	if !strings.Contains(page, "<td>Underweight</td>") {
		t.Fatalf("table body row missing:\n%s", page)
	}
	if strings.Contains(page, "Range|Class") || strings.Contains(page, "--|--") {
		t.Fatalf("raw table markup leaked into output:\n%s", page)
	}
	// MathJax should be loaded only because the page uses math.
	if !strings.Contains(page, "tex-chtml.js") {
		t.Fatalf("MathJax script was not included on a page with math:\n%s", page)
	}
	home := readTestFile(t, root, "index.html")
	if strings.Contains(home, "tex-chtml.js") {
		t.Fatalf("MathJax script leaked onto a page without math:\n%s", home)
	}
}

func TestBuildWithoutIniUsesDocsAndIndexTitle(t *testing.T) {
	root := t.TempDir()
	// No onyx.ini: the docs/ folder marks the root and the home page title
	// becomes the site title.
	writeTestFile(t, root, "docs/index.md", "---\ntitle: My Notebook\n---\n# Welcome\n\n[[Foo]]\n")
	writeTestFile(t, root, "docs/Foo.md", "# Foo\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{root}, &stdout, &stderr); code != 0 {
		t.Fatalf("run failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	index := readTestFile(t, root, "index.html")
	if !strings.Contains(index, "My Notebook") {
		t.Fatalf("site title did not fall back to index.md title:\n%s", index)
	}
	if !strings.Contains(index, `href="public/Foo/"`) {
		t.Fatalf("homepage did not link to Foo without an onyx.ini:\n%s", index)
	}
	if _, err := os.Stat(filepath.Join(root, ".nojekyll")); err != nil {
		t.Fatalf(".nojekyll was not created: %v", err)
	}
}

func TestBuildSingleNonDocsSourceBuildsAsIs(t *testing.T) {
	root := t.TempDir()
	// No onyx.ini and no docs/: a lone wiki/ folder marks the root and builds
	// exactly like a docs-only site, with no section prefix in the paths.
	writeTestFile(t, root, "wiki/index.md", "---\ntitle: Wiki Home\n---\n# Welcome\n\n[[Foo]]\n")
	writeTestFile(t, root, "wiki/Foo.md", "# Foo\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{root}, &stdout, &stderr); code != 0 {
		t.Fatalf("run failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	index := readTestFile(t, root, "index.html")
	if !strings.Contains(index, "Wiki Home") {
		t.Fatalf("site title did not fall back to wiki/index.md title:\n%s", index)
	}
	if !strings.Contains(index, `href="public/Foo/"`) {
		t.Fatalf("single wiki source should build without a section prefix:\n%s", index)
	}
	if _, err := os.Stat(filepath.Join(root, "public", "Foo", "index.html")); err != nil {
		t.Fatalf("Foo page was not generated at the unprefixed path: %v", err)
	}
}

func TestBuildMultipleSourcesSectionsAndIndex(t *testing.T) {
	root := t.TempDir()
	// docs/ and wiki/ are both present and there is no top-level index, so Onyx
	// builds a sectioned site with an autogenerated index.
	writeTestFile(t, root, "docs/Alpha.md", "# Alpha\n\nSee [[Beta]] in the wiki.\n")
	writeTestFile(t, root, "wiki/Beta.md", "# Beta\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{root}, &stdout, &stderr); code != 0 {
		t.Fatalf("run failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), "unresolved wikilink") {
		t.Fatalf("cross-source wikilink was not resolved:\n%s", stderr.String())
	}

	// Each source becomes a section prefix in the output tree.
	if _, err := os.Stat(filepath.Join(root, "public", "docs", "Alpha", "index.html")); err != nil {
		t.Fatalf("docs section page missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "public", "wiki", "Beta", "index.html")); err != nil {
		t.Fatalf("wiki section page missing: %v", err)
	}

	// The home page is an autogenerated index grouped by section.
	index := readTestFile(t, root, "index.html")
	if !strings.Contains(index, "<h2>Docs</h2>") || !strings.Contains(index, "<h2>Wiki</h2>") {
		t.Fatalf("home index was not grouped into sections:\n%s", index)
	}
	if !strings.Contains(index, `href="public/docs/Alpha/"`) || !strings.Contains(index, `href="public/wiki/Beta/"`) {
		t.Fatalf("home index did not link to section pages:\n%s", index)
	}

	// The nav nests each source as its own folder.
	if !strings.Contains(index, "<summary>") || !strings.Contains(index, ">docs<") || !strings.Contains(index, ">wiki<") {
		t.Fatalf("nav did not nest sources as folders:\n%s", index)
	}

	// The cross-source wikilink points from docs/Alpha to wiki/Beta.
	alpha := readTestFile(t, root, "public/docs/Alpha/index.html")
	if !strings.Contains(alpha, `href="../../../public/wiki/Beta/"`) {
		t.Fatalf("cross-source wikilink href missing:\n%s", alpha)
	}
}

func TestBuildExplicitMultipleSources(t *testing.T) {
	root := t.TempDir()
	// An explicit comma-separated source list opts a folder outside the default
	// set (notes/) into a multi-source build.
	writeTestFile(t, root, "onyx.ini", "source = docs, notes\n")
	writeTestFile(t, root, "docs/Alpha.md", "# Alpha\n")
	writeTestFile(t, root, "notes/Gamma.md", "# Gamma\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{root}, &stdout, &stderr); code != 0 {
		t.Fatalf("run failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	if _, err := os.Stat(filepath.Join(root, "public", "notes", "Gamma", "index.html")); err != nil {
		t.Fatalf("explicit notes section page missing: %v", err)
	}
	index := readTestFile(t, root, "index.html")
	if !strings.Contains(index, "<h2>Notes</h2>") {
		t.Fatalf("explicit source was not rendered as a section:\n%s", index)
	}
}

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
