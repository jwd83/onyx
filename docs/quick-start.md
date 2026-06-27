# Quick Start

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

Onyx writes a homepage to `index.html` plus the generated site into `public/`.
Open `index.html` in a browser, or serve the folder with any static file server.

## What Onyx builds

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

See [[configuration|Configuration]] to choose which folders build, and
[[markdown|Markdown Support]] for the syntax Onyx understands.
