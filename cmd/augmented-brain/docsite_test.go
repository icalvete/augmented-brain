package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// The documentation as people read it: every link and anchor leads somewhere, every
// guide exists in both languages with the same shape, sits in the site's sidebar, and
// every example command is one the program has, with flags it reads. What the guides
// claim is checked by reading them; this keeps what a machine can check from rotting.

const docRoot = "../.."

// docPages are the Markdown pages people read: the READMEs, the guides and the spec.
func docPages(t *testing.T) []string {
	t.Helper()
	pages := []string{filepath.Join(docRoot, "README.md"), filepath.Join(docRoot, "README.es.md"),
		filepath.Join(docRoot, "docs", "spec", "memory-format.md")}
	guides, err := filepath.Glob(filepath.Join(docRoot, "docs", "guide", "*.md"))
	if err != nil || len(guides) == 0 {
		t.Fatalf("no guides: %v", err)
	}
	return append(pages, guides...)
}

var (
	fenceRE  = regexp.MustCompile("(?ms)^```.*?^```")
	linkRE   = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	headRE   = regexp.MustCompile(`(?m)^(#{1,6}) (.*)$`)
	anchorRE = regexp.MustCompile(`<a id="([\w\-]+)"></a>`)
	tagRE    = regexp.MustCompile(`<[^>]+>`)
)

// slug is a heading's anchor as the site (scripts/build-guides.py) and GitHub make it:
// lowercase, no punctuation, spaces as dashes, accented letters kept; -1, -2 when repeated.
func slug(text string, seen map[string]bool) string {
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

// anchors are the targets a page offers: its headings' slugs and its <a id> marks.
func anchors(text string) map[string]bool {
	out, seen := map[string]bool{}, map[string]bool{}
	body := fenceRE.ReplaceAllString(text, "")
	for _, m := range headRE.FindAllStringSubmatch(body, -1) {
		out[slug(strings.ReplaceAll(m[2], "`", ""), seen)] = true
	}
	for _, m := range anchorRE.FindAllStringSubmatch(body, -1) {
		out[m[1]] = true
	}
	return out
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Every relative link leads to a file that is there, and every #anchor to a heading or
// mark that page has.
func TestDocLinks(t *testing.T) {
	cache := map[string]map[string]bool{}
	for _, page := range docPages(t) {
		text := fenceRE.ReplaceAllString(read(t, page), "")
		for _, m := range linkRE.FindAllStringSubmatch(text, -1) {
			url := m[1]
			if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "mailto:") {
				continue
			}
			path, frag, _ := strings.Cut(url, "#")
			// A Spanish page sends its reader to the Spanish guide, when there is one.
			// Its own English twin ("Read in English") is the one link meant to cross.
			own := strings.TrimSuffix(filepath.Base(page), ".es.md") + ".md"
			if strings.HasSuffix(page, ".es.md") && strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".es.md") && path != own {
				twin := filepath.Join(filepath.Dir(page), filepath.FromSlash(strings.TrimSuffix(path, ".md")+".es.md"))
				if _, err := os.Stat(twin); err == nil {
					t.Errorf("%s: link to %s, the English guide; the Spanish one is %s", rel(page), url, filepath.Base(twin))
				}
			}
			target := page
			if path != "" {
				target = filepath.Join(filepath.Dir(page), filepath.FromSlash(path))
				if _, err := os.Stat(target); err != nil {
					t.Errorf("%s: link to %s, which is not there", rel(page), url)
					continue
				}
			}
			if frag == "" || !strings.HasSuffix(target, ".md") {
				continue
			}
			if cache[target] == nil {
				cache[target] = anchors(read(t, target))
			}
			if !cache[target][frag] {
				t.Errorf("%s: link to %s, whose page has no #%s", rel(page), url, frag)
			}
		}
	}
}

