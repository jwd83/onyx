# Architectural Review — Onyx

*Reviewed 2026-06-25 against `master` @ `73c2b90`.*

Onyx is a deliberately small Go static-site generator for Markdown vaults. Its
strongest architectural asset is also its product promise: one installed command,
one binary, no third-party dependencies, no npm, no runtime server. That promise
is real in the current codebase and should remain a hard constraint for any
improvement campaign. The main carrying cost is not a wrong design; it is that
several mature subsystems now occupy one 3,141-line file. The best order of
attack is to add focused stdlib-only tests around the markdown/link/config
surfaces, split the existing package into subsystem files, then consolidate the
duplicated path-resolution logic. Related active plan: this document supersedes
the previous active review in `plans/in-progress/architectural-review.md`, which
was written against `f802ac2`.

## Snapshot

| Metric | Value |
| --- | --- |
| Repository shape | 10 tracked files; Go command in `onyx/`; vanity import HTML at `index.html` and `onyx/index.html` |
| Go module | `onyx.jwd.me`, package import path `onyx.jwd.me/onyx` |
| Go version | `go 1.26` in `go.mod`; local toolchain `go1.26.3 darwin/arm64` |
| Third-party dependencies | **0**; `go list` shows only stdlib imports and there is no `go.sum` |
| Source footprint | `onyx/onyx.go` 3,141 lines; 98 package functions, 15 methods, 14 struct types |
| Test footprint | `onyx/onyx_test.go` 295 lines; 9 `Test*` functions, mostly end-to-end temp-dir builds |
| Embedded defaults | `defaultPageTemplate` 77 lines, `defaultCSS` 429 lines, `defaultJS` 420 lines; 926 lines total, about 29% of `onyx.go` |
| Documentation | `README.md` 85 lines; no `docs/` tree; active plan at this file |
| Formatting/static checks | `gofmt -l onyx/onyx.go onyx/onyx_test.go` clean; `go vet ./...` passes |
| Test baseline | `go test ./... -cover` passes; coverage 60.5% of statements |

Interpretation: the project is still compact and dependency-clean, but no
longer "tiny" internally. The 3,141-line source file contains command wiring,
config and INI parsing, source discovery, vault indexing, output safety checks,
page writing, navigation, search/graph JSON, a Markdown renderer, wikilink and
asset resolution, and the built-in HTML/CSS/JS theme. The tests give useful
user-level coverage for builds, drafts, relative URLs, math, tables, and
multi-source sections, but the coverage profile shows several high-change
helpers are barely or not directly tested: `renderInline` 18.6%,
`resolveMarkdownHref` 0%, `resolveMarkdownAsset` 0%, `resolveAsset` 0%,
`consumeBareURL` 0%, `parseMarkdownLink` 0%, `renderBlockquote` 0%, and
`copyDir` 0%.

## Structurally sound elements

- **The no-dependency contract is intact and valuable.** `go.mod` has no
  `require` block, imports are stdlib-only, and the README's "no npm, no build
  system, no runtime server" claim matches the code. This is a constraint to
  preserve, not technical debt to remove.
- **The top-level build pipeline is clear.** `buildSite` (around line 420) reads
  as a stable sequence: protect existing root output, ensure `.nojekyll`, load
  the vault, render pages, compute backlinks, prepare `public/`, write assets,
  write pages, then emit search and graph data. That linear shape is simple and
  should survive refactoring.
- **Generated-output safety is treated as architecture.** `ensureRootIndexWritable`
  (around line 458) and `preparePublic` (around line 499) refuse to overwrite
  unmarked user content. The tests assert the public-directory guard. This is a
  load-bearing behavioral contract.
- **Relative URL behavior is a first-class invariant.** `relativeRoot` and
  `relativeURL` (around lines 2192-2212) support GitHub Pages project paths, and
  tests assert that nested pages link back to `../../public/...` instead of
  root-relative `/public/...`.
