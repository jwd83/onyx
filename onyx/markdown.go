package main

import (
	"fmt"
	"html"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type MarkdownRenderer struct {
	vault      *Vault
	current    *Page
	warnings   *[]string
	headingIDs map[string]int
	outgoing   map[string]bool
	headings   []string
	hasMath    bool
}

type RenderResult struct {
	HTML     string
	Text     string
	Headings []string
	Tags     []string
	HasMath  bool
}

func (r *MarkdownRenderer) Render(markdown string) RenderResult {
	markdown = normalizeNewlines(markdown)
	htmlOut := r.renderBlocks(strings.Split(markdown, "\n"))
	text := plainText(markdown)
	return RenderResult{
		HTML:     htmlOut,
		Text:     text,
		Headings: r.headings,
		Tags:     extractTags(markdown),
		HasMath:  r.hasMath,
	}
}

func (r *MarkdownRenderer) renderBlocks(lines []string) string {
	var b strings.Builder
	for i := 0; i < len(lines); {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			i++
			continue
		}

		if fence, info, ok := fenceStart(trimmed); ok {
			j := i + 1
			var code []string
			for ; j < len(lines); j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), fence) {
					break
				}
				code = append(code, lines[j])
			}
			class := ""
			if info != "" {
				class = ` class="language-` + html.EscapeString(sanitizeClass(info)) + `"`
			}
			b.WriteString("<pre><code")
			b.WriteString(class)
			b.WriteString(">")
			b.WriteString(html.EscapeString(strings.Join(code, "\n")))
			b.WriteString("</code></pre>\n")
			if j < len(lines) {
				i = j + 1
			} else {
				i = j
			}
			continue
		}

		if mathLines, next, ok := mathBlock(lines, i); ok {
			r.hasMath = true
			b.WriteString(`<div class="onyx-math">`)
			b.WriteString("$$\n")
			b.WriteString(html.EscapeString(strings.Join(mathLines, "\n")))
			b.WriteString("\n$$")
			b.WriteString("</div>\n")
			i = next
			continue
		}

		if isTableStart(lines, i) {
			j := i + 2
			for j < len(lines) && strings.Contains(lines[j], "|") && strings.TrimSpace(lines[j]) != "" {
				j++
			}
			b.WriteString(r.renderTable(lines[i:j]))
			i = j
			continue
		}

		if level, text, ok := heading(trimmed); ok {
			title := stripInlineMarkdown(text)
			id := r.uniqueHeadingID(slugify(title))
			r.headings = append(r.headings, title)
			fmt.Fprintf(&b, `<h%d id="%s">%s</h%d>`+"\n", level, html.EscapeString(id), r.renderInline(text), level)
			i++
			continue
		}

		if isHorizontalRule(trimmed) {
			b.WriteString("<hr>\n")
			i++
			continue
		}

		if strings.HasPrefix(strings.TrimLeft(line, " "), ">") {
			j := i
			var quote []string
			for j < len(lines) {
				if strings.TrimSpace(lines[j]) == "" {
					quote = append(quote, "")
					j++
					continue
				}
				if !strings.HasPrefix(strings.TrimLeft(lines[j], " "), ">") {
					break
				}
				quote = append(quote, stripBlockquoteMarker(lines[j]))
				j++
			}
			b.WriteString(r.renderBlockquote(quote))
			i = j
			continue
		}

		if ordered, _, ok := listMarker(line); ok {
			tag := "ul"
			if ordered {
				tag = "ol"
			}
			b.WriteString("<" + tag + ">\n")
			j := i
			for j < len(lines) {
				nextOrdered, nextContent, ok := listMarker(lines[j])
				if !ok || nextOrdered != ordered {
					break
				}
				b.WriteString("<li>")
				b.WriteString(r.renderListItem(nextContent))
				b.WriteString("</li>\n")
				j++
			}
			b.WriteString("</" + tag + ">\n")
			i = j
			continue
		}

		j := i
		var para []string
		for j < len(lines) {
			if strings.TrimSpace(lines[j]) == "" || startsBlock(lines, j) {
				break
			}
			para = append(para, strings.TrimSpace(lines[j]))
			j++
		}
		b.WriteString("<p>")
		b.WriteString(r.renderInline(strings.Join(para, " ")))
		b.WriteString("</p>\n")
		i = j
	}
	return b.String()
}

