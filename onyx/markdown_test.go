package main

import (
	"strings"
	"testing"
)

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
