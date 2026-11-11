package backup

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// copiesFixture is a space with a backup in a throwaway repository.
func copiesFixture(t *testing.T) (*config.Config, *config.Space) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := filepath.Join(root, "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", remote).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Language: "en", Space: "work", WorkFolder: "~/work", Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := c.Spaces["work"]
	if err := Setup(c, s, remote); err != nil {
		t.Fatal(err)
	}
	return c, s
}

func lesson(id, body string) string {
	return "---\nid: " + id + "\ntype: lesson\ntitle: " + id + "\n---\n" + body + "\n"
}

func writeNote(t *testing.T, s *config.Space, id, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(s.Memory, "knowledge", id+".md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readNote(s *config.Space, id string) (string, bool) {
	b, err := os.ReadFile(filepath.Join(s.Memory, "knowledge", id+".md"))
	return string(b), err == nil
}

func backupNow(t *testing.T, c *config.Config, s *config.Space) Copy {
	t.Helper()
	if _, err := Run(c, s, "one"); err != nil {
		t.Fatal(err)
	}
	cs, err := Copies(s.Memory, 1)
	if err != nil || len(cs) != 1 {
		t.Fatalf("copies: %v %v", cs, err)
	}
	return cs[0]
}

func find(ch Changes, name string) *FileChange {
	for i := range ch.Files {
		if ch.Files[i].Name == name {
			return &ch.Files[i]
		}
	}
	return nil
}

// Each copy says what it changed; going back to one leaves the notes as they were, as a
// new copy, and is undone by going back to the copy made just before.
func TestCopiesChangesAndGoingBack(t *testing.T) {
	c, s := copiesFixture(t)
	writeNote(t, s, "lesson-x", lesson("lesson-x", "First."))
	first := backupNow(t, c, s)
	if first.Host != "one" || first.Back != "" {
		t.Errorf("first copy: %+v", first)
	}

	writeNote(t, s, "lesson-x", lesson("lesson-x", "Second."))
	writeNote(t, s, "lesson-y", lesson("lesson-y", "New."))
	second := backupNow(t, c, s)

	ch, err := ChangesOf(s.Memory, second.Rev)
	if err != nil {
		t.Fatal(err)
	}
	x, y := find(ch, "knowledge/lesson-x.md"), find(ch, "knowledge/lesson-y.md")
	if x == nil || x.New || x.Gone || y == nil || !y.New {
		t.Fatalf("what the second copy changed: %+v", ch)
	}
	var del, add bool
	for _, l := range x.Lines {
		del = del || (l.Kind() == "del" && l.Text == "First.")
		add = add || (l.Kind() == "add" && l.Text == "Second.")
	}
	if !del || !add {
		t.Errorf("the lines of lesson-x: %+v", x.Lines)
	}
	for _, f := range ch.Files {
		if f.Name == "MEMORY.md" || strings.HasPrefix(f.Name, MetaDir) {
			t.Errorf("a generated file is shown as a change: %s", f.Name)
		}
	}
	if ch, err := ChangesOf(s.Memory, first.Rev); err != nil || find(ch, "knowledge/lesson-x.md") == nil {
		t.Errorf("the first copy is compared with nothing: %+v %v", ch, err)
	}

	// A note no copy holds yet, and one removed.
	writeNote(t, s, "lesson-z", lesson("lesson-z", "Not backed up."))
	if err := os.Remove(filepath.Join(s.Memory, "knowledge", "lesson-y.md")); err != nil {
		t.Fatal(err)
	}
	back, err := ChangesBack(s.Memory, first.Rev)
	if err != nil {
		t.Fatal(err)
	}
	if z := find(back, "knowledge/lesson-z.md"); z == nil || !z.Gone {
		t.Errorf("going back must say the note with no copy goes: %+v", back)
	}
	if find(back, "knowledge/lesson-x.md") == nil || find(back, "knowledge/lesson-y.md") != nil {
		t.Errorf("what going back would change: %+v", back)
	}
	if out, _ := git(s.Memory, "status", "--porcelain"); !strings.Contains(out, "lesson-z") {
		t.Errorf("looking changed the repository: %q", out)
	}

	if _, err := GoBack(c, s, first.Rev, "one"); err != nil {
		t.Fatal(err)
	}
	if got, _ := readNote(s, "lesson-x"); !strings.Contains(got, "First.") {
		t.Errorf("lesson-x did not go back: %q", got)
	}
	for _, id := range []string{"lesson-y", "lesson-z"} {
		if _, ok := readNote(s, id); ok {
			t.Errorf("%s is still there", id)
		}
	}
	if out, _ := git(s.Memory, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		t.Errorf("going back left changes out of a copy: %q", out)
	}
	if idx, _ := os.ReadFile(filepath.Join(s.Memory, "MEMORY.md")); strings.Contains(string(idx), "lesson-z") {
		t.Error("the index still names a note that went")
	}
	cs, _ := Copies(s.Memory, 5)
	if len(cs) < 4 || cs[0].Back == "" || cs[1].Back != "" {
		t.Fatalf("the copies after going back: %+v", cs)
	}

	// Undone: back to the copy made just before, with the note that had no copy.
	if _, err := GoBack(c, s, cs[1].Rev, "one"); err != nil {
		t.Fatal(err)
	}
	if got, _ := readNote(s, "lesson-z"); !strings.Contains(got, "Not backed up.") {
		t.Errorf("undoing did not bring lesson-z back: %q", got)
	}
	if got, _ := readNote(s, "lesson-x"); !strings.Contains(got, "Second.") {
		t.Errorf("undoing did not bring lesson-x back: %q", got)
	}
}

// Only copies of these notes: not a commit a person made, not anything else.
func TestGoBackOnlyToACopy(t *testing.T) {
	c, s := copiesFixture(t)
	writeNote(t, s, "lesson-x", lesson("lesson-x", "First."))
	backupNow(t, c, s)
	writeNote(t, s, "lesson-x", lesson("lesson-x", "By hand."))
	if _, err := git(s.Memory, append(ident, "commit", "-q", "-am", "a change of mine")...); err != nil {
		t.Fatal(err)
	}
	mine, _ := git(s.Memory, "rev-parse", "HEAD")
	for _, rev := range []string{strings.TrimSpace(mine), "HEAD", "--all", "0000000", "../x", ""} {
		if _, err := GoBack(c, s, rev, "one"); !errors.Is(err, ErrNotACopy) {
			t.Errorf("going back to %q: %v", rev, err)
		}
		if _, err := ChangesOf(s.Memory, rev); !errors.Is(err, ErrNotACopy) {
			t.Errorf("changes of %q: %v", rev, err)
		}
	}
	if got, _ := readNote(s, "lesson-x"); !strings.Contains(got, "By hand.") {
		t.Errorf("a refused going back changed a note: %q", got)
	}
}

// A copy whose notes would not load is refused, and the notes stay as they were.
func TestGoBackRefusesNotesNotValid(t *testing.T) {
	c, s := copiesFixture(t)
	writeNote(t, s, "lesson-x", lesson("lesson-x", "Fine."))
	backupNow(t, c, s)
	writeNote(t, s, "lesson-bad", "no header at all\n")
	if _, err := git(s.Memory, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	if _, err := git(s.Memory, append(ident, "commit", "-q", "-m", "backup other 2026-01-01 10:00")...); err != nil {
		t.Fatal(err)
	}
	bad, _ := git(s.Memory, "rev-parse", "HEAD")
	if err := os.Remove(filepath.Join(s.Memory, "knowledge", "lesson-bad.md")); err != nil {
		t.Fatal(err)
	}
	writeNote(t, s, "lesson-x", lesson("lesson-x", "Now."))
	backupNow(t, c, s)
	before, _ := git(s.Memory, "rev-parse", "HEAD")

	_, err := GoBack(c, s, strings.TrimSpace(bad), "one")
	var nv *NotValidError
	if !errors.As(err, &nv) || len(nv.Problems) == 0 {
		t.Fatalf("going back to notes not valid: %v", err)
	}
	if after, _ := git(s.Memory, "rev-parse", "HEAD"); after != before {
		t.Errorf("a refused going back left a commit")
	}
	if out, _ := git(s.Memory, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		t.Errorf("a refused going back left changes: %q", out)
	}
	if got, _ := readNote(s, "lesson-x"); !strings.Contains(got, "Now.") {
		t.Errorf("lesson-x: %q", got)
	}
}

func TestParseDiffCaps(t *testing.T) {
	var b strings.Builder
	for i := 0; i < MaxChangedFiles+3; i++ {
		b.WriteString("diff --git n n\nnew file mode 100644\n--- /dev/null\n+++ n\n@@ -0,0 +1 @@\n+x\n")
	}
	b.WriteString("diff --git big big\n--- big\n+++ big\n@@ -1 +1,300 @@\n")
	for i := 0; i < 300; i++ {
		b.WriteString("+line\n")
	}
	ch := parseDiff(b.String())
	if len(ch.Files) != MaxChangedFiles || ch.More != 4 {
		t.Errorf("files %d, more %d", len(ch.Files), ch.More)
	}
	ch = parseDiff("diff --git big big\n--- big\n+++ big\n@@ -1 +1,300 @@\n" + strings.Repeat("+l\n", 300) +
		"diff --git p.png p.png\nBinary files /dev/null and p.png differ\n")
	if n := len(ch.Files[0].Lines); n != maxChangedLines+1 || ch.Files[0].Lines[n-1].Op != '~' {
		t.Errorf("a long change is not cut: %d lines", n)
	}
	if !ch.Files[1].Binary || ch.Files[1].Name != "p.png" {
		t.Errorf("binary: %+v", ch.Files[1])
	}
}
