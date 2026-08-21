package content

import (
	"fmt"
	"strings"
)

// Slugify lowercases s and reduces it to hyphen-separated ASCII alphanumeric
// runs: every other character collapses into a single hyphen, with no leading
// or trailing hyphen. Non-ASCII letters are dropped rather than transliterated,
// so a slug is always URL-safe without escaping. The result can be empty.
func Slugify(s string) string {
	var b strings.Builder
	pendingHyphen := false
	for _, r := range strings.ToLower(s) {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if !isAlnum {
			pendingHyphen = b.Len() > 0
			continue
		}
		if pendingHyphen {
			b.WriteByte('-')
			pendingHyphen = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

// dedupe returns base if unused, otherwise base-2, base-3, and so on, marking
// the result in seen. An empty base becomes "section" so every heading gets a
// non-empty id.
func dedupe(seen map[string]bool, base string) string {
	if base == "" {
		base = "section"
	}
	if !seen[base] {
		seen[base] = true
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !seen[candidate] {
			seen[candidate] = true
			return candidate
		}
	}
}
