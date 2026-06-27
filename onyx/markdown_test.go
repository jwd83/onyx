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