func fenceStart(line string) (string, string, bool) {
	if strings.HasPrefix(line, "```") {
		return "```", strings.TrimSpace(line[3:]), true
	}
	if strings.HasPrefix(line, "~~~") {
		return "~~~", strings.TrimSpace(line[3:]), true
	}
	return "", "", false
}

func sanitizeClass(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	s = fields[0]
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func startsBlock(lines []string, i int) bool {
	trimmed := strings.TrimSpace(lines[i])
	if trimmed == "" {
		return true
	}
	if _, _, ok := fenceStart(trimmed); ok {
		return true
	}
	if _, _, ok := heading(trimmed); ok {
		return true
	}
	if isHorizontalRule(trimmed) {
		return true
	}
	if strings.HasPrefix(strings.TrimLeft(lines[i], " "), ">") {
		return true
	}
	if _, _, ok := listMarker(lines[i]); ok {
		return true
	}
	if strings.HasPrefix(trimmed, "$$") {
		return true
	}
	return isTableStart(lines, i)
}

func heading(line string) (int, string, bool) {
	count := 0
	for count < len(line) && line[count] == '#' {
		count++
	}
	if count == 0 || count > 6 || count >= len(line) || line[count] != ' ' {
		return 0, "", false
	}
	return count, strings.TrimSpace(line[count+1:]), true
}

func isHorizontalRule(line string) bool {
	if len(line) < 3 {
		return false
	}
	for _, marker := range []rune{'-', '*', '_'} {
		count := 0
		ok := true
		for _, r := range line {
			if unicode.IsSpace(r) {
				continue
			}
			if r != marker {
				ok = false
				break
			}
			count++
		}
		if ok && count >= 3 {
			return true
		}
	}
	return false
}

func stripBlockquoteMarker(line string) string {
	line = strings.TrimLeft(line, " ")
	if strings.HasPrefix(line, ">") {
		line = line[1:]
	}
	return strings.TrimPrefix(line, " ")
}

func (r *MarkdownRenderer) renderBlockquote(lines []string) string {
	first := ""
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			first = strings.TrimSpace(line)
			break
		}
	}

	if strings.HasPrefix(first, "[!") {
		end := strings.Index(first, "]")
		if end > 2 {
			kind := strings.ToLower(strings.TrimSpace(first[2:end]))
			title := strings.TrimSpace(first[end+1:])
			if title == "" {
				title = titleCase(kind)
			}
			body := dropFirstNonBlank(lines)
			var b strings.Builder
			b.WriteString(`<aside class="callout callout-`)
			b.WriteString(html.EscapeString(sanitizeClass(kind)))
			b.WriteString(`"><p class="callout-title">`)
			b.WriteString(html.EscapeString(title))
			b.WriteString("</p>\n")
			b.WriteString(r.renderBlocks(body))
			b.WriteString("</aside>\n")
			return b.String()
		}
	}

	return "<blockquote>\n" + r.renderBlocks(lines) + "</blockquote>\n"
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(strings.ToLower(s))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func dropFirstNonBlank(lines []string) []string {
	out := append([]string(nil), lines...)
	for i, line := range out {
		if strings.TrimSpace(line) != "" {
			return out[i+1:]
		}
	}
	return nil
}

func listMarker(line string) (bool, string, bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(trimmed) >= 2 && (trimmed[0] == '-' || trimmed[0] == '*' || trimmed[0] == '+') && trimmed[1] == ' ' {
		return false, strings.TrimSpace(trimmed[2:]), true
	}
	i := 0
	for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
		i++
	}
	if i > 0 && i+1 < len(trimmed) && trimmed[i] == '.' && trimmed[i+1] == ' ' {
		return true, strings.TrimSpace(trimmed[i+2:]), true
	}
	return false, "", false
}

