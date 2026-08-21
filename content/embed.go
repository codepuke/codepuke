// Package contentfs embeds the synced content tree that cmd/sync rebuilds:
// example snippets, docs markdown, and the manifest. The server reads it via
// io/fs so tests can substitute any directory.
package contentfs

import "embed"

//go:embed all:examples all:docs manifest.json
var FS embed.FS
