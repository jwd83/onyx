# Configuration

Configuration is optional. Without `onyx.ini`, Onyx builds every conventional
content folder it finds, uses `theme/` for overrides if present, and enables
search, graph data, and source links.

Example `onyx.ini`:

```ini
site_title = My Notes
source = docs, wiki
theme = theme
search = true
graph = true
show_source = true
```

## Supported keys

| Key | Default | Notes |
| --- | --- | --- |
| `site_title` | Home note title, then `Onyx` | Sets the site title shown by the template. |
| `source` | Existing conventional folders | Comma- or whitespace-separated list of content folders. Explicit sources must exist. |
| `theme` | `theme` | Relative path to theme overrides. Absolute paths are rejected. |
| `search` | `true` | Writes `public/search-index.json` and enables search UI data. |
| `graph` | `true` | Writes `public/graph.json` and enables graph UI data. |
| `show_source` | `true` | Shows links to original Markdown files when templates expose them. |

Unknown keys are ignored for compatibility. Legacy aliases are still accepted:
`build.search`, `build.graph`, and `publish_raw_markdown`. When both a modern key
and a legacy alias are present, the modern key wins.

## This site's configuration

For reference, the `onyx.ini` that builds the site you are reading is:

```ini
site_title = Onyx
source = docs
theme = theme
search = true
graph = true
show_source = true
```

The explicit `source = docs` keeps Onyx's internal `plans/` folder out of the
published site. To restyle the output, see [[themes|Theme Overrides]].
