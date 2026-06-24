# Onyx

A tiny static-site generator that publishes a folder of Markdown notes (e.g. an
Obsidian vault) as a static website. Intentionally minimal: one `onyx.go` file,
a single `go.mod` with no third-party dependencies, no npm, no build system, and
no runtime server.

## Quick start

```sh
go run onyx.jwd.me/onyx@latest
```

Run that from the root of a site (a directory containing `docs/`), or pass a
site path in the same one-liner:

```sh
go run onyx.jwd.me/onyx@latest path/to/site
```

If you want a reusable `onyx` command instead:

```sh
go install onyx.jwd.me/onyx@latest
```

## Usage

With the installed command:

```sh
onyx              # build the site rooted at the current directory
onyx path/to/site # build a site elsewhere
```

Onyx finds the site root (the nearest ancestor containing `docs/`), renders the
Markdown in `docs/`, writes the homepage to `index.html`, and writes generated
assets and pages into `public/`.

## Configuration

There is no config file. The defaults are the convention:

| Setting | Default |
| --- | --- |
| Source folder | `docs/` |
| Site title | the `docs/index.md` title (frontmatter `title:` or first `# heading`), falling back to `Onyx` |
| Theme overrides | `theme/` if present, otherwise the built-in CSS, JS, and templates |
| Search, graph, and Markdown-source links | on |

Everything is optional. If a `theme/` folder exists it overrides the built-in
`style.css`, `onyx.js`, `page.html`, and `home.html`. Per-note `publish: false`
or `draft: true` frontmatter excludes a note from the build.

An optional `onyx.ini` at the site root can override the defaults
(`site_title`, `source`, `theme`, and the `search`, `graph`, and `show_source`
toggles), but it is not required.

## Deploying to GitHub Pages

Generated links are relative, so the site works from a project URL such as
`/repo/` without hardcoded root paths. Onyx writes a `.nojekyll` file so GitHub
Pages serves the generated files directly instead of running Jekyll over the
repo.

## Development

```sh
go test ./...
```
