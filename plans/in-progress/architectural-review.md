# Architectural Review — Onyx

*Reviewed 2026-06-26 against `master` @ `6b3e4f8`.*

Onyx is a zero-dependency, single-binary Go static-site generator that publishes a
folder of Markdown notes (an Obsidian-style vault) as a static website. The prior
review (now at `plans/completed/2026-06-27-architectural-review.md`) drove a
monolith-splitting and test-hardening campaign to completion; recent follow-ups
centralized dangerous-scheme handling in `hasDangerousHrefScheme`/`normalizedScheme`,
hoisted the per-call Markdown regexps to package level, added config/link/inline
tests, added CI, and overhauled the README. The result is a clean, navigable,
well-tested small command: `gofmt` clean, `go vet` passes, tests pass at 88.9%
coverage in under half a second. This is therefore a *finishing-pass* review, not a
structural-overhaul one. The dominant remaining carrying cost is the hand-rolled
Markdown renderer (the one large, safety-sensitive, manually-escaped surface),
followed by small output-guard and metadata/template edges. A minimal CI workflow now
enforces the checks the README already documents, config/source-resolution edges
are pinned with direct tests, and (2026-06-27) the previously under-covered renderer
branches plus an attribute-injection regression are now pinned as well. The remaining
recommended steps are the small metadata/template edges — documenting and pinning
`extractTags` current behavior and the default home/page template fallback — then
optional cohesion cleanups, with explicit restraint against rewriting the parser or
adding dependencies.

## Snapshot

| Metric | Value |
| --- | --- |
| Repository shape | 26 tracked files; Go command in `onyx/`; vanity-import HTML at root `index.html` and `onyx/index.html`; prior review under `plans/completed/` |
| Go module | `onyx.jwd.me`; `go 1.21`; built/tested with `go1.26.3 darwin/arm64` |
| Third-party dependencies | **0**; no `go.sum`; stdlib-only imports |
| Source footprint | 11 non-test Go files, 2,388 lines; largest: `markdown.go` 760, `vault.go` 412, `links.go` 272, `config.go` 265, `page.go` 174; `onyx.go` 71, `theme.go` 18 |
| Test footprint | 4 test files, 1,420 lines, 32 `Test*` functions |
| Embedded default theme | 921 asset lines under `onyx/theme/default/` (`style.css` 427, `onyx.js` 419, `page.html` 75), embedded via `theme.go` |
| Documentation | `README.md` 204 lines |
| Formatting / static checks | `gofmt -l` clean; `go vet ./...` passes |
| Test baseline | `go test ./... -coverprofile` passes; 91.5% of statements; ~0.36s |

Interpretation: the structural metrics are healthy and stable. `markdown.go` is the
one outlier at 760 lines — roughly a third of all source and nearly double the next
file — because it carries the entire bespoke block+inline parser plus text
extraction. Coverage is high overall but unevenly distributed: the strongest coverage
is on inline/link/config logic (`renderInline` 98.6%, `sanitizeHref`/`parseINI`/the
resolvers mostly 95–100%, `findConfig` 88.9%, `resolveSources` 88.0%) and, as of the
2026-06-27 renderer pass, the previously weak renderer branches (`startsBlock`
100%, `mathBlock` 100%, `parseMarkdownLink` 100%, `renderWiki` 96.3%). The
weakest spots are now concentrated in filesystem output guards (`preparePublic`
61.5%, `buildSite` 68.2%, `ensureNoJekyll` 66.7%) and tag extraction
(`extractTags` 54.5%).

Progress update 2026-06-27: `.github/workflows/ci.yml` now runs the README's
documented Go checks on push and pull request, and guards the zero-dependency
contract by failing if extra modules or `go.sum` appear.

Progress update 2026-06-27: `config_test.go` now pins root discovery and
source-resolution behavior: walk-up discovery, `onyx.ini`-marked and
content-folder-marked roots, explicit single/multi source handling, missing
explicit sources, non-directory sources, and the no-content-directory error.
This moved `findConfig` from 61.1% to 88.9% and `resolveSources` from 68.0% to
88.0%.

Progress update 2026-06-27: `markdown_test.go` now pins the under-covered
renderer branches called out in Risk 1 and step 3. It covers every
paragraph→block `startsBlock` transition, the single-line / open-fence-content /
close-fence-content / unterminated `mathBlock` forms, `parseMarkdownLink`'s three
rejection paths plus its success shape, and `renderWiki`'s asset-as-link,
note-embed, and embed-falls-through-to-broken-link branches. It also adds the
attribute-injection regression the review asked for: quote/angle characters in a
link href, image `alt`, and wikilink alias are asserted to be HTML-escaped so they
cannot break out of their attribute. This moved `startsBlock` 62.5%→100%,
`mathBlock` 72.2%→100%, `parseMarkdownLink` 72.7%→100%, `renderWiki` 77.8%→96.3%,
and `renderInline` 85.7%→98.6% (with incidental gains on the table-alignment
helpers), lifting overall statement coverage from 88.9% to 91.5%. No behavior
changed; these are characterization tests only.

## Structurally sound elements

These are load-bearing; treat them as a do-not-break list.

