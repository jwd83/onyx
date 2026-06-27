# Architectural Review — Onyx

*Reviewed 2026-06-27 against `master` @ `0325866`.*

Onyx is a zero-dependency, single-binary Go static-site generator that publishes a
folder of Markdown notes (an Obsidian-style vault) as a plain static website. This
is a *clean* baseline review taken after two prior campaigns drove the codebase from
a monolith to a clean split and then through a test-hardening finishing pass (both
now under `plans/completed/`): the subsystem split, centralized href-scheme guards,
hoisted regexps, CI, and a 34-test suite at 91.9% coverage are all in place. The
result is a genuinely healthy small command — `gofmt`-clean, `go vet`-clean, tests
green in ~0.3s, stdlib-only. Because the structural risks the earlier reviews named
are largely retired, this review is a re-baseline, not a remediation backlog. The one
real remaining carrying cost is `markdown.go`: it is a third of all source and holds
the entire hand-rolled parser whose HTML safety rests on per-branch manual escaping.
Secondary costs are the link-resolution logic spread across five parallel entry
points, and a few destructive filesystem guards that are lightly tested. None of
these warrant a rewrite, a framework, or a Markdown dependency — the recommended plan
is short and mostly optional, and most of it should be done only when the relevant
file is next touched for other reasons.

## Snapshot

| Metric | Value |
| --- | --- |
| Reviewed | `master` @ `0325866`, 2026-06-27 |
| Repository shape | 28 tracked files; Go command in `onyx/`; vanity-import HTML at root `index.html` and `onyx/index.html`; two prior reviews under `plans/completed/` |
| Go module | `onyx.jwd.me`; `go 1.21`; built/tested with `go1.26.3 darwin/arm64` |
| Third-party dependencies | **0**; no `go.sum`; stdlib-only imports |
| Source footprint | 11 non-test Go files, 2,393 lines. Largest: `markdown.go` 760, `vault.go` 412, `links.go` 272, `config.go` 265, `page.go` 179; then `build.go` 127, `nav.go` 115, `search_graph.go` 107, `onyx.go` 71, `assets.go` 67, `theme.go` 18 |
| Test footprint | 4 test files, 1,514 lines, 34 `Test*` functions |
| Embedded default theme | 921 asset lines under `onyx/theme/default/` (`style.css` 427, `onyx.js` 419, `page.html` 75), embedded via `theme.go` |
| Documentation | `README.md` 204 lines |
| CI | `.github/workflows/ci.yml` — gofmt, vet, test, zero-dependency guard on push/PR |
| Formatting / static checks | `gofmt -l` clean; `go vet ./...` passes |
| Test baseline | `go test ./...` passes; **91.9%** of statements; ~0.3s |

Interpretation: every structural metric is stable and healthy. `markdown.go` is the
sole outlier at 760 lines — ≈32% of all source and nearly double the next file —
because it carries the complete bespoke block+inline parser plus text extraction.
Coverage is high and now fairly even; the remaining weak spots are concentrated in
the destructive filesystem guards (`preparePublic` 61.5%, `buildSite` 68.2%,
`ensureNoJekyll` 66.7%) and the asset/theme copy paths (`writeThemeOrDefault` 80%,
`copyDir` 81%) — exactly the I/O-error branches that are low-value to chase for their
own sake, with the one caveat noted in Risk 3.

## Structurally sound elements

These are load-bearing. Treat them as a do-not-break list, not praise.

- **The zero-dependency, single-binary contract holds.** `go.mod` has no `require`
  block, there is no `go.sum`, and the default theme compiles in via stdlib
  `go:embed` (`onyx/theme.go:11-18`). CI fails if a module or `go.sum` ever appears.
  This is a hard product constraint, now machine-enforced.
- **The subsystem file split matches the mental model.** `build.go`, `config.go`,
  `vault.go`, `markdown.go`, `links.go`, `nav.go`, `assets.go`, `page.go`,
  `search_graph.go`, `theme.go`, `onyx.go` each own one responsibility while staying
  in `package main`. That is the right amount of architecture for a command this size
  — resist adding packages or layers it doesn't need.
- **The build pipeline is explicit and self-documenting.** `buildSite`
  (`onyx/build.go:11`) carries a load-order comment spelling out why the sequence
  (guard outputs → load → render → backlinks → write) cannot be reordered, and
  `loadVault` (`onyx/vault.go:45`) documents the `Page` identity lifecycle. The linear
  flow is a genuine strength and the reason the renderer's size is tolerable.
