package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/memory"
)

// A review that cannot be saved takes nothing out of the inbox (N-03).
func TestFinishReviewRefusedKeepsCaptures(t *testing.T) {
	b := open(t, "en")
	for _, c := range []string{"idea A that matters", "idea B that matters"} {
		if err := b.Capture(c, ""); err != nil {
			t.Fatal(err)
		}
	}
	processed := []string{"idea A that matters", "idea B that matters"}
	token := "ghp_" + strings.Repeat("a1B2c3D4e5", 4)[:36]
	for _, summary := range []string{"   ", "Decided everything; the key was " + token} {
		_, _, err := b.FinishReview(summary, processed)
		if err == nil {
			t.Fatalf("FinishReview(%q) saved", summary)
		}
		if strings.TrimSpace(summary) != "" && !strings.Contains(err.Error(), "Secrets never go into notes") {
			t.Fatalf("refused for another reason: %v", err)
		}
		in, err := b.Inbox()
		if err != nil || len(in) != 2 {
			t.Fatalf("after a refused review (%q) the inbox has %d captures (%v), want 2", summary, len(in), err)
		}
	}
	if got := b.Mem.OfType("review"); len(got) != 0 {
		t.Fatalf("a refused review was saved: %v", got)
	}
}

// A review that is saved holds the captures it took out, and only then are they gone.
func TestFinishReviewWritesCaptures(t *testing.T) {
	b := open(t, "en")
	_ = b.Capture("idea one", "")
	_ = b.Capture("idea two", "")
	_ = b.Capture("idea three", "")
	n, missing, err := b.FinishReview("Went through it", []string{"idea one", "idea two", "not there"})
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 1 || missing[0] != "not there" {
		t.Errorf("missing = %v", missing)
	}
	for _, want := range []string{"Went through it", "## Captures taken out of the inbox", "— idea one", "— idea two"} {
		if !strings.Contains(n.Body, want) {
			t.Errorf("review body lacks %q:\n%s", want, n.Body)
		}
	}
	if strings.Contains(n.Body, "idea three") {
		t.Errorf("review names a capture it did not take:\n%s", n.Body)
	}
	in, _ := b.Inbox()
	if len(in) != 1 || in[0].Text != "idea three" {
		t.Errorf("inbox after the review: %+v", in)
	}
}

// A review with no inbox at all is saved, naming what it could not find.
func TestFinishReviewNoInbox(t *testing.T) {
	b := open(t, "en")
	_ = os.Remove(b.inboxPath())
	_, missing, err := b.FinishReview("Nothing to see", []string{"gone already"})
	if err != nil || len(missing) != 1 {
		t.Fatalf("missing=%v err=%v", missing, err)
	}
}

