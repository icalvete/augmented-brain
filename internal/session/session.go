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
// text does not fit, the user's rules (shared and of the space) are listed by title and
// path for the assistant to read when needed, and if that is still too long the index is
// cut.
func Context(c *config.Config, s *config.Space) string { return ContextWith(c, s, "") }

// ContextWith is Context with a note for the assistant right after the heading, kept
// whole when the rest is cut to fit.
func ContextWith(c *config.Config, s *config.Space, note string) string {
	m, shared, problems, failed := loadSpace(c, s)
	if failed != "" {
		return failed
	}
	return fit(parts{
		head:    heading(c, s, m, problems, note),
		profile: section("Who the user is", profileText(c, s)),
		common:  section("Rules", rules.Common(c.Language)),
		index:   "\n" + m.Index(c.Language) + sharedKnowledge(shared),
	}, c, s, m, shared)
}

// Parts is how many SessionStart hooks share the session text. Claude Code cuts each
// hook's text at 10,000 characters, so one hook cannot carry the index of a big space;
// each part has a Budget of its own.
const Parts = 1 + indexParts + rulesParts

// indexParts carry the index (parts 2 to 4), rulesParts the user's rules (5 to 9),
// whole: in real sessions the assistant does not open a rule's file before acting.
const indexParts, rulesParts = 3, 5

// Part returns part n (1 to Parts) of the session text: 1 is the heading with note, the
// profile and the common rules; then the index; then the user's rules. Each part is
// built on its own, so the hooks can run at the same time, in any order. A part with
// nothing to say is "".
func Part(c *config.Config, s *config.Space, note string, n int) string {
	m, shared, problems, failed := loadSpace(c, s)
	if failed != "" {
		if n == 1 {
			return failed
		}
		return ""
	}
	tag := fmt.Sprintf("# augmented brain · space %s · part %d of %d\n\nThe start of this session comes in %d parts, which may arrive in any order: read them together.\n", s.Name, n, Parts, Parts)
	switch {
	case n == 1:
		head := heading(c, s, m, problems, note) + fmt.Sprintf("\n(This is part 1 of %d: the index of the space and the user's rules come in the other parts.)\n", Parts)
		profile := section("Who the user is", profileText(c, s))
		common := section("Rules", rules.Common(c.Language))
		if size(head+profile+common) > Budget && size(profile) > profileBudget {
			profile = cut(profile, profileBudget) + fmt.Sprintf("\n(The profile was cut to fit. Read it whole in %s.)\n", filepath.Join(c.Dir, "profile.md"))
		}
		return within(head+profile+common, "\n(Cut to fit. Read the profile in "+filepath.Join(c.Dir, "profile.md")+".)\n")
	case n > 1 && n <= 1+indexParts:
		where := filepath.Join(s.Memory, memory.IndexFile)
		if len(shared.Notes) > 0 {
			where += "; the shared notes are in " + shared.Dir
		}
		cutNote := fmt.Sprintf("\n\n(The index was cut to fit. Read it whole in %s.)\n", where)
		chunks := lines("\n"+m.Index(c.Language)+sharedKnowledge(shared), Budget-size(tag+cutNote))
		if n-2 >= len(chunks) {
			return ""
		}
		text := tag + chunks[n-2]
		if n == 1+indexParts && len(chunks) > indexParts {
			text += cutNote
		}
		return text
	case n > 1+indexParts && n <= Parts:
		end := "\n(Cut to fit. Read the rules in " + s.Memory + " and the shared notes in " + shared.Dir + ".)\n"
		pieces := rulePieces(m, shared, Budget-size(tag+end))
		k := n - 2 - indexParts
		if k >= len(pieces) {
			return ""
		}
		return within(tag+pieces[k], end)
	}
	return ""
}

