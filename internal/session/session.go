// Package session builds what the assistant reads at the start of every session: who
// the user is, the rules, and the index of the space it works in.
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/rules"
)

// Budget is the most a hook may add to the context: Claude Code cuts anything longer
// than 10,000 characters to a short preview, so stay well under it.
const Budget = 9500

// Context returns the text for a session started in a folder of space s. When the full
// text does not fit, space rules are listed by title and path for the assistant to read
// when needed, and if that is still too long the index is cut.
func Context(c *config.Config, s *config.Space) string {
	m, problems, err := memory.Load(s.Memory)
	if err != nil {
		return fmt.Sprintf("augmented brain: the memory folder of space %q (%s) cannot be read: %v. Run `augmented-brain doctor`.", s.Name, s.Memory, err)
	}
	head := fmt.Sprintf("# augmented brain · space %s\n\nThis session works in space %q. Its memory is in %s; use the augmented-brain tools to work with it.\n",
		s.Name, s.Name, s.Memory)
	if len(problems) > 0 {
		head += fmt.Sprintf("\nWarning: %d notes of this space are not valid; tell the user to run `augmented-brain check`.\n", len(problems))
	}
	profile := section("Who the user is", profileText(c, s))
	common := section("Rules", rules.Common(c.Language))
	index := "\n" + m.Index(c.Language)

	var full, short []string
	for _, r := range m.OfType("rule") {
		full = append(full, "### "+r.Header.Title+"\n"+strings.TrimSpace(r.Body)+"\n")
		short = append(short, fmt.Sprintf("- %s: %s", r.Header.Title, filepath.Join(s.Memory, filepath.FromSlash(r.Path))))
	}
	spaceRules := ""
	if len(full) > 0 {
		spaceRules = section("Rules of this space", strings.Join(full, "\n"))
	}
	text := head + profile + common + spaceRules + index
	if utf8.RuneCountInString(text) <= Budget {
		return text
	}
	if len(short) > 0 {
		spaceRules = section("Rules of this space (read each file before acting on its subject)", strings.Join(short, "\n"))
	}
	text = head + profile + common + spaceRules + index
	if utf8.RuneCountInString(text) <= Budget {
		return text
	}
	note := fmt.Sprintf("\n\n(The index was cut to fit. Read it whole in %s.)\n", filepath.Join(s.Memory, memory.IndexFile))
	keep := Budget - utf8.RuneCountInString(head+profile+common+spaceRules+note)
	if keep < 0 {
		keep = 0
	}
	return head + profile + common + spaceRules + string([]rune(index)[:min(keep, utf8.RuneCountInString(index))]) + note
}

// NoSpace is the text for a session that starts outside every space.
func NoSpace(c *config.Config, dir string) string {
	return fmt.Sprintf("augmented brain: %s belongs to no space (spaces: %s), so no memory is loaded in this session. If the user wants one here, they can run `augmented-brain space add`.",
		dir, strings.Join(c.SpaceNames(), ", "))
}

// profileText joins the person's profile and the space's own, without their headers.
func profileText(c *config.Config, s *config.Space) string {
	var parts []string
	for _, p := range []string{filepath.Join(c.Dir, "profile.md"), filepath.Join(s.Memory, "profile.md")} {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if n, err := memory.Parse(filepath.Base(p), data); err == nil {
			parts = append(parts, strings.TrimSpace(n.Body))
		}
	}
	return strings.Join(parts, "\n\n")
}

func section(title, body string) string {
	if strings.TrimSpace(body) == "" {
		return ""
	}
	return "\n## " + title + "\n\n" + strings.TrimSpace(body) + "\n"
}
