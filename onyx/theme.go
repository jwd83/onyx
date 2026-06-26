package main

import _ "embed"

// Default theme assets, embedded from theme/default/* at build time. These are
// the fallback template, styles, and script used when a site provides no
// override of its own. Edit the real files under theme/default/ — they compile
// into the single binary via go:embed, so the no-dependency, one-binary
// contract is unchanged.

//go:embed theme/default/page.html
var defaultPageTemplate string

//go:embed theme/default/style.css
var defaultCSS string

//go:embed theme/default/onyx.js
var defaultJS string
