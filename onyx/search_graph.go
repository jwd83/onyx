package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

type SearchItem struct {
	Title    string   `json:"title"`
	URL      string   `json:"url"`
	Path     string   `json:"path"`
	Excerpt  string   `json:"excerpt"`
	Headings []string `json:"headings,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

type GraphNode struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Degree int    `json:"degree"`
}

type GraphLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Links []GraphLink `json:"links"`
}

func writeSearchIndex(vault *Vault) error {
	items := make([]SearchItem, 0, len(vault.Notes))
	for _, page := range vault.Notes {
		items = append(items, SearchItem{
			Title:    page.Title,
			URL:      page.URL,
			Path:     page.SourceRel,
			Excerpt:  page.Excerpt,
			Headings: page.Headings,
			Tags:     page.Tags,
		})
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(vault.Config.Root, "public", "search-index.json"), data, 0o644)
}

// writeGraph emits public/graph.json: one node per published note and one link
// per resolved outgoing wikilink, so the browser can draw the knowledge graph.
func writeGraph(vault *Vault) error {
	indexByRel := map[string]int{}
	nodes := make([]GraphNode, 0, len(vault.Notes))
	for _, page := range vault.Notes {
		if page.Generated {
			continue
		}
		indexByRel[page.PageRel] = len(nodes)
		nodes = append(nodes, GraphNode{
			ID:    page.PageRel,
			Title: page.Title,
			URL:   page.URL,
		})
	}

	links := make([]GraphLink, 0)
	seen := map[string]bool{}
	for _, page := range vault.Notes {
		srcIdx, ok := indexByRel[page.PageRel]
		if !ok {
			continue
		}
		targets := make([]string, 0, len(page.Outgoing))
		for target := range page.Outgoing {
			targets = append(targets, target)
		}
		sort.Strings(targets)
		for _, target := range targets {
			tgtIdx, ok := indexByRel[target]
			if !ok || target == page.PageRel {
				continue
			}
			key := page.PageRel + "\x00" + target
			if seen[key] {
				continue
			}
			seen[key] = true
			links = append(links, GraphLink{Source: page.PageRel, Target: target})
			nodes[srcIdx].Degree++
			nodes[tgtIdx].Degree++
		}
	}

	data, err := json.MarshalIndent(GraphData{Nodes: nodes, Links: links}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(vault.Config.Root, "public", "graph.json"), data, 0o644)
}
