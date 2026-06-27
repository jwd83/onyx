package main

import (
	"reflect"
	"strings"
	"testing"
)

func renderTestMarkdown(markdown string) RenderResult {
	v := testVault(Config{}, []string{"index.md"}, nil)
	r, _ := testRenderer(v, "index.md")
	return r.Render(markdown)
}

func TestRenderInline(t *testing.T) {
	v := testVault(Config{}, []string{"index.md"}, nil)
	r, _ := testRenderer(v, "index.md")

	cases := []struct{ name, in, want string }{
		{"strong asterisks", "**bold**", "<strong>bold</strong>"},
		{"strong underscores", "__bold__", "<strong>bold</strong>"},
		{"emphasis asterisk", "*it*", "<em>it</em>"},
		// Documented gap: there is no single-underscore emphasis rule, so
		// _x_ renders literally. Pinned so the behavior is a deliberate choice.
		{"single underscore is literal", "say _hi_ ok", "say _hi_ ok"},
		{"inline code", "`x<y`", "<code>x&lt;y</code>"},
		{"code escapes html", "`<b>&`", "<code>&lt;b&gt;&amp;</code>"},
		{"plain text is escaped", "<b> & 'q'", "&lt;b&gt; &amp; &#39;q&#39;"},
		{"nested emphasis", "**a *b* c**", "<strong>a <em>b</em> c</strong>"},
		// Documented gap: no flanking rule, so loose asterisks still pair up.
		{"loose asterisks pair", "a * b * c", "a <em> b </em> c"},
		{"bare url", "see https://ex.com now", `see <a href="https://ex.com">https://ex.com</a> now`},
		{"bare url trailing punctuation excluded", "https://ex.com.", `<a href="https://ex.com">https://ex.com</a>.`},
		{"markdown link external", "[go](https://go.dev)", `<a href="https://go.dev">go</a>`},
		{"markdown link dangerous scheme blocked", "[bad](javascript:alert)", `<a href="#">bad</a>`},
		{"markdown link dangerous tab scheme blocked", "[bad](java\tscript:alert)", `<a href="#">bad</a>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := r.renderInline(tc.in); got != tc.want {
				t.Errorf("renderInline(%q) =\n  %q\nwant\n  %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRenderInlineWikilinks(t *testing.T) {
	v := testVault(Config{}, []string{"index.md", "Guide.md"}, []string{"img/diagram.png"})

	t.Run("resolved wikilink", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		want := `<a class="wiki-link" href="public/Guide/">Guide</a>`
		if got := r.renderInline("[[Guide]]"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		if !r.outgoing["Guide"] {
			t.Errorf("outgoing not recorded: %v", r.outgoing)
		}
	})

	t.Run("alias and heading", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		want := `<a class="wiki-link" href="public/Guide/#setup">Click here</a>`
		if got := r.renderInline("[[Guide#Setup|Click here]]"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("broken wikilink renders span and warns", func(t *testing.T) {
		r, warnings := testRenderer(v, "index.md")
		want := `<span class="broken-link">[[Nope]]</span>`
		if got := r.renderInline("[[Nope]]"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		if len(*warnings) != 1 || !strings.Contains((*warnings)[0], "unresolved wikilink [[Nope]]") {
			t.Errorf("warnings = %v, want one unresolved-wikilink warning", *warnings)
		}
	})

	t.Run("image embed", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		want := `<img src="img/diagram.png" alt="diagram.png">`
		if got := r.renderInline("![[img/diagram.png]]"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestRenderBlocks(t *testing.T) {
	cases := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			name:     "blockquote with nested paragraph and list",
			markdown: "> Quote **bold**\n>\n> - item",
			want: "<blockquote>\n" +
				"<p>Quote <strong>bold</strong></p>\n" +
				"<ul>\n" +
				"<li>item</li>\n" +
				"</ul>\n" +
				"</blockquote>\n",
		},
		{
			name:     "callout uses title and renders body blocks",
			markdown: "> [!note] Heads up\n> Remember **this**.\n>\n> 1. Step",
			want: `<aside class="callout callout-note"><p class="callout-title">Heads up</p>` + "\n" +
				"<p>Remember <strong>this</strong>.</p>\n" +
				"<ol>\n" +
				"<li>Step</li>\n" +
				"</ol>\n" +
				"</aside>\n",
		},
		{
			name:     "task list checkboxes and plain items",
			markdown: "- [ ] Todo **one**\n- [x] Done <two>\n+ plain",
			want: "<ul>\n" +
				`<li><input type="checkbox" disabled> Todo <strong>one</strong></li>` + "\n" +
				`<li><input type="checkbox" disabled checked> Done &lt;two&gt;</li>` + "\n" +
				"<li>plain</li>\n" +
				"</ul>\n",
		},
		{
			name:     "fenced code sanitizes info string class",
			markdown: "```go<script> extra\nfmt.Println(\"<x>\")\n```",
			want:     `<pre><code class="language-goscript">fmt.Println(&#34;&lt;x&gt;&#34;)</code></pre>` + "\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderTestMarkdown(tc.markdown).HTML; got != tc.want {
				t.Errorf("Render(%q).HTML =\n%s\nwant\n%s", tc.markdown, got, tc.want)
			}
		})
	}
}