- **The zero-dependency, single-binary contract holds.** `go.mod` has no `require`
  block, there is no `go.sum`, and the default theme compiles in via stdlib
  `go:embed` (`theme.go:11-18`). The README's "no npm, no build system, no runtime
  server" promise still matches the code. This is a hard product constraint.
- **The subsystem file split matches the mental model.** `build.go`, `config.go`,
  `vault.go`, `markdown.go`, `links.go`, `nav.go`, `assets.go`, `page.go`,
  `search_graph.go`, `theme.go`, `onyx.go` each own one responsibility while staying
  in `package main`. That is the right amount of architecture for a command this size.
- **The build pipeline is explicit and documented.** `buildSite` (`onyx/build.go:11`)
  carries a load-order comment spelling out why the sequence (guard outputs → load →
  render → backlinks → write) cannot be reordered, and `loadVault`
  (`onyx/vault.go:45`) documents the `Page` identity lifecycle. The linear flow is a
  genuine strength.
- **Conservative overwrite safety is real and tested.** `ensureRootIndexWritable`
  refuses to clobber an unmarked root `index.html`, and `preparePublic` refuses to
  delete a `public/` directory missing its `.onyx-generated` marker
  (`onyx/build.go:66,107`). Integration tests assert both refusals.
- **Href safety is now centralized.** Dangerous `javascript:`/`data:` schemes route
  through `hasDangerousHrefScheme`→`normalizedScheme` (`onyx/links.go:247-272`), which
  strips control/whitespace bytes before comparing — closing tab/newline/NUL bypasses.
  Both `resolveMarkdownHref` and `resolveMarkdownAsset` call it, and `sanitizeHref`
  delegates to it. This is the right shape: one chokepoint, covered 100%.
- **Relative-URL output keeps GitHub Pages working.** `relativeRoot`/`relativeURL`
  (`onyx/page.go:152-174`) emit `../`-relative asset and page links so the site serves
  from a project URL such as `/repo/` with no hardcoded root. Tests assert nested-page
  relative paths.
- **The theme is editable as source.** Default CSS/JS/template live as real files
  under `onyx/theme/default/` and are embedded, removing the old "frontend trapped in
  Go string literals" problem with no change to install or runtime behavior.

## Structural risks and costs

Ranked by ongoing development cost.

1. **The hand-rolled Markdown renderer is the dominant complexity and safety surface.**
   *Evidence:* `markdown.go` is 760 lines (≈32% of source), holding three
   distinguishable jobs — block parsing (`renderBlocks`, fences, math, tables,
   headings, HR, blockquotes/callouts, lists), inline parsing (`renderInline`), and
   text extraction (`plainText`, `extractTags`, `excerpt`, `slugify`). HTML is built by
   string concatenation, and safety depends on *every* branch remembering to call
   `html.EscapeString` on the right substrings (e.g. `onyx/markdown.go:114`,
   `510-602`). There is no structural guard that a new branch escapes its output; the
   discipline is manual. As of 2026-06-27 the previously least-covered branches are
   pinned (`startsBlock` 100%, `mathBlock` 100%, `parseMarkdownLink` 100%,
   `renderWiki` 96.3%, `renderInline` 98.6%) and an attribute-injection regression
   test now guards link/image/wikilink escaping, so the manual discipline is at least
   characterized by tests.
   *Consequence:* this is still where correctness and injection regressions are most
   likely to appear and most expensive to review, and where new Markdown features
   carry the most risk. There is no *known* hole today (escaping and scheme checks are
   in place, and now exercised), but every renderer change re-opens the question.
   *Fix direction:* keep tests as the primary guardrail; extend the attribute-injection
   cases whenever link/image rendering is touched. If renderer churn continues,
   introduce one tiny shared helper that emits an escaped `name="value"` attribute so
   the discipline becomes structural rather than per-branch. **Do not** replace the
   parser with a third-party Markdown library — that breaks the zero-dependency
   contract — and do not rewrite it speculatively.

2. **The source-selection contract is now pinned by direct tests.**
   *Evidence:* `config_test.go` now covers walk-up root discovery, `onyx.ini`-marked
   roots, content-folder-marked roots without config, explicit single-source and
   multi-source config, missing explicit sources, non-directory explicit sources, and
   the no-content-directory error. Coverage rose to `findConfig` 88.9%,
   `hasDefaultSource` 100.0%, `readConfig` 85.7%, and `resolveSources` 88.0%.
   *Consequence:* the README's core convention rules ("run from a child directory",
   "list `source = docs, wiki`", "an explicit source that doesn't exist should fail
   loudly") now have focused regression coverage.
   *Fix direction:* treat this area as protected unless product behavior changes. If
   new source-selection features are added, extend these temp-directory table tests
   before editing the resolver.

