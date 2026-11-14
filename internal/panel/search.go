package panel

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// searchMax is how many notes a search shows at most.
const searchMax = 50

type searchHit struct {
	Title, Snippet []part
	Kind           string
	Shared         bool
	Space          string // the space it is in; "" for a shared note
	Link           string // the page that shows it, after Base
	score          int
	updated        string
	here           bool // in the space the page is for
	// snip makes Snippet, once the hit is among the ones shown: a snippet folds every
	// line a rune at a time, too much for every hit of a common word (P-11).
	snip func() []part
}

// part is a piece of a title or snippet, Mark when it is a word the search asked for.
// The template writes each piece as text, inside <mark> when marked: nothing in a note
// becomes HTML.
type part struct {
	Text string
	Mark bool
}

// marked cuts text into pieces, marking every place a word matches, folded as fold
// folds (one rune for one rune, so a place in the folded text is a place in the text).
func marked(text string, words []string) []part {
	r := []rune(text)
	f := []rune(fold(text))
	if len(f) != len(r) {
		return []part{{Text: text}}
	}
	on := make([]bool, len(r))
	for _, w := range words {
		ws := []rune(w)
		if len(ws) == 0 {
			continue
		}
		for i := 0; i+len(ws) <= len(f); i++ {
			if string(f[i:i+len(ws)]) == w {
				for k := i; k < i+len(ws); k++ {
					on[k] = true
				}
			}
		}
	}
	var out []part
	for i := 0; i < len(r); {
		j := i
		for j < len(r) && on[j] == on[i] {
			j++
		}
		out = append(out, part{Text: string(r[i:j]), Mark: on[i]})
		i = j
	}
	return out
}

// unaccent takes the accents off the letters of Latin languages, one letter for one,
// so a search for "revision" finds "revisión" (no extra library for a whole Unicode
// normalization: this covers what notes are written in).
var unaccent = strings.NewReplacer(
	"á", "a", "à", "a", "ä", "a", "â", "a", "ã", "a", "å", "a",
	"é", "e", "è", "e", "ë", "e", "ê", "e",
	"í", "i", "ì", "i", "ï", "i", "î", "i",
	"ó", "o", "ò", "o", "ö", "o", "ô", "o", "õ", "o",
	"ú", "u", "ù", "u", "ü", "u", "û", "u",
	"ñ", "n", "ç", "c",
)

// fold lowers text and takes accents off, keeping one rune per rune.
func fold(s string) string { return unaccent.Replace(strings.ToLower(s)) }

// snippetOf is snippet; tests count the calls.
var snippetOf = snippet

// lazySnippet is the snip of a hit: the snippet of body, marked.
func lazySnippet(body string, words []string) func() []part {
	return func() []part { return marked(snippetOf(body, words), words) }
}

// snippet is a short piece of text around the first place a line matches one of the
// words. It folds the line a rune at a time, so where the match is in the folded text
// says where it is in the line, whatever the folding does to a rune.
func snippet(body string, words []string) string {
	for _, l := range strings.Split(body, "\n") {
		r := []rune(strings.TrimSpace(l))
		var folded strings.Builder
		at := make([]int, 0, len(r)+1) // at[k]: where rune k starts in the folded text
		for _, c := range r {
			at = append(at, folded.Len())
			folded.WriteString(fold(string(c)))
		}
		f := folded.String()
		for _, w := range words {
			i := strings.Index(f, w)
			if i < 0 {
				continue
			}
			if len(r) <= 220 {
				return string(r)
			}
			k := 0
			for k+1 < len(at) && at[k+1] <= i {
				k++
			}
			from := min(max(0, k-80), len(r))
			to := min(len(r), from+220)
			out := string(r[from:to])
			if from > 0 {
				out = "…" + out
			}
			if to < len(r) {
				out += "…"
			}
			return out
		}
	}
	return ""
}

