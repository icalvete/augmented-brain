package panel

import (
	"fmt"
	"html"
	"html/template"
	"net/url"
	"regexp"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// The panel shows notes as HTML, not as the Markdown they are written in. This is a
// small renderer of the part of Markdown notes use: headings, paragraphs, lists (nested,
// with checkboxes), tables, quotes, code, bold, italics, links and [[wikilinks]]. Every
// piece of text is escaped first, and only these tags are written, so a note can never
// put its own HTML or scripts in the page. Links go only to http(s) addresses or within
// the panel; anything else stays as text.

// linker turns the id of a [[wikilink]] into the address of its note, and says whether
// the note exists.
type linker func(id string) (href string, ok bool)

var (
	reHeading  = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*\s*$`)
	reItem     = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])\s+(.*)$`)
	reTableSep = regexp.MustCompile(`^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?\s*$`)
	reRule     = regexp.MustCompile(`^\s*([-*_])(\s*[-*_]){2,}\s*$`)
	reWiki     = regexp.MustCompile(`\[\[([^\[\]|\x00]+?)(?:\|([^\[\]]+?))?\]\]`)
	// Neither a link's address nor a bare address can take in a token (\x00): what a
	// token stands for is finished HTML, which must never land inside an attribute.
	reLink      = regexp.MustCompile(`\[([^\[\]]+)\]\(([^()\s\x00]+)\)`)
	reURL       = regexp.MustCompile(`https?://(?:[^\s<>()"'&\x00]|&amp;)*[^\s<>()"'&.,;:!?\x00]`)
	reBold      = regexp.MustCompile(`\*\*([^*]+?)\*\*`)
	reItalic    = regexp.MustCompile(`(^|[^*\w])\*([^*\s][^*]*?)\*`)
	reStrike    = regexp.MustCompile(`~~([^~]+?)~~`)
	reToken     = regexp.MustCompile("\x00(\\d+)\x00")
	reSafeHref  = regexp.MustCompile(`^(https?://|/|#)`)
	reTaskCheck = regexp.MustCompile(`^\[([ xX])\]\s+`)
)

// markdown renders src as HTML.
func markdown(src string, link linker) template.HTML {
	r := mdRenderer{link: link}
	r.blocks(strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n"))
	return template.HTML(r.out.String())
}

type mdRenderer struct {
	out  strings.Builder
	link linker
}

func (r *mdRenderer) blocks(lines []string) {
	for i := 0; i < len(lines); {
		l := lines[i]
		t := strings.TrimSpace(l)
		switch {
		case t == "":
			i++
		case strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~"):
			fence := t[:3]
			j := i + 1
			var code []string
			for j < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[j]), fence) {
				code = append(code, lines[j])
				j++
			}
			r.out.WriteString("<pre><code>" + html.EscapeString(strings.Join(code, "\n")) + "</code></pre>\n")
			i = j + 1
		case reHeading.MatchString(t):
			m := reHeading.FindStringSubmatch(t)
			level := min(len(m[1])+1, 6) // the page's own title is the h1
			fmt.Fprintf(&r.out, "<h%d>%s</h%d>\n", level, r.inline(m[2]), level)
			i++
		case strings.HasPrefix(t, "|") && i+1 < len(lines) && reTableSep.MatchString(lines[i+1]):
			i = r.table(lines, i)
		case reRule.MatchString(t):
			r.out.WriteString("<hr>\n")
			i++
		case strings.HasPrefix(t, ">"):
			var q []string
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
				q = append(q, strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(lines[i]), ">"), " "))
				i++
			}
			r.out.WriteString("<blockquote>")
			r.blocks(q)
			r.out.WriteString("</blockquote>\n")
		case reItem.MatchString(l):
			i = r.list(lines, i)
		default:
			var p []string
			for i < len(lines) {
				t := strings.TrimSpace(lines[i])
				if t == "" || reHeading.MatchString(t) || reItem.MatchString(lines[i]) ||
					strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") || strings.HasPrefix(t, ">") ||
					strings.HasPrefix(t, "|") && i+1 < len(lines) && reTableSep.MatchString(lines[i+1]) {
					break
				}
				p = append(p, t)
				i++
			}
			if len(p) == 0 {
				// A line no other block took (a "|" line that is not a table): it is
				// text. Never leave the line where it is, or the loop would not end.
				p = append(p, strings.TrimSpace(lines[i]))
				i++
			}
			r.out.WriteString("<p>" + r.inline(strings.Join(p, " ")) + "</p>\n")
		}
	}
}

