# Architectural Review — Onyx

*Reviewed 2026-06-25 against `master` @ `f802ac2`.*

Onyx is a single-file Markdown static-site generator (`onyx/onyx.go`, ~3,140
lines) with no third-party dependencies, a vanity-import distribution path, and
a strong "one file, one binary, no build step" design ethos. The code is
genuinely well-factored for a single file: types are small and purposeful, the
build pipeline is a clear linear sequence, and behavior is locked in by a
focused test suite that targets the riskiest features (relative URLs, draft
exclusion, multi-source sections, math/tables). The dominant structural cost is
**monolithic scaling**: one 3.1k-line file holds config parsing, INI parsing,
Markdown block/inline rendering, wikilink resolution, nav tree, search/graph
JSON, HTML/CSS/JS assets, and the page template. The highest-leverage move is
not a rewrite but a disciplined internal split of that file along the seams
that already exist as comment headers — preserving the zero-dependency,
single-binary contract the README promises.

## Snapshot

| Metric | Value |
| --- | --- |
| Source file | `onyx/onyx.go` — 3,141 lines |
| Test file | `onyx/onyx_test.go` — 295 lines, 9 `Test*` functions |
| Total Go LOC | ~3,436 |
| Top-level funcs / methods | 98 / 15 |
| Type declarations | 14 structs (no interfaces) |
| Third-party deps | **0** (stdlib only; no `go.sum`) |
| Embedded asset strings | `defaultPageTemplate` 76 lines, `defaultCSS` 428 lines, `defaultJS` 420 lines (~1,924 lines combined, ~55% of file) |
| Largest single function | `renderBlocks` ~130 lines (1249–1378), `renderInline` ~94 lines (1706–1799) |
| Build/release model | `go run onyx.jwd.me/onyx@latest`; vanity import served by `index.html` + `.nojekyll` |
| `gofmt` | clean |
| Test baseline | **Could not run.** `go.mod` declares `go 1.26`; only `go1.22.2` is installed and the `go1.26` toolchain is "not available" for download in this environment. `GOTOOLCHAIN=local` blocks at the version gate. CI/author environment presumably has 1.26+. |

Interpretation: the project is small by intent and the dependency story is
excellent. The flip side is that more than half the file's line count is
string-literal CSS/JS/template, and the remaining ~1,200 lines of Go carry
roughly eight distinct subsystems. The test suite is good for what it covers
(relative paths, drafts, sections, math, tables, config fallbacks) but it is
**9 tests for a Markdown renderer + SSG pipeline**, which is thin for the
surface area — there are no dedicated tests for inline formatting (emphasis,
links, bare URLs), nav tree construction, backlink ordering, graph JSON shape,
or INI edge cases.

## Structurally sound elements

- **Zero-dependency, single-binary contract.** `go.mod` has no requires; all
  imports are stdlib. The README's "no npm, no build system, no runtime server"
  promise is actually enforced by the dependency graph. This is the core asset
  and must be preserved through any refactor — splitting files is fine, pulling
  in `goldmark`, `blackfriday`, or a config library is not, unless a deliberate
  decision overturns it.
- **Linear, ordered build pipeline.** `buildSite` (line 420) is a textbook
  sequence: validate root → ensure nojekyll → load vault → render → backlinks →
  prepare public → assets → pages → search → graph. Each step is independently
  readable and the warning/error contract (`[]string, error`) threads through
  cleanly. This is the right shape to keep.
- **Strong relative-URL discipline.** The whole site is designed to deploy under
  a project subpath (`/repo/`) without hardcoded roots. `relativeURL`/
  `relativeRoot` (2192–2213) and the tests' assertions on `../../public/...`
  paths show this is a load-bearing invariant, tested at the homepage and at
  nested depth. A refactor must keep this invariant in tests.
- **Safety rails against clobbering user content.** `ensureRootIndexWritable`
  (458), `preparePublic`'s `.onyx-generated` marker check (499–519), and
  `ensureNoJekyll` (474) make the tool non-destructive by default. The
  "refusing to overwrite" behavior is tested (`TestBuildRefusesUnmarkedPublicDirectory`).
- **Concept-dense, well-commented code.** `sourcePrefix`, `resolveNote`,
  `nearestPage`/`folderDistance`, and the multi-source vs single-source
  branching carry real domain logic and are documented with *why*, not *what*.
  The dual-key INI storage (`section.key` + bare `key`) is a clever touch.
- **Behavior-preserving test intent.** Existing tests assert exact relative
  hrefs and the absence of root-relative paths — they encode the invariants
  that matter most for a deployable SSG.

## Structural risks and costs