- **`main` is thin and the core is testable.** `main` only calls `os.Exit(run(...))`;
  `run`/`runErr` take explicit `args`/`stdout`/`stderr` (`onyx/onyx.go:16-57`), and the
  integration tests drive `buildSite` against temp directories directly. The design is
  test-friendly without ceremony.
- **Conservative overwrite safety is real and tested.** `ensureRootIndexWritable`
  refuses to clobber an unmarked root `index.html`, and `preparePublic` refuses to
  `RemoveAll` a `public/` directory missing its `.onyx-generated` marker
  (`onyx/build.go:66,107`). Integration tests assert both refusals.
- **Href safety is centralized at one chokepoint.** Dangerous `javascript:`/`data:`
  schemes route through `hasDangerousHrefScheme`→`normalizedScheme`
  (`onyx/links.go:247-272`), which strips control/whitespace bytes before comparing, so
  tab/newline/NUL obfuscation can't bypass it. Both resolvers call it and `sanitizeHref`
  delegates to it. One place to audit, covered 100%.
- **Relative-URL output keeps GitHub Pages working.** `relativeRoot`/`relativeURL`
  (`onyx/page.go:157-179`) emit `../`-relative links so the site serves from a project
  URL such as `/repo/` with no hardcoded root; tests assert nested-page paths.
- **The theme is editable as source.** Default CSS/JS/template live as real files under
  `onyx/theme/default/` and are embedded, with no change to the one-binary contract.

## Structural risks and costs

Ranked by ongoing development cost.

1. **`markdown.go` concentrates complexity and the entire manual-escaping safety
   surface.**
   *Evidence:* one 760-line file (≈32% of source) holds three distinguishable jobs —
   block parsing (`renderBlocks` and its fence/math/table/heading/HR/blockquote/list
   helpers), inline parsing (`renderInline`, `onyx/markdown.go:510-602`), and text
   extraction (`plainText`, `extractTags`, `excerpt`, `slugify`). All HTML is built by
   string concatenation, and output safety depends on *every* branch remembering to call
   `html.EscapeString` on the right substrings (e.g. the heading writer at
   `onyx/markdown.go:114`, every attribute emitted in `renderInline`/`renderWiki`). There
   is no structural guard that a newly added branch escapes its output; the discipline is
   manual and per-branch. The earlier pass added an attribute-injection regression test
   and pinned the least-covered branches, so the discipline is at least *characterized* —
   but it remains the place where a future Markdown feature is most likely to introduce an
   injection or correctness regression and most expensive to review.
   *Consequence:* every renderer change re-opens the escaping question by hand. There is
   no *known* hole today, but the cost is paid on each edit and at each review.
   *Fix direction:* keep tests as the primary guardrail and extend the
   attribute-injection cases whenever link/image rendering is touched. When the renderer
   is next modified for other reasons, introduce one tiny shared helper that emits an
   escaped `name="value"` attribute (and/or an escaped-text writer) so escaping becomes
   structural rather than remembered; optionally split the file along its three existing
   seams. **Do not** swap in a third-party Markdown library (breaks the zero-dependency
   contract) and **do not** rewrite the parser speculatively — the file is still navigable
   and the split is a convenience, not a necessity.

2. **Link/asset resolution is spread across five parallel entry points.**
   *Evidence:* wikilink and Markdown-link resolution lives in `resolveNote`,
   `resolveAsset`, `resolveMarkdownHref`, `resolveMarkdownAsset` (`onyx/links.go`) plus
   the `renderWiki` dispatcher (`onyx/markdown.go:636`). The note and asset resolvers
   share the same shape — build `relativeCandidates`, probe `ByPath`/`AssetsByPath`, fall
   back to the `ByBase`/`AssetsByBase` basename index, then break ties with
   `nearestByFolder` — but each re-implements that shape against its own map and value
   type (`*Page` vs `string`). Anchor handling, `.md` stripping, and the
   current-folder-relative base are repeated with small variations between the wikilink
   and Markdown-link paths.
   *Consequence:* a change to resolution semantics (case-folding, anchor escaping, a new
   tie-break rule, a new candidate order) must be mirrored in up to five places, and a
   miss shows up only as a subtly wrong or broken link in a published vault. This is the
   second-most-likely place for a quiet regression after the renderer.
   *Fix direction:* extract the shared "candidates → exact-path index → basename index →
   `nearestByFolder`" lookup into one generic helper that both the note and asset
   resolvers call (a small generic or an index-abstraction over the two map pairs), so the
   tie-break and candidate-order policy lives in exactly one place. Behavior-preserving;
   the existing `TestResolveNote`/`TestResolveAsset`/`TestResolveMarkdown*` tests pin the
   contract to refactor against. Keep the wikilink-vs-Markdown *dispatch* separate — only
   the lookup core is duplicated.

