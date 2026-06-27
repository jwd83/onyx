---
title: Onyx
---

Onyx is a tiny static-site generator for Markdown note folders. Point it at a
folder of notes — an Obsidian-style vault, a `docs/` directory, a pile of
`wiki/` pages — and it writes a clean, fast, linkable website. No database, no
server, no npm pipeline, and no third-party Go dependencies.

This very site is built by Onyx from a folder of Markdown files. What you are
reading is the output.

## Why Onyx

- **One command, no setup.** `go run onyx.jwd.me/onyx@latest` in a notes folder produces a finished site — nothing to install, configure, or wire together.
- **Zero dependencies.** Onyx is stdlib-only Go: it compiles to a single binary and pulls in no third-party packages — `go.sum` does not exist on purpose.
- **Obsidian-friendly Markdown.** Wikilinks, callouts, task lists, tables, math blocks, and `![[embeds]]` all work, alongside ordinary Markdown.
- **Backlinks and a graph.** Onyx derives backlinks and an interactive knowledge graph from your links, so a vault becomes a navigable web.
- **Built-in search.** A client-side search index ships with the site — no external service.
- **Safe by default.** Dangerous link schemes are blocked, raw HTML is escaped, and Onyx refuses to clobber files it did not generate.
- **Deploys anywhere static.** Relative URLs mean the output works on GitHub Pages, including under project subpaths, or any static host.

## How it works

1. Onyx finds the site root by looking for `onyx.ini` or a conventional content folder (`doc/`, `docs/`, `plan/`, `plans/`, `wiki/`).
2. It reads your Markdown, resolves wikilinks and assets across folders, and renders each note to HTML.
3. It writes a homepage to `index.html` and the generated site into `public/`.

## Explore the docs

- [[quick-start|Quick Start]] — install Onyx and build your first site.
- [[markdown|Markdown Support]] — the Markdown and Obsidian syntax Onyx renders.
- [[frontmatter|Frontmatter]] — per-note titles and publish controls.
- [[configuration|Configuration]] — `onyx.ini` keys and sources.
- [[themes|Theme Overrides]] — restyle the site or replace templates.
- [[deploying|Deploying]] — publish to GitHub Pages and output safety.
- [[development|Development]] — the project's stdlib-only contract.

Onyx lives on [GitHub](https://github.com/jwd83/onyx).