func (r *MarkdownRenderer) renderListItem(content string) string {
	if strings.HasPrefix(content, "[ ] ") || strings.HasPrefix(strings.ToLower(content), "[x] ") {
		checked := strings.HasPrefix(strings.ToLower(content), "[x] ")
		rest := strings.TrimSpace(content[4:])
		box := `<input type="checkbox" disabled>`
		if checked {
			box = `<input type="checkbox" disabled checked>`
		}
		return box + " " + r.renderInline(rest)
	}
	return r.renderInline(content)
}

// mathBlock detects a display math block delimited by $$ and returns the LaTeX
// content lines (without the $$ fences) along with the index of the line that
// follows the block. Both the single-line form `$$ ... $$` and the multi-line
// form with `$$` on its own line are supported. The content is preserved
// verbatim so MathJax can typeset it on the client.
func mathBlock(lines []string, i int) ([]string, int, bool) {
	trimmed := strings.TrimSpace(lines[i])
	if !strings.HasPrefix(trimmed, "$$") {
		return nil, 0, false
	}
	rest := trimmed[2:]
	// Single-line: $$ ... $$
	if strings.HasSuffix(rest, "$$") && len(rest) >= 2 {
		return []string{strings.TrimSpace(rest[:len(rest)-2])}, i + 1, true
	}
	// Multi-line: opening $$ (optionally with trailing content) until closing $$.
	var body []string
	if head := strings.TrimSpace(rest); head != "" {
		body = append(body, head)
	}
	for j := i + 1; j < len(lines); j++ {
		lt := strings.TrimSpace(lines[j])
		if lt == "$$" {
			return body, j + 1, true
		}
		if strings.HasSuffix(lt, "$$") {
			body = append(body, strings.TrimSpace(strings.TrimSuffix(lt, "$$")))
			return body, j + 1, true
		}
		body = append(body, lines[j])
	}
	// Unterminated block: keep the remaining lines as math rather than mangling them.
	return body, len(lines), true
}

func isTableStart(lines []string, i int) bool {
	if i+1 >= len(lines) {
		return false
	}
	if !strings.Contains(lines[i], "|") {
		return false
	}
	return isTableSeparator(lines[i+1])
}

func isTableSeparator(line string) bool {
	cells := splitTableRow(line)
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			return false
		}
		cell = strings.Trim(cell, ":")
		if cell == "" {
			return false
		}
		for _, r := range cell {
			if r != '-' {
				return false
			}
		}
	}
	return true
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// parseTableAlignments reads the separator row of a table and returns the
// CSS text-align value for each column. An empty string means no explicit
// alignment (the default).
func parseTableAlignments(line string) []string {
	cells := splitTableRow(line)
	aligns := make([]string, len(cells))
	for i, cell := range cells {
		cell = strings.TrimSpace(cell)
		left := strings.HasPrefix(cell, ":")
		right := strings.HasSuffix(cell, ":")
		switch {
		case left && right:
			aligns[i] = "center"
		case right:
			aligns[i] = "right"
		case left:
			aligns[i] = "left"
		}
	}
	return aligns
}

func alignAttr(aligns []string, i int) string {
	if i < len(aligns) && aligns[i] != "" {
		return ` style="text-align:` + aligns[i] + `"`
	}
	return ""
}

