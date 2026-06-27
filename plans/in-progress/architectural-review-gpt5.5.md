# Architectural Review — Onyx

*Reviewed 2026-06-26 local EDT against `master` @ `47518fb`.*

Onyx has made the jump the previous review was aiming for: the project is still
a zero-dependency, single-binary Go static-site generator, but it is no longer
presented as a 3,141-line monolith. The source is now split across focused files,
the built-in theme lives as real embedded assets, the link/asset resolver paths
share mechanics, and the highest-risk inline/link/config surfaces have direct
tests. The dominant carrying cost has shifted from "too much in one file" to a
smaller, sharper set of risks: generated-output behavior, theme/static override
behavior, and the still-manual HTML safety model need continued regression
coverage. This file is now the single active architectural review artifact.

## Snapshot

| Metric | Value |
| --- | --- |
| Repository shape | 26 tracked files; Go command in `onyx/`; vanity import HTML at root and `onyx/index.html`; one active review artifact |
| Go module | `onyx.jwd.me`; `go 1.21`; tested with `go1.26.3 darwin/arm64` |
| Third-party dependencies | **0**; no `go.sum`; imports are stdlib-only |
| Source footprint | 11 non-test Go files, 2,385 lines; largest: `markdown.go` 757, `vault.go` 412, `links.go` 272, `config.go` 265; `onyx.go` is now 71 lines |
| Test footprint | 4 Go test files, 880 lines, 19 `Test*` functions |
| Embedded default theme | 921 real asset lines under `onyx/theme/default/`, embedded through `theme.go`: CSS 427, JS 419, page template 75 |
| Documentation | `README.md` 83 lines; active review refresh at this file |
| Formatting/static checks | `gofmt -l $(rg --files -g '*.go')` clean; `go vet ./...` passes |
| Test baseline | `go test ./... -coverprofile=/tmp/onyx-cover.out` passes in 0.240s; coverage is 84.0% of statements |

Interpretation: the earlier structural campaign paid off. Since the previous
`73c2b90` review baseline, the repo lowered the Go directive (`4eff5ff`), split
the former monolith across subsystem files (`afef272`), extracted the theme into
real embedded files (`bac7313` and `32b0573`), added 487 lines of parser and
resolver tests (`12bd59a`), consolidated resolver mechanics (`4da549c`), and
documented the load-bearing build order (`47518fb`). Function coverage confirms
the improvement where it mattered most: `renderInline` is 85.7% covered,
`renderBlocks` is 93.8%, `renderBlockquote` is 95.7%,
`renderListItem` is 100%, `parseINI` and `sanitizeHref` are 100%, and the
link/asset resolvers are mostly 95-100%. The remaining weak spots are
concentrated and visible: `copyDir` 0%, `isBlankFile` 0%,
`updateGeneratedHome` 18.8%, `boolOr` 50%, and `extractTags` 58.3%.

## Structurally sound elements

- **The no-dependency, single-binary contract survived the refactor.** `go.mod`
  has no `require` block, the default theme is compiled with stdlib `go:embed`,
  and the README's "no npm, no build system, no runtime server" claim still
  matches the code. This remains a hard product constraint.
- **Subsystem files now match the mental model.** `build.go`, `config.go`,
  `vault.go`, `markdown.go`, `links.go`, `nav.go`, `assets.go`, `page.go`,
  `search_graph.go`, and `theme.go` keep `package main` but give the project a
  navigable table of contents. That is the right amount of architecture for a
  small command.
- **The build pipeline is explicit and documented.** `buildSite` now carries a
  clear load-order comment at `onyx/build.go:11`, and `loadVault` documents the
  `Page` lifecycle at `onyx/vault.go:45`. The linear flow remains a strength:
  validate outputs, load pages, render, compute backlinks, then write.
- **Resolver behavior has a real safety net.** `onyx/links_test.go` covers
  wikilink resolution, asset resolution, Markdown hrefs, anchors, ambiguous
  basenames, current-directory preference, and warning behavior. That directly
  addresses one of the riskiest old gaps.
