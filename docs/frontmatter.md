# Frontmatter

Files may start with simple YAML-style frontmatter:

```markdown
---
title: My Note
publish: true
---
```

Recognized fields:

| Field | Effect |
| --- | --- |
| `title` | Sets the page title. Without it, Onyx uses the first `# Heading`, then the filename. |
| `publish: false` | Excludes the note from the build. |
| `draft: true` | Excludes the note from the build. |

Frontmatter parsing is intentionally small: `key: value` lines between opening
and closing `---` markers.

See [[configuration|Configuration]] for site-wide settings in `onyx.ini`.