1. **Single 3,140-line file taxes every change.**
   *Evidence:* `onyx/onyx.go` holds config/INI parsing, Markdown block + inline
   rendering, wikilink/asset resolution, nav tree, search/graph emitters, and
   ~1,924 lines of embedded CSS/JS/HTML as `const` strings.
   *Consequence:* any edit forces navigating a 3k-line buffer; reviewers can't
   see diffs in context; grep returns hits across unrelated subsystems; IDE
   outlining and jump-to-definition are less useful. The markdown renderer and
   the asset strings live in the same compilation unit, so a CSS tweak and a
   parser fix produce the same noisy diff. New contributors face a wall of code
   with no module boundaries to orient on.
   *Fix direction:* split into a `package onyx` (or keep `package main` but
   multiple files in `onyx/`): `config.go`, `markdown.go`, `links.go`,
   `vault.go`, `nav.go`, `assets.go` (move the three big `const` strings to
   `assets_css.go`/`assets_js.go`/`assets_template.go`), `graph_search.go`,
   `page.go`. `go` lets one package span many files for free; the installed
   binary and `go run` UX are unchanged.

2. **No test baseline could be run here, and the suite is thin for the surface area.**
   *Evidence:* `go.mod` requires `go 1.26`; only 1.22 is locally available and
   the toolchain auto-download is unavailable. The suite has 9 tests; inline
   Markdown (`renderInline`) — emphasis, code spans, bare URLs, images,
   `[[wikilink|alias#heading]]` — has **no direct tests**; `renderNav`,
   `writeGraph`, `writeSearchIndex`, `parseINI` edge cases, and backlink
   ordering are untested.
   *Consequence:* the markdown renderer is the most edit-prone surface and also
   the least verified; regressions in inline parsing or link resolution will
   land silently. The inability to run tests in a fresh checkout (without the
   exact toolchain) raises the barrier for contributors and CI.
   *Fix direction:* (a) lower `go.mod`'s `go` directive to the minimum the code
   actually needs (the code uses only widely-available stdlib APIs — likely
   `go 1.21` is sufficient) so the toolchain gate stops blocking contributors;
   (b) add table-driven tests for `renderInline`, `resolveNote`/`resolveAsset`
   ambiguity, `parseINI` (BOM, comments, sections), and the nav tree.

3. **The hand-rolled Markdown renderer is a growing correctness/edge-case liability.**
   *Evidence:* `renderBlocks` (1249) and `renderInline` (1706) reimplement
   block + inline parsing with hand-rolled scanners. Known subtle behaviors
   already exist and are tested (math `$$` must survive verbatim, compact
   2-dash tables). `renderInline` is a left-to-right `switch` over prefix
   matches; nested emphasis (`**a *b* c**`), `__` vs `_` word-boundary rules,
   and intraword emphasis are not handled per CommonMark. `stripInlineMarkdown`
   (2098) compiles two regexps *per call*.
   *Consequence:* every new Markdown feature (footnotes, autolinks, task lists
   beyond the simple `[ ]` case) is a bespoke addition; divergence from user
   expectations is the silent cost. Per-call regexp compilation is a minor
   perf wart that signals the renderer wasn't built for reuse.
   *Fix direction:* do **not** rewrite to a CommonMark library (that breaks the
   zero-dependency promise and is out of scope). Instead: extract the renderer
   to its own file, hoist the two regexps in `stripInlineMarkdown` to package
   `var` (compile once), and add the inline/block test table from step 2 so
   future tweaks are guarded. Reserve a real CommonMark migration as an
   explicit product decision, not a side effect of refactoring.

4. **Embedded CSS/JS/template as string literals conflates "source" with "code".**
   *Evidence:* `defaultCSS` (429–2720), `defaultJS` (2722–end), and
   `defaultPageTemplate` (2216–2291) are backtick strings totaling ~1,924 lines
   inside `onyx.go`. The JS implements a force-directed graph canvas
   simulation, search, and keyboard shortcuts — non-trivial frontend code with
   no syntax check, no lint, and edits mixed into Go commits.
   *Consequence:* frontend changes can't be validated except by building a site
   and eyeballing it; the JS `tick()` layout math and pointer/wheel handlers
   are the most complex non-Go logic in the repo and have zero tests. Go diffs
   show 400-line CSS changes alongside parser fixes.
   *Fix direction:* keep the embed (preserves the single-binary contract) but
   move the strings to their own files (`assets_js.go` etc., or use
   `//go:embed` from real `*.js`/`*.css` files in a `theme/default/` dir once
   on Go 1.16+, which is already satisfied). `//go:embed` lets the CSS/JS be
   authored as real files with tooling while still compiling into the binary,
   with **zero behavioral change** to the install/`go run` UX. This is the
   single highest payoff refactor.

5. **Wikilink/asset resolution logic is subtle and duplicated across link types.**
   *Evidence:* `resolveNote`, `resolveAsset`, `resolveMarkdownHref`,
   `resolveMarkdownAsset` (1908–2067) each independently implement
   "candidate path list (current-dir-relative + as-is) → lookup" with slightly
   different normalization, lowercase keys, and ambiguity handling
   (`nearestPage`/`folderDistance` for notes vs. a sort-then-distance check for
   assets). `sourcePrefix` (49) and the `Multi` flag thread through four call
   sites.
   *Consequence:* the most bug-prone domain logic (does `[[Foo]]` resolve,
   ambiguously, or to an asset?) is duplicated across four near-parallel
   resolvers; a fix to one rarely propagates. Ambiguity warnings differ in
   shape between notes and assets.
   *Fix direction:* extract a small "path resolver" abstraction
   (`resolveRelative(base, target, index)`) shared by note and asset lookup,
   with one ambiguity policy. Cover it with the table tests from step 2. This
   is internal cleanup, not a public-API change.

