package backup

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// ConflictError stops a backup because another machine changed the same notes in a
// way that cannot be joined by itself. Its text is in English; the command says it in
// the user's language.
type ConflictError struct {
	Notes []string // with forward slashes, from the memory folder
	Dir   string   // the memory folder, where the person merges by hand
}

func (e *ConflictError) Error() string {
	return "nothing was backed up: another machine changed the same notes (" + strings.Join(e.Notes, ", ") + "). Open " + e.Dir +
		", run `git merge origin/main`, keep the right version of each, commit, and back up again"
}

// joinAdditions resolves a note both machines changed, when each only added dated lines
// to it (journal entries) and moved its `updated:` date: the result keeps every entry
// either added, in date order, and the later date. It reports whether it could; when not,
// the note is left as the merge left it.
func joinAdditions(dir, rel string) bool {
	show := func(stage string) (string, bool) {
		out, err := git(dir, "show", stage+":"+rel)
		return out, err == nil
	}
	base, ok1 := show(":1")
	ours, ok2 := show(":2")
	theirs, ok3 := show(":3")
	if !ok2 || !ok3 {
		return false // removed on one side: for the person
	}
	date := max(updatedOf(ours), updatedOf(theirs))
	ours, theirs = withUpdated(ours, date), withUpdated(theirs, date)
	if ok1 {
		base = withUpdated(base, date)
	} else {
		base = commonStart(ours, theirs) // written on both machines: what both begin with
	}
	if !onlyAddsDated(base, ours) || !onlyAddsDated(base, theirs) {
		return false
	}
	tmp, err := os.MkdirTemp("", "ab-merge-")
	if err != nil {
		return false
	}
	defer os.RemoveAll(tmp)
	files := map[string]string{"ours": ours, "base": base, "theirs": theirs}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(text), 0o600); err != nil {
			return false
		}
	}
	joined, err := git(tmp, "merge-file", "-p", "--union", "ours", "base", "theirs")
	if err != nil {
		return false
	}
	joined = byDate(joined)
	if _, err := memory.Parse(rel, []byte(joined)); err != nil {
		return false
	}
	if err := paths.WriteAtomic(filepath.Join(dir, filepath.FromSlash(rel)), []byte(joined), 0o644); err != nil {
		return false
	}
	_, err = git(dir, "add", "--", rel)
	return err == nil
}

// updatedOf returns the note's `updated:` date, or "".
func updatedOf(note string) string {
	for _, l := range strings.Split(note, "\n") {
		if v, ok := strings.CutPrefix(l, "updated:"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// withUpdated sets the note's `updated:` date, when it has one.
func withUpdated(note, date string) string {
	lines := strings.Split(note, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "updated:") {
			lines[i] = "updated: " + date
			break
		}
	}
	return strings.Join(lines, "\n")
}

// onlyAddsDated reports whether after is before with dated lines added ("- date —
// text") and nothing changed or removed: every line of before is in after, in order,
// and every other line of after is a dated line or blank.
func onlyAddsDated(before, after string) bool {
	a := strings.Split(after, "\n")
	j := 0
	added := func(l string) bool {
		_, _, dated := memory.ParseDatedLine(l)
		return dated || strings.TrimSpace(l) == ""
	}
	for _, l := range strings.Split(before, "\n") {
		for j < len(a) && a[j] != l {
			if !added(a[j]) {
				return false
			}
			j++
		}
		if j == len(a) {
			return false
		}
		j++
	}
	for ; j < len(a); j++ {
		if !added(a[j]) {
			return false
		}
	}
	return true
}

// byDate orders each run of dated lines by date, keeping the order of the same day.
func byDate(text string) string {
	lines := strings.Split(text, "\n")
	date := func(l string) (string, bool) {
		d, _, ok := memory.ParseDatedLine(l)
		return d, ok
	}
	for i := 0; i < len(lines); {
		if _, ok := date(lines[i]); !ok {
			i++
			continue
		}
		j := i
		for j < len(lines) {
			if _, ok := date(lines[j]); !ok {
				break
			}
			j++
		}
		run := lines[i:j]
		sort.SliceStable(run, func(x, y int) bool { dx, _ := date(run[x]); dy, _ := date(run[y]); return dx < dy })
		i = j
	}
	return strings.Join(lines, "\n")
}

// commonStart returns the lines a and b begin with.
func commonStart(a, b string) string {
	la, lb := strings.SplitAfter(a, "\n"), strings.SplitAfter(b, "\n")
	var out strings.Builder
	for i := 0; i < len(la) && i < len(lb) && la[i] == lb[i]; i++ {
		out.WriteString(la[i])
	}
	return out.String()
}
