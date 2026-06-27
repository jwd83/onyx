# Theme Overrides

Onyx ships with an embedded default theme. A site can override any of these files
inside its configured theme folder:

```text
theme/
  style.css   # copied to public/onyx.css
  onyx.js     # copied to public/onyx.js
  page.html   # template for notes
  home.html   # template for the homepage
  static/     # copied to public/theme/
```

Missing theme files fall back to the embedded defaults. `home.html` falls back
to the embedded page template when no custom homepage template exists.

Templates are Go `html/template` files. The default data includes the site title,
current page, rendered nav, backlinks, feature toggles, generated marker, and
relative URLs for the homepage, CSS, and JavaScript assets.

## How this site uses a theme

This site overrides only `theme/home.html` to give the landing page you arrive on
its hero and install command, while every documentation page uses Onyx's embedded
default template. That custom `home.html` also carries the Go vanity-import
`<meta>` tags, so `go install onyx.jwd.me/onyx@latest` keeps resolving even though
the homepage is now a generated Onyx page rather than a hand-written redirect.

When you build a site, deploy it as described in [[deploying|Deploying]].
