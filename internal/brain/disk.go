package brain

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// Mismatch is something a note says about this computer that is not so. Today, one
// kind: a repository in its repos: that is in none of the space's work folders (it was
// moved, renamed or deleted, or this computer never had it). It is only reported:
// nothing is changed to make the note and the disk agree.
type Mismatch struct {
	Note, Title, Repo string
}

// Mismatches returns the mismatches of every note but closed tasks (a repository
// deleted after its task closed is no surprise), by note.
func (b *Brain) Mismatches() []Mismatch {
	var out []Mismatch
	for id, n := range b.Mem.Notes {
		if n.Header.Type == "task" && n.Header.Status == "closed" {
			continue
		}
		for _, r := range b.MissingRepos(n) {
			out = append(out, Mismatch{Note: id, Title: n.Header.Title, Repo: r})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Note != out[j].Note {
			return out[i].Note < out[j].Note
		}
		return out[i].Repo < out[j].Repo
	})
	return out
}

// MissingRepos returns the repositories of a note that are not on this computer. A
// repository is written relative to a work folder of the space (an absolute path or
// one starting with ~ is taken as it is) and is there when one of the work folders has
// that folder. A space with no work folders cannot tell, so nothing is missing there.
func (b *Brain) MissingRepos(n *memory.Note) []string {
	var out []string
	for _, r := range n.Header.Repos {
		if r = strings.TrimSpace(r); r != "" && !b.repoHere(r) {
			out = append(out, r)
		}
	}
	return out
}

func (b *Brain) repoHere(r string) bool {
	if strings.HasPrefix(r, "~") || filepath.IsAbs(r) {
		p, err := paths.Resolve(r)
		return err != nil || isDir(p) // a path it cannot read is not called missing
	}
	if len(b.Space.WorkFolders) == 0 {
		return true
	}
	for _, wf := range b.Space.WorkFolders {
		if isDir(filepath.Join(wf, filepath.FromSlash(r))) {
			return true
		}
	}
	return false
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
