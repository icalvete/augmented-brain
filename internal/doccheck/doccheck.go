// Package doccheck reads the reference pages of the guide (docs/guide/commands.md,
// mcp-tools.md, configuration.md and their Spanish twins) for the tests that check they
// still cover the program: every command and flag, every MCP tool and parameter, every
// setting. The prose is written by hand; these tests only say what it leaves out, and
// what it documents that no longer exists. Only tests use this package.
package doccheck

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// DirEnv, when set, is the folder the pages are read from instead of docs/guide, to try
// the checks on pages written elsewhere.
const DirEnv = "AB_DOCS_DIR"

// Languages are the suffixes of each page's versions: commands.md, commands.es.md.
var Languages = []string{"", ".es"}

// Files are the file names of the page called name, one per language.
func Files(name string) []string {
	var out []string
	for _, l := range Languages {
		out = append(out, name+l+".md")
	}
	return out
}

// Dir is the folder of the reference pages: $AB_DOCS_DIR, or docs/guide of the
// repository the test runs in (found by going up to go.mod).
func Dir() (string, error) {
	if d := os.Getenv(DirEnv); d != "" {
		return d, nil
	}
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return filepath.Join(d, "docs", "guide"), nil
		}
		up := filepath.Dir(d)
		if up == d {
			return "", errors.New("no go.mod above the working folder")
		}
		d = up
	}
}

// Page is a reference page, cut into its sections.
type Page struct {
	File string
	// Sections are the `## ` sections in the order of the page: the heading's text
	// (without the hashes), and the inline code of the section, subsections (### and
	// deeper) included.
	Sections []Section
	// Code is the inline code of the whole page.
	Code []string
}

// Section is one `## ` heading and the inline code under it.
type Section struct {
	Heading string
	Code    []string
}

// Read reads file of dir.
func Read(dir, file string) (*Page, error) {
	b, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return nil, err
	}
	return Parse(file, string(b)), nil
}

// Parse cuts text into sections. Fenced code blocks are skipped: a heading or a flag
// inside one does not count, only inline code (`--flag`) does.
func Parse(file, text string) *Page {
	p := &Page{File: file}
	var cur *Section
	fence := ""
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if f := fenceOf(trimmed); f != "" {
			fence = f
			continue
		}
		if strings.HasPrefix(line, "## ") {
			h := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			p.Sections = append(p.Sections, Section{Heading: h})
			cur = &p.Sections[len(p.Sections)-1]
			if strings.HasPrefix(h, "`") {
				// The heading is code itself (## `task_new`): it names the section, it
				// does not document anything inside it.
				continue
			}
		}
		code := Spans(line)
		p.Code = append(p.Code, code...)
		if cur != nil {
			cur.Code = append(cur.Code, code...)
		}
	}
	return p
}

// fenceOf is the fence a line opens (``` or ~~~, three or more), or "".
func fenceOf(line string) string {
	for _, c := range []string{"`", "~"} {
		n := 0
		for n < len(line) && line[n:n+1] == c {
			n++
		}
		if n >= 3 {
			return strings.Repeat(c, n)
		}
	}
	return ""
}

// Spans are the inline code spans of a line, as Markdown reads them: a run of
// backticks opens one, and the next run of the same length closes it.
func Spans(line string) []string {
	var out []string
	for i := 0; i < len(line); {
		if line[i] != '`' {
			i++
			continue
		}
		n := run(line, i)
		closed := false
		for j := i + n; j < len(line); {
			if line[j] != '`' {
				j++
				continue
			}
			m := run(line, j)
			if m == n {
				out = append(out, strings.TrimSpace(line[i+n:j]))
				i, closed = j+m, true
				break
			}
			j += m
		}
		if !closed {
			i += n
		}
	}
	return out
}

func run(s string, i int) int {
	n := 0
	for i+n < len(s) && s[i+n] == '`' {
		n++
	}
	return n
}

// Mentions says whether token is in one of the code spans as a whole word: `--all`
// is in `--all` and in `backup --all`, `stop` is not in `stopped`, `id` is not in
// `task_id`.
func Mentions(code []string, token string) bool {
	for _, c := range code {
		for i := 0; ; {
			j := strings.Index(c[i:], token)
			if j < 0 {
				break
			}
			start, end := i+j, i+j+len(token)
			if (start == 0 || !word(c[start-1])) && (end == len(c) || !word(c[end])) {
				return true
			}
			i = start + 1
		}
	}
	return false
}

// word says whether b can be part of a name: a flag, a tool, a parameter, a setting.
func word(b byte) bool {
	return b == '_' || b == '-' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

// Check compares the sections of p with want: for each heading, the names its section
// must mention in inline code. ours says which headings are of this kind (a command, a
// tool): one of those that is not a key of want documents something that no longer
// exists. It returns every gap, the sections in the order given by order (want's keys).
func Check(p *Page, order []string, want map[string][]string, ours func(heading string) bool, kind string) []string {
	code := map[string][]string{}
	seen := map[string]bool{}
	for _, s := range p.Sections {
		if !ours(s.Heading) {
			continue
		}
		seen[s.Heading] = true
		code[s.Heading] = append(code[s.Heading], s.Code...)
	}
	var gaps []string
	for _, h := range order {
		if !seen[h] {
			gaps = append(gaps, p.File+": no section \"## "+h+"\"")
			continue
		}
		for _, w := range want[h] {
			if !Mentions(code[h], w) {
				gaps = append(gaps, p.File+": section \"## "+h+"\" lacks `"+w+"`")
			}
		}
	}
	for _, s := range p.Sections {
		if _, known := want[s.Heading]; ours(s.Heading) && !known && seen[s.Heading] {
			gaps = append(gaps, p.File+": section \"## "+s.Heading+"\" documents a "+kind+" that does not exist")
			seen[s.Heading] = false // once
		}
	}
	return gaps
}
