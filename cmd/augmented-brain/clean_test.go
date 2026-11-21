package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/leftovers"
	"github.com/icalvete/augmented-brain/internal/remote"
)

// clean removes the background sessions left over (21-nov): those of tasks parked,
// put off, closed or gone with the rest, after one yes (or --yes); those no task
// opened one by one, never with --yes. A session of a task going on, one answering
// now, and one no task opened that has not ended stay.
func TestCleanSpareSessions(t *testing.T) {
	h, cfg := home(t)
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := brain.Open(c, c.Spaces["trabajo"])
	if err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"Parked", "Going"} {
		if _, err := b.NewTask(brain.NewTaskInput{Title: title}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := b.Park(brain.ParkInput{ID: "task-parked", Next: "later"}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(cfg, "state", "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for task, id := range map[string]string{"task-parked": "aaa11111", "task-going": "bbb22222", "task-gone": "ccc33333"} {
		if err := os.WriteFile(filepath.Join(dir, task+".json"), []byte(`{"id":"`+id+`","task":"`+task+`","name":"box · `+task+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	oldEnv := cleanEnv
	t.Cleanup(func() { cleanEnv = oldEnv })
	cleanEnv = func() (leftovers.Env, bool) {
		return leftovers.Env{ClaudeJSON: filepath.Join(h, "none.json"), ClaudeDir: filepath.Join(h, ".claude")}, true
	}
	var calls []string
	agents := `[
	 {"id":"aaa11111","kind":"background","state":"done"},
	 {"id":"bbb22222","kind":"background","pid":7,"status":"idle","state":"working"},
	 {"id":"ccc33333","kind":"background","pid":8,"status":"busy","state":"working"},
	 {"id":"ddd44444","name":"vigilante","cwd":"/w/repo/.claude/worktrees/fix","kind":"background","state":"stopped"},
	 {"id":"eee55555","name":"other","cwd":"/w","kind":"background","state":"done"},
	 {"id":"fff66666","name":"still","cwd":"/w","kind":"background","pid":9,"status":"idle","state":"blocked"},
	 {"name":"desk","kind":"interactive","pid":10,"status":"idle"}]`
	fake := remoteSystem
	remoteSystem = func() (remote.System, error) {
		s, err := fake()
		s.Run = func(name string, args ...string) (string, error) {
			calls = append(calls, strings.Join(args, " "))
			if len(args) > 0 && args[0] == "agents" {
				return agents, nil
			}
			return "", nil
		}
		return s, err
	}

	// With --yes: the task's session goes (the one answering stays), the others are
	// only listed.
	var out, errOut bytes.Buffer
	if code := run([]string{"clean", "--yes"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("clean --yes: %d\n%s%s", code, out.String(), errOut.String())
	}
	got := out.String()
	for _, want := range []string{"«box · task-parked», de task-parked (aparcada)", "«vigilante» (ddd44444), en ", "fix, parada · su worktree", "su worktree también se va", "«other» (eee55555)", "Ninguna de estas se va con --yes", "Sesiones de tareas cerradas: 1."} {
		if !strings.Contains(got, want) {
			t.Errorf("lacks %q:\n%s", want, got)
		}
	}
	for _, not := range []string{"task-going", "task-gone", "still", "desk"} {
		if strings.Contains(got, not) {
			t.Errorf("names %q:\n%s", not, got)
		}
	}
	all := strings.Join(calls, "\n")
	if !strings.Contains(all, "rm aaa11111") || strings.Contains(all, "rm ddd44444") || strings.Contains(all, "rm eee55555") || strings.Contains(all, "rm ccc33333") {
		t.Errorf("calls:\n%s", all)
	}
	if _, err := os.Stat(filepath.Join(dir, "task-parked.json")); !os.IsNotExist(err) {
		t.Errorf("the parked task's record stays: %v", err)
	}

	// Without --yes, each other one is asked: yes to the first, no to the second.
	agents = strings.Replace(agents, `{"id":"aaa11111","kind":"background","state":"done"},`, "", 1)
	calls = nil
	out.Reset()
	errOut.Reset()
	if code := cmdClean(nil, strings.NewReader("sí\nno\n"), &out, &errOut); code != 0 {
		t.Fatalf("clean: %d\n%s%s", code, out.String(), errOut.String())
	}
	all = strings.Join(calls, "\n")
	if !strings.Contains(all, "rm ddd44444") || strings.Contains(all, "rm eee55555") || !strings.Contains(out.String(), "He quitado «vigilante»") {
		t.Errorf("calls:\n%s\n%s%s", all, out.String(), errOut.String())
	}
}