// Every guide is in both languages, with the same headings and the same number of
// examples, so a reader of either gets the same guide.
func TestGuidesInBothLanguages(t *testing.T) {
	guides, _ := filepath.Glob(filepath.Join(docRoot, "docs", "guide", "*.md"))
	for _, en := range guides {
		if strings.HasSuffix(en, ".es.md") {
			continue
		}
		es := strings.TrimSuffix(en, ".md") + ".es.md"
		if _, err := os.Stat(es); err != nil {
			t.Errorf("%s has no Spanish twin", rel(en))
			continue
		}
		a, b := read(t, en), read(t, es)
		ha, hb := headRE.FindAllStringSubmatch(fenceRE.ReplaceAllString(a, ""), -1), headRE.FindAllStringSubmatch(fenceRE.ReplaceAllString(b, ""), -1)
		if len(ha) != len(hb) {
			t.Errorf("%s: %d headings in English, %d in Spanish", rel(en), len(ha), len(hb))
		} else {
			for i := range ha {
				if len(ha[i][1]) != len(hb[i][1]) {
					t.Errorf("%s: heading %d is level %d in English and %d in Spanish (%q / %q)", rel(en), i+1, len(ha[i][1]), len(hb[i][1]), ha[i][2], hb[i][2])
				}
			}
		}
		if ca, cb := len(fenceRE.FindAllString(a, -1)), len(fenceRE.FindAllString(b, -1)); ca != cb {
			t.Errorf("%s: %d examples in English, %d in Spanish", rel(en), ca, cb)
		}
	}
}

// Every guide is in the site's sidebar (GROUPS in scripts/build-guides.py), or nobody
// finds it.
func TestGuidesInTheSidebar(t *testing.T) {
	builder := read(t, filepath.Join(docRoot, "scripts", "build-guides.py"))
	guides, _ := filepath.Glob(filepath.Join(docRoot, "docs", "guide", "*.md"))
	for _, g := range guides {
		if strings.HasSuffix(g, ".es.md") {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(g), ".md")
		if !strings.Contains(builder, `slug="`+name+`"`) {
			t.Errorf("%s is not in GROUPS of scripts/build-guides.py", rel(g))
		}
	}
}

// Every `augmented-brain …` line of an example is a command the program has, with flags
// that command reads.
func TestDocExamples(t *testing.T) {
	type known struct{ flags []string }
	cmds := map[string]known{}
	for _, c := range commands {
		var flags []string
		if c.flags != nil {
			flags = c.flags()
		}
		cmds[c.name] = known{flags}
		for _, a := range c.aliases {
			cmds[a] = known{flags}
		}
	}
	checked := 0
	defer func() {
		if checked < 30 {
			t.Errorf("only %d example lines checked: the reader of examples is broken", checked)
		}
	}()
	for _, page := range docPages(t) {
		for _, block := range fenceRE.FindAllString(read(t, page), -1) {
			for _, line := range strings.Split(block, "\n") {
				line = strings.TrimSpace(line)
				i := strings.Index(line, "augmented-brain ")
				if i < 0 || (i > 0 && !strings.HasSuffix(line[:i], "&& ") && !strings.HasSuffix(line[:i], "$ ")) {
					continue // not a command line that starts with the program
				}
				checked++
				words := strings.Fields(line[i+len("augmented-brain "):])
				name, n := "", 0
				for k := len(words); k > 0; k-- {
					if _, ok := cmds[strings.Join(words[:k], " ")]; ok {
						name, n = strings.Join(words[:k], " "), k
						break
					}
				}
				if name == "" {
					if len(words) > 0 && !strings.HasPrefix(words[0], "-") {
						t.Errorf("%s: example %q: no command %q", rel(page), line, words[0])
					}
					continue
				}
				for _, w := range words[n:] {
					if w == "&&" || w == "|" || w == ";" {
						break
					}
					if !strings.HasPrefix(w, "-") || w == "-" {
						continue
					}
					f := strings.TrimLeft(strings.SplitN(w, "=", 2)[0], "-")
					if f == "help" || f == "h" || f == "version" || f == "v" {
						continue
					}
					if !slices.Contains(cmds[name].flags, f) {
						t.Errorf("%s: example %q: %s has no --%s", rel(page), line, name, f)
					}
				}
			}
		}
	}
}

func rel(p string) string { r, _ := filepath.Rel(docRoot, p); return filepath.ToSlash(r) }

// Both READMEs link every guide, so whoever starts there can reach them all.
func TestReadmesLinkEveryGuide(t *testing.T) {
	guides, _ := filepath.Glob(filepath.Join(docRoot, "docs", "guide", "*.md"))
	for _, readme := range []string{"README.md", "README.es.md"} {
		text := read(t, filepath.Join(docRoot, readme))
		for _, g := range guides {
			if strings.HasSuffix(g, ".es.md") {
				continue
			}
			name := strings.TrimSuffix(filepath.Base(g), ".md")
			if !strings.Contains(text, "/guide/"+name+".") {
				t.Errorf("%s does not link the guide %s", readme, name)
			}
		}
	}
}