func TestRenderBlocksHeadingIDsAndHorizontalRule(t *testing.T) {
	got := renderTestMarkdown("# Same\n---\n# Same")

	wantHTML := `<h1 id="same">Same</h1>` + "\n" +
		"<hr>\n" +
		`<h1 id="same-2">Same</h1>` + "\n"
	if got.HTML != wantHTML {
		t.Errorf("HTML =\n%s\nwant\n%s", got.HTML, wantHTML)
	}

	wantHeadings := []string{"Same", "Same"}
	if !reflect.DeepEqual(got.Headings, wantHeadings) {
		t.Errorf("Headings = %v, want %v", got.Headings, wantHeadings)
	}
}

// TestParagraphBlockBoundaries pins startsBlock's transitions: a paragraph that
// runs directly into a block starter (no blank separator line) must close at the
// boundary and let the following block render.
func TestParagraphBlockBoundaries(t *testing.T) {
	cases := []struct{ name, markdown, want string }{
		{
			name:     "paragraph into heading",
			markdown: "para\n# Heading",
			want:     "<p>para</p>\n" + `<h1 id="heading">Heading</h1>` + "\n",
		},
		{
			name:     "paragraph into fenced code",
			markdown: "para\n```\ncode\n```",
			want:     "<p>para</p>\n<pre><code>code</code></pre>\n",
		},
		{
			name:     "paragraph into horizontal rule",
			markdown: "para\n---",
			want:     "<p>para</p>\n<hr>\n",
		},
		{
			name:     "paragraph into blockquote",
			markdown: "para\n> quote",
			want:     "<p>para</p>\n<blockquote>\n<p>quote</p>\n</blockquote>\n",
		},
		{
			name:     "paragraph into list",
			markdown: "para\n- item",
			want:     "<p>para</p>\n<ul>\n<li>item</li>\n</ul>\n",
		},
		{
			name:     "paragraph into math block",
			markdown: "para\n$$x^2$$",
			want:     "<p>para</p>\n" + `<div class="onyx-math">$$` + "\nx^2\n$$</div>\n",
		},
		{
			name:     "paragraph into aligned table",
			markdown: "para\n| A | B |\n| :-- | --: |\n| 1 | 2 |",
			want: "<p>para</p>\n" +
				"<table>\n<thead><tr>" +
				`<th style="text-align:left">A</th>` +
				`<th style="text-align:right">B</th>` +
				"</tr></thead>\n<tbody>\n<tr>" +
				`<td style="text-align:left">1</td>` +
				`<td style="text-align:right">2</td>` +
				"</tr>\n</tbody>\n</table>\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderTestMarkdown(tc.markdown).HTML; got != tc.want {
				t.Errorf("Render(%q).HTML =\n%s\nwant\n%s", tc.markdown, got, tc.want)
			}
		})
	}
}