// A refused change to an oversized task writes no archive of its journal (N-15).
func TestRefusedChangeLeavesNoArchive(t *testing.T) {
	b := open(t, "en")
	n, err := b.NewTask(NewTaskInput{Title: "Big", Summary: "s"})
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(b.Mem.Dir, n.Path)
	data, _ := os.ReadFile(p)
	var sb strings.Builder
	sb.Write(data)
	for i := range 400 {
		fmt.Fprintf(&sb, "- 2026-09-%02d — entry %d %s\n", 1+i%28, i, strings.Repeat("x", 80))
	}
	if err := os.WriteFile(p, []byte(sb.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, err := open2(t, b).Update(UpdateInput{ID: n.Header.ID, Status: "bogus"}); err == nil {
			t.Fatal("a bogus status was accepted")
		}
	}
	files, _ := filepath.Glob(filepath.Join(b.Mem.Dir, filepath.FromSlash(memory.JournalArchiveDir), "*"))
	if len(files) != 0 {
		t.Errorf("refused changes left %d archive files: %v", len(files), files)
	}
}

// A refused change to a rule keeps no copy of it in archive/ (N-15).
func TestRefusedNoteUpdateLeavesNoCopy(t *testing.T) {
	b := open(t, "en")
	r, err := b.NewRule("Be kind", "always", "Be kind.\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.UpdateNote(r.Header.ID, "rule", "", "", "", "the token "+"ghp_"+strings.Repeat("Zx9", 12)); err == nil || !strings.Contains(err.Error(), "Secrets never go into notes") {
		t.Fatalf("a secret was not refused as one: %v", err)
	}
	files, _ := filepath.Glob(filepath.Join(b.Mem.Dir, "archive", "*", "*"))
	if len(files) != 0 {
		t.Errorf("a refused change left copies: %v", files)
	}
}

// An index that cannot be written does not turn a saved note into an error (N-15).
func TestIndexFailureIsNotAnError(t *testing.T) {
	b := open(t, "en")
	n, err := b.NewTask(NewTaskInput{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	idx := filepath.Join(b.Mem.Dir, "MEMORY.md")
	_ = os.Remove(idx)
	if err := os.Mkdir(idx, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Journal(n.Header.ID, "one entry"); err != nil {
		t.Fatalf("journal: %v", err)
	}
	if b.IndexProblem == nil {
		t.Error("the index problem was not told")
	}
	got, _ := os.ReadFile(filepath.Join(b.Mem.Dir, n.Path))
	if strings.Count(string(got), "one entry") != 1 {
		t.Errorf("the entry is not there once:\n%s", got)
	}
}

// A closed task is not parked, nor are its steps changed (N-16).
func TestClosedTaskStaysClosed(t *testing.T) {
	b := open(t, "en")
	n, _ := b.NewTask(NewTaskInput{Title: "x"})
	if _, err := b.AddStep(n.Header.ID, "first"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Close(n.Header.ID, "done and delivered"); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(b.Mem.Dir, n.Path)
	before, _ := os.ReadFile(p)
	if _, err := b.Park(ParkInput{ID: n.Header.ID, Next: "later", WaitingOn: "me"}); err == nil {
		t.Error("a closed task was parked")
	}
	if _, err := b.Park(ParkInput{ID: n.Header.ID, Next: "later", Essentials: "all new"}); err == nil {
		t.Error("a closed task was parked with new essentials")
	}
	if _, err := b.AddStep(n.Header.ID, "new step"); err == nil {
		t.Error("a step was added to a closed task")
	}
	if _, err := b.Step(StepInput{ID: n.Header.ID, Number: 1, Done: true}); err == nil {
		t.Error("a step of a closed task was ticked")
	}
	after, _ := os.ReadFile(p)
	if string(before) != string(after) {
		t.Errorf("the closed task changed:\n%s", after)
	}
}

// A capture after a last line with no newline keeps a line of its own (N-17).
func TestCaptureAfterNoTrailingNewline(t *testing.T) {
	b := open(t, "en")
	p := b.inboxPath()
	if err := os.WriteFile(p, []byte("# Inbox\n\n- 2026-10-01 — written by hand, no newline at the end"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := b.Capture("second capture", ""); err != nil {
		t.Fatal(err)
	}
	in, _ := b.Inbox()
	if len(in) != 2 || in[1].Text != "second capture" {
		t.Errorf("captures read back: %+v", in)
	}
}

// Taking a name out of a list written by hand with quotes keeps the others (N-19).
func TestSecretsFileQuotes(t *testing.T) {
	b := open(t, "en")
	path := filepath.Join(b.ConfigDir, "secrets.toml")
	if err := os.WriteFile(path, []byte("secrets = [\n  'svc/a: the \"main\" one',\n  \"svc/b: other\",\n  \"svc/c: third\",\n]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := b.ForgetSecretName("svc", "b"); err != nil {
		t.Fatal(err)
	}
	want := []string{`svc/a: the "main" one`, "svc/c: third"}
	if got := SecretList(b.ConfigDir); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("after forget: %q, want %q", got, want)
	}
	if _, err := b.SaveSecretName(SecretName{Service: "svc", Name: "d", What: "new"}); err != nil {
		t.Fatal(err)
	}
	if got := SecretList(b.ConfigDir); len(got) != 3 {
		t.Errorf("after save: %q", got)
	}
}

// A secrets file that cannot be read is not written over (N-19).
func TestSecretsFileUnreadableNotOverwritten(t *testing.T) {
	b := open(t, "en")
	path := filepath.Join(b.ConfigDir, "secrets.toml")
	broken := "secrets = [\n  \"svc/a: one\",\n  \"svc/b: broken \"quote\",\n]\n"
	if err := os.WriteFile(path, []byte(broken), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SaveSecretName(SecretName{Service: "svc", Name: "d", What: "new"}); err == nil {
		t.Error("saved over a file it could not read")
	}
	if err := b.ForgetSecretName("svc", "a"); err == nil {
		t.Error("forgot over a file it could not read")
	}
	if got, _ := os.ReadFile(path); string(got) != broken {
		t.Errorf("the file changed:\n%s", got)
	}
}

func TestTOMLString(t *testing.T) {
	for in, want := range map[string]string{
		`plain`:     `"plain"`,
		`a "q" \ b`: `"a \"q\" \\ b"`,
		"tab\there": `"tab\u0009here"`,
	} {
		if got := tomlString(in); got != want {
			t.Errorf("tomlString(%q) = %s, want %s", in, got, want)
		}
	}
}
