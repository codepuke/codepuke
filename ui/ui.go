// Package ui holds the templ components, the hand-written stylesheet, and
// the two custom elements. ui/design-system.md is the source of truth for
// everything in here.
package ui

import (
	"embed"
	"fmt"
	"io/fs"
	"time"

	"github.com/codepuke/codepuke/internal/store"
)

//go:embed static
var staticFiles embed.FS

// StaticFS serves /static/*: the stylesheet and the JS inventory.
var StaticFS, _ = fs.Sub(staticFiles, "static")

// Offset renders a record row's positional offset (design-system.md 3.5):
// position times 0x40, page-local, pure presentation.
func Offset(index int) string {
	return fmt.Sprintf("0x%04x", index*0x40)
}

// NavNum renders a docs nav item number (design-system.md 4d): the
// zero-padded position of the page in the project's nav order.
func NavNum(index int) string {
	return fmt.Sprintf("%02d", index)
}

// Date renders every visible date; ISO keeps the columns aligned in mono.
func Date(t time.Time) string {
	return t.Format("2006-01-02")
}

// ProjectCard pairs a project row with its resolved open link: the first
// docs page when the project has docs, else the repository.
type ProjectCard struct {
	Project store.Project
	OpenURL string
}

// FamilyGroup is one family with its projects, query-ordered.
type FamilyGroup struct {
	Family store.Family
	Cards  []ProjectCard
}

// ArchiveYear is one year head plus its articles, newest first.
type ArchiveYear struct {
	Year     int
	Articles []store.ArticleSummary
}

// DocSection is one h2 of the active docs page, shown as a subdued anchor
// link under its nav item.
type DocSection struct {
	ID    string
	Label string // the 0x01-style chip text
	Title string
}

// DocsNavItem is one entry of a project's docs navigation. Sections is
// populated only on the active item.
type DocsNavItem struct {
	Title    string
	URL      string
	Active   bool
	Sections []DocSection
}

// DocsNavModel feeds the docs nav partial, which renders twice per page
// (design-system.md 4d).
type DocsNavModel struct {
	ProjectName string
	Version     string
	Items       []DocsNavItem
}

// AdminEditor feeds the article editor. The three states the admin board
// defines map onto it: Unsaved false is "saved SavedAt" with the save button
// disabled, Unsaved true is "unsaved changes" (the no-JS preview round trip
// lands here), and the in-flight state exists only in admin.js.
type AdminEditor struct {
	ID          int64
	Slug        string
	Title       string
	Author      string
	Date        string // publish date field, yyyy-mm-dd
	Tags        string // comma-separated category slugs
	BodyMD      string
	PreviewHTML string
	Published   bool
	Unsaved     bool
	SavedAt     string // updated_at clock time, UTC
	User        string
}