3. **The destructive filesystem guards are the least-tested code in the build.**
   *Evidence:* `preparePublic` (61.5%), `buildSite` (68.2%), and `ensureNoJekyll`
   (66.7%) are the lowest-covered functions, and `preparePublic` is precisely where
   `os.RemoveAll(publicDir)` runs (`onyx/build.go:116`). The happy path and the
   missing-marker refusal are covered; the "`public/` exists but is a file", "exists with
   marker → removed and recreated", and `ensureNoJekyll` write-vs-skip branches are not
   directly asserted.
   *Consequence:* the guards that exist specifically to prevent deleting a user's
   non-Onyx `public/` are the ones with the thinnest regression net. Low probability, high
   blast radius if it ever regresses.
   *Fix direction:* add one focused test that exercises the marker round-trip (marked
   dir is replaced; file-at-`public/` errors; `.nojekyll` is created when absent and left
   alone when present). This is the single coverage gap worth closing on its merits rather
   than for the percentage; the remaining low-coverage I/O-error branches in
   `assets.go`/`build.go` are *not* worth chasing.

### Smaller frictions

- **`countPages` re-walks `public/` after the build** (`onyx/onyx.go:59`) to report a
  count the in-memory vault already knows (`len(vault.Notes)`, minus/plus the home).
  Harmless but redundant I/O and a second source of truth for "how many pages"; fold the
  count into the build result if `build.go`/`onyx.go` is touched.
- **Two near-duplicate vanity-import HTML files** (`index.html`, `onyx/index.html`)
  carry the `go-import`/`go-source` meta tags and must stay in sync if the module path
  ever changes. Intentional for `go get`; a one-line cross-reference comment in each would
  make the coupling explicit.
