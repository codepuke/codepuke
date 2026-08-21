// Command sync pulls documentation and snippet regions from the sibling
// repositories listed in a sources file and rebuilds the committed content
// tree (content/examples, content/docs, content/manifest.json), which the
// server embeds. CI never needs the sibling repos checked out; only the
// operator running sync does.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	sourcesPath := flag.String("sources", "sources.json", "sources file listing repo paths and refs")
	outDir := flag.String("out", "content", "content tree to rebuild")
	flag.Parse()

	if err := run(*sourcesPath, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		os.Exit(1)
	}
}
