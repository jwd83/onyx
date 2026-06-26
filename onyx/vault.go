package main

import (
	"html"
	"html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type Vault struct {
	Config       Config
	Notes        []*Page
	Home         *Page
	ByPath       map[string]*Page
	ByBase       map[string][]*Page
	AssetsByPath map[string]string
	AssetsByBase map[string][]string
}

type Page struct {
	SourceRel   string
	SourceAbs   string
	PageRel     string
	URL         string
	SourceURL   string
	Title       string
	Body        string
	HTML        template.HTML
	Text        string
	Excerpt     string
	Headings    []string
	Tags        []string
	Outgoing    map[string]bool
	Backlinks   []LinkView
	FrontMatter map[string]string
	IsHome      bool
	Generated   bool
	HasMath     bool
}

// loadVault builds the in-memory vault and fixes each Page's identity in a
// fixed order that the rest of the pipeline depends on:
//
//   - read pages from every source into Notes, then sort them for stable output;
//   - assign each note its PageRel and register it in ByPath/ByBase;
//   - choose the home page, then resolve the site title (generatedHome reads it),
//     synthesizing a home if none exists;
//   - assign canonical URL/SourceURL to every note;
//   - if the home is generated, build its HTML now — renderVault later skips
//     generated pages, so this is the only place that HTML is produced.
//
// After loadVault returns, every page has a stable PageRel and URL; later stages
// only read those fields (writePage re-relativizes URLs on a per-page copy).
func loadVault(cfg Config) (*Vault, []string, error) {
	vault := &Vault{
		Config:       cfg,
		ByPath:       map[string]*Page{},
		ByBase:       map[string][]*Page{},
		AssetsByPath: map[string]string{},
		AssetsByBase: map[string][]string{},
	}
	var warnings []string

	for _, src := range cfg.Sources {
		sourceDir := filepath.Join(cfg.Root, filepath.FromSlash(src))
		// With a single source, paths are relative to that folder (it builds
		// as-is). With several, they are relative to the site root, so each
		// note carries its source folder as a section prefix.
		relBase := sourceDir
		if cfg.Multi {
			relBase = cfg.Root
		}

		err := filepath.WalkDir(sourceDir, func(abs string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if abs == sourceDir {
				return nil
			}
			if shouldSkipName(d.Name()) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			relOS, err := filepath.Rel(relBase, abs)
			if err != nil {
				return err
			}
			rel := filepath.ToSlash(relOS)
			if hasSkippedComponent(rel) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}

			if strings.EqualFold(filepath.Ext(d.Name()), ".md") {
				page, excluded, err := readPage(abs, rel)
				if err != nil {
					return err
				}
				if excluded {
					warnings = append(warnings, rel+": excluded by frontmatter")
					return nil
				}
				vault.Notes = append(vault.Notes, page)
				return nil
			}

			vault.addAsset(rel)
			return nil
		})
		if err != nil {
			return nil, warnings, err
		}
	}

	sort.Slice(vault.Notes, func(i, j int) bool {
		return strings.ToLower(vault.Notes[i].SourceRel) < strings.ToLower(vault.Notes[j].SourceRel)
	})

	for _, note := range vault.Notes {
		note.PageRel = strings.TrimSuffix(note.SourceRel, path.Ext(note.SourceRel))
		vault.ByPath[strings.ToLower(note.PageRel)] = note
		base := strings.ToLower(path.Base(note.PageRel))
		vault.ByBase[base] = append(vault.ByBase[base], note)
	}

	for _, list := range vault.ByBase {
		sort.Slice(list, func(i, j int) bool {
			return strings.ToLower(list[i].SourceRel) < strings.ToLower(list[j].SourceRel)
		})
	}

	vault.Home = chooseHome(vault)
	// Resolve the site title before any generated home is built (generatedHome
	// reads it): an explicit onyx.ini title wins, else the home page's title,
	// else "Onyx".
	if vault.Config.SiteTitle == "" {
		if vault.Home != nil {
			vault.Config.SiteTitle = vault.Home.Title
		} else {
			vault.Config.SiteTitle = "Onyx"
		}
	}
	if vault.Home == nil {
		vault.Home = generatedHome(vault)
		vault.Notes = append([]*Page{vault.Home}, vault.Notes...)
	}
	vault.Home.IsHome = true

	for _, note := range vault.Notes {
		if note.IsHome {
			note.URL = ""
		} else {
			note.URL = "public/" + escapeURLPath(note.PageRel) + "/"
		}
		if note.SourceRel != "" {
			note.SourceURL = escapeURLPath(path.Join(cfg.sourcePrefix(), note.SourceRel))
		}
	}
	if vault.Home.Generated {
		updateGeneratedHome(vault)
	}

	return vault, warnings, nil
}

func shouldSkipName(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch strings.ToLower(name) {
	case ".obsidian", ".git", ".trash":
		return true
	}
	return false
}

func hasSkippedComponent(rel string) bool {
	for _, part := range strings.Split(rel, "/") {
		if shouldSkipName(part) {
			return true
		}
	}
	return false
}

func readPage(abs, rel string) (*Page, bool, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, false, err
	}
	text := normalizeNewlines(stripUTF8BOM(string(data)))
	frontMatter, body := splitFrontMatter(text)

	if strings.EqualFold(frontMatter["publish"], "false") || truthy(frontMatter["draft"]) {
		return nil, true, nil
	}

	title := strings.TrimSpace(frontMatter["title"])
	if title == "" {
		title = firstHeading(body)
	}
	if title == "" {
		title = strings.TrimSuffix(path.Base(rel), path.Ext(rel))
	}

	return &Page{
		SourceRel:   rel,
		SourceAbs:   abs,
		Title:       title,
		Body:        body,
		FrontMatter: frontMatter,
		Outgoing:    map[string]bool{},
	}, false, nil
}

