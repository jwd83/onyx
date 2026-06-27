# Architectural Review — Onyx

*Reviewed 2026-06-26 local EDT against `master` @ `47518fb`. Completed
2026-06-27 after the config-contract, contributor-guardrail, and review
lifecycle follow-ups were applied.*

Onyx has made the jump the previous review was aiming for: the project is still
a zero-dependency, single-binary Go static-site generator, but it is no longer
presented as a 3,141-line monolith. The source is now split across focused files,
the built-in theme lives as real embedded assets, the link/asset resolver paths
share mechanics, and the highest-risk inline/link/config surfaces have direct
tests. The dominant carrying cost has shifted from "too much in one file" to a
smaller, sharper set of risks: generated-output behavior, theme/static override
behavior, and the still-manual HTML safety model need continued regression
coverage. This completed review is retained as the record of that campaign.

## Snapshot

| Metric | Value |
| --- | --- |
| Repository shape | 26 tracked files; Go command in `onyx/`; vanity import HTML at root and `onyx/index.html`; completed review artifact under `plans/completed/` |
| Go module | `onyx.jwd.me`; `go 1.21`; tested with `go1.26.3 darwin/arm64` |
| Third-party dependencies | **0**; no `go.sum`; imports are stdlib-only |
| Source footprint | 11 non-test Go files, 2,388 lines; largest: `markdown.go` 760, `vault.go` 412, `links.go` 272, `config.go` 265; `onyx.go` is now 71 lines |
| Test footprint | 4 Go test files, 986 lines, 22 `Test*` functions |
| Embedded default theme | 921 real asset lines under `onyx/theme/default/`, embedded through `theme.go`: CSS 427, JS 419, page template 75 |
| Documentation | `README.md` 83 lines at review time; completed review refresh at this file |
| Formatting/static checks | `gofmt -l $(rg --files -g '*.go')` clean; `go vet ./...` passes |
| Test baseline | `go test ./... -coverprofile=/tmp/onyx-cover.out` passes; coverage is 87.7% of statements |

Interpretation: the earlier structural campaign paid off. Since the previous
`73c2b90` review baseline, the repo lowered the Go directive (`4eff5ff`), split
the former monolith across subsystem files (`afef272`), extracted the theme into
real embedded files (`bac7313` and `32b0573`), added 487 lines of parser and
resolver tests (`12bd59a`), consolidated resolver mechanics (`4da549c`), and
documented the load-bearing build order (`47518fb`). Function coverage confirms
the improvement where it mattered most: `renderInline` is 85.7% covered,
`renderBlocks` is 93.8%, `renderBlockquote` is 95.7%,
`renderListItem` is 100%, `parseINI` and `sanitizeHref` are 100%, and the
link/asset resolvers are mostly 95-100%. The newest build-level tests now cover
custom theme CSS/JS/template overrides, `theme/static` copying, blank root
`index.html` replacement, generated single-source home output, and
`isBlankFile`. The per-call Markdown regexps have also been hoisted to
package-level compiled values. The remaining weak spots are concentrated and
visible: `boolOr` 50% and `extractTags` 58.3%.

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

1. **The output/theme test net now covers the highest-risk write paths.**
   *Evidence:* Function coverage now exercises the block renderer directly:
   `renderBlocks` is 93.8%, `renderBlockquote` is 95.7%,
   `stripBlockquoteMarker`, `listMarker`, and `renderListItem` are 100%, and
   `sanitizeClass` is 88.9%. The custom-theme test moved `copyDir` to 81%
   coverage, while the blank-index/generated-home tests moved
   `ensureRootIndexWritable` to 75%, `isBlankFile` to 100%, and
   `updateGeneratedHome` to 100%. The existing integration tests now cover
   custom theme CSS/JS/template overrides, `theme/static` copying, blank root
   `index.html` replacement, generated single-source home output, relative
   public asset paths, sectioned generated homes, and unmarked `public/`
   refusal.
   *Consequence:* the highest-risk write paths are now pinned by tests. The
   remaining risk is less about missing output coverage and more about keeping
   the small helpers simple enough that future output-path changes are easy to
   review.
   *Fix direction:* stop expanding broad build fixtures for now; move to the
   small cleanup items unless a new output feature changes the risk profile.

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

3. **Theme extraction is complete, and the extension surface now has a first integration check.**
   *Evidence:* `writeThemeOrDefault` is 80% covered, `copyDir` is 81% covered,
   and `loadTemplateSource` is 85.7% covered. The default JS is a real 419-line
   file, but no check verifies it parses after edits. The README documents
   `theme/` overrides and `theme/static`, and the build tests now assert custom
   CSS, JS, page/home templates, copied static assets, relative asset URLs, and
   skipped dotfiles.
   *Consequence:* the new file layout is much better for editing, and override
   lookup/static copying are now pinned. JS syntax mistakes may still only show
   up after a generated site is opened in a browser.
   *Fix direction:* if a JS runtime is available in development, add an optional
   local `node --check onyx/theme/default/onyx.js` note; do not make the project
   depend on Node.

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

5. **The public config contract is now explicit, but should stay small.**
   *Evidence:* README documents `site_title`, `source`, `theme`, `search`,
   `graph`, and `show_source`, and now states that unknown keys are ignored.
   `onyx/config_test.go` pins the legacy aliases `build.search`,
   `build.graph`, and `publish_raw_markdown`, including modern-key precedence
   when both forms are present. Stale integration fixtures no longer include
   ignored `base_url = /` lines.
   *Consequence:* future config edits have a clearer compatibility target. The
   remaining tradeoff is intentional: tolerating unknown keys keeps old configs
   building, but it also means misspelled keys do not fail loudly.
   *Fix direction:* keep the current small key set unless product needs force an
   expansion. Add direct config tests for any new key or alias before changing
   README guidance.

### Smaller frictions

- `stripInlineMarkdown` and `extractTags` now reuse package-level compiled
  regexps instead of compiling them per call.
- `isBlankFile` now has direct ASCII and UTF-16LE coverage. Keep the branch
  unless a later compatibility decision removes support for UTF-16LE blank root
  files.
- This artifact has been moved through the completed-plan lifecycle to
  `plans/completed/2026-06-27-architectural-review.md`.
- The README now documents the local contributor guardrails: `gofmt -l`,
  `go vet`, `go test`, the zero-dependency contract, relative generated URLs,
  and conservative generated-output safety. There is still no CI/workflow file,
  so automated PR checks remain an optional follow-up if repository activity
  makes them useful.

## Remaining Follow-Up

1. **Optionally add CI when automation is worth the upkeep.** A lightweight
   workflow can mirror the README checks: `gofmt -l`, `go vet`, and
   `go test ./...`.

## Closing assessment

The architecture is in a much better place than the previous baseline. Onyx no
longer needs a monolith-splitting campaign; it needs a finishing pass that makes
the newly modular code as well protected at the block/output/theme edges as it
now is at the inline/link/config edges. The best leverage point is tests first,
then small hardening and cleanup. Preserve the restraint that makes Onyx good:
stdlib-only Go, one installed command, relative static output, conservative
overwrite safety, and a readable linear build pipeline.
