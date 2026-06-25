# Architectural Review — Onyx (independent)

*Reviewed 2026-06-25 against `master` @ `73c2b90`, by Claude Opus 4.8.*
*A prior review by GLM 5.2 lives at [`architectural-review.md`](architectural-review.md); this is an independent second opinion, and it corrects three of that review's measurements (see the box at the end).*

Onyx is a single-file, zero-dependency Markdown static-site generator: ~2,215 lines of Go plus ~924 lines of embedded CSS/JS/HTML, one binary, no build step, no runtime. The design is genuinely good for its size — a clean linear build pipeline, small purposeful types, strict relative-URL discipline so sites deploy under any subpath, and real safety rails that refuse to clobber files the tool didn't generate. It builds and **its tests pass here in 0.24 s at 60.5% statement coverage** on the declared toolchain. The dominant structural cost is **monolithic packaging**: one 3,141-line `package main` file holds eight subsystems, and the project's correctness-critical surfaces — the hand-rolled inline Markdown renderer and the four link/asset resolvers — sit almost entirely *outside* test coverage. The highest-leverage move is not a rewrite but (a) covering the inline renderer + resolvers with table tests, then (b) splitting the file along seams that already exist as comment headers — all while preserving the no-dependency, single-binary contract the README sells.

## Snapshot

| Metric | Value (measured 2026-06-25) |
| --- | --- |
| Source file | `onyx/onyx.go` — 3,141 lines, `package main` |
| Go logic | ~2,215 lines (lines 1–2215) |
| Embedded asset literals | `defaultPageTemplate` 76 + `defaultCSS` 428 + `defaultJS` 420 ≈ **924 lines (~29.5%)** |
| Test file | `onyx/onyx_test.go` — 295 lines, **9** `Test*` functions, all end-to-end |
| Top-level funcs / methods | 98 / 15 |
| Types | 14 structs, **0 interfaces** |
| Third-party deps | **0** (stdlib only; no `go.sum`) |
| `go.mod` directive | `go 1.26`; installed toolchain `go1.26.3` |
| Test baseline | **Runs and passes**: `go test ./...` → `ok 0.237s`, **coverage 60.5%**; `go vet` clean; `gofmt` clean |
| Largest functions | `renderInline` ~92 lines (1706), `renderBlocks` ~130 lines (1249) |
| Per-call regexp compiles | **3** (`stripInlineMarkdown` ×2, `extractTags` ×1) |

**Interpretation.** The dependency story is the project's crown jewel and it is real, not aspirational. The file is large but only ~30% of it is embedded front-end strings (not the ~55% the prior review reported), so the actual Go surface is ~2,200 lines carrying config/INI parsing, Markdown block + inline rendering, wikilink/asset resolution, nav-tree building, and search/graph emitters. Coverage at 60.5% sounds healthy but is unevenly distributed in exactly the wrong way: the all-integration test suite drives the happy paths well and leaves the most edit-prone, bug-prone logic near zero.

## Structurally sound elements

*(Do-not-break list, not praise.)*

