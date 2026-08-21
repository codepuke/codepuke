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

// DocsNavItem is one entry of a project's docs navigation.
type DocsNavItem struct {
	Title  string
	URL    string
	Active bool
}

// DocsNavModel feeds the docs nav partial, which renders twice per page
// (design-system.md 4d).
type DocsNavModel struct {
	ProjectName string
	Version     string
	Items       []DocsNavItem
}