- **Inline Markdown behavior is now pinned.** `onyx/markdown_test.go` covers
  strong/emphasis/code escaping, bare URLs, Markdown links, wikilinks, aliases,
  headings, broken links, and image embeds. It also deliberately pins known
  behavior gaps such as single-underscore emphasis and loose asterisk pairing.
- **Block-level Markdown behavior is now pinned.** `onyx/markdown_test.go`
  covers blockquotes, callouts, task-list checkboxes, fenced-code class
  sanitization, horizontal rules, duplicate heading IDs, and heading collection.
- **Generated-output safety and relative URLs remain load-bearing tests.**
  Integration tests still assert refusal to overwrite an unmarked `public/`
  directory, creation of `.nojekyll`, relative `public/onyx.css` and
  `public/onyx.js` paths, nested-page `../../` links, and cross-source paths.
- **Theme source is editable as source.** The default CSS/JS/template now live
  under `onyx/theme/default/` and are embedded from `theme.go:11-18`. This
  removes the old "frontend trapped in Go string literals" problem without
  changing install or runtime behavior.

## Structural risks and costs

1. **The remaining test gap is now output/theme behavior, not Markdown parsing.**
   *Evidence:* Function coverage now exercises the block renderer directly:
   `renderBlocks` is 93.8%, `renderBlockquote` is 95.7%,
   `stripBlockquoteMarker`, `listMarker`, and `renderListItem` are 100%, and
   `sanitizeClass` is 88.9%. The low-coverage areas have moved to filesystem and
   generated-output paths: `copyDir` 0%, `isBlankFile` 0%, and
   `updateGeneratedHome` 18.8%. The existing integration tests still do not
   cover custom theme overrides, `theme/static` copying, blank root `index.html`
   replacement, or generated single-source home output.
   *Consequence:* the most likely future regressions now sit at the boundary
   where Onyx writes files or honors user theme/config choices. A change to
   theme override lookup, static asset copying, blank-file safety, or generated
   home pages could slip through even though parser coverage is healthy.
   *Fix direction:* add build-level tests for custom theme overrides,
   `theme/static` copying, blank root `index.html` handling, and generated
   single-source home output.

2. **The HTML/href safety model is still manual, but dangerous schemes are now centralized.**
   *Evidence:* Markdown output is constructed by hand and then stored as
   `template.HTML` (`onyx/vault.go:379`, `onyx/page.go:102`). Most branches call
   `html.EscapeString`, and raw Markdown HTML is not passed through. Dangerous
   `javascript:` and `data:` schemes now route through a shared stdlib-only
   helper in `onyx/links.go` before Markdown destinations can fall back to asset
   paths, and tests cover tab/newline/NUL scheme bypasses plus rendered Markdown
   links.
   *Consequence:* there is no known raw-HTML or dangerous-scheme hole today, but
   link safety still depends on current and future rendering branches using the
   central helper and the right escaping calls.
   *Fix direction:* keep href handling centralized and add attribute-injection
   regression tests when touching Markdown link/image rendering. If renderer
   churn continues, consider a tiny shared escaping helper for link and image
   attributes.

3. **Theme extraction is complete, but the theme extension surface is thinly verified.**
   *Evidence:* `writeThemeOrDefault` is 60% covered and `copyDir` is 0% covered.
   The default JS is a real 419-line file, but no check verifies it parses after
   edits. The README documents `theme/` overrides and `theme/static`, while tests
   primarily exercise the built-in default asset paths.
   *Consequence:* the new file layout is much better for editing, but mistakes
   in override lookup, static asset copying, or JS syntax may only show up after
   a generated site is opened in a browser.
   *Fix direction:* add one integration test that supplies custom `theme/style.css`,
   `theme/onyx.js`, `theme/page.html`, and `theme/static/*`, then asserts output
   bytes and relative links. If a JS runtime is available in development, add an
   optional local `node --check onyx/theme/default/onyx.js` note; do not make the
   project depend on Node.