// search finds, in every space and the shared notes, the notes that hold every word of
// the query in their title, summary or body; also the captures of each space's inbox
// and the journals archived from its tasks. A title that matches goes first, then what
// is in the space the page is for, then the most recently changed. The words found are
// marked.
func (s *Server) search(v *view, r *http.Request) (string, page, error) {
	q := strings.TrimSpace(r.FormValue("q"))
	words := strings.Fields(fold(q))
	var hits []searchHit
	var unread []string
	if len(words) > 0 {
		here := v.b.Space.Name
		hits = append(hits, noteHits(v, v.shared, "", words)...)
		for _, name := range v.c.SpaceNames() {
			b := v.b
			if name != here {
				var err error
				if b, err = brain.Open(v.c, v.c.Spaces[name]); err != nil {
					unread = append(unread, name)
					continue
				}
			}
			hits = append(hits, noteHits(v, b.Mem, name, words)...)
			hits = append(hits, inboxHits(v, b, words)...)
			hits = append(hits, journalHits(v, b, words)...)
		}
		for i := range hits {
			hits[i].here = hits[i].Space == "" || hits[i].Space == here
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		if hits[i].here != hits[j].here {
			return hits[i].here
		}
		return hits[i].updated > hits[j].updated
	})
	more := 0
	if len(hits) > searchMax {
		more, hits = len(hits)-searchMax, hits[:searchMax]
	}
	for i := range hits {
		if hits[i].snip != nil {
			hits[i].Snippet = hits[i].snip()
		}
	}
	p := v.frame("search", v.t("search"))
	p.D = struct {
		Q      string
		Hits   []searchHit
		More   int
		Asked  bool
		Here   string
		Unread string
	}{q, hits, more, len(words) > 0, v.b.Space.Name, strings.Join(unread, ", ")}
	return "search", p, nil
}

// holds says whether text, folded, holds every word, and how many of them title holds.
func holds(text, title string, words []string) (bool, int) {
	all, t := fold(text), fold(title)
	score := 0
	for _, w := range words {
		if !strings.Contains(all, w) {
			return false, 0
		}
		if strings.Contains(t, w) {
			score++
		}
	}
	return true, score
}

// noteHits are the notes of m (of space; "" for the shared notes) that hold every word.
func noteHits(v *view, m *memory.Memory, space string, words []string) []searchHit {
	var out []searchHit
	for _, n := range m.Notes {
		h := n.Header
		ok, score := holds(h.Title+"\n"+h.Summary+"\n"+h.ID+"\n"+n.Body, h.Title, words)
		if !ok {
			continue
		}
		kind := h.Type
		if kind != "" {
			kind = v.t("kind." + h.Type)
		}
		s := space
		if s == "" {
			s = v.b.Space.Name // a shared note opens from the page's own space
		}
		out = append(out, searchHit{Title: marked(h.Title, words), Kind: kind, Shared: space == "", Space: space,
			Link: "/note?" + url.Values{"s": {s}, "id": {h.ID}}.Encode(), snip: lazySnippet(h.Summary+"\n"+n.Body, words),
			score: score, updated: h.Updated})
	}
	return out
}

// inboxHits are the captures of b's inbox that hold every word.
func inboxHits(v *view, b *brain.Brain, words []string) []searchHit {
	entries, _ := b.Inbox()
	var out []searchHit
	for _, e := range entries {
		if ok, _ := holds(e.Text, "", words); ok {
			out = append(out, searchHit{Title: marked(e.Text, words), Kind: v.t("search.kind.inbox"), Space: b.Space.Name,
				Link: "/inbox?" + url.Values{"s": {b.Space.Name}}.Encode(), updated: e.Date})
		}
	}
	return out
}

// archivedName is the name of a file of archive/journals/: the task's id, then the
// dates of the first and last entry it holds, and a number when the name was taken.
var archivedName = regexp.MustCompile(`^(.+)-\d{4}-\d{2}-\d{2}-(\d{4}-\d{2}-\d{2})(?:-\d+)?\.md$`)

// journalHits are the journals archived from b's tasks that hold every word. Each
// leads to its task.
func journalHits(v *view, b *brain.Brain, words []string) []searchHit {
	dir := filepath.Join(b.Mem.Dir, filepath.FromSlash(memory.JournalArchiveDir))
	files, _ := os.ReadDir(dir)
	var out []searchHit
	for _, f := range files {
		m := archivedName.FindStringSubmatch(f.Name())
		if f.IsDir() || m == nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			continue
		}
		text := string(data)
		title, rest, _ := strings.Cut(strings.TrimPrefix(text, "# "), "\n")
		if ok, _ := holds(text, "", words); !ok {
			continue
		}
		out = append(out, searchHit{Title: marked(title, words), Kind: v.t("search.kind.journal"), Space: b.Space.Name,
			Link: "/task?" + url.Values{"s": {b.Space.Name}, "id": {m[1]}}.Encode(), snip: lazySnippet(rest, words), updated: m[2]})
	}
	return out
}
