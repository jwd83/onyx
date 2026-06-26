package main

import (
	"net/url"
	"path"
	"sort"
	"strings"
)

func splitAlias(s string) (string, string) {
	target, alias, ok := strings.Cut(s, "|")
	if !ok {
		return strings.TrimSpace(s), ""
	}
	return strings.TrimSpace(target), strings.TrimSpace(alias)
}

func splitHeading(s string) (string, string) {
	target, heading, ok := strings.Cut(s, "#")
	if !ok {
		return strings.TrimSpace(s), ""
	}
	return strings.TrimSpace(target), strings.TrimSpace(heading)
}

func looksLikeAsset(target string) bool {
	ext := strings.ToLower(path.Ext(target))
	if strings.ContainsAny(ext, " \t") {
		return false
	}
	return ext != "" && ext != ".md"
}

func isImage(rel string) bool {
	switch strings.ToLower(path.Ext(rel)) {
	case ".avif", ".gif", ".jpeg", ".jpg", ".png", ".svg", ".webp":
		return true
	default:
		return false
	}
}

func (r *MarkdownRenderer) resolveNote(target string) *Page {
	target = strings.TrimSpace(strings.TrimSuffix(target, ".md"))
	if target == "" {
		return r.current
	}

	candidates := []string{}
	if path.Dir(r.current.PageRel) != "." {
		candidates = append(candidates, path.Join(path.Dir(r.current.PageRel), target))
	}
	candidates = append(candidates, target)
	for _, candidate := range candidates {
		if page := r.vault.ByPath[strings.ToLower(path.Clean(candidate))]; page != nil {
			return page
		}
	}

	if strings.Contains(target, "/") {
		return nil
	}

	matches := r.vault.ByBase[strings.ToLower(target)]
	if len(matches) == 0 {
		return nil
	}
	if len(matches) == 1 {
		return matches[0]
	}

	best := nearestPage(r.current, matches)
	if best == nil {
		r.warn("ambiguous wikilink [[" + target + "]]")
	}
	return best
}

func nearestPage(current *Page, matches []*Page) *Page {
	bestDistance := 1 << 30
	var best *Page
	tie := false
	for _, candidate := range matches {
		distance := folderDistance(path.Dir(current.PageRel), path.Dir(candidate.PageRel))
		if distance < bestDistance {
			bestDistance = distance
			best = candidate
			tie = false
		} else if distance == bestDistance {
			tie = true
		}
	}
	if tie {
		return nil
	}
	return best
}

func folderDistance(a, b string) int {
	if a == "." {
		a = ""
	}
	if b == "." {
		b = ""
	}
	ap := splitPath(a)
	bp := splitPath(b)
	common := 0
	for common < len(ap) && common < len(bp) && strings.EqualFold(ap[common], bp[common]) {
		common++
	}
	return (len(ap) - common) + (len(bp) - common)
}

func splitPath(s string) []string {
	if s == "" || s == "." {
		return nil
	}
	return strings.Split(s, "/")
}

func (r *MarkdownRenderer) resolveAsset(target string) (string, bool) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", false
	}
	target = strings.TrimPrefix(path.Clean(target), "./")
	var candidates []string
	if path.Dir(r.current.SourceRel) != "." {
		candidates = append(candidates, path.Join(path.Dir(r.current.SourceRel), target))
	}
	candidates = append(candidates, target)
	for _, candidate := range candidates {
		if rel, ok := r.vault.AssetsByPath[strings.ToLower(candidate)]; ok {
			return rel, true
		}
	}
	if strings.Contains(target, "/") {
		return "", false
	}
	matches := r.vault.AssetsByBase[strings.ToLower(path.Base(target))]
	if len(matches) == 1 {
		return matches[0], true
	}
	if len(matches) > 1 {
		currentDir := path.Dir(r.current.SourceRel)
		sort.Slice(matches, func(i, j int) bool {
			return folderDistance(currentDir, path.Dir(matches[i])) < folderDistance(currentDir, path.Dir(matches[j]))
		})
		if folderDistance(currentDir, path.Dir(matches[0])) != folderDistance(currentDir, path.Dir(matches[1])) {
			return matches[0], true
		}
		r.warn("ambiguous asset [[" + target + "]]")
	}
	return "", false
}

func (r *MarkdownRenderer) resolveMarkdownHref(dest string) string {
	dest = strings.TrimSpace(dest)
	if isExternalOrAbsolute(dest) || strings.HasPrefix(dest, "#") {
		return sanitizeHref(dest)
	}

	cleanDest, anchor := splitURLAnchor(dest)
	if strings.EqualFold(path.Ext(cleanDest), ".md") {
		target := strings.TrimSuffix(cleanDest, ".md")
		candidates := []string{}
		if !strings.HasPrefix(target, "/") && path.Dir(r.current.PageRel) != "." {
			candidates = append(candidates, path.Join(path.Dir(r.current.PageRel), target))
		}
		candidates = append(candidates, strings.TrimPrefix(target, "/"))
		for _, candidate := range candidates {
			if page := r.vault.ByPath[strings.ToLower(path.Clean(candidate))]; page != nil {
				href := relativeURL(r.current, page.URL)
				if anchor != "" {
					href += "#" + url.PathEscape(slugify(anchor))
				}
				r.outgoing[page.PageRel] = true
				return href
			}
		}
	}

	return r.resolveMarkdownAsset(dest)
}

func (r *MarkdownRenderer) resolveMarkdownAsset(dest string) string {
	dest = strings.TrimSpace(dest)
	if isExternalOrAbsolute(dest) || strings.HasPrefix(dest, "#") {
		return sanitizeHref(dest)
	}
	cleanDest, anchor := splitURLAnchor(dest)
	if path.Dir(r.current.SourceRel) != "." && !strings.HasPrefix(cleanDest, "/") {
		cleanDest = path.Join(path.Dir(r.current.SourceRel), cleanDest)
	}
	cleanDest = strings.TrimPrefix(path.Clean(cleanDest), "/")
	href := relativeURL(r.current, escapeURLPath(path.Join(r.vault.Config.sourcePrefix(), cleanDest)))
	if anchor != "" {
		href += "#" + url.PathEscape(anchor)
	}
	return href
}

func splitURLAnchor(s string) (string, string) {
	before, after, ok := strings.Cut(s, "#")
	if !ok {
		return s, ""
	}
	return before, after
}

func isExternalOrAbsolute(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "tel:") ||
		strings.HasPrefix(s, "/")
}

func sanitizeHref(href string) string {
	lower := strings.ToLower(strings.TrimSpace(href))
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") {
		return "#"
	}
	return href
}