4. **`Page` mutation is documented but still spread across the pipeline.**
   *Evidence:* `loadVault` sets `PageRel`, `URL`, `SourceURL`, home status, and
   generated-home HTML; `renderVault` fills `HTML`, `Text`, `Excerpt`,
   `Headings`, `Tags`, `HasMath`, and `Outgoing`; `computeBacklinks` mutates
   backlinks; `writePage` copies a page and rewrites URL fields for the template
   view. The comments now describe the order, which is a major improvement.
   *Consequence:* ordinary maintenance is fine, but features like aliases,
   custom output paths, drafts by folder, or alternate page views would still
   require careful cross-file reasoning about which `Page` fields are canonical
   and which are render-time or view-time.
   *Fix direction:* leave the current model alone until a feature forces it. If
   output-path work arrives, introduce a narrow page-view constructor or helper
   around the `writePage` copy instead of a broad builder framework.

5. **The public config contract still has drift signals.**
   *Evidence:* README documents `site_title`, `source`, `theme`, `search`,
   `graph`, and `show_source`. The implementation also accepts `build.search`,
   `build.graph`, and `publish_raw_markdown` (`onyx/config.go:111-113`), while
   several integration fixtures still include ignored `base_url = /` keys.
   Unknown keys are tolerated.
   *Consequence:* this is not a current bug, but it leaves the true compatibility
   contract fuzzy. Future config edits may accidentally preserve, remove, or
   document legacy keys without an explicit decision.
   *Fix direction:* remove stale `base_url` fixture lines, add config tests for
   legacy toggles or deliberately drop/document them, and add a short README
   development note saying unknown keys are tolerated intentionally if that is
   the chosen contract.

### Smaller frictions

- `stripInlineMarkdown` and `extractTags` still compile regexps per call at
  `onyx/markdown.go:677` and `onyx/markdown.go:716`. Hoist them to
  package-level `regexp.MustCompile` vars once the current tests are green.
- `isBlankFile` still has an untested UTF-16LE blank-file branch
  (`onyx/build.go:96-103`). Either test it or remove it if the behavior is not
  intentional.
- `plans/in-progress/architectural-review-gpt5.5.md` is the only active review
  artifact now. Consider renaming it to a model-neutral
  `architectural-review.md` if the review will remain the canonical plan.
- There is no CI/workflow file. For a project this small, a basic `go test`,
  `go vet`, and `gofmt -l` workflow would give high confidence with low upkeep.

## Recommended order of attack

1. **Finish the current test net around output/theme behavior.** Add build-level
   tests for custom `style.css`, `onyx.js`, `page.html`/`home.html`, copied
   `theme/static` files, blank root `index.html` handling, generated single-source
   home output, and `isBlankFile`.
2. **Hoist the regexps and clean the small hot spots.** Move the three per-call
   regexps to package vars; either test or simplify the UTF-16 blank-file branch;
   keep `countPages` as success-message-only code.
3. **Clarify the config compatibility contract.** Remove stale `base_url`
   fixtures, decide whether legacy keys are supported, and document/test that
   decision.
4. **Document contributor guardrails.** Add a short README development note or
   lightweight CI workflow covering `go test ./...`, `go vet ./...`, `gofmt -l`,
   zero third-party dependencies, relative URL behavior, and generated-output
   safety.
5. **Rename or complete the active review artifact when this campaign is done.**
   If this remains the canonical review, use a model-neutral filename; otherwise
   move it through the completed-plan lifecycle.

## Closing assessment

The architecture is in a much better place than the previous baseline. Onyx no
longer needs a monolith-splitting campaign; it needs a finishing pass that makes
the newly modular code as well protected at the block/output/theme edges as it
now is at the inline/link/config edges. The best leverage point is tests first,
then small hardening and cleanup. Preserve the restraint that makes Onyx good:
stdlib-only Go, one installed command, relative static output, conservative
overwrite safety, and a readable linear build pipeline.
