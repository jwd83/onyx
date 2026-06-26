package main

import (
	"html"
	"sort"
	"strings"
)

type NavNode struct {
	Name     string
	Path     string
	Page     *Page
	Children map[string]*NavNode
}

func renderNav(vault *Vault, current *Page) string {
	root := &NavNode{Children: map[string]*NavNode{}}
	for _, page := range vault.Notes {
		if page.IsHome {
			continue
		}
		insertNav(root, page)
	}

	var b strings.Builder
	b.WriteString(`<ul class="nav-tree">`)
	b.WriteString(`<li><a`)
	if current.IsHome {
		b.WriteString(` aria-current="page"`)
	}
	b.WriteString(` href="`)
	b.WriteString(html.EscapeString(relativeURL(current, "")))
	b.WriteString(`">Home</a></li>`)
	writeNavChildren(&b, root, current)
	b.WriteString(`</ul>`)
	return b.String()
}

func insertNav(root *NavNode, page *Page) {
	segments := strings.Split(page.PageRel, "/")
	node := root
	prefix := ""
	for _, segment := range segments {
		key := strings.ToLower(segment)
		if prefix == "" {
			prefix = key
		} else {
			prefix += "/" + key
		}
		if node.Children == nil {
			node.Children = map[string]*NavNode{}
		}
		child := node.Children[key]
		if child == nil {
			child = &NavNode{Name: segment, Path: prefix, Children: map[string]*NavNode{}}
			node.Children[key] = child
		}
		node = child
	}
	node.Page = page
	node.Name = page.Title
}

func writeNavChildren(b *strings.Builder, node *NavNode, current *Page) {
	children := make([]*NavNode, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool {
		iFolder := len(children[i].Children) > 0
		jFolder := len(children[j].Children) > 0
		if iFolder != jFolder {
			return iFolder
		}
		return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
	})

	currentPath := strings.ToLower(current.PageRel)
	for _, child := range children {
		b.WriteString("<li>")
		if len(child.Children) > 0 {
			onPath := currentPath == child.Path || strings.HasPrefix(currentPath, child.Path+"/")
			b.WriteString("<details")
			if onPath {
				b.WriteString(" open")
			}
			b.WriteString("><summary>")
			writeNavLabel(b, child, current)
			b.WriteString("</summary><ul>")
			writeNavChildren(b, child, current)
			b.WriteString("</ul></details>")
		} else {
			writeNavLabel(b, child, current)
		}
		b.WriteString("</li>")
	}
}

func writeNavLabel(b *strings.Builder, node *NavNode, current *Page) {
	if node.Page == nil {
		b.WriteString(`<span>`)
		b.WriteString(html.EscapeString(node.Name))
		b.WriteString(`</span>`)
		return
	}
	b.WriteString(`<a`)
	if node.Page == current {
		b.WriteString(` aria-current="page"`)
	}
	b.WriteString(` href="`)
	b.WriteString(html.EscapeString(relativeURL(current, node.Page.URL)))
	b.WriteString(`">`)
	b.WriteString(html.EscapeString(node.Name))
	b.WriteString(`</a>`)
}