- **The source-folder model is compact but expressive.** `defaultSources`,
  `resolveSources`, `Config.Multi`, and `sourcePrefix` handle single-source,
  multi-source, and explicit-source builds without introducing configuration
  machinery. The comments explain the important path distinction.
- **Tests exercise real workflows.** The existing tests build temporary sites
  through `run`, then inspect generated files. That style catches integration
  regressions that isolated unit tests would miss.

## Structural risks and costs

1. **One source file now carries too many subsystem boundaries.**
   *Evidence:* `onyx/onyx.go` is 3,141 lines and contains CLI entrypoints,
   config parsing, vault loading, output safety, template loading, page writing,
   nav/search/graph emission, markdown block parsing, inline parsing, wikilink
   and asset resolution, and 926 lines of embedded default assets.
   *Consequence:* unrelated changes collide in one file, diffs are harder to
   review, and new contributors must build their own mental table of contents.
   The design is simple, but the file shape makes routine work more expensive
   than it needs to be.
   *Fix direction:* keep `package main` and split by existing responsibilities:
   `config.go`, `vault.go`, `output.go`, `templates.go`, `nav.go`,
   `search_graph.go`, `markdown.go`, `links.go`, and `assets.go`. This changes
   no public behavior, adds no dependencies, and preserves the same binary and
   `go run onyx.jwd.me/onyx@latest` path.

2. **The highest-risk parsing and linking code is under-tested.**
   *Evidence:* Total statement coverage is 60.5%, but `renderInline` is 18.6%.
   Markdown links, bare URLs, image paths, asset resolution, external-link
   sanitization, blockquotes/callouts, task-list items, theme static copying,
   and resolver ambiguity have little or no direct coverage.
   *Consequence:* the code most likely to change is the least protected. A
   behavior-preserving file split would be mostly mechanical, but a small edit
   to inline parsing or link resolution could silently alter output.
   *Fix direction:* before structural edits, add focused table-driven tests for
   `renderInline`, `resolveMarkdownHref`, `resolveMarkdownAsset`, `resolveAsset`,
   `parseINI`, blockquotes/callouts, task-list items, and theme overrides. Keep
   the existing integration tests; add unit tests only where they make later
   refactors safer. No parser library is needed.

3. **Path resolution has duplicated rules with subtly different semantics.**
   *Evidence:* `resolveNote`, `resolveAsset`, `resolveMarkdownHref`, and
   `resolveMarkdownAsset` (roughly lines 1908-2067) each build candidate paths
   from current page/source paths, normalize case, handle anchors, and decide
   whether to warn. Notes use `nearestPage`; assets sort matches by distance and
   warn separately.
   *Consequence:* fixes to one link form can miss another. The distinction
   between `PageRel`, `SourceRel`, source prefixes, anchors, assets, and notes
   is valid, but it is currently encoded four times.
   *Fix direction:* after tests land, extract a small candidate-resolution helper
   that accepts current directory, target, exact index, basename index, and an
   ambiguity policy. Keep note and asset behavior explicit at the call sites,
   but share the path-cleaning and nearest-match machinery.

4. **`Page` state changes across the pipeline are order-sensitive.**
   *Evidence:* `loadVault` sets `PageRel`, indexes maps, chooses or generates
   home, assigns canonical `URL` and `SourceURL`, then mutates generated home
   `HTML`. `renderVault` skips generated pages. `computeBacklinks` derives a
   local `byRel` map from rendered outgoing links. `writePage` copies a `Page`
   and changes the view copy's URLs to be relative.
   *Consequence:* the current order works, but the invariant is carried in the
   reader's head. A future feature such as custom output paths, aliases, or
   alternate home generation could easily break when a field is read before it
   has reached its expected lifecycle phase.
   *Fix direction:* document the build-order invariant near `buildSite` and
   `loadVault`. If churn continues, separate canonical page data from
   per-render view data more deliberately. Do not introduce a heavy builder or
   framework; the linear pipeline is a strength.

