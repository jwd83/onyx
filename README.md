# Onyx

[![CI](https://github.com/jwd83/onyx/actions/workflows/ci.yml/badge.svg)](https://github.com/jwd83/onyx/actions/workflows/ci.yml)

Onyx is a tiny static-site generator for Markdown note folders. It is built for
publishing an Obsidian-style vault as a plain static website: no database, no
server, no npm pipeline, and no third-party Go dependencies.

Run it in a folder with notes, and it writes a homepage to `index.html` plus the
generated site into `public/`.

## Quick Start

From the root of a site that contains `docs/`, `wiki/`, `plans/`, or another
recognized content folder:

```sh
go run onyx.jwd.me/onyx@latest
```

Or point Onyx at a site from anywhere:

```sh
go run onyx.jwd.me/onyx@latest path/to/site
```

To install a reusable command:

```sh
go install onyx.jwd.me/onyx@latest
onyx path/to/site
```

## What Onyx Builds

Onyx looks for a site root by walking upward from the current directory until it
finds `onyx.ini` or one of its conventional content folders:

- `doc/`
- `docs/`
- `plan/`
- `plans/`
- `wiki/`

When one content folder exists, notes publish at the top of the generated tree.
For example, `docs/Projects/Onyx.md` becomes:

```text
public/Projects/Onyx/index.html
```

When several content folders exist, Onyx publishes a sectioned site and keeps the
folder name in each URL. For example, `wiki/Onyx.md` becomes:

```text
public/wiki/Onyx/index.html
```

The homepage is chosen from `index.md` first, then `home.md`. If neither exists,
Onyx generates an index page from the published notes. With several content
folders, the generated homepage groups notes by section.

## Markdown Support

Onyx intentionally supports a practical subset of Markdown instead of trying to
be a full CommonMark implementation. The supported surface includes:

- headings, paragraphs, horizontal rules, blockquotes, and Obsidian-style
  callouts
- unordered, ordered, and task-list items
- fenced code blocks with language classes
- tables with optional alignment markers
- inline code, emphasis, strong text, Markdown links, Markdown images, and bare
  `http://` or `https://` links
- display math blocks delimited with `$$`
- wikilinks such as `[[Note]]`, `[[Note#Heading]]`, `[[Note|Alias]]`, and image
  embeds such as `![[diagram.png]]`

Wikilinks resolve across all configured content folders. If several notes or
assets share a name, Onyx prefers the nearest matching folder and emits a warning
when a match is still ambiguous.

Dangerous `javascript:` and `data:` link schemes are blocked before link or asset
resolution. Raw Markdown HTML is escaped rather than passed through.

## Frontmatter

Files may start with simple YAML-style frontmatter:

```markdown
---
title: My Note
publish: true
---
```

Recognized fields:

| Field | Effect |
| --- | --- |
| `title` | Sets the page title. Without it, Onyx uses the first `# Heading`, then the filename. |
| `publish: false` | Excludes the note from the build. |
| `draft: true` | Excludes the note from the build. |

Frontmatter parsing is intentionally small: `key: value` lines between opening
and closing `---` markers.

## Configuration

Configuration is optional. Without `onyx.ini`, Onyx builds every conventional
content folder it finds, uses `theme/` for overrides if present, and enables
search, graph data, and source links.

Example `onyx.ini`:

```ini
site_title = My Notes
source = docs, wiki
theme = theme
search = true
graph = true
show_source = true
```

Supported keys:

| Key | Default | Notes |
| --- | --- | --- |
| `site_title` | Home note title, then `Onyx` | Sets the site title shown by the template. |
| `source` | Existing conventional folders | Comma- or whitespace-separated list of content folders. Explicit sources must exist. |
| `theme` | `theme` | Relative path to theme overrides. Absolute paths are rejected. |
| `search` | `true` | Writes `public/search-index.json` and enables search UI data. |
| `graph` | `true` | Writes `public/graph.json` and enables graph UI data. |
| `show_source` | `true` | Shows links to original Markdown files when templates expose them. |

Unknown keys are ignored for compatibility. Legacy aliases are still accepted:
`build.search`, `build.graph`, and `publish_raw_markdown`. When both a modern key
and a legacy alias are present, the modern key wins.

## Theme Overrides

Onyx ships with an embedded default theme. A site can override any of these files
inside its configured theme folder:

```text
theme/
  style.css   # copied to public/onyx.css
  onyx.js     # copied to public/onyx.js
  page.html   # template for notes
  home.html   # template for the homepage
  static/     # copied to public/theme/
```

Missing theme files fall back to the embedded defaults. `home.html` falls back
to the embedded page template when no custom homepage template exists.

Templates are Go `html/template` files. The default data includes the site title,
current page, rendered nav, backlinks, feature toggles, generated marker, and
relative URLs for the homepage, CSS, and JavaScript assets.

## Output Safety

Onyx is conservative about overwriting files:

- `index.html` at the site root is replaced only if it is blank or already marked
  as generated by Onyx.
- `public/` is deleted and rebuilt only when it contains `.onyx-generated`.
- `.nojekyll` is written so GitHub Pages serves the generated static files
  directly.
- Generated page and asset URLs are relative, so the site works under GitHub
  Pages project paths such as `/repo/`.

## Deploying

Build locally, then publish the site root with GitHub Pages or any static file
host. The generated files live in:

```text
index.html
public/
.nojekyll
```

For GitHub Pages, commit those generated files on the branch Pages serves, or run
Onyx as part of your publishing workflow.

## Development

Onyx is deliberately boring Go. The local contract is the same one enforced by
CI:

```sh
files="$(gofmt -l $(git ls-files '*.go'))"
test -z "$files" || { echo "$files"; exit 1; }
go vet ./...
go test ./...
```

The project is stdlib-only. Do not add third-party requirements or `go.sum`
unless the no-dependency contract is being changed deliberately.

When changing generated output, preserve relative URLs, the `.nojekyll` behavior,
the conservative overwrite guards, and the generated-file markers. Update tests
alongside behavior changes.