// rulePieces packs the user's rules, shared and of the space, into at most rulesParts
// pieces of max units: each whole, without its History section, in the first piece
// with room, and one longer than a piece across pieces of its own. A rule that cannot
// go whole goes in one line, with the path of its file.
func rulePieces(m, shared *memory.Memory, max int) []string {
	type item struct {
		group         int
		whole, line   string
		listed, split bool
	}
	titles := []string{"Rules for every space", "Rules of this space"}
	var items []*item
	for g, mem := range []*memory.Memory{shared, m} {
		for _, r := range mem.OfType("rule") {
			path := filepath.Join(mem.Dir, filepath.FromSlash(r.Path))
			body, cutHistory := withoutHistory(strings.TrimSpace(r.Body))
			it := &item{group: g, whole: "### " + r.Header.Title + "\n" + body + "\n"}
			if cutHistory {
				it.whole += "(Its History is in " + path + ".)\n"
			}
			it.whole += "\n"
			it.line = "- " + r.Header.Title
			if r.Header.Summary != "" {
				it.line += " — " + r.Header.Summary // the gist, so the rule applies without opening it
			}
			it.line += " (" + path + ")\n"
			items = append(items, it)
		}
	}
	render := func(p []*item) string {
		var out strings.Builder
		for _, listed := range []bool{false, true} {
			group := -1
			for _, it := range p {
				if it.listed != listed {
					continue
				}
				if listed && group == -1 {
					out.WriteString(listedNote)
				}
				if it.group != group {
					group = it.group
					out.WriteString("\n## " + titles[group] + "\n\n")
				}
				if listed {
					out.WriteString(it.line)
				} else {
					out.WriteString(it.whole)
				}
			}
		}
		return out.String()
	}
	var pieces [][]*item
	var raw []string // a rule longer than a piece, cut in pieces of its own
	slots := func() int { return rulesParts - len(pieces) - len(raw) }
	place := func(it *item) bool {
		for i, p := range pieces {
			if size(render(append(p[:len(p):len(p)], it))) <= max {
				pieces[i] = append(p, it)
				return true
			}
		}
		if slots() > 0 && size(render([]*item{it})) <= max {
			pieces = append(pieces, []*item{it})
			return true
		}
		return false
	}
	for _, it := range items {
		if place(it) {
			continue
		}
		head := "\n## " + titles[it.group] + "\n\n"
		if chunks := lines(head+it.whole, max); size(head+it.whole) > max && len(chunks) <= slots()-1 {
			raw = append(raw, chunks...) // one slot stays for the rules that go in a line
			continue
		}
		it.listed = true
	}
	var out, dropped []string
	for _, it := range items {
		if it.listed && !place(it) {
			dropped = append(dropped, it.line)
		}
	}
	for _, p := range pieces {
		out = append(out, render(p))
	}
	out = append(out, raw...)
	if len(dropped) > 0 && len(out) > 0 {
		out[len(out)-1] += fmt.Sprintf("\n(%d more rules did not fit; their files are in the rules folders of the space and the shared notes.)\n", len(dropped))
	}
	return out
}

// listedNote heads the rules that go in one line each.
const listedNote = "\n(The rules below did not fit whole, so each goes in one line: follow it, and read its file before acting on its subject.)\n"

