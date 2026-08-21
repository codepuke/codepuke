package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

var frontMatterLineRe = regexp.MustCompile(`^([A-Za-z0-9_-]+):[ \t]*(.*)$`)

// parseFrontMatter splits an optional leading front matter block from a
// markdown document:
//
//	---
//	title: Reading Streams
//	---
//
// Only flat "key: value" string pairs are supported. Documents without a
// leading "---" line pass through unchanged with an empty map. The returned
// body is always a suffix of data.
func parseFrontMatter(data []byte) (map[string]string, []byte, error) {
	meta := map[string]string{}
	if !bytes.HasPrefix(data, []byte("---\n")) && !bytes.HasPrefix(data, []byte("---\r\n")) {
		return meta, data, nil
	}

	offset := bytes.IndexByte(data, '\n') + 1
	for lineNo := 2; ; lineNo++ {
		if offset >= len(data) {
			return nil, nil, fmt.Errorf("front matter is never closed")
		}
		end := bytes.IndexByte(data[offset:], '\n')
		if end < 0 {
			end = len(data) - offset
		}
		line := strings.TrimSuffix(string(data[offset:offset+end]), "\r")
		offset += end + 1

		if line == "---" {
			if offset > len(data) {
				offset = len(data)
			}
			return meta, data[offset:], nil
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		m := frontMatterLineRe.FindStringSubmatch(line)
		if m == nil {
			return nil, nil, fmt.Errorf("front matter line %d is not \"key: value\": %q", lineNo, line)
		}
		meta[m[1]] = strings.TrimSpace(m[2])
	}
}