6. **Dual representation of published state in `Page` and `Vault` invites drift.**
   *Evidence:* `vault.Notes` holds home + real notes; `ByPath`/`ByBase` are
   keyed maps; `computeBacklinks` (852) rebuilds a *third* `byRel` map. Home
   can be real or `Generated`; generated home HTML is mutated in place after
   construction (`updateGeneratedHome`/`updateSectionedHome`). `Page.URL` is
   mutated again at write time (`pageView.URL = relativeURL(page, page.URL)`).
   *Consequence:* the lifecycle of a `Page` (read → render → assign URL →
   relativize → write) is spread across `loadVault`, `renderVault`,
   `computeBacklinks`, `writePage`, each mutating shared struct fields. It
   works, but reasoning about "what's in `Page.URL` at this point" requires
   reading four functions. Generated home's `HTML` is set in
   `updateGeneratedHome` *before* `renderVault` runs, which skips generated
   pages — the ordering dependency is implicit.
   *Fix direction:* not urgent. Once the file is split (step 1) and
   resolution is consolidated (step 5), the lifecycle becomes more legible.
   Document the build order invariant in `buildSite`'s comment so future edits
   respect it. Avoid introducing a heavier "builder" abstraction — the linear
   pipeline is fine.

### Smaller frictions

- `countPages` (182) walks `public/` counting `index.html` *and* special-cases
  the root `index.html`; it's only used for the success message and is
  fragile (counts `.nojekyll`-adjacent files). Low priority.
- `isBlankFile` (484) has a UTF-16LE BOM branch that is essentially untested
  and may be dead in practice.
- `stripInlineComment`/`stripQuotes`/`valueOr`/`boolOr`/`truthy` are small
  config helpers that would naturally live in `config.go` after step 1.
- The vanity-import `index.html` and the source `onyx/index.html` share a
  name in different dirs; harmless but mildly confusing on a directory listing.

## Recommended order of attack

1. **Lower the `go.mod` toolchain gate.** Change `go 1.26` to the minimum
   version the code actually requires (verify with the oldest local toolchain
   that builds cleanly). This unblocks contributors and CI and lets the test
   baseline actually run in more environments. *Behavior-preserving; verify
   `go test ./...` passes on the chosen version.*
2. **Establish a runnable test baseline and broaden it.** Confirm the existing
   9 tests pass, then add table-driven tests for `renderInline` (emphasis,
   code, links, bare URLs, images, wikilink alias/heading), `parseINI` (BOM,
   inline comments, sections), and nav-tree shape. This guards every later
   step.
3. **Extract embedded assets to real files via `//go:embed`.** Move
   `defaultPageTemplate`, `defaultCSS`, `defaultJS` into `theme/default/*.html`
   / `*.css` / `*.js` and embed them. Author the JS/CSS as real files. *Zero
   change to the install UX or output; verify generated `public/onyx.css` and
   `public/onyx.js` are byte-identical before/after.*
4. **Split `onyx.go` into one-file-per-subsystem.** Along the existing comment
   seams: `config.go`, `vault.go`, `markdown.go`, `links.go`, `nav.go`,
   `assets.go`, `graph_search.go`, `page.go`, `template.go`. Same package,
   same binary, no behavior change. Run the suite after each coherent move.
5. **Consolidate the four link/asset resolvers.** Extract a shared
   relative-path resolver with one ambiguity policy; route note, asset,
   markdown-href, and markdown-asset lookups through it. Update the step-2
   table tests to cover cross-source and ambiguous-base cases.
6. **Hoist per-call regexps and small cleanups.** Move the two regexps in
   `stripInlineMarkdown` (and any others compiled inside hot functions) to
   package-level `var` with `regexp.MustCompile`. Tidy `countPages` and
   remove the UTF-16LE BOM branch if confirmed dead. *Behavior-preserving
   micro-cleanup; covered by the step-2 tests.*
7. **Document the build-order invariant.** Add a short comment block on
   `buildSite` (and `loadVault`) stating the required ordering
   (resolve sources → load pages → choose/generate home → assign URLs →
   render → backlinks → write) so future edits preserve the lifecycle that
   `Page` mutation depends on.

## Closing assessment

The dominant risk is **not** a wrong architecture — the pipeline, types, and
zero-dep contract are sound — but **monolithic scaling**: one file carrying
eight subsystems and ~1,900 lines of embedded frontend strings, with a test
suite too thin to safely evolve the markdown renderer and link resolver. The
best leverage point is `//go:embed` extraction of the asset strings (step 3),
because it converts the largest chunk of the file into editable, lintable,
real files at zero behavioral cost and immediately shrinks every future diff.
Pair that with the file split (step 4) and a real inline-markdown test table
(step 2), and Onyx becomes substantially easier to maintain while remaining
the single-binary, no-dependency tool the README promises. Reserve any move to
a real CommonMark library or a frontend build pipeline for an explicit product
decision — the current design's restraint is a feature, not a defect.