func stripUTF8BOM(s string) string {
	return strings.TrimPrefix(s, "\ufeff")
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func splitFrontMatter(text string) (map[string]string, string) {
	frontMatter := map[string]string{}
	if !strings.HasPrefix(text, "---\n") {
		return frontMatter, text
	}

	lines := strings.Split(text, "\n")
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			for _, line := range lines[1:i] {
				key, value, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				frontMatter[strings.ToLower(strings.TrimSpace(key))] = stripQuotes(strings.TrimSpace(value))
			}
			return frontMatter, strings.Join(lines[i+1:], "\n")
		}
	}
	return map[string]string{}, text
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func firstHeading(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if level, text, ok := heading(trimmed); ok && level == 1 {
			return stripInlineMarkdown(text)
		}
	}
	return ""
}

func chooseHome(vault *Vault) *Page {
	if page := vault.ByPath["index"]; page != nil {
		return page
	}
	if page := vault.ByPath["home"]; page != nil {
		return page
	}
	return nil
}

func generatedHome(vault *Vault) *Page {
	return &Page{
		Title:     vault.Config.SiteTitle,
		Text:      vault.Config.SiteTitle,
		Excerpt:   "Generated index",
		Outgoing:  map[string]bool{},
		IsHome:    true,
		Generated: true,
	}
}

func updateGeneratedHome(vault *Vault) {
	if vault.Config.Multi {
		updateSectionedHome(vault)
		return
	}

	var b strings.Builder
	b.WriteString("<p>This site was generated from a folder of Markdown notes.</p>\n")
	b.WriteString("<ul>\n")
	for _, note := range vault.Notes {
		if note.Generated {
			continue
		}
		b.WriteString(`<li><a href="`)
		b.WriteString(html.EscapeString(relativeURL(vault.Home, note.URL)))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(note.Title))
		b.WriteString("</a></li>\n")
	}
	b.WriteString("</ul>\n")
	vault.Home.HTML = template.HTML(b.String())
}

// updateSectionedHome renders the multi-source landing page: one section per
// source folder, each listing its notes. The left-hand nav already nests the
// same folders, so the two stay in step.
func updateSectionedHome(vault *Vault) {
	var b strings.Builder
	b.WriteString("<p>This site collects several sets of notes. Choose a section to start.</p>\n")
	for _, src := range vault.Config.Sources {
		section := strings.ToLower(path.Clean(filepath.ToSlash(src))) + "/"
		var items []*Page
		for _, note := range vault.Notes {
			if note.Generated {
				continue
			}
			if strings.HasPrefix(strings.ToLower(note.PageRel)+"/", section) {
				items = append(items, note)
			}
		}
		if len(items) == 0 {
			continue
		}
		b.WriteString(`<section class="onyx-index-section">` + "\n")
		b.WriteString("<h2>")
		b.WriteString(html.EscapeString(titleCase(path.Base(path.Clean(filepath.ToSlash(src))))))
		b.WriteString("</h2>\n<ul>\n")
		for _, note := range items {
			b.WriteString(`<li><a href="`)
			b.WriteString(html.EscapeString(relativeURL(vault.Home, note.URL)))
			b.WriteString(`">`)
			b.WriteString(html.EscapeString(note.Title))
			b.WriteString("</a></li>\n")
		}
		b.WriteString("</ul>\n</section>\n")
	}
	vault.Home.HTML = template.HTML(b.String())
}

func (v *Vault) addAsset(rel string) {
	key := strings.ToLower(strings.TrimPrefix(path.Clean(rel), "./"))
	v.AssetsByPath[key] = rel
	base := strings.ToLower(path.Base(rel))
	v.AssetsByBase[base] = append(v.AssetsByBase[base], rel)
}

func renderVault(vault *Vault, warnings *[]string) {
	for _, note := range vault.Notes {
		if note.Generated {
			continue
		}
		renderer := &MarkdownRenderer{
			vault:      vault,
			current:    note,
			warnings:   warnings,
			headingIDs: map[string]int{},
			outgoing:   map[string]bool{},
		}
		result := renderer.Render(note.Body)
		note.HTML = template.HTML(result.HTML)
		note.Text = result.Text
		note.Excerpt = excerpt(result.Text, 220)
		note.Headings = result.Headings
		note.Tags = result.Tags
		note.HasMath = result.HasMath
		note.Outgoing = renderer.outgoing
	}
}

func computeBacklinks(vault *Vault) {
	byRel := map[string]*Page{}
	for _, note := range vault.Notes {
		if !note.Generated {
			byRel[note.PageRel] = note
		}
	}
	for _, note := range vault.Notes {
		for target := range note.Outgoing {
			if page := byRel[target]; page != nil && page != note {
				page.Backlinks = append(page.Backlinks, LinkView{
					Title: note.Title,
					URL:   note.URL,
					Path:  note.SourceRel,
				})
			}
		}
	}
	for _, note := range vault.Notes {
		sort.Slice(note.Backlinks, func(i, j int) bool {
			return strings.ToLower(note.Backlinks[i].Title) < strings.ToLower(note.Backlinks[j].Title)
		})
	}
}
