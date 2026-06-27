package main

import (
	"bytes"
	"html/template"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type LinkView struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Path  string `json:"path"`
}

type TemplateData struct {
	Site       SiteView
	Page       *Page
	Nav        template.HTML
	Backlinks  []LinkView
	Search     bool
	Graph      bool
	ShowSource bool
	Generated  template.HTML
	RootScript template.JS
	PageID     template.JS
	HomeURL    string
	CSSURL     string
	JSURL      string
}

type SiteView struct {
	Title string
}

func writePages(vault *Vault) (int, error) {
	pageTemplate, err := loadTemplateSource(vault, "page.html", defaultPageTemplate)
	if err != nil {
		return 0, err
	}
	// The home page renders through the theme's home.html when present, otherwise
	// it falls back to defaultPageTemplate — the embedded default page template,
	// which the default theme ships no home.html override for. The fallback is the
	// *default* template, not whatever page.html resolved to above, so a custom
	// page.html does not silently restyle the home page.
	homeTemplate, err := loadTemplateSource(vault, "home.html", defaultPageTemplate)
	if err != nil {
		return 0, err
	}

	written := 0
	for _, page := range vault.Notes {
		tpl := pageTemplate
		if page.IsHome {
			tpl = homeTemplate
		}
		if err := writePage(vault, page, tpl); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

type templateSource struct {
	name   string
	source string
}

func loadTemplateSource(vault *Vault, name, fallback string) (templateSource, error) {
	themePath := filepath.Join(vault.Config.Root, filepath.FromSlash(vault.Config.Theme), name)
	source := fallback
	if data, err := os.ReadFile(themePath); err == nil {
		source = string(data)
	} else if !os.IsNotExist(err) {
		return templateSource{}, err
	}
	return templateSource{name: name, source: source}, nil
}

func parseTemplateForPage(vault *Vault, page *Page, source templateSource) (*template.Template, error) {
	funcs := template.FuncMap{
		"asset": func(name string) string {
			return relativeURL(page, "public/"+escapeURLPath(name))
		},
	}
	return template.New(source.name).Funcs(funcs).Parse(source.source)
}

func writePage(vault *Vault, page *Page, source templateSource) error {
	tpl, err := parseTemplateForPage(vault, page, source)
	if err != nil {
		return err
	}
	pageView := *page
	pageView.URL = relativeURL(page, page.URL)
	if page.SourceURL != "" {
		pageView.SourceURL = relativeURL(page, page.SourceURL)
	}
	data := TemplateData{
		Site: SiteView{
			Title: vault.Config.SiteTitle,
		},
		Page:       &pageView,
		Nav:        template.HTML(renderNav(vault, page)),
		Backlinks:  relativeLinkViews(page, page.Backlinks),
		Search:     vault.Config.Search,
		Graph:      vault.Config.Graph,
		ShowSource: vault.Config.ShowSource && page.SourceURL != "",
		Generated:  template.HTML(generatedMarker),
		RootScript: template.JS(strconv.Quote(relativeRoot(page))),
		PageID:     template.JS(strconv.Quote(page.PageRel)),
		HomeURL:    relativeURL(page, ""),
		CSSURL:     relativeURL(page, "public/onyx.css"),
		JSURL:      relativeURL(page, "public/onyx.js"),
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		return err
	}

	if page.IsHome {
		return os.WriteFile(filepath.Join(vault.Config.Root, "index.html"), out.Bytes(), 0o644)
	}

	dest := filepath.Join(vault.Config.Root, "public", filepath.FromSlash(page.PageRel), "index.html")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, out.Bytes(), 0o644)
}

func relativeLinkViews(from *Page, links []LinkView) []LinkView {
	out := make([]LinkView, 0, len(links))
	for _, link := range links {
		link.URL = relativeURL(from, link.URL)
		out = append(out, link)
	}
	return out
}

func escapeURLPath(rel string) string {
	rel = strings.TrimPrefix(path.Clean(strings.ReplaceAll(rel, "\\", "/")), "/")
	if rel == "." || rel == "" {
		return ""
	}
	parts := strings.Split(rel, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func relativeRoot(from *Page) string {
	if from == nil || from.IsHome || from.URL == "" {
		return ""
	}
	trimmed := strings.Trim(strings.TrimSuffix(from.URL, "/"), "/")
	if trimmed == "" {
		return ""
	}
	levels := len(strings.Split(trimmed, "/"))
	return strings.Repeat("../", levels)
}

func relativeURL(from *Page, target string) string {
	target = strings.TrimPrefix(target, "/")
	root := relativeRoot(from)
	if target == "" {
		if root == "" {
			return "./"
		}
		return root
	}
	return root + target
}