3. **`extractTags` harvests tags from raw Markdown, including code blocks — and is the
   single lowest-covered logic function (54.5%).**
   *Evidence:* `extractTags` (`onyx/markdown.go:720`) runs `tagRE` over the entire raw
   document with no awareness of fenced code. A line like `#define` or `#include` inside
   a ```` ```c ```` block is extracted as a site tag, as is any `#word` after
   whitespace in a code sample. `plainText` strips fences for the search excerpt, but
   tag extraction does not.
   *Consequence:* search/graph metadata can be polluted by code content. It is a
   contained correctness wart, not a safety issue, but it is invisible until a vault
   with code blocks is published.
   *Fix direction:* first pin current behavior with a direct test (including the
   code-block case) so the gap is documented like the other deliberately-pinned
   Markdown quirks. If the behavior is judged wrong, the smallest fix is to extract
   tags from the already-computed fence-stripped text rather than the raw source — a
   behavior change, so make it a separate, explicitly-authorized step.

4. **CI now enforces the checks the README already documents.**
   *Evidence:* `.github/workflows/ci.yml` runs on push and pull request, sets up Go
   from `go.mod`, checks `gofmt`, runs `go vet ./...`, runs `go test ./...`, and fails
   if additional modules or `go.sum` appear without changing the zero-dependency
   contract deliberately.
   *Consequence:* the gofmt-clean, vet-clean, tests-green, stdlib-only invariants no
   longer depend only on local contributor discipline.
   *Fix direction:* keep the workflow in sync with the README's Development section if
   the project deliberately changes its local checks or dependency policy.

5. **The default theme ships no `home.html`, so the home/page split is implicit.**
   *Evidence:* `writePages` loads `home.html` with `defaultPageTemplate` as its
   fallback (`onyx/page.go:45`), but `onyx/theme/default/` contains only `page.html`,
   `style.css`, and `onyx.js`. So out of the box the generated home and ordinary pages
   render through the same template; the distinct `home.html` override the README
   advertises only matters when a user supplies one.
   *Consequence:* harmless today, but the "home uses `home.html`, falling back to the
   page template" rule is undocumented in code and untested for the default (no
   override) path. A future change to either template could diverge unnoticed.
   *Fix direction:* add a one-line comment at the `home.html` load site recording the
   fallback intent, and a small test asserting the default home renders via the page
   template fallback. No behavior change.

### Smaller frictions

- `countPages` (`onyx/onyx.go:59`) re-walks `public/` after the build to report a page
  count the in-memory vault already knows (`len(vault.Notes)`). Harmless but redundant
  I/O; fold into the build result if that path is touched.
- Two near-duplicate vanity-import files (`index.html`, `onyx/index.html`) carry the
  `go-import`/`go-source` meta tags. Intentional for `go get`, but worth a comment
  noting they must stay in sync if the module path ever changes.
- Filesystem error branches across `assets.go`/`build.go` (e.g. `writeThemeOrDefault`
  80%, `copyDir` 81%, `ensureNoJekyll` 66.7%) are genuinely low-value to test; do not
  chase coverage here for its own sake.

## Recommended order of attack

Behavior-preserving unless a step says otherwise. Each step is independently useful.

1. **Done 2026-06-27: add a minimal CI workflow** mirroring the README checks
   (`gofmt -l`, `go vet`, `go test ./...`) plus a zero-dependency guard. Highest
   leverage, lowest cost; every later step is now self-verifying on GitHub.
2. **Done 2026-06-27: pin the config/source-resolution edges with table tests**
   against temp directory trees: walk-up root discovery, `onyx.ini`-marked vs.
   content-folder-marked roots, explicit single/multi source, missing explicit source
   (must error), non-directory source. (Risk 2.)
3. **Done 2026-06-27: pin the under-covered renderer branches**: `parseMarkdownLink`
   rejection/success paths, `renderWiki` asset-vs-note-embed-vs-broken resolution,
   single-line/open/close/unterminated `mathBlock`, and every paragraph→block
   `startsBlock` transition, plus an attribute-injection regression test for Markdown
   link/image/wikilink rendering. (Risk 1.)
4. **Document and pin `extractTags` current behavior**, including the code-block case,
   the way other known Markdown quirks are already pinned. Decide separately, and only
   with explicit authorization, whether to change it to skip fenced code. (Risk 3.)
5. **Make the home/page template fallback explicit**: a comment at the `home.html`
   load site and a test that the default (no-override) home renders via the page
   template. (Risk 5.)
6. **Optional cohesion cleanup**: if `markdown.go` keeps growing, split it along its
   three existing seams (`markdown_block.go` / `markdown_inline.go` / text extraction)
   and, when next touching link/image output, introduce the tiny shared
   attribute-escaping helper from Risk 1. Do this only if it reduces real friction —
   the file is still navigable today.

## Closing assessment

The dominant risk is no longer structure; it is the hand-rolled renderer's
manually-enforced escaping discipline plus a few small, visible metadata/template
edges. With CI and config/source tests in place, the best next leverage is focused
renderer regression coverage, then the `extractTags` and home-template fallback
items. Expected payoff is disproportionate to effort — the code is already good, so
a small finishing pass buys durable confidence. Above all, preserve the restraint that makes Onyx good:
stdlib-only Go, one installed command, relative static output, conservative overwrite
safety, and a readable linear build pipeline. Resist the two tempting overcorrections —
swapping in a Markdown dependency or building a `Page` builder framework — neither of
which this project needs.