// TestStartsBlockBlankLine covers the one startsBlock branch renderBlocks cannot
// reach: it short-circuits on blank lines before calling startsBlock, so the
// blank-line case is only observable by calling the predicate directly.
func TestStartsBlockBlankLine(t *testing.T) {
	lines := []string{"prose", "", "more"}
	if startsBlock(lines, 0) {
		t.Errorf("startsBlock on prose = true, want false")
	}
	if !startsBlock(lines, 1) {
		t.Errorf("startsBlock on blank line = false, want true")
	}
}

// TestMathBlocks pins the display-math forms beyond the bare multi-line block the
// integration test already covers: single-line, content on the opening or closing
// fence, and an unterminated block.
func TestMathBlocks(t *testing.T) {
	wrap := func(body string) string {
		return `<div class="onyx-math">$$` + "\n" + body + "\n$$</div>\n"
	}
	cases := []struct{ name, markdown, want string }{
		{"single line escapes html", "$$a<b$$", wrap("a&lt;b")},
		{"open fence carries content", "$$ a + b\n$$", wrap("a + b")},
		{"close fence carries content", "$$\nfoo\nbar$$", wrap("foo\nbar")},
		{"unterminated keeps remaining lines", "$$\nx + y", wrap("x + y")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := renderTestMarkdown(tc.markdown)
			if got.HTML != tc.want {
				t.Errorf("Render(%q).HTML =\n%q\nwant\n%q", tc.markdown, got.HTML, tc.want)
			}
			if !got.HasMath {
				t.Errorf("Render(%q).HasMath = false, want true", tc.markdown)
			}
		})
	}
}

// TestParseMarkdownLink pins the three rejection paths and the success shape of
// the Markdown-link scanner, including the defensive non-bracket guard that
// renderInline never reaches.
func TestParseMarkdownLink(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		label    string
		dest     string
		consumed int
		ok       bool
	}{
		{"well formed", "[label](dest)rest", "label", "dest", 13, true},
		{"missing destination open", "[label]", "", "", 0, false},
		{"missing destination close", "[label](dest", "", "", 0, false},
		{"not a link", "plain text", "", "", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			label, dest, consumed, ok := parseMarkdownLink(tc.in)
			if label != tc.label || dest != tc.dest || consumed != tc.consumed || ok != tc.ok {
				t.Errorf("parseMarkdownLink(%q) = (%q, %q, %d, %v), want (%q, %q, %d, %v)",
					tc.in, label, dest, consumed, ok, tc.label, tc.dest, tc.consumed, tc.ok)
			}
		})
	}
}