- **Zero-dependency, single-binary contract.** `go.mod` has no `require` block; every import is stdlib. The README's "no npm, no build system, no runtime server" is enforced by the dependency graph itself. Splitting files is fine; pulling in `goldmark`/`blackfriday`/an INI library is not, absent a deliberate reversal.
- **Linear, legible build pipeline.** `buildSite` (420) reads top-to-bottom: validate root → `.nojekyll` → load vault → render → backlinks → prepare `public/` → assets → pages → search → graph. The `([]string, error)` warning/error contract threads cleanly. Keep this shape.
- **Relative-URL discipline is load-bearing and tested.** `relativeRoot`/`relativeURL` (2192) plus the tests asserting exact `../../public/...` hrefs and the *absence* of root-relative paths encode the invariant that lets a site deploy under `/repo/`. Any refactor must keep these assertions green.
- **Non-destructive by default.** `ensureRootIndexWritable` (458), the `.onyx-generated` marker gate in `preparePublic` (499), and `ensureNoJekyll` (474) mean Onyx won't overwrite hand-authored files. The refusal path is tested (`TestBuildRefusesUnmarkedPublicDirectory`). This is a strong, deliberate safety property.
- **No raw-HTML passthrough.** Inline/block rendering escapes text through `html.EscapeString`; there is no path that copies user Markdown verbatim into output. That closes the most common SSG XSS vector by construction (see risk #4 for the caveat).
- **Comments explain *why*.** `sourcePrefix`, the single-vs-multi source branching, the math-block verbatim handling, and the JS `tick()` distance-flooring note all document intent rather than restating code.

## Structural risks and costs

**1. The correctness-critical surface is the least-tested surface.**
*Evidence (measured per-function coverage):* `renderInline` 18.6%; `sanitizeHref` **0%**; `resolveAsset`, `resolveMarkdownHref`, `resolveMarkdownAsset` **0%**; `renderBlockquote`/callouts **0%**; `consumeBareURL` (bare-URL autolinking) **0%**; `nearestPage`/`folderDistance` (ambiguity resolution) **0%**; `parseMarkdownLink` **0%**. All 9 tests go through `run()` end-to-end, so these run only incidentally on the happy path.
*Consequence:* the inline renderer and the link resolvers are the most edit-prone code in the repo and also the least verified. A regression in emphasis nesting, ambiguous-wikilink tie-breaking, or `sanitizeHref` (the `javascript:`/`data:` guard — currently **0% covered**) would ship silently. 60.5% overall coverage masks this because the integration tests inflate coverage of orchestration code while leaving the parsers thin.
*Fix direction:* add table-driven *unit* tests against the pure functions — `renderInline` (emphasis, code, links, images, bare URLs, `[[alias|target#heading]]`), `resolveNote`/`resolveAsset` (current-dir-relative, base-name, ambiguous, cross-source), `sanitizeHref`, and `parseINI` (BOM, inline comments, sections). This is the single highest-leverage step and it guards every later one.

**2. One 3,141-line file taxes every change.**
*Evidence:* a single `package main` file holds config + INI parsing, Markdown block + inline rendering, four resolvers, nav-tree construction, search/graph JSON, and ~924 lines of embedded CSS/JS/template.
*Consequence:* every edit means navigating a 3k-line buffer; a CSS tweak and a parser fix produce the same noisy diff; grep returns cross-subsystem hits; new contributors get no module boundaries to orient on.
*Fix direction:* split into one file per subsystem in the *same* package — `config.go`, `vault.go`, `markdown.go`, `links.go`, `nav.go`, `assets.go`, `graph_search.go`, `page.go`. Go lets a package span many files for free; the binary and `go run`/`go install` UX are unchanged. Do this **after** step 1 so the tests catch any accidental behavior change during the move.

**3. The hand-rolled Markdown renderer has real, demonstrable CommonMark gaps.**
*Evidence:* `renderInline` (1706) is a left-to-right prefix `switch`. It handles `**`/`__` (strong) and `*` (em) but has **no single-`_` case** — so `_italic_` renders literally, which will surprise Obsidian users who write that form. `*` emphasis matches the next `*` with no flanking rule, so `a * b * c` wraps ` b ` in `<em>`. Nested/intraword emphasis is not CommonMark-correct.
*Consequence:* each new Markdown feature is a bespoke addition, and silent divergence from author expectations is the ongoing cost. These are correctness gaps, not crashes, so they surface as "why didn't my italics render?" bug reports rather than build failures.
*Fix direction:* do **not** adopt a CommonMark library (that breaks the zero-dep contract and is a product decision, not a refactor). Instead, treat the renderer's *documented* behavior as the spec, add the inline test table from step 1, and decide deliberately whether single-`_` emphasis and flanking rules are in scope. Lock whatever you choose with tests.

**4. The HTML-safety model is manual and its one guard is untested.**
*Evidence:* the renderer builds `template.HTML` (trusted) by hand-concatenating strings and calling `html.EscapeString` at each site; `html/template`'s contextual auto-escaping is therefore bypassed for all rendered content. `sanitizeHref` (2086) is the lone guard against `javascript:`/`data:` URLs — and it has **0% test coverage**.
*Consequence:* safety depends entirely on every current and future writer in `renderInline`/`renderBlocks` remembering to escape, with no test asserting the `javascript:` guard actually fires. There is no *known* hole today (raw HTML is escaped, hrefs are sanitized), but the architecture has no backstop if a future branch forgets.
*Fix direction:* add direct tests for `sanitizeHref` (including `JavaScript:`/whitespace/encoded variants) and for an attribute-injection attempt in a link label/title. Consider a single `writeEscaped` helper so escaping isn't re-implemented at each call site. Low effort, removes a latent risk.

**5. Four near-parallel link/asset resolvers duplicate the trickiest logic.**
*Evidence:* `resolveNote` (1908), `resolveAsset` (1987), `resolveMarkdownHref` (2023), `resolveMarkdownAsset` (2052) each independently build a candidate list (current-dir-relative + as-is), lowercase-key a lookup, and handle ambiguity — but with subtly different normalization (`ByPath` vs `AssetsByPath`) and different tie-break shapes (`nearestPage`/`folderDistance` for notes; an inline sort-then-compare for assets).
*Consequence:* the single most bug-prone question in the tool — "does `[[Foo]]` resolve to a note, an asset, ambiguously, or break?" — is answered in four places, so a fix to one rarely propagates, and ambiguity warnings differ in wording between notes and assets. All four are at 0% coverage today.
*Fix direction:* extract one `resolveRelative(base, target, index)` helper with a single ambiguity policy; route all four call paths through it. Cover with the step-1 table tests. Internal cleanup, no public-surface change.

**6. `Page` is mutated across four functions, with an implicit ordering dependency.**
*Evidence:* `vault.Notes` holds home + notes; `ByPath`/`ByBase` are parallel indexes; `computeBacklinks` builds a *third* `byRel` map. `Page.URL` is assigned in `loadVault` then re-relativized in `writePage` (`pageView.URL = relativeURL(...)`). The generated home's `HTML` is set by `updateGeneratedHome` *before* `renderVault` runs — and `renderVault` skips generated pages, so the order is load-bearing but undocumented.
*Consequence:* answering "what's in `Page.URL` at this point?" requires reading four functions, and the home-generation ordering is a tripwire for a future edit that reorders the pipeline.
*Fix direction:* not urgent. After steps 2 and 5 the lifecycle is more legible; in the meantime, add a short comment on `buildSite`/`loadVault` stating the required order (resolve sources → load pages → choose/generate home → assign URLs → render → backlinks → write). Resist introducing a heavier "builder" abstraction — the linear pipeline is the right shape.

### Smaller frictions

- **Three regexps compiled per call** (`stripInlineMarkdown` lines 2099/2107, `extractTags` 2138). `stripInlineMarkdown` runs once per heading (via `slugify`) and per page (via `plainText`), so this is real but minor. Hoist to package-level `var` with `regexp.MustCompile`.
- **`go 1.26` directive is ~8 versions higher than the code needs.** The newest API used is `strings.Cut` (Go 1.18); everything else predates it. *Verified empirically:* compiling against a real `go1.17.13` fails (`undefined: strings.Cut`, 5 sites) while `go1.18.10` builds and passes the suite. So the directive can be lowered to **`go 1.18`** (true floor) or `go 1.21` (a "modern" floor with headroom for `min`/`max`/`slices`). This is a portability courtesy — **not** a blocker, since the suite runs fine on the declared toolchain — but it removes the gate that `GOTOOLCHAIN=local` contributors hit (and the reason the prior review believed the tests couldn't run).
- **`isBlankFile` (484)** has a UTF-16LE BOM branch that is untested and likely dead in practice.
- **`countPages` (182)** double-counts logic (walks `public/` *and* special-cases root `index.html`) purely for a success message; fragile but low-stakes.
- The vanity-import `index.html` (repo root) and `onyx/index.html` share a name in different dirs — harmless, mildly confusing in a listing.

## Recommended order of attack

1. **Unit-test the parsers and resolvers first.** Table-driven tests for `renderInline`, `resolveNote`/`resolveAsset` (relative, base-name, ambiguous, cross-source), `sanitizeHref`, and `parseINI`. Target the 0%/18% functions. *This guards every subsequent step and is the highest-leverage move — do it before touching structure.*
2. **Decide and pin the Markdown spec.** Using the new test table, make an explicit call on single-`_` emphasis and `*` flanking (risk #3). Encode the decision as tests; change behavior only if you choose to.
3. **Add the missing safety test + escaping helper.** Cover `sanitizeHref` and one attribute-injection attempt; optionally centralize escaping behind one helper (risk #4).
4. **Consolidate the four resolvers** into one shared relative-path resolver with a single ambiguity policy (risk #5). Run the step-1 table tests after.
5. **Extract embedded assets via `//go:embed`.** Move `defaultPageTemplate`/`defaultCSS`/`defaultJS` to real `theme/default/*.{html,css,js}` files and embed them, so the front-end can be linted/edited with tooling. *Verify generated `public/onyx.css` and `public/onyx.js` are byte-identical before/after; zero change to install UX.*
6. **Split `onyx.go` into one file per subsystem** (same package, same binary): `config.go`, `vault.go`, `markdown.go`, `links.go`, `nav.go`, `assets.go`, `graph_search.go`, `page.go`. Run the suite after each coherent move.
7. **Micro-cleanups:** hoist the three per-call regexps; document the build-order invariant on `buildSite`; remove the dead UTF-16LE branch in `isBlankFile` if confirmed.

## Closing assessment

The architecture is not wrong — the pipeline, the small types, the relative-URL invariant, and the zero-dependency contract are all sound and worth defending. The dominant risk is the combination of **monolithic packaging** and a **test suite that covers orchestration but not the parsers**: 60.5% overall coverage hides an inline renderer at 18.6% and four link resolvers plus the `javascript:` guard at 0%. The best leverage point — contrary to the prior review's "extract assets first" — is to **put the inline renderer and resolvers under unit test before restructuring anything**, because those are both the most bug-prone code and the least guarded. With that net in place, the asset extraction (`//go:embed`) and the file split become safe, mechanical wins. Reserve any move to a real CommonMark library or a front-end build pipeline for an explicit product decision; the current restraint is a feature.

---

### Where this review differs from the prior (GLM 5.2) review

I agree with its six core risks. I diverge on three points of fact, each verified by measurement on this checkout:

1. **Test baseline.** The prior review states the suite "could not be run" because `go 1.26` was unavailable. On this machine `go1.26.3` is installed; `go test ./...` **passes in 0.237 s at 60.5% coverage** and `go vet`/`gofmt` are clean. Consequently I demote "lower the `go.mod` gate" from its **Step 1** to an optional smaller friction, and I make *unit-testing the parsers* the real Step 1.
2. **Embedded-asset size.** It reports the three asset literals as "~1,924 lines combined, ~55%" with `defaultCSS` at "429–2720". Actual: `defaultCSS` is `2293–2720`; the three literals total **~924 lines (~29.5%)**, and the Go logic is **~2,215 lines** (not ~1,200). The asset extraction is still worthwhile, but it is not "more than half the file," so I rank it below the testing and resolver-consolidation work rather than as the top payoff.
3. **Per-call regexps.** It cites two (`stripInlineMarkdown`); there are **three** — `extractTags` compiles one per page too.