func (r *MarkdownRenderer) renderTable(lines []string) string {
	header := splitTableRow(lines[0])
	aligns := parseTableAlignments(lines[1])
	var b strings.Builder
	b.WriteString("<table>\n<thead><tr>")
	for i, cell := range header {
		b.WriteString("<th")
		b.WriteString(alignAttr(aligns, i))
		b.WriteString(">")
		b.WriteString(r.renderInline(cell))
		b.WriteString("</th>")
	}
	b.WriteString("</tr></thead>\n<tbody>\n")
	for _, line := range lines[2:] {
		cells := splitTableRow(line)
		if len(cells) == 0 {
			continue
		}
		b.WriteString("<tr>")
		for i, cell := range cells {
			b.WriteString("<td")
			b.WriteString(alignAttr(aligns, i))
			b.WriteString(">")
			b.WriteString(r.renderInline(cell))
			b.WriteString("</td>")
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	return b.String()
}

func (r *MarkdownRenderer) uniqueHeadingID(base string) string {
	if base == "" {
		base = "section"
	}
	count := r.headingIDs[base]
	r.headingIDs[base] = count + 1
	if count == 0 {
		return base
	}
	return base + "-" + strconv.Itoa(count+1)
}

func (r *MarkdownRenderer) renderInline(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		switch {
		case strings.HasPrefix(s[i:], "`"):
			if end := strings.Index(s[i+1:], "`"); end >= 0 {
				code := s[i+1 : i+1+end]
				b.WriteString("<code>")
				b.WriteString(html.EscapeString(code))
				b.WriteString("</code>")
				i += end + 2
				continue
			}
		case strings.HasPrefix(s[i:], "![["):
			if end := strings.Index(s[i+3:], "]]"); end >= 0 {
				raw := s[i+3 : i+3+end]
				b.WriteString(r.renderWiki(raw, true))
				i += end + 5
				continue
			}
		case strings.HasPrefix(s[i:], "[["):
			if end := strings.Index(s[i+2:], "]]"); end >= 0 {
				raw := s[i+2 : i+2+end]
				b.WriteString(r.renderWiki(raw, false))
				i += end + 4
				continue
			}
		case strings.HasPrefix(s[i:], "!["):
			if alt, dest, consumed, ok := parseMarkdownLink(s[i+1:]); ok {
				src := r.resolveMarkdownAsset(dest)
				b.WriteString(`<img src="`)
				b.WriteString(html.EscapeString(src))
				b.WriteString(`" alt="`)
				b.WriteString(html.EscapeString(stripInlineMarkdown(alt)))
				b.WriteString(`">`)
				i += consumed + 1
				continue
			}
		case strings.HasPrefix(s[i:], "["):
			if label, dest, consumed, ok := parseMarkdownLink(s[i:]); ok {
				href := r.resolveMarkdownHref(dest)
				b.WriteString(`<a href="`)
				b.WriteString(html.EscapeString(href))
				b.WriteString(`">`)
				b.WriteString(r.renderInline(label))
				b.WriteString(`</a>`)
				i += consumed
				continue
			}
		case strings.HasPrefix(s[i:], "https://") || strings.HasPrefix(s[i:], "http://"):
			href, consumed := consumeBareURL(s[i:])
			b.WriteString(`<a href="`)
			b.WriteString(html.EscapeString(sanitizeHref(href)))
			b.WriteString(`">`)
			b.WriteString(html.EscapeString(href))
			b.WriteString(`</a>`)
			i += consumed
			continue
		case strings.HasPrefix(s[i:], "**"):
			if end := strings.Index(s[i+2:], "**"); end >= 0 {
				b.WriteString("<strong>")
				b.WriteString(r.renderInline(s[i+2 : i+2+end]))
				b.WriteString("</strong>")
				i += end + 4
				continue
			}
		case strings.HasPrefix(s[i:], "__"):
			if end := strings.Index(s[i+2:], "__"); end >= 0 {
				b.WriteString("<strong>")
				b.WriteString(r.renderInline(s[i+2 : i+2+end]))
				b.WriteString("</strong>")
				i += end + 4
				continue
			}
		case strings.HasPrefix(s[i:], "*"):
			if end := strings.Index(s[i+1:], "*"); end >= 0 {
				b.WriteString("<em>")
				b.WriteString(r.renderInline(s[i+1 : i+1+end]))
				b.WriteString("</em>")
				i += end + 2
				continue
			}
		}

		rn, size := utf8.DecodeRuneInString(s[i:])
		if rn == utf8.RuneError && size == 0 {
			break
		}
		b.WriteString(html.EscapeString(string(rn)))
		i += size
	}
	return b.String()
}

func consumeBareURL(s string) (string, int) {
	end := 0
	for end < len(s) {
		r, size := utf8.DecodeRuneInString(s[end:])
		if unicode.IsSpace(r) || r == '<' || r == '"' {
			break
		}
		end += size
	}
	for end > 0 && strings.ContainsRune(".,;:", rune(s[end-1])) {
		end--
	}
	return s[:end], end
}

func parseMarkdownLink(s string) (string, string, int, bool) {
	if !strings.HasPrefix(s, "[") {
		return "", "", 0, false
	}
	closeLabel := strings.Index(s, "](")
	if closeLabel < 0 {
		return "", "", 0, false
	}
	closeDest := strings.Index(s[closeLabel+2:], ")")
	if closeDest < 0 {
		return "", "", 0, false
	}
	label := s[1:closeLabel]
	dest := s[closeLabel+2 : closeLabel+2+closeDest]
	return label, dest, closeLabel + 2 + closeDest + 1, true
}

func (r *MarkdownRenderer) renderWiki(raw string, embed bool) string {
	target, alias := splitAlias(raw)
	target, heading := splitHeading(target)
	display := alias
	if display == "" {
		display = heading
	}
	if display == "" {
		display = path.Base(strings.TrimSuffix(target, ".md"))
	}
	if display == "" {
		display = raw
	}

	if embed || looksLikeAsset(target) {
		if asset, ok := r.resolveAsset(target); ok {
			href := relativeURL(r.current, escapeURLPath(path.Join(r.vault.Config.sourcePrefix(), asset)))
			if isImage(asset) && embed {
				return `<img src="` + html.EscapeString(href) + `" alt="` + html.EscapeString(display) + `">`
			}
			return `<a href="` + html.EscapeString(href) + `">` + html.EscapeString(display) + `</a>`
		}
		if embed {
			if page := r.resolveNote(target); page != nil {
				r.outgoing[page.PageRel] = true
				return `<a class="embed-note" href="` + html.EscapeString(relativeURL(r.current, page.URL)) + `">` + html.EscapeString(display) + `</a>`
			}
		}
	}

	if page := r.resolveNote(target); page != nil {
		r.outgoing[page.PageRel] = true
		href := relativeURL(r.current, page.URL)
		if heading != "" {
			href += "#" + url.PathEscape(slugify(heading))
		}
		return `<a class="wiki-link" href="` + html.EscapeString(href) + `">` + html.EscapeString(display) + `</a>`
	}

	r.warn("unresolved wikilink [[" + raw + "]]")
	return `<span class="broken-link">[[` + html.EscapeString(raw) + `]]</span>`
}

func (r *MarkdownRenderer) warn(message string) {
	*r.warnings = append(*r.warnings, r.current.SourceRel+": "+message)
}

func stripInlineMarkdown(s string) string {
	wikiRE := regexp.MustCompile(`!?\[\[([^|\]#]+)(?:#[^|\]]*)?(?:\|([^\]]+))?\]\]`)
	s = wikiRE.ReplaceAllStringFunc(s, func(match string) string {
		m := wikiRE.FindStringSubmatch(match)
		if len(m) > 2 && m[2] != "" {
			return m[2]
		}
		return path.Base(m[1])
	})
	linkRE := regexp.MustCompile(`!?\[([^\]]*)\]\([^)]+\)`)
	s = linkRE.ReplaceAllString(s, "$1")
	for _, marker := range []string{"**", "__", "*", "`"} {
		s = strings.ReplaceAll(s, marker, "")
	}
	return strings.TrimSpace(s)
}

func plainText(markdown string) string {
	text := stripInlineMarkdown(markdown)
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence || trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, ">") {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
		}
		out = append(out, strings.Trim(trimmed, "#-* "))
	}
	return strings.Join(strings.Fields(strings.Join(out, " ")), " ")
}

func extractTags(markdown string) []string {
	re := regexp.MustCompile(`(^|\s)#([A-Za-z0-9_/-]+)`)
	matches := re.FindAllStringSubmatch(markdown, -1)
	seen := map[string]bool{}
	var tags []string
	for _, match := range matches {
		tag := match[2]
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

func excerpt(text string, limit int) string {
	text = strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	if len(text) <= limit {
		return text
	}
	return strings.TrimSpace(text[:limit]) + "..."
}

func slugify(s string) string {
	s = strings.ToLower(stripInlineMarkdown(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
