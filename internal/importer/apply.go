package importer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/format"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// Applied says what Apply wrote, per space: notes, files copied as they were, and
// captures added to the inbox.
type Applied struct {
	Notes, Copies, Inbox map[string]int
}

// Apply writes a plan: each note into its space, each archive file into the space's
// archive/ as it was, and then the index of every space it changed. It never writes
// over a file that is already there (one written since the plan was made is left as
// it is and reported). The caller holds the notes lock.
func Apply(p *Plan, c *config.Config, lang string) (Applied, error) {
	done := Applied{Notes: map[string]int{}, Copies: map[string]int{}, Inbox: map[string]int{}}
	// Notes written by a newer program are never written into.
	checked := map[string]bool{}
	for _, it := range p.Items {
		if it.Space != "" && !checked[it.Space] {
			checked[it.Space] = true
			mem := memoryOf(c, it.Space)
			if _, err := os.Stat(mem); errors.Is(err, os.ErrNotExist) && it.Space == config.SharedFolder {
				if err := format.Write(mem, format.Notes()); err != nil { // the first shared notes
					return done, err
				}
			}
			if err := format.CheckNotes(mem); err != nil {
				return done, err
			}
		}
	}
	var skipped []string
	for _, it := range p.Items {
		if it.Space == "" {
			continue
		}
		mem := memoryOf(c, it.Space)
		if it.Inbox != nil {
			n, err := addCaptures(filepath.Join(mem, "inbox.md"), it.Inbox)
			if err != nil {
				return done, err
			}
			done.Inbox[it.Space] += n
			continue
		}
		var dst string
		if it.Copy {
			dst = filepath.Join(mem, filepath.FromSlash(it.Source))
		} else {
			dst = filepath.Join(mem, filepath.FromSlash(it.Note.Path))
		}
		if _, err := os.Stat(dst); err == nil {
			skipped = append(skipped, dst)
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return done, err
		}
		if it.Copy {
			data, err := os.ReadFile(filepath.Join(p.Source, filepath.FromSlash(it.Source)))
			if err != nil {
				return done, err
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return done, err
			}
			if err := paths.WriteAtomic(dst, data, 0o644); err != nil {
				return done, err
			}
			done.Copies[it.Space]++
			continue
		}
		if err := memory.Save(mem, it.Note); err != nil {
			return done, fmt.Errorf("%s: %w", it.Source, err)
		}
		done.Notes[it.Space]++
	}
	var spaces []string
	for sp := range done.Notes {
		spaces = append(spaces, sp)
	}
	sort.Strings(spaces)
	for _, sp := range spaces {
		m, _, err := memory.Load(memoryOf(c, sp))
		if err != nil {
			return done, err
		}
		m.Shared = sp == config.SharedFolder
		if _, err := m.WriteIndex(lang); err != nil {
			return done, err
		}
	}
	if len(skipped) > 0 {
		return done, fmt.Errorf("%d files appeared after the plan was made and were left as they are: %v", len(skipped), skipped)
	}
	return done, nil
}

// memoryOf returns the memory folder of a space of the plan: the shared notes have
// theirs in the configuration folder.
func memoryOf(c *config.Config, sp string) string {
	if sp == config.SharedFolder {
		return c.SharedDir()
	}
	return c.Spaces[sp].Memory
}

// addCaptures appends to an inbox the captures it does not have yet, so a second import
// adds nothing, and returns how many it added.
func addCaptures(path string, lines []string) (int, error) {
	old, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	have := map[string]bool{}
	for _, l := range strings.Split(string(old), "\n") {
		have[l] = true
	}
	text := string(old)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	n := 0
	for _, l := range lines {
		if !have[l] {
			have[l] = true
			text += l + "\n"
			n++
		}
	}
	if n == 0 {
		return 0, nil
	}
	return n, paths.WriteAtomic(path, []byte(text), 0o644)
}