- **`extractTags` harvests `#tags` from raw Markdown, including fenced code blocks**
  (`onyx/markdown.go:720`): `#include`/`#define` inside a ```` ```c ```` block become site
  tags. This is a pinned, documented wart (`TestExtractTags`), not a safety issue. Whether
  to extract from the fence-stripped text instead is a deliberate behavior change, left as
  an explicit decision rather than an implied fix.
- **No recursion-depth bound** in `renderInline`/`renderBlocks` (both recurse on
  emphasis and on blockquotes/callouts). Input is the author's own trusted vault, so this
  is contained; only worth a thought if Onyx ever renders untrusted Markdown.

## Recommended order of attack

Behavior-preserving unless a step says otherwise. This is a short, mostly-optional
plan — the codebase is already in good shape, so each item should be taken up when it
buys real safety or rides along with other work, not as a standalone backlog to burn
down.

1. **Add the destructive-guard test (Risk 3).** Highest value-per-line: one table test
   over a temp `public/` that asserts the marker round-trip, the file-at-`public/` error,
   and `ensureNoJekyll`'s create-vs-skip. Closes the one coverage gap that protects against
   data loss. Do this first; it is independently useful and cheap.

   *Done 2026-06-27*, in commit `1a96c0a`:
   - Added `TestDestructiveOutputGuards` in `onyx/onyx_test.go`, covering marked
     `public/` replacement, refusal to treat a file at `public` as a generated output
     directory, `.nojekyll` creation, and preservation of an existing `.nojekyll`.

   Result: `gofmt -l $(git ls-files '*.go')`, `go vet ./...`, and
   `go test -count=1 ./... -coverprofile=/tmp/onyx-cover-after.out` pass. Overall
   statement coverage is now 92.1%; `preparePublic` moved from 61.5% to 76.9% and
   `ensureNoJekyll` moved from 66.7% to 83.3%. No behavior changed.

2. **Consolidate the resolver lookup core (Risk 2).** Extract the shared
   "candidates → exact index → basename index → `nearestByFolder`" path so note and asset
   resolution share one implementation, leaving the wikilink/Markdown dispatch in place.
   Refactor against the existing resolver tests; add none unless a gap appears.

   *Done 2026-06-27*, in commit `84dcf73`:
   - `onyx/links.go`: added generic `lookupRelativeTarget` and routed `resolveNote`
     and `resolveAsset` through it, preserving their separate warning messages and
     leaving Markdown-link dispatch unchanged.

   Result: focused resolver/link tests pass with
   `go test -count=1 ./... -run 'TestResolve|TestRenderWiki|TestRenderInline'`.
   Full local checks also pass: `gofmt -l $(git ls-files '*.go')`, `go vet ./...`,
   and `go test -count=1 ./... -coverprofile=/tmp/onyx-cover-next-step.out`.
   Overall statement coverage is 92.2%; `lookupRelativeTarget` is 100% covered,
   `resolveNote` remains 100%, and `resolveAsset` is 90.9%. No behavior changed.

3. **Fold `countPages` into the build result (smaller friction).** When `build.go` is
   next touched, have `buildSite`/`writePages` return the page count instead of re-walking
   `public/`, removing the redundant walk and the second source of truth. Trivial,
   behavior-preserving.

   *Done 2026-06-27*, in commit `84dcf73`:
   - `onyx/build.go`: added `buildResult` so `buildSite` returns warnings plus the
     generated page count.
   - `onyx/page.go`: changed `writePages` to count successfully written pages.
   - `onyx/onyx.go`: removed the `countPages` `public/` walk and prints the count from
     the build result instead.
   - `onyx/onyx_test.go`: pinned CLI counts for excluded notes and generated-home builds.

   Result: focused build-count tests pass with
   `go test -count=1 ./... -run 'TestBuildExcludesDraftAndPublishFalse|TestBuildReplacesBlankRootIndexWithGeneratedHome|TestBuildRendersHomepageWikilinksAndBacklinks'`.
   Full local checks also pass: `gofmt -l $(git ls-files '*.go')`, `go vet ./...`, and
   `go test -count=1 ./... -coverprofile=/tmp/onyx-cover-count.out`. Overall statement
   coverage remains 92.2%; `buildSite` is 69.6%, `writePages` is 80.0%, and `runErr` is
   83.3%. No behavior changed.

4. **When the renderer is next edited, make escaping structural (Risk 1).** Introduce a
   tiny escaped-attribute (and/or escaped-text) writer and route the inline/wiki/heading
   emitters through it, so a new branch can't forget to escape. Optionally split
   `markdown.go` along its three seams (`markdown_block.go` / `markdown_inline.go` / text
   extraction). Explicitly *deferred*: do this only as a rider on real renderer work, never
   as a speculative rewrite, and never by adding a Markdown dependency.

5. **(Optional, behavior-changing) Decide `extractTags` fenced-code handling.** If code
   blocks polluting tags is judged wrong, switch tag extraction to the already-computed
   fence-stripped text and update the pinned `TestExtractTags` case. Authorize explicitly
   before doing it; it changes published output.

6. **Add cross-reference comments to the dual vanity-import HTML files.** One line in each
   noting the other must stay in sync if the module path changes. Documentation only.

   *Done 2026-06-27*, in current working-tree changes:
   - `index.html`: added a comment tying its vanity-import tags to `onyx/index.html`.
   - `onyx/index.html`: added the reciprocal comment tying its vanity-import tags to
     `../index.html`.

   Result: `git diff --check` and `go test -count=1 ./...` pass. No generated output
   or Go behavior changed.

## Closing assessment

The dominant risk is no longer structure — two prior campaigns retired that — it is the
hand-rolled renderer's manually-enforced escaping discipline, with link-resolution
duplication and a lightly-tested set of destructive filesystem guards behind it. The
best leverage points are the cheapest: one guard test (step 1) and the resolver-core
consolidation (step 2) remove the two most likely sources of a quiet regression, and the
renderer hardening (step 4) is best carried in on the back of real feature work rather
than pursued for its own sake. Expected payoff is high relative to effort precisely
because the code is already good. Above all, preserve the restraint that makes Onyx good:
stdlib-only Go, one installed command, relative static output, conservative overwrite
safety, and a readable linear build pipeline. The two tempting overcorrections — swapping
in a Markdown library or building a `Page`/template framework — would each cost more than
they return; this project needs neither.