5. **Built-in assets are valid source files trapped inside Go strings.**
   *Evidence:* `defaultPageTemplate`, `defaultCSS`, and `defaultJS` start around
   lines 2216, 2293, and 2722. The JS contains search and graph behavior; the
   CSS contains the whole default theme.
   *Consequence:* frontend edits share the same diff context as Go parser
   changes, syntax mistakes are only caught by rendering a site in a browser,
   and the theme cannot be inspected as normal `*.css`/`*.js` files.
   *Fix direction:* either move the constants to `assets.go` as a first pass or,
   better, use stdlib `//go:embed` to keep real `theme/default/page.html`,
   `style.css`, and `onyx.js` files compiled into the binary. This preserves the
   no-dependency and single-binary contract.

6. **The public configuration surface has a few drift signals.**
   *Evidence:* README documents `site_title`, `source`, `theme`, `search`,
   `graph`, and `show_source`. Tests still write `base_url = /` in several
   fixtures, but the implementation ignores unknown keys and has no `base_url`
   behavior. `boolOr` also accepts legacy-ish `build.search`, `build.graph`,
   and `publish_raw_markdown` keys that are not documented.
   *Consequence:* this is not currently a bug, but it makes the true contract
   less obvious. Future config changes may accidentally preserve, remove, or
   document the wrong keys.
   *Fix direction:* clean stale test fixture keys and add a short config contract
   table in tests or README. Keep unknown-key tolerance unless there is a clear
   product reason to reject it.

### Smaller frictions

- `stripInlineMarkdown` and `extractTags` compile regexps per call. Hoisting
  them to package-level `regexp.MustCompile` vars is a small, safe cleanup once
  tests are in place.
- `isBlankFile` has UTF-16LE handling with 0% coverage. Either test it or remove
  it if that behavior is no longer intentional.
- `copyDir` and theme static asset handling have 0% coverage despite being the
  extension point for custom themes.
- `countPages` is used only for the success message. It is fine as-is, but it
  should not become a source of release logic.
- The two vanity-import HTML files are intentionally simple but easy to confuse
  with generated site HTML. A short comment in README's development section
  would help.

## Recommended order of attack

1. **Add focused safety tests before moving code.** Cover inline markdown,
   markdown links/images, asset resolution, ambiguous note/asset basenames,
   callouts, task lists, theme overrides, and config parsing. Keep using only
   Go's stdlib test tools.
2. **Split `onyx.go` by subsystem inside the same package.** Preserve the
   current binary, module path, CLI behavior, and no-dependency contract. This
   should be mostly mechanical after step 1.
3. **Extract the built-in theme into real files with stdlib embedding.** Use
   `//go:embed` for the default template/CSS/JS, or move strings to `assets.go`
   first if a smaller step is preferred. Verify generated asset bytes before
   and after.
4. **Consolidate resolver mechanics.** Share candidate path generation,
   basename lookup, nearest-match scoring, and ambiguity warning paths between
   note and asset resolution while keeping their product behavior explicit.
5. **Clarify page lifecycle invariants.** Add concise comments around
   `buildSite`/`loadVault`; consider separating canonical URLs from render-view
   URLs if future work touches output paths.
6. **Clean low-risk hot spots.** Hoist regexps, remove stale `base_url` test
   fixture keys, and either test or simplify the UTF-16 blank-file branch.
7. **Document contributor constraints.** Add a short development note that says
   architecture changes must preserve stdlib-only dependencies, single-binary
   install, relative URLs, and generated-output safety.

## Closing assessment

Onyx should not be "modernized" by adding parser libraries, frontend tooling,
or a broader framework. Its architecture is strongest where it is restrained:
stdlib-only Go, a linear build pipeline, relative static output, and conservative
overwrite safety. The dominant risk is that the implementation has outgrown the
single-file presentation and the tests are thinnest around the parsing/linking
code that future changes will touch most. A short campaign of focused tests,
same-package file splits, stdlib embedding, and resolver cleanup would make the
project easier to change while keeping the no-dependency promise intact.