// list renders the list that starts at line i, nested by indentation, and returns the
// line after it. A line indented under an item, that is not an item itself, continues
// that item's text.
func (r *mdRenderer) list(lines []string, i int) int {
	type open struct {
		indent  int
		ordered bool
	}
	var stack []open
	closeTo := func(n int) {
		for len(stack) > n {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if top.ordered {
				r.out.WriteString("</li></ol>\n")
			} else {
				r.out.WriteString("</li></ul>\n")
			}
		}
	}
	for i < len(lines) {
		l := lines[i]
		m := reItem.FindStringSubmatch(l)
		if m == nil {
			t := strings.TrimSpace(l)
			// A continuation line: indented, not blank, not the start of another block.
			if t != "" && len(stack) > 0 && len(l)-len(strings.TrimLeft(l, " \t")) > 0 && !strings.HasPrefix(t, "```") {
				r.out.WriteString(" " + r.inline(t))
				i++
				continue
			}
			// A blank line inside a list ends it only if the next line is not an item.
			if t == "" && i+1 < len(lines) && reItem.MatchString(lines[i+1]) && len(stack) > 0 {
				i++
				continue
			}
			break
		}
		indent := len(strings.ReplaceAll(m[1], "\t", "    "))
		ordered := m[2] != "-" && m[2] != "*" && m[2] != "+"
		for len(stack) > 0 && indent < stack[len(stack)-1].indent {
			closeTo(len(stack) - 1)
		}
		switch {
		case len(stack) == 0 || indent > stack[len(stack)-1].indent:
			stack = append(stack, open{indent, ordered})
			if ordered {
				r.out.WriteString("<ol>\n<li>")
			} else {
				r.out.WriteString("<ul>\n<li>")
			}
		default:
			r.out.WriteString("</li>\n<li>")
		}
		text := m[3]
		if c := reTaskCheck.FindStringSubmatch(text); c != nil {
			box := `<span class="box-off">☐</span> `
			if c[1] != " " {
				box = `<span class="box-on">☑</span> `
			}
			r.out.WriteString(box)
			text = text[len(c[0]):]
		}
		r.out.WriteString(r.inline(text))
		i++
	}
	closeTo(0)
	return i
}

func (r *mdRenderer) table(lines []string, i int) int {
	cells := func(l string) []string {
		l = strings.TrimSpace(l)
		l = strings.TrimSuffix(strings.TrimPrefix(l, "|"), "|")
		parts := strings.Split(l, "|")
		for k := range parts {
			parts[k] = strings.TrimSpace(parts[k])
		}
		return parts
	}
	r.out.WriteString(`<div class="table"><table><thead><tr>`)
	for _, c := range cells(lines[i]) {
		r.out.WriteString("<th>" + r.inline(c) + "</th>")
	}
	r.out.WriteString("</tr></thead><tbody>\n")
	i += 2
	for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
		r.out.WriteString("<tr>")
		for _, c := range cells(lines[i]) {
			r.out.WriteString("<td>" + r.inline(c) + "</td>")
		}
		r.out.WriteString("</tr>\n")
		i++
	}
	r.out.WriteString("</tbody></table></div>\n")
	return i
}

// inline renders the text of one line. Code spans are kept apart first, then every
// other piece is escaped, and only then are the marks turned into tags. The tags made
// along the way are held as tokens, so a later rule never looks inside them.
func (r *mdRenderer) inline(s string) string {
	var held []string
	hold := func(h string) string {
		held = append(held, h)
		return fmt.Sprintf("\x00%d\x00", len(held)-1)
	}
	// Take out the NUL a note could hold, so it never passes for a token.
	s = strings.ReplaceAll(s, "\x00", "")
	var b strings.Builder
	for {
		start := strings.IndexByte(s, '`')
		if start < 0 {
			break
		}
		end := strings.IndexByte(s[start+1:], '`')
		if end < 0 {
			break
		}
		b.WriteString(s[:start])
		b.WriteString(hold("<code>" + html.EscapeString(s[start+1:start+1+end]) + "</code>"))
		s = s[start+1+end+1:]
	}
	b.WriteString(s)
	// The tokens of the code spans survive escaping: they have no special characters.
	t := html.EscapeString(b.String())

	t = reWiki.ReplaceAllStringFunc(t, func(m string) string {
		g := reWiki.FindStringSubmatch(m)
		id := html.UnescapeString(strings.TrimSpace(g[1]))
		label := g[1]
		if g[2] != "" {
			label = g[2]
		}
		if r.link != nil {
			if href, ok := r.link(id); ok {
				return hold(`<a href="` + html.EscapeString(href) + `">` + marks(label) + `</a>`)
			}
		}
		return hold(`<span class="nolink" title="` + html.EscapeString(id) + `">` + marks(label) + `</span>`)
	})
	t = reLink.ReplaceAllStringFunc(t, func(m string) string {
		g := reLink.FindStringSubmatch(m)
		href := html.UnescapeString(g[2])
		if !reSafeHref.MatchString(href) || strings.ContainsRune(href, 0) {
			return m
		}
		return hold(`<a href="` + html.EscapeString(href) + `" rel="noreferrer noopener">` + marks(g[1]) + `</a>`)
	})
	t = reURL.ReplaceAllStringFunc(t, func(m string) string {
		return hold(`<a href="` + m + `" rel="noreferrer noopener">` + m + `</a>`)
	})
	t = marks(t)
	// Tokens can hold tokens (a code span inside a link's text): restore until none is left.
	for strings.IndexByte(t, 0) >= 0 {
		t = reToken.ReplaceAllStringFunc(t, func(m string) string {
			var n int
			fmt.Sscanf(strings.Trim(m, "\x00"), "%d", &n)
			if n < len(held) {
				return held[n]
			}
			return ""
		})
	}
	return t
}

