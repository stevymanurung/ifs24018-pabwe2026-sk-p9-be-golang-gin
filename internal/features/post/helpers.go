package post

import (
	"regexp"
	"strings"
	"unicode"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonSlug.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
func ReadingTime(content string) int {
	words := 0
	in := false
	for _, r := range content {
		if unicode.IsSpace(r) {
			in = false
		} else if !in {
			words++
			in = true
		}
	}
	n := (words + 199) / 200
	if n < 1 {
		return 1
	}
	return n
}
