package content

import "github.com/microcosm-cc/bluemonday"

// buildPolicy is the sanitizer for rendered markdown. It starts from the UGC
// policy and adds exactly what the design system's markup contracts emit:
// classed chroma spans, the code-tabs and scroll-box custom elements, figures
// (hexdump strips), goldmark footnote roles, and heading anchor ids. Mermaid
// SVG never passes through here; it is inlined after sanitization because the
// sidecar output is trusted.
func buildPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowTables()

	p.AllowElements(
		"section", "figure", "figcaption",
		"code-tabs", "scroll-box",
		"sup", "sub", "hr",
	)
	// Allowed elements are still dropped when they carry no attributes
	// unless listed here; scroll-box in particular is usually bare.
	p.AllowNoAttrs().OnElements(
		"section", "figure", "figcaption",
		"code-tabs", "scroll-box",
		"sup", "sub", "hr",
	)

	// All content is written by trusted authors or synced from our own
	// repos; forcing rel=nofollow would also mangle the internal anchor
	// chips and footnote links.
	p.RequireNoFollowOnLinks(false)

	p.AllowAttrs("class").Globally()
	p.AllowAttrs("id").Globally()
	p.AllowAttrs("role").OnElements("a", "div", "ol", "li", "sup")

	p.AllowAttrs("data-topic", "data-active", "data-default").OnElements("code-tabs")
	p.AllowAttrs("data-lang", "hidden").OnElements("section")
	p.AllowAttrs("data-overflow").OnElements("scroll-box")
	p.AllowAttrs("tabindex").OnElements("pre")

	return p
}
