package main

import (
	"net/url"
	"path"
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

	for _, key := range relativeCandidates(path.Dir(r.current.PageRel), target) {
		if page := r.vault.ByPath[key]; page != nil {
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

	folders := make([]string, len(matches))
	for i, page := range matches {
		folders[i] = path.Dir(page.PageRel)
	}
	if idx, ok := nearestByFolder(path.Dir(r.current.PageRel), folders); ok {
		return matches[idx]
	}
	r.warn("ambiguous wikilink [[" + target + "]]")
	return nil
}

// relativeCandidates returns the ordered lookup keys for target as seen from
// baseDir: the current-directory-relative form first (when baseDir is a real
// subfolder), then target as-is. Keys are cleaned and lowercased to match the
// vault's path indexes. It does no leading-slash handling; callers that allow
// root-absolute targets must strip the slash and pass baseDir "." themselves.
func relativeCandidates(baseDir, target string) []string {
	var keys []string
	if baseDir != "." && baseDir != "" {
		keys = append(keys, indexKey(path.Join(baseDir, target)))
	}
	return append(keys, indexKey(target))
}

func indexKey(p string) string {
	return strings.ToLower(path.Clean(p))
}

// nearestByFolder picks the single match whose folder is closest to currentDir
// and returns its index. ok is false when the list is empty or the nearest
// distance is shared by more than one match — i.e. the reference is ambiguous.
func nearestByFolder(currentDir string, folders []string) (int, bool) {
	best := 1 << 30
	bestIdx := -1
	tie := false
	for i, folder := range folders {
		distance := folderDistance(currentDir, folder)
		switch {
		case distance < best:
			best = distance
			bestIdx = i
			tie = false
		case distance == best:
			tie = true
		}
	}
	if bestIdx < 0 || tie {
		return -1, false
	}
	return bestIdx, true
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

	for _, key := range relativeCandidates(path.Dir(r.current.SourceRel), target) {
		if rel, ok := r.vault.AssetsByPath[key]; ok {
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
		folders := make([]string, len(matches))
		for i, rel := range matches {
			folders[i] = path.Dir(rel)
		}
		if idx, ok := nearestByFolder(path.Dir(r.current.SourceRel), folders); ok {
			return matches[idx], true
		}
		r.warn("ambiguous asset [[" + target + "]]")
	}
	return "", false
}

func (r *MarkdownRenderer) resolveMarkdownHref(dest string) string {
	dest = strings.TrimSpace(dest)
	if hasDangerousHrefScheme(dest) {
		return "#"
	}
	if isExternalOrAbsolute(dest) || strings.HasPrefix(dest, "#") {
		return sanitizeHref(dest)
	}

	cleanDest, anchor := splitURLAnchor(dest)
	if strings.EqualFold(path.Ext(cleanDest), ".md") {
		// Root-absolute dests ("/x.md") never reach here — isExternalOrAbsolute
		// returns them above — so target is always source-relative.
		target := strings.TrimSuffix(cleanDest, ".md")
		for _, key := range relativeCandidates(path.Dir(r.current.PageRel), target) {
			if page := r.vault.ByPath[key]; page != nil {
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
	if hasDangerousHrefScheme(dest) {
		return "#"
	}
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
	if hasDangerousHrefScheme(href) {
		return "#"
	}
	return href
}

func hasDangerousHrefScheme(href string) bool {
	scheme := normalizedScheme(href)
	return scheme == "javascript" || scheme == "data"
}

func normalizedScheme(href string) string {
	href = strings.TrimSpace(href)
	var b strings.Builder
	for _, r := range href {
		switch {
		case r == ':':
			return strings.ToLower(b.String())
		case r == '/' || r == '?' || r == '#':
			return ""
		case isASCIIControlOrSpace(r):
			continue
		default:
			b.WriteRune(r)
		}
	}
	return ""
}

func isASCIIControlOrSpace(r rune) bool {
	return r >= 0 && r <= ' ' || r == 0x7f
}
