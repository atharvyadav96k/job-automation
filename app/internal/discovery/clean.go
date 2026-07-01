package discovery

import (
	"html"
	"regexp"
	"strings"
)

var (
	scriptOrStyle = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	tag           = regexp.MustCompile(`(?s)<[^>]+>`)
	whitespace    = regexp.MustCompile(`[ \t]+`)
	blankLines    = regexp.MustCompile(`\n{3,}`)
	blockClose    = regexp.MustCompile(`(?i)</(p|div|li|br|h[1-6])\s*>`)
)

// CleanHTML strips a job description down to plain text good enough for
// storage/search/prompting. It's a best-effort stripper, not a full HTML
// parser — job board descriptions are simple enough that this holds up.
func CleanHTML(raw string) string {
	s := scriptOrStyle.ReplaceAllString(raw, "")
	s = blockClose.ReplaceAllString(s, "\n")
	s = tag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = whitespace.ReplaceAllString(s, " ")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	s = strings.Join(lines, "\n")
	s = blankLines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