// withoutHistory drops a rule's History section (## History or ## Historia, to the next
// heading of its level), which tells how the rule came to be, not what to do.
func withoutHistory(body string) (string, bool) {
	var out []string
	skip, level, cut := false, 0, false
	for _, l := range strings.Split(body, "\n") {
		if h := strings.TrimLeft(l, "#"); h != l && strings.HasPrefix(h, " ") {
			lv := len(l) - len(h)
			title := strings.ToLower(strings.TrimSpace(h))
			switch {
			case title == "history" || title == "historia":
				skip, level, cut = true, lv, true
				continue
			case skip && lv <= level:
				skip = false
			}
		}
		if !skip {
			out = append(out, l)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n")), cut
}

// loadSpace reads space s with the shared notes; failed is the text for the assistant
// when they cannot be read.
func loadSpace(c *config.Config, s *config.Space) (m, shared *memory.Memory, problems []memory.Problem, failed string) {
	shared, sharedProblems, err := memory.LoadShared(c.SharedDir())
	if err != nil {
		return nil, nil, nil, fmt.Sprintf("augmented brain: the shared notes (%s) cannot be read: %v. Run `augmented-brain doctor`.", c.SharedDir(), err)
	}
	m, problems, err = memory.Load(s.Memory, shared)
	if err != nil {
		return nil, nil, nil, fmt.Sprintf("augmented brain: the memory folder of space %q (%s) cannot be read: %v. Run `augmented-brain doctor`.", s.Name, s.Memory, err)
	}
	for _, p := range sharedProblems {
		problems = append(problems, memory.Problem{Path: filepath.Join(c.SharedDir(), filepath.FromSlash(p.Path)), Err: p.Err})
	}
	return m, shared, problems, ""
}

// within returns text, or its start and end when it is over Budget.
func within(text, end string) string {
	if size(text) <= Budget {
		return text
	}
	return cut(text, Budget-size(end)) + end
}

// lines splits text into pieces of at most max units, at line ends when it can.
func lines(text string, max int) []string {
	var out []string
	cur := ""
	for _, l := range strings.SplitAfter(text, "\n") {
		for size(l) > max { // a line longer than a piece: cut it
			if cur != "" {
				out, cur = append(out, cur), ""
			}
			head := cut(l, max)
			out, l = append(out, head), l[len(head):]
		}
		if size(cur+l) > max {
			out, cur = append(out, cur), ""
		}
		cur += l
	}
	if strings.TrimSpace(cur) != "" {
		out = append(out, cur)
	}
	return out
}

// parts are the pieces of the session text, in the order they are joined.
type parts struct{ head, profile, common, userRules, index string }

func (p parts) join() string { return p.head + p.profile + p.common + p.userRules + p.index }

// sharedKnowledge lists the shared references and lessons, one line each, for the
// assistant to open when one bears on the work.
func sharedKnowledge(shared *memory.Memory) string {
	var lines []string
	for _, typ := range []string{"reference", "lesson"} {
		for _, n := range shared.OfType(typ) {
			line := fmt.Sprintf("- [%s](%s)", n.Header.Title, filepath.Join(shared.Dir, filepath.FromSlash(n.Path)))
			if n.Header.Summary != "" {
				line += " — " + n.Header.Summary
			}
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return section("Knowledge for every space (open the file when it bears on the work)", strings.Join(lines, "\n"))
}

// heading is what always goes whole: where the memory is, the caller's note, broken
// files, and either the first-session interview or what comes up today.
func heading(c *config.Config, s *config.Space, m *memory.Memory, problems []memory.Problem, note string) string {
	head := fmt.Sprintf("# augmented brain · space %s\n\nThis session works in space %q. Its memory is in %s; use the augmented-brain tools to work with it.\n",
		s.Name, s.Name, s.Memory)
	head += note
	if len(problems) > 0 {
		head += "\nWarning: " + brain.ProblemsNote(problems, 3)
	}
	if brain.ProfileEmpty(c.Dir) {
		return head + firstSession(c.Language)
	}
	b := &brain.Brain{Space: s, Language: c.Language, Mem: m, ConfigDir: c.Dir, Now: Now}
	return head + dueNote(b) + reviewNote(b)
}

// fit adds the user's rules, shared and of the space, and returns the text within
// Budget, giving up detail in steps when it does not fit; the heading is never cut
// before the last step.
func fit(p parts, c *config.Config, s *config.Space, m, shared *memory.Memory) string {
	full, short := userRules(m, shared)
	p.userRules = full
	if text := p.join(); size(text) <= Budget {
		return text
	}
	// 1. The user's rules by title and path, to be read when needed.
	p.userRules = short
	// 2. A long profile, cut, saying where the whole one is.
	if size(p.profile) > profileBudget {
		p.profile = cut(p.profile, profileBudget) + fmt.Sprintf("\n(The profile was cut to fit. Read it whole in %s.)\n", filepath.Join(c.Dir, "profile.md"))
	}
	if text := p.join(); size(text) <= Budget {
		return text
	}
	// 3. The index, cut, saying where the whole one is.
	where := filepath.Join(s.Memory, memory.IndexFile)
	if len(shared.Notes) > 0 {
		where += "; the shared notes are in " + shared.Dir
	}
	cutNote := fmt.Sprintf("\n\n(The index was cut to fit. Read it whole in %s.)\n", where)
	rest := p.head + p.profile + p.common + p.userRules
	text := rest + cut(p.index, Budget-size(rest+cutNote)) + cutNote
	// 4. Whatever is left (hundreds of rules), never over the budget.
	if size(text) > Budget {
		end := "\n(Cut to fit. Read the rules and index in " + s.Memory + " and the shared notes in " + shared.Dir + ".)\n"
		text = cut(text, Budget-size(end)) + end
	}
	return text
}

// userRules is the user's rules, shared and of the space: full, with their text, and
// short, one line each with its summary and path.
func userRules(m, shared *memory.Memory) (full, short string) {
	for _, g := range []struct {
		title string
		mem   *memory.Memory
	}{{"Rules for every space", shared}, {"Rules of this space", m}} {
		var whole, listed []string
		for _, r := range g.mem.OfType("rule") {
			whole = append(whole, "### "+r.Header.Title+"\n"+strings.TrimSpace(r.Body)+"\n")
			line := "- " + r.Header.Title
			if r.Header.Summary != "" {
				line += " — " + r.Header.Summary // the gist, so the rule applies without opening it
			}
			listed = append(listed, line+" ("+filepath.Join(g.mem.Dir, filepath.FromSlash(r.Path))+")")
		}
		full += section(g.title, strings.Join(whole, "\n"))
		short += section(g.title+" (each in one line: follow it, and read its file before acting on its subject)", strings.Join(listed, "\n"))
	}
	return full, short
}

// Now gives today's date; tests replace it.
var Now = time.Now

// maxDue is how many due items the session start lists; the rest are counted.
const maxDue = 15

// soonDays is how far ahead the session start looks: what is due in the next week is
// mentioned too, so it can be prepared.
const soonDays = 7

// dueNote lists what comes up today: watch dates and inbox reminders due today or
// tomorrow, and those already past.
func dueNote(b *brain.Brain) string {
	due := b.Reminders(soonDays)
	if len(due) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\n## Due\n\nMention these in your first answer, briefly, before anything else (overdue first, then what comes in the next days):\n")
	for i, d := range due {
		if i == maxDue {
			fmt.Fprintf(&out, "- and %d more: see the pending tool\n", len(due)-maxDue)
			break
		}
		when := "today"
		switch {
		case d.Days == 1:
			when = "tomorrow"
		case d.Days > 1:
			when = fmt.Sprintf("in %d days", d.Days)
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
	return "\n## First session: get to know the user\n\n" +
		"The user's profile is blank, so this is their first session. Before anything else, run this short interview, in the language the user writes in (" + Language(lang) + " if you cannot tell yet), and keep to it.\n\n" +
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
		"Save the profiles and hosts with safety_save: they apply on this whole machine, in every space.\n" +
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
	note := fmt.Sprintf("\nThis session runs in %s, outside every space's folder, so the default space, %s, is loaded. "+
		"Say so in one short line of your first answer, %s.", dir, s.Name, InUserLanguage(c.Language))
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
	reason = shorten(reason, 240)
	return fmt.Sprintf("\nThe last backup of space %s, at %s, failed: %s. %stell them their notes are not backed up, and offer to look into it "+
		"(running `augmented-brain backup` in the space's folder shows the problem).\n", space, when.Format("2006-01-02 15:04"), reason, FirstAnswerLine(lang))
}

// BackupWarningNote tells the assistant that the last backup of the space went through
// but left something out (files too big for it).
func BackupWarningNote(lang, space string, when time.Time, warning string) string {
	warning = shorten(warning, 400)
	return fmt.Sprintf("\nThe last backup of space %s, at %s, went through, but %s. %stell them which files are not backed up and why.\n",
		space, when.Format("2006-01-02 15:04"), warning, FirstAnswerLine(lang))
}

// shorten keeps the start of s that fits in max units, counted as the budget is, and
// marks the cut with "…".
func shorten(s string, max int) string {
	if size(s) <= max {
		return s
	}
	return cut(s, max) + "…"
}

// UpdateNote tells the assistant that a newer augmented brain is out.
func UpdateNote(lang, current, latest string) string {
	return fmt.Sprintf("\naugmented brain %s is available (this machine has %s). %stell them and offer to update it: you can run "+
		"`augmented-brain update` yourself if they say yes. Afterwards they restart Claude to use it.\n", latest, current, FirstAnswerLine(lang))
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
	return fmt.Sprintf("augmented brain: this session runs in %s, which is outside the user's spaces, so none of their memory, rules or tasks is loaded.\n\n"+
		"In your first answer, before anything else, tell the user this %s, in one or two short sentences, and how to fix it: "+
		"start a new session in the work folder of one of their spaces (%s). "+
		"In the Claude desktop app that is \"+ New\", then \"Choose…\" (the folder button under the message box) before writing; in a terminal, `cd <folder> && claude`. "+
		"If they really want to work here, carry on without memory.",
		dir, InTheirLanguage(c.Language), strings.Join(where, "; "))
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
