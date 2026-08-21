package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// snippet is one region extracted from a source file.
type snippet struct {
	Topic string
	Code  string
}

var snippetTopicRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

type markerKind int

const (
	markerNone markerKind = iota
	markerStart
	markerEnd
)

// extractSnippets scans source for regions delimited by marker comments:
//
//	// snippet:start encode-struct
//	...
//	// snippet:end
//
// prefix is the language's line-comment token (// or #). Regions may nest; a
// bare snippet:end closes the most recently opened region, and snippet:end
// with a topic closes that specific one. Marker lines never appear in any
// captured region. Captured code is dedented by the common leading
// whitespace of its non-blank lines and trimmed of surrounding blank lines.
// A topic may appear once per file; unclosed or unmatched markers are
// errors, as is anything after "snippet:" that is not a well-formed marker,
// so typos surface at sync time instead of silently dropping a region.
func extractSnippets(source []byte, prefix string) ([]snippet, error) {
	type region struct {
		topic string
		lines []string
	}
	var (
		out  []snippet
		open []*region
		seen = map[string]bool{}
	)

	for i, raw := range strings.Split(string(source), "\n") {
		line := strings.TrimSuffix(raw, "\r")
		kind, topic, err := parseMarker(line, prefix)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		switch kind {
		case markerNone:
			for _, r := range open {
				r.lines = append(r.lines, line)
			}
		case markerStart:
			if seen[topic] {
				return nil, fmt.Errorf("line %d: duplicate snippet topic %q", i+1, topic)
			}
			seen[topic] = true
			open = append(open, &region{topic: topic})
		case markerEnd:
			idx := len(open) - 1
			if topic != "" {
				idx = -1
				for j, o := range slices.Backward(open) {
					if o.topic == topic {
						idx = j
						break
					}
				}
			}
			if idx < 0 {
				return nil, fmt.Errorf("line %d: snippet:end without a matching snippet:start", i+1)
			}
			r := open[idx]
			open = slices.Delete(open, idx, idx+1)
			out = append(out, snippet{Topic: r.topic, Code: dedent(r.lines)})
		}
	}
	if len(open) > 0 {
		return nil, fmt.Errorf("snippet %q is never closed", open[len(open)-1].topic)
	}
	return out, nil
}

// parseMarker classifies one line. Lines that are not comments, or comments
// that do not begin with "snippet:", are content.
func parseMarker(line, prefix string) (markerKind, string, error) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, prefix) {
		return markerNone, "", nil
	}
	comment := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
	if !strings.HasPrefix(comment, "snippet:") {
		return markerNone, "", nil
	}

	verb, arg, _ := strings.Cut(comment, " ")
	arg = strings.TrimSpace(arg)
	switch verb {
	case "snippet:start":
		if arg == "" {
			return markerNone, "", fmt.Errorf("snippet:start requires a topic")
		}
		if !snippetTopicRe.MatchString(arg) {
			return markerNone, "", fmt.Errorf("invalid snippet topic %q", arg)
		}
		return markerStart, arg, nil
	case "snippet:end":
		if arg != "" && !snippetTopicRe.MatchString(arg) {
			return markerNone, "", fmt.Errorf("invalid snippet topic %q", arg)
		}
		return markerEnd, arg, nil
	default:
		return markerNone, "", fmt.Errorf("unknown snippet marker %q", verb)
	}
}

// dedent trims surrounding blank lines, strips the longest whitespace prefix
// shared by every non-blank line, and returns the code with a trailing
// newline. An empty region returns "".
func dedent(lines []string) string {
	start, end := 0, len(lines)
	for start < end && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	lines = lines[start:end]
	if len(lines) == 0 {
		return ""
	}

	indent := ""
	first := true
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		ws := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		if first {
			indent = ws
			first = false
			continue
		}
		indent = commonPrefix(indent, ws)
	}

	var b strings.Builder
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			b.WriteByte('\n')
			continue
		}
		b.WriteString(strings.TrimPrefix(line, indent))
		b.WriteByte('\n')
	}
	return b.String()
}

func commonPrefix(a, b string) string {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	return a[:n]
}
