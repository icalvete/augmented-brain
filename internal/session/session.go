// Package session builds what the assistant reads at the start of every session: who
// the user is, the rules, and the index of the space it works in.
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
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
func Context(c *config.Config, s *config.Space) string { return ContextWith(c, s, "") }

// ContextWith is Context with a note for the assistant right after the heading, kept
// whole when the rest is cut to fit.
func ContextWith(c *config.Config, s *config.Space, note string) string {
	m, problems, err := memory.Load(s.Memory)
	if err != nil {
		return fmt.Sprintf("augmented brain: the memory folder of space %q (%s) cannot be read: %v. Run `augmented-brain doctor`.", s.Name, s.Memory, err)
	}
	head := fmt.Sprintf("# augmented brain · space %s\n\nThis session works in space %q. Its memory is in %s; use the augmented-brain tools to work with it.\n",
		s.Name, s.Name, s.Memory)
	head += note
	if len(problems) > 0 {
		head += fmt.Sprintf("\nWarning: %d notes of this space are not valid; tell the user to run `augmented-brain check`.\n", len(problems))
	}
	if brain.ProfileEmpty(c.Dir) {
		head += firstSession(c.Language)
	} else if len(problems) == 0 {
		b := &brain.Brain{Space: s, Language: c.Language, Mem: m, ConfigDir: c.Dir, Now: Now}
		head += dueNote(b) + reviewNote(b)
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
	if size(text) <= Budget {
		return text
	}
	// 1. Space rules by title and path, to be read when needed.
	if len(short) > 0 {
		spaceRules = section("Rules of this space (read each file before acting on its subject)", strings.Join(short, "\n"))
	}
	// 2. A long profile, cut, saying where the whole one is.
	if size(profile) > profileBudget {
		profile = cut(profile, profileBudget) + fmt.Sprintf("\n(The profile was cut to fit. Read it whole in %s.)\n", filepath.Join(c.Dir, "profile.md"))
	}
	text = head + profile + common + spaceRules + index
	if size(text) <= Budget {
		return text
	}
	// 3. The index, cut, saying where the whole one is.
	cutNote := fmt.Sprintf("\n\n(The index was cut to fit. Read it whole in %s.)\n", filepath.Join(s.Memory, memory.IndexFile))
	text = head + profile + common + spaceRules + cut(index, Budget-size(head+profile+common+spaceRules+cutNote)) + cutNote
	// 4. Whatever is left (hundreds of space rules), never over the budget.
	if size(text) > Budget {
		end := "\n(Cut to fit. Read the space's rules and index in " + s.Memory + ".)\n"
		text = cut(text, Budget-size(end)) + end
	}
	return text
}

// Now gives today's date; tests replace it.
var Now = time.Now

// maxDue is how many due items the session start lists; the rest are counted.
const maxDue = 8

// dueNote lists what comes up today: watch dates and inbox reminders due today or
// tomorrow, and those already past.
func dueNote(b *brain.Brain) string {
	due := b.Reminders(1)
	if len(due) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\n## Due\n\nMention these in your first answer, briefly, before anything else:\n")
	for i, d := range due {
		if i == maxDue {
			fmt.Fprintf(&out, "- and %d more: see the pending tool\n", len(due)-maxDue)
			break
		}
		when := "today"
		switch {
		case d.Days == 1:
			when = "tomorrow"
		case d.Days < 0:
			when = fmt.Sprintf("%d days overdue", -d.Days)
		}
		where := "inbox"
		if d.TaskID != "" {
			where = d.TaskID
		}
		fmt.Fprintf(&out, "- %s (%s): %s [%s]\n", d.Date, when, d.What, where)
	}
	out.WriteString("When one is dealt with, mark it: task_update with watch_done for a task, or take the capture out in the weekly review.\n")
	return out.String()
}

// reviewNote asks the assistant to offer the weekly review when it is due.
func reviewNote(b *brain.Brain) string {
	due, days := b.ReviewDue()
	if !due {
		return ""
	}
	r := b.Review()
	return fmt.Sprintf("\nThe weekly review is due: %d days since the last one, with %d captures in the inbox and %d tasks untouched for %d days or more. "+
		"Offer it once, in one short line, when it does not interrupt other work (about ten minutes); if the user agrees, start with weekly_review.\n",
		days, len(r.Inbox), len(r.Stale), brain.StaleDays)
}

// profileBudget is the most of the profile loaded when everything does not fit.
const profileBudget = 2500

// size counts like Claude Code may: in UTF-16 units, so a character outside the basic
// plane (an emoji) counts twice. It is never less than the number of characters.
func size(s string) int {
	n := 0
	for _, r := range s {
		n++
		if r > 0xFFFF {
			n++
		}
	}
	return n
}

// cut keeps the start of s that fits in max units, without splitting a character.
func cut(s string, max int) string {
	if max <= 0 {
		return ""
	}
	n := 0
	for i, r := range s {
		w := 1
		if r > 0xFFFF {
			w = 2
		}
		if n+w > max {
			return s[:i]
		}
		n += w
	}
	return s
}

// firstSession is the interview the assistant runs while the user's profile is blank:
// a few guided questions, each with its reason, whose answers set everything up.
func firstSession(lang string) string {
	language := map[string]string{"es": "Spanish"}[lang]
	if language == "" {
		language = "English"
	}
	return "\n## First session: get to know the user\n\n" +
		"The user's profile is blank, so this is their first session. Before anything else, run this short interview, in the language the user writes in (" + language + " if you cannot tell yet), and keep to it.\n\n" +
		"Start by saying, in two sentences, what augmented brain does for them (you remember their tasks, decisions and people from one session to the next, " +
		"and you check risky commands with them first), and that six quick questions will set it up for them: about five minutes, any question can be skipped, " +
		"and everything can be changed later by asking. Then ask one question at a time. Before each, say in one short sentence why you ask it. " +
		"Wait for the answer, repeat back what you understood, and go on.\n\n" +
		"1. Their name and what they do. Why: to talk to them in their own context.\n" +
		"2. What they look after, the responsibilities that never end (a website, some servers, a team, the home). " +
		"Why: each one becomes an area that groups its tasks and what they learn. Create the ones that belong in this space with area_new.\n" +
		"3. How they want your answers: language, short or detailed, a recommendation first or the options. Why: so they never have to repeat it.\n" +
		"4. How much you may do without asking (for example: edit freely, but ask before deleting or publishing), and what must never be touched lightly: " +
		"AWS profiles that reach production, servers by name. Why: the guard will stop and ask them before anything that touches those. " +
		"Save the profiles and hosts with safety_save.\n" +
		"5. The tools they use (calendar, mail, documents, Jira…). Why: some can be connected to you; if they want, point them to the guide " +
		"https://icalvete.github.io/augmented-brain/guide/connect-services.html. Now save the profile once with profile_save: name, what they do and their tools in about, " +
		"answers and autonomy in work_style.\n" +
		"6. The people they deal with most, and what they call them. Why: so a first name like \"Marcos\" leads to the right person. " +
		"Save each one with person_save, with their nicknames as aliases.\n\n" +
		"At the end, show a short summary of what you saved and where, tell them they can change any of it just by asking, " +
		"and offer two things: to set up the backup of their notes (backup_setup: a private repository, so losing the machine loses nothing), " +
		"and to open their first task.\n"
}

// DefaultNote tells the assistant that the session runs outside every space's folder,
// so the default space was loaded, and how to work in another one.
func DefaultNote(c *config.Config, s *config.Space, dir string) string {
	var others []string
	for _, name := range c.SpaceNames() {
		if name != s.Name && len(c.Spaces[name].WorkFolders) > 0 {
			others = append(others, fmt.Sprintf("%s: %s", name, c.Spaces[name].WorkFolders[0]))
		}
	}
	language := map[string]string{"es": "Spanish"}[c.Language]
	if language == "" {
		language = "English"
	}
	note := fmt.Sprintf("\nThis session runs in %s, outside every space's folder, so the default space, %s, is loaded. "+
		"Say so in one short line of your first answer, in the language the user writes in (%s if you cannot tell).", dir, s.Name, language)
	if len(others) > 0 {
		note += fmt.Sprintf(" To work in another space, the user opens a session in its folder (%s): in the Claude desktop app, "+
			"\"+ New\" then \"Choose…\" under the message box; in a terminal, `cd <folder> && claude`.", strings.Join(others, "; "))
	}
	return note + "\n"
}

// BackupFailedNote tells the assistant that the last backup of the space failed.
func BackupFailedNote(lang, space string, when time.Time, reason string) string {
	// git's first line says what happened; the rest is advice for people at a terminal.
	if i := strings.IndexAny(reason, "\r\n"); i >= 0 {
		reason = reason[:i]
	}
	if len(reason) > 240 {
		reason = cut(reason, 240) + "…"
	}
	language := map[string]string{"es": "Spanish"}[lang]
	if language == "" {
		language = "English"
	}
	return fmt.Sprintf("\nThe last backup of space %s, at %s, failed: %s. In one short line of your first answer, in the language "+
		"the user writes in (%s if you cannot tell), tell them their notes are not backed up, and offer to look into it "+
		"(running `augmented-brain backup` in the space's folder shows the problem).\n", space, when.Format("2006-01-02 15:04"), reason, language)
}

// UpdateNote tells the assistant that a newer augmented brain is out.
func UpdateNote(lang, current, latest string) string {
	language := map[string]string{"es": "Spanish"}[lang]
	if language == "" {
		language = "English"
	}
	return fmt.Sprintf("\naugmented brain %s is available (this machine has %s). In one short line of your first answer, "+
		"in the language the user writes in (%s if you cannot tell), tell them and offer to update it: you can run "+
		"`augmented-brain update` yourself if they say yes. Afterwards they restart Claude to use it.\n", latest, current, language)
}

// NoSpace is the text for a session that starts outside every space. The assistant
// tells the user at once, with the folders to open instead: in the desktop app a new
// session starts in a scratch folder unless the user picks one, and that is easy to miss.
func NoSpace(c *config.Config, dir string) string {
	var where []string
	for _, name := range c.SpaceNames() {
		if f := c.Spaces[name].WorkFolders; len(f) > 0 {
			where = append(where, fmt.Sprintf("%s: %s", name, f[0]))
		}
	}
	language := map[string]string{"es": "Spanish"}[c.Language]
	if language == "" {
		language = "English"
	}
	return fmt.Sprintf("augmented brain: this session runs in %s, which is outside the user's spaces, so none of their memory, rules or tasks is loaded.\n\n"+
		"In your first answer, before anything else, tell the user this in the language they write in (%s if you cannot tell), in one or two short sentences, and how to fix it: "+
		"start a new session in the work folder of one of their spaces (%s). "+
		"In the Claude desktop app that is \"+ New\", then \"Choose…\" (the folder button under the message box) before writing; in a terminal, `cd <folder> && claude`. "+
		"If they really want to work here, carry on without memory.",
		dir, language, strings.Join(where, "; "))
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