// TestRenderInlineMalformedLinks asserts the observable behavior of the rejected
// link forms: a `[` that does not complete a Markdown link renders literally
// rather than being dropped.
func TestRenderInlineMalformedLinks(t *testing.T) {
	r, _ := testRenderer(testVault(Config{}, []string{"index.md"}, nil), "index.md")
	cases := []struct{ name, in, want string }{
		{"missing destination renders literally", "[label without dest", "[label without dest"},
		{"unclosed destination renders literally", "[label](dest", "[label](dest"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := r.renderInline(tc.in); got != tc.want {
				t.Errorf("renderInline(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestRenderInlineAttributeInjection is a regression guard: quote and angle
// characters in a link destination, image alt, or wikilink alias must be escaped
// so they cannot break out of the surrounding HTML attribute.
func TestRenderInlineAttributeInjection(t *testing.T) {
	v := testVault(Config{}, []string{"index.md", "Guide.md"}, []string{"img/diagram.png"})
	cases := []struct{ name, in, want string }{
		{
			name: "link href quote is escaped",
			in:   `[x](https://e.com/a"onmouseover=1)`,
			want: `<a href="https://e.com/a&#34;onmouseover=1">x</a>`,
		},
		{
			name: "image alt markup is escaped",
			in:   `![evil"><img>](img/diagram.png)`,
			want: `<img src="img/diagram.png" alt="evil&#34;&gt;&lt;img&gt;">`,
		},
		{
			name: "wikilink alias markup is escaped",
			in:   `[[Guide|evil"><b>]]`,
			want: `<a class="wiki-link" href="public/Guide/">evil&#34;&gt;&lt;b&gt;</a>`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := testRenderer(v, "index.md")
			if got := r.renderInline(tc.in); got != tc.want {
				t.Errorf("renderInline(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestRenderWikiAssetAndEmbed covers renderWiki's asset-as-link and note-embed
// branches and the embed path that falls through to a broken-link span.
func TestRenderWikiAssetAndEmbed(t *testing.T) {
	v := testVault(Config{}, []string{"index.md", "Guide.md"}, []string{"img/diagram.png"})

	t.Run("image asset wikilink renders a link, not an embed", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		want := `<a href="img/diagram.png">diagram.png</a>`
		if got := r.renderInline("[[img/diagram.png]]"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("note embed renders an embed-note link", func(t *testing.T) {
		r, _ := testRenderer(v, "index.md")
		want := `<a class="embed-note" href="public/Guide/">Guide</a>`
		if got := r.renderInline("![[Guide]]"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		if !r.outgoing["Guide"] {
			t.Errorf("note embed did not record an outgoing link: %v", r.outgoing)
		}
	})

	t.Run("broken embed warns and renders a broken-link span", func(t *testing.T) {
		r, warnings := testRenderer(v, "index.md")
		want := `<span class="broken-link">[[missing.png]]</span>`
		if got := r.renderInline("![[missing.png]]"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		if len(*warnings) != 1 || !strings.Contains((*warnings)[0], "unresolved wikilink [[missing.png]]") {
			t.Errorf("warnings = %v, want one unresolved-wikilink warning", *warnings)
		}
	})
}

// TestExtractTags characterizes extractTags as it behaves today. It is a
// characterization test, not a specification: it pins current behavior so the
// known quirks stay visible and intentional, including the documented wart that
// tags are harvested from the raw document with no awareness of fenced code, so
// `#include`/`#define` inside a code block become site tags. Changing that
// (harvesting from fence-stripped text) is a separate, behavior-changing step.
func TestExtractTags(t *testing.T) {
	cases := []struct {
		name     string
		markdown string
		want     []string
	}{
		{"single tag", "#foo", []string{"foo"}},
		{"requires preceding start or space, sorted", "a #foo and #bar", []string{"bar", "foo"}},
		{"not matched mid-word", "word#foo", nil},
		{"deduplicated", "#foo #foo", []string{"foo"}},
		{"sorted ascending", "#zebra #apple #mango", []string{"apple", "mango", "zebra"}},
		{"stops at disallowed char", "#foo.bar", []string{"foo"}},
		{"slashes form nested tags", "#foo/bar/baz", []string{"foo/bar/baz"}},
		{"hyphen and underscore allowed", "#foo-bar #baz_qux", []string{"baz_qux", "foo-bar"}},
		{"digits only", "#123", []string{"123"}},
		{"leading hyphen kept", "#-foo", []string{"-foo"}},
		{"headings are not tags", "# heading\n## sub", nil},
		{"case sensitive, not deduped across case", "text #Tag and #tag", []string{"Tag", "tag"}},
		// The wart: code fences are not stripped, so code content pollutes tags.
		{"tags harvested from inside code fences", "```c\n#include <stdio.h>\n#define X 1\n```", []string{"define", "include"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractTags(tc.markdown); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("extractTags(%q) = %#v, want %#v", tc.markdown, got, tc.want)
			}
		})
	}
}
