// Package migrations embeds the goose SQL migrations so the server can
// apply them on boot without shipping loose files.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