// bodySection is one level-2 section of a note, rendered: its heading and its HTML.
type bodySection struct {
	Title         string
	HTML          template.HTML
	Open, Journal bool
	Entries       int // dated entries, for a journal
}

// noteLink is the address of a note of the space or the shared ones, relative to the
// page (every page lives under the same /s/<session>/), and whether it exists.
func (v *view) noteLink(id string) (string, bool) {
	_, ok := v.b.Mem.Notes[id]
	if !ok && v.shared != nil {
		_, ok = v.shared.Notes[id]
	}
	if !ok {
		return "", false
	}
	return "note?s=" + url.QueryEscape(v.b.Space.Name) + "&id=" + url.QueryEscape(id), true
}

// sections splits a note's body at its level-2 headings and renders each. What comes
// before the first heading has no title. For a task, the first section (its essentials)
// comes open, and the last one is its journal: shown newest first, folded.
func (v *view) sections(body string, task bool) []bodySection {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	heads := memory.Headings(lines)
	var out []bodySection
	add := func(title string, part []string, open, journal bool) {
		n := 0
		if journal {
			part, n = newestFirst(part)
		}
		if strings.TrimSpace(strings.Join(part, "")) == "" && title == "" {
			return
		}
		out = append(out, bodySection{Title: title, HTML: markdown(strings.Join(part, "\n"), v.noteLink),
			Open: open, Journal: journal, Entries: n})
	}
	if len(heads) == 0 {
		add("", lines, true, false)
		return out
	}
	add("", lines[:heads[0]], true, false)
	for k, h := range heads {
		end := len(lines)
		if k+1 < len(heads) {
			end = heads[k+1]
		}
		title := strings.TrimSpace(strings.TrimPrefix(lines[h], "## "))
		journal := task && k == len(heads)-1 && k > 0 && isJournal(title)
		add(title, lines[h+1:end], !task || k == 0, journal)
	}
	return out
}

// newestFirst puts the dated entries of a journal ("- 2026-10-17 — …", with the lines
// that continue each) last to first; lines before the first entry stay on top.
func newestFirst(lines []string) ([]string, int) {
	var head []string
	var entries [][]string
	for _, l := range lines {
		if _, _, ok := memory.ParseDatedLine(l); ok {
			entries = append(entries, []string{l})
			continue
		}
		if len(entries) == 0 {
			head = append(head, l)
		} else {
			entries[len(entries)-1] = append(entries[len(entries)-1], l)
		}
	}
	out := head
	for k := len(entries) - 1; k >= 0; k-- {
		out = append(out, entries[k]...)
	}
	return out, len(entries)
}

// marks turns bold, italics and strike-through into tags, on text already escaped.
func marks(t string) string {
	t = reBold.ReplaceAllString(t, "<strong>$1</strong>")
	t = reItalic.ReplaceAllString(t, "$1<em>$2</em>")
	return reStrike.ReplaceAllString(t, "<s>$1</s>")
}

// inlineHTML renders one line, with its marks but no blocks: a next step, a summary, a
// step. A [[wikilink]] there shows as text, since the line has no space to link into.
func inlineHTML(text string) template.HTML {
	r := mdRenderer{}
	return template.HTML(r.inline(text))
}

// isJournal says whether a section's heading is a task's journal ("## Journal",
// "## 4. Diario"), in any language the program writes.
func isJournal(title string) bool {
	t := strings.ToLower(title)
	for _, w := range i18n.All("note.journal") {
		if strings.Contains(t, strings.ToLower(w)) {
			return true
		}
	}
	return false
}
