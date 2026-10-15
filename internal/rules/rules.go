// Package rules holds the common rules that augmented brain loads at the start of
// every session, in every supported language. Each space can add its own in rules/.
package rules

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
)

//go:embed en es
var files embed.FS

// Common returns the common rules in lang (English if lang is not supported), in order.
func Common(lang string) string {
	if _, err := fs.Stat(files, lang); err != nil {
		lang = "en"
	}
	names, _ := fs.Glob(files, lang+"/*.md")
	sort.Strings(names)
	var b strings.Builder
	for i, n := range names {
		data, _ := files.ReadFile(n)
		if i > 0 {
			b.WriteString("\n")
		}
		b.Write(data)
	}
	return b.String()
}
