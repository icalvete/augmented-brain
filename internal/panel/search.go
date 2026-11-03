package panel

import (
	"net/http"
	"sort"
	"strings"

	"github.com/icalvete/augmented-brain/internal/memory"
)

// searchMax is how many notes a search shows at most.
const searchMax = 50

type searchHit struct {
	ID, Title, Kind, Snippet string
	Shared                   bool
	score                    int
	updated                  string
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

// search finds the notes of the space, and the shared ones, that hold every word of
// the query, in their title, summary or body. A title that matches goes first; then
// the most recently changed.
func (s *Server) search(v *view, r *http.Request) (string, page, error) {
	q := strings.TrimSpace(r.FormValue("q"))
	words := strings.Fields(fold(q))
	var hits []searchHit
	if len(words) > 0 {
		for _, m := range []*memory.Memory{v.b.Mem, v.shared} {
			for _, n := range m.Notes {
				h := n.Header
				title, all := fold(h.Title), fold(h.Title+"\n"+h.Summary+"\n"+h.ID+"\n"+n.Body)
				ok, score := true, 0
				for _, w := range words {
					if !strings.Contains(all, w) {
						ok = false
						break
					}
					if strings.Contains(title, w) {
						score++
					}
				}
				if !ok {
					continue
				}
				kind := h.Type
				if kind != "" {
					kind = v.t("kind." + h.Type)
				}
				hits = append(hits, searchHit{ID: h.ID, Title: h.Title, Kind: kind, Shared: m == v.shared,
					Snippet: snippet(h.Summary+"\n"+n.Body, words), score: score, updated: h.Updated})
			}
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].updated > hits[j].updated
	})
	more := 0
	if len(hits) > searchMax {
		more, hits = len(hits)-searchMax, hits[:searchMax]
	}
	p := v.frame("search", v.t("search"))
	p.D = struct {
		Q     string
		Hits  []searchHit
		More  int
		Asked bool
	}{q, hits, more, len(words) > 0}
	return "search", p, nil
}
