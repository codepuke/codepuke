package web

import (
	stdhtml "html"
	"regexp"

	"github.com/codepuke/codepuke/ui"
)

// h2Re matches the heading-anchor contract the render pipeline emits
// (design-system.md 4f): an h2 with a text-derived id and a leading
// offset-anchor chip.
var (
	h2Re  = regexp.MustCompile(`(?s)<h2 id="([^"]+)"><a class="offset-anchor"[^>]*>([^<]+)</a>\s*(.*?)</h2>`)
	tagRe = regexp.MustCompile(`<[^>]*>`)
)

// extractSections pulls the h2 outline out of stored body HTML for the docs
// nav sub-list. This reads the already-rendered HTML; no markdown runs at
// request time.
func extractSections(bodyHTML string) []ui.DocSection {
	var sections []ui.DocSection
	for _, m := range h2Re.FindAllStringSubmatch(bodyHTML, -1) {
		title := stdhtml.UnescapeString(tagRe.ReplaceAllString(m[3], ""))
		sections = append(sections, ui.DocSection{
			ID:    m[1],
			Label: m[2],
			Title: title,
		})
	}
	return sections
}
