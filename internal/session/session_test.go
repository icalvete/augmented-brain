package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/rules"
	"github.com/icalvete/augmented-brain/internal/setup"
)

func space(t *testing.T, lang string) (*config.Config, *config.Space) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfgDir, setup.Options{Language: lang, Space: "work", WorkFolder: work, Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	return c, c.Spaces["work"]
}

func TestContext(t *testing.T) {
	c, s := space(t, "es")
	rule := "---\nid: rule-db\ntype: rule\ntitle: Las bases de datos no se tocan\n---\nNunca reiniciar db01 sin permiso.\n"
	if err := os.WriteFile(filepath.Join(s.Memory, "rules", "rule-db.md"), []byte(rule), 0o644); err != nil {
		t.Fatal(err)
	}
	text := Context(c, s)
	for _, want := range []string{
		`space "work"`,
		"## Who the user is", "Idioma: español.", // the profile body, without its header
		"### Cómo trabajamos", "### Seguridad", // common rules in the user's language
		"### Las bases de datos no se tocan\nNunca reiniciar db01 sin permiso.",
		"# Índice de la memoria", "[General](areas/area-general.md)",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("context lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "type: profile") {
		t.Error("a header leaked into the context")
	}
}

// The shared notes reach every space: their rules whole, their knowledge listed, and a
// space note may link to them.
func TestContextShared(t *testing.T) {
	c, s := space(t, "en")
	files := map[string]string{
		filepath.Join(c.SharedDir(), "rules", "rule-brief.md"):             "---\nid: rule-brief\ntype: rule\ntitle: Answer briefly\n---\nShort answers, one idea each.\n",
		filepath.Join(c.SharedDir(), "knowledge", "lesson-small-steps.md"): "---\nid: lesson-small-steps\ntype: lesson\ntitle: Small steps win\nsummary: one change at a time\n---\n",
		filepath.Join(c.SharedDir(), "tasks", "task-lost.md"):              "---\nid: task-lost\ntype: task\ntitle: Lost\nstatus: parked\nwaiting_on: me\nnext: x\n---\n",
		filepath.Join(s.Memory, "knowledge", "ref-deploy.md"):              "---\nid: ref-deploy\ntype: reference\ntitle: Deploy\nrelated: [lesson-small-steps]\n---\n",
	}
	for path, text := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	text := Context(c, s)
	for _, want := range []string{
		"## Rules for every space", "### Answer briefly\nShort answers, one idea each.",
		"## Knowledge for every space", "In " + filepath.Join(c.SharedDir(), "knowledge") + ", each in <id>.md:\n- Small steps win (lesson-small-steps)",
		"a task belongs to one space", // a task in the shared folder is reported, not loaded
	} {
		if !strings.Contains(text, want) {
			t.Errorf("context lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "lesson-small-steps\" does not exist") {
		t.Errorf("a link to a shared note is reported as broken:\n%s", text)
	}
}

func TestContextBudget(t *testing.T) {
	c, s := space(t, "en")
	long := strings.Repeat("Explain every step before doing it. ", 60)
	for i := 0; i < 8; i++ {
		rule := fmt.Sprintf("---\nid: rule-%d\ntype: rule\ntitle: Rule %d\nsummary: step %d first\n---\n%s\n", i, i, i, long)
		if err := os.WriteFile(filepath.Join(s.Memory, "rules", fmt.Sprintf("rule-%d.md", i)), []byte(rule), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	text := Context(c, s)
	if n := utf8.RuneCountInString(text); n > Budget {
		t.Fatalf("context is %d characters, over the %d budget", n, Budget)
	}
	if !strings.Contains(text, "read its file before acting") || !strings.Contains(text, "- Rule 3 — step 3 first (") || !strings.Contains(text, "rule-3.md)") {
		t.Errorf("long rules are not listed with their summary and path:\n%s", text)
	}
	if strings.Contains(text, long) {
		t.Error("a long rule body is still inlined")
	}

	// A huge index is cut, with a note saying where the whole one is.
	for i := 0; i < 400; i++ {
		task := fmt.Sprintf("---\nid: task-%03d\ntype: task\ntitle: Task number %03d with a long title\nstatus: plan\nwaiting_on: me\nnext: do the next thing for task %03d\n---\n", i, i, i)
		if err := os.WriteFile(filepath.Join(s.Memory, "tasks", fmt.Sprintf("task-%03d.md", i)), []byte(task), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	text = Context(c, s)
	if n := utf8.RuneCountInString(text); n > Budget || !strings.Contains(text, "The index was cut") {
		t.Errorf("big index: %d characters, cut note present: %v", n, strings.Contains(text, "The index was cut"))
	}
}

// Found in the review before 0.1: a huge profile, hundreds of space rules and emojis
// must never push the session context over the budget.
func TestContextNeverOverBudget(t *testing.T) {
	c, s := space(t, "en")
	profile := "---\nid: profile\ntype: profile\ntitle: Profile\n---\n" + strings.Repeat("I like short answers 🙂. ", 600) + "\n"
	if err := os.WriteFile(filepath.Join(c.Dir, "profile.md"), []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 300; i++ {
		rule := fmt.Sprintf("---\nid: rule-%03d\ntype: rule\ntitle: A rule with a fairly long title number %03d\n---\nbody\n", i, i)
		if err := os.WriteFile(filepath.Join(s.Memory, "rules", fmt.Sprintf("rule-%03d.md", i)), []byte(rule), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	text := Context(c, s)
	if n := size(text); n > Budget {
		t.Fatalf("context is %d units, over %d", n, Budget)
	}
	if !strings.Contains(text, "The profile was cut to fit") || !strings.Contains(text, "### How we work") {
		t.Errorf("the profile note or the common rules are missing")
	}
	if utf8.RuneCountInString(text) > size(text) {
		t.Error("size must never count less than the characters")
	}
}

// While the profile is blank the session asks the assistant to meet the user first; once
// it is filled in, it does not.
func TestFirstSession(t *testing.T) {
	c, s := space(t, "es")
	if text := Context(c, s); !strings.Contains(text, "## First session: get to know the user") || !strings.Contains(text, "the language the user writes in (Spanish if you cannot tell yet)") {
		t.Errorf("a blank profile must start with the first-session note:\n%s", text)
	}
	profile := filepath.Join(c.Dir, "profile.md")
	data, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	filled := strings.Replace(string(data), "- Nombre:", "- Nombre: Ana", 1)
	if err := os.WriteFile(profile, []byte(filled), 0o644); err != nil {
		t.Fatal(err)
	}
	if text := Context(c, s); strings.Contains(text, "First session") {
		t.Error("a filled profile must not ask to meet the user again")
	}
}

// Outside every space the assistant must tell the user, with the folders to open.
func TestNoSpace(t *testing.T) {
	c, s := space(t, "es")
	text := NoSpace(c, "/tmp/scratch")
	for _, want := range []string{"/tmp/scratch", "outside the user's spaces", "Spanish if you cannot tell", "work: " + s.WorkFolders[0], "Choose…"} {
		if !strings.Contains(text, want) {
			t.Errorf("no-space text lacks %q:\n%s", want, text)
		}
	}
}

func TestContextDueAndReview(t *testing.T) {
	c, s := space(t, "en")
	Now = func() time.Time { return time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC) }
	defer func() { Now = time.Now }()
	b, err := brain.Open(c, s)
	if err != nil {
		t.Fatal(err)
	}
	b.Now = func() time.Time { return time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC) }
	if _, err := b.SaveProfile(brain.ProfileInput{Name: "Dana", About: "Runs a bakery's IT."}); err != nil {
		t.Fatal(err)
	}
	task, _ := b.NewTask(brain.NewTaskInput{Title: "Certificates"})
	if _, err := b.AddWatch(task.Header.ID, "2026-10-18", "renewed?"); err != nil {
		t.Fatal(err)
	}
	if err := b.Capture("call the insurance", "2026-10-21"); err != nil {
		t.Fatal(err)
	}
	if err := b.Capture("renew the domain", "2026-10-25"); err != nil {
		t.Fatal(err)
	}
	if err := b.Capture("too far to mention", "2026-10-30"); err != nil {
		t.Fatal(err)
	}

	// A file that is not a valid note is named, and the rest keeps working.
	if err := os.WriteFile(filepath.Join(s.Memory, "tasks", "task-broken.md"), []byte("no header here\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	text := Context(c, s)
	for _, want := range []string{
		"1 files of this space are not valid notes",
		"tasks/task-broken.md: missing header",
		"## Due",
		"- 2026-10-18 (2 days overdue): renewed? [" + task.Header.ID + "]",
		"- 2026-10-21 (tomorrow): call the insurance [inbox]",
		"- 2026-10-25 (in 5 days): renew the domain [inbox]",
		"The weekly review is due: 10 days since the last one, with 3 captures in the inbox",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("context lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "too far to mention") {
		t.Errorf("a reminder 10 days ahead is mentioned:\n%s", text)
	}

	// A session opened for the task hears only that task's dates: not the inbox's, not
	// the weekly review. One opened for another task hears no dates at all.
	text = ContextFor(c, s, "", task.Header.ID)
	if !strings.Contains(text, "renewed? ["+task.Header.ID+"]") || !strings.Contains(text, "opened for task "+task.Header.ID) {
		t.Errorf("the task's session lacks its own date:\n%s", text)
	}
	for _, not := range []string{"call the insurance", "weekly review is due"} {
		if strings.Contains(text, not) {
			t.Errorf("the task's session hears %q:\n%s", not, text)
		}
	}
	if text := PartFor(c, s, "", "task-other", 1); strings.Contains(text, "## Due") || strings.Contains(text, "weekly review is due") {
		t.Errorf("another task's session hears this one's dates:\n%s", text)
	}

	// A blank profile means the first session: the interview, nothing else.
	c2, s2 := space(t, "en")
	if text := Context(c2, s2); strings.Contains(text, "## Due") || strings.Contains(text, "weekly review is due") {
		t.Errorf("the first session must only run the interview:\n%s", text)
	}
}

// shorten counts as the budget does (UTF-16 units), never splits a character, and
// marks only a real cut.
func TestShorten(t *testing.T) {
	if got := shorten(strings.Repeat("ñ", 200), 240); got != strings.Repeat("ñ", 200) {
		t.Errorf("fits in units, longer in bytes: %q", got)
	}
	got := shorten(strings.Repeat("😀", 5), 5)
	if got != "😀😀…" || !utf8.ValidString(got) {
		t.Errorf("emoji count twice: %q", got)
	}
}

// Found in the rehearsal of 0.5.4: one hook's text could not hold the index of a space
// with 43 tasks. In parts, every task of such a space arrives, each part within Budget.
func TestParts(t *testing.T) {
	c, s := space(t, "en")
	long := strings.Repeat("Explain every step before doing it. ", 60)
	for i := 0; i < 6; i++ {
		rule := fmt.Sprintf("---\nid: rule-%d\ntype: rule\ntitle: Rule %d\nsummary: step %d first\n---\n%s\n", i, i, i, long)
		if err := os.WriteFile(filepath.Join(s.Memory, "rules", fmt.Sprintf("rule-%d.md", i)), []byte(rule), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write := func(n int) {
		for i := 0; i < n; i++ {
			task := fmt.Sprintf("---\nid: task-%03d\ntype: task\ntitle: Task number %03d with a long title\nstatus: plan\nwaiting_on: me\nnext: do the next thing for task %03d, which takes a sentence to say\n---\n", i, i, i)
			if err := os.WriteFile(filepath.Join(s.Memory, "tasks", fmt.Sprintf("task-%03d.md", i)), []byte(task), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	write(80)
	all := ""
	for n := 1; n <= Parts; n++ {
		p := Part(c, s, "NOTE FOR TODAY\n", n)
		if size(p) > Budget {
			t.Errorf("part %d is %d units, over %d", n, size(p), Budget)
		}
		all += p
	}
	for _, want := range []string{"NOTE FOR TODAY", "Who the user is", "### How we work", "Task number 000", "Task number 079", "### Rule 5"} {
		if !strings.Contains(all, want) {
			t.Errorf("the parts lack %q", want)
		}
	}
	if strings.Contains(all, "was cut") || strings.Count(all, "NOTE FOR TODAY") != 1 {
		t.Errorf("80 tasks and 6 rules should fit whole, the note once:\n%s", all)
	}
	// Found in the rehearsal of 0.5.10: empty parts made the assistant wait for parts
	// that were not coming. The parts with something count only themselves, 1 to total.
	full, seen := 0, map[string]bool{}
	for n := 1; n <= Parts; n++ {
		if Part(c, s, "", n) != "" {
			full++
		}
	}
	for n := 2; n <= Parts; n++ {
		if p := Part(c, s, "", n); p != "" {
			tag := strings.SplitN(p, "\n", 2)[0]
			if !strings.HasSuffix(tag, fmt.Sprintf(" of %d", full)) || seen[tag] {
				t.Errorf("part %d says %q, with %d parts that have something", n, tag, full)
			}
			seen[tag] = true
		}
	}
	if !strings.Contains(Part(c, s, "", 1), fmt.Sprintf("This is part 1 of %d", full)) {
		t.Error("part 1 does not say how many parts there are")
	}

	// A rule's History stays in its file; rules beyond three parts go one line each.
	hist := "---\nid: rule-00-hist\ntype: rule\ntitle: With history\nsummary: keep it short\n---\nKeep answers short.\n\n## History\nAsked on a Monday.\n"
	if err := os.WriteFile(filepath.Join(s.Memory, "rules", "rule-00-hist.md"), []byte(hist), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 6; i < 30; i++ {
		rule := fmt.Sprintf("---\nid: rule-%d\ntype: rule\ntitle: Rule %d\nsummary: step %d first\n---\n%s\n", i, i, i, long)
		if err := os.WriteFile(filepath.Join(s.Memory, "rules", fmt.Sprintf("rule-%d.md", i)), []byte(rule), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rulesText := ""
	for n := 2 + indexParts; n <= Parts; n++ {
		p := Part(c, s, "", n)
		if size(p) > Budget {
			t.Errorf("rules part %d is %d units", n, size(p))
		}
		rulesText += p
	}
	if !strings.Contains(rulesText, "Keep answers short.") || strings.Contains(rulesText, "Asked on a Monday") || !strings.Contains(rulesText, "Its History is in") {
		t.Errorf("the History section is not left in the file:\n%s", rulesText)
	}
	if !strings.Contains(rulesText, "did not fit whole") || strings.Contains(rulesText, "more rules did not fit") {
		t.Errorf("rules beyond the parts are not listed in one line:\n%s", rulesText)
	}
	for i := 0; i < 30; i++ {
		if !strings.Contains(rulesText, fmt.Sprintf("### Rule %d\n", i)) && !strings.Contains(rulesText, fmt.Sprintf("- Rule %d — step %d first (", i, i)) {
			t.Errorf("rule %d is neither whole nor in a line", i)
		}
	}

	// A space too big even for the parts: the index says where the whole one is.
	write(400)
	if p := Part(c, s, "", 1+indexParts); size(p) > Budget || !strings.Contains(p, "The index was cut") {
		t.Errorf("400 tasks: %d units, cut note: %v", size(p), strings.Contains(p, "The index was cut"))
	}
}

func TestLines(t *testing.T) {
	text := "a\nbb\nccc\n" + strings.Repeat("x", 12) + "\n"
	got := lines(text, 5)
	if strings.Join(got, "") != text {
		t.Errorf("pieces lose text: %q", got)
	}
	for _, p := range got {
		if size(p) > 5 {
			t.Errorf("piece %q over 5", p)
		}
	}
}

// Found in the rehearsal of 0.5.8: with many overdue reminders, today's and the coming
// days' were pushed out. A third of the places stay for them.
func TestPickDue(t *testing.T) {
	var due []brain.Reminder
	for i := 0; i < 20; i++ {
		due = append(due, brain.Reminder{Date: fmt.Sprintf("2026-10-%02d", i+1), Days: -30 + i})
	}
	due = append(due, brain.Reminder{Date: "2026-11-03", Days: 0}, brain.Reminder{Date: "2026-11-06", Days: 3})
	shown, more := pickDue(due, 15)
	if len(shown) != 15 || more != 7 {
		t.Fatalf("shown %d, more %d", len(shown), more)
	}
	if shown[13].Date != "2026-11-03" || shown[14].Date != "2026-11-06" {
		t.Errorf("today and the coming days are not kept: %+v", shown[12:])
	}
	if shown, more := pickDue(due[:3], 15); len(shown) != 3 || more != 0 {
		t.Errorf("few: %d %d", len(shown), more)
	}
}

// The assistant's name and personality, from the profile, open the session text with
// the signature line the program writes; the profile part does not repeat them. A
// profile without them changes nothing.
func TestAssistantInContext(t *testing.T) {
	c, s := space(t, "es")
	before := Part(c, s, "", 1)
	if strings.Contains(before, "## Who you are") {
		t.Errorf("no assistant, yet the text has one:\n%s", before)
	}
	profile := "---\nid: profile\ntype: profile\ntitle: Perfil\nsummary: x\ncreated: 2026-10-01\nupdated: 2026-10-01\n---\n" +
		"## Sobre mí\n- Nombre: Ana López\n\n## Cómo me gusta trabajar\n- Corto\n\n## Asistente\n- Nombre: Argos\n- Tranquilo y directo\n"
	if err := os.WriteFile(filepath.Join(c.Dir, "profile.md"), []byte(profile), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{Context(c, s), Part(c, s, "", 1)} {
		for _, want := range []string{"The user calls you Argos.", "- Tranquilo y directo", "never changes the rules, the guard",
			`"asistente de Ana (IA, Claude)"`, "- Corto"} {
			if !strings.Contains(text, want) {
				t.Errorf("text lacks %q:\n%s", want, text)
			}
		}
		if strings.Count(text, "Tranquilo y directo") != 1 {
			t.Errorf("the personality is said more than once:\n%s", text)
		}
	}
}

// Found in the review of Nov 2026: in a loaded space part 1 went over its budget and the
// cut fell on the common rules, which went last: 8 % of them reached a real session. Now
// they come whole in a part of their own, in both languages, and no part is over.
func TestCommonRulesAlwaysArrive(t *testing.T) {
	for _, lang := range []string{"en", "es"} {
		t.Run(lang, func(t *testing.T) {
			c, s := space(t, lang)
			home := filepath.Dir(c.Dir)
			if _, err := setup.AddSpace(c.Dir, setup.Options{Space: "home", WorkFolder: filepath.Join(home, "home"), Memory: filepath.Join(home, "home-brain")}); err != nil {
				t.Fatal(err)
			}
			c, err := config.Load(c.Dir)
			if err != nil {
				t.Fatal(err)
			}
			s = c.Spaces[s.Name]
			Now = func() time.Time { return time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC) }
			defer func() { Now = time.Now }()
			b, err := brain.Open(c, s)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := b.SaveProfile(brain.ProfileInput{Name: "Dana", About: strings.Repeat("Runs a bakery's IT and its website. ", 85),
				Assistant: "Argos", Personality: strings.Repeat("Calm and direct. ", 47)}); err != nil {
				t.Fatal(err)
			}
			long := func(what string, i int) string {
				return fmt.Sprintf("%s %d: %s", what, i, strings.Repeat("a long sentence of words ", 16))
			}
			var inbox strings.Builder
			inbox.WriteString("# Inbox\n\n")
			for i := range 10 {
				fmt.Fprintf(&inbox, "- 2026-10-1%d — %s (from home)\n", i, long("Sent", i))
			}
			for i := range 15 {
				fmt.Fprintf(&inbox, "- 2026-10-0%d — %s (remind: 2026-10-%d)\n", i%9+1, long("Due", i), 14+i%10)
			}
			if err := os.WriteFile(filepath.Join(s.Memory, "inbox.md"), []byte(inbox.String()), 0o600); err != nil {
				t.Fatal(err)
			}

			note := "\n" + strings.Repeat("A note from the program for this session. ", 28) + "\n"
			all := ""
			for n := 1; n <= Parts; n++ {
				p := Part(c, s, note, n)
				if size(p) > Budget {
					t.Errorf("part %d is %d units, over %d", n, size(p), Budget)
				}
				all += p
			}
			for _, l := range strings.Split(rules.Common(lang), "\n") {
				if !strings.Contains(all, l) {
					t.Errorf("the common rules lose a line: %q", l)
				}
			}
			for _, want := range []string{"Sent 0:", "Due 0:", "…", "Calm and direct.", "Runs a bakery"} {
				if !strings.Contains(all, want) {
					t.Errorf("the parts lack %q", want)
				}
			}
		})
	}
}
