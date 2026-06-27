# Markdown Support

Onyx intentionally supports a practical subset of Markdown instead of trying to
be a full CommonMark implementation. The supported surface includes:

- headings, paragraphs, horizontal rules, blockquotes, and Obsidian-style callouts
- unordered, ordered, and task-list items
- fenced code blocks with language classes
- tables with optional alignment markers
- inline code, emphasis, strong text, Markdown links, Markdown images, and bare `http://` or `https://` links
- display math blocks delimited with `$$`
- wikilinks such as `[[Note]]`, `[[Note#Heading]]`, `[[Note|Alias]]`, and image embeds such as `![[diagram.png]]`

## Wikilinks

Wikilinks resolve across all configured content folders. If several notes or
assets share a name, Onyx prefers the nearest matching folder and emits a warning
when a match is still ambiguous. Every wikilink also becomes a backlink on the
note it points to and an edge in the graph view.

> [!note] Callouts
> Obsidian-style callouts render as titled boxes. Start a blockquote with
> `> [!note]`, `> [!warning]`, and friends.

## Math

Display math blocks delimited with `$$` are passed through to MathJax, which is
loaded only on pages that contain math:

$$
e^{i\pi} + 1 = 0
$$

## Safety

Dangerous `javascript:` and `data:` link schemes are blocked before link or asset
resolution. Raw Markdown HTML is escaped rather than passed through, so notes
cannot inject arbitrary markup into the generated pages.

Next: [[frontmatter|Frontmatter]] for per-note metadata.
