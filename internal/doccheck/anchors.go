package doccheck

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	fenceRE  = regexp.MustCompile("(?ms)^```.*?^```")
	headRE   = regexp.MustCompile(`(?m)^(#{1,6}) (.*)$`)
	anchorRE = regexp.MustCompile(`<a id="([\w\-]+)"></a>`)
	tagRE    = regexp.MustCompile(`<[^>]+>`)
)

// Slug is the anchor of a heading, as scripts/build-guides.py and GitHub make it:
// lowercase, no punctuation, spaces as dashes, accented letters kept. seen holds the
// page's anchors so far; a repeated one gets -1, -2…
func Slug(text string, seen map[string]bool) string {
	s := strings.ToLower(strings.TrimSpace(tagRE.ReplaceAllString(text, "")))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	base, out := b.String(), b.String()
	for n := 1; seen[out]; n++ {
		out = base + "-" + strconv.Itoa(n)
	}
	seen[out] = true
	return out
}

// Anchors are the targets a guide's Markdown offers, the ids its page on the website
// has: its headings' slugs and its <a id> marks. Code blocks are left out. Reading the
// Markdown, not the built page, the tests need no generated file (the website's pages
// are not committed).
func Anchors(text string) map[string]bool {
	out, seen := map[string]bool{}, map[string]bool{}
	body := fenceRE.ReplaceAllString(text, "")
	for _, m := range headRE.FindAllStringSubmatch(body, -1) {
		out[Slug(strings.ReplaceAll(m[2], "`", ""), seen)] = true
	}
	for _, m := range anchorRE.FindAllStringSubmatch(body, -1) {
		out[m[1]] = true
	}
	return out
}
