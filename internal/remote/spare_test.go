package remote

import (
	"errors"
	"strings"
	"testing"
)

// RemoveSpare says why Claude refused to remove a session (a worktree with work of its
// own), in Claude's words, after waking its daemon once.
func TestRemoveSpareSaysWhy(t *testing.T) {
	var calls []string
	s := System{GOOS: "linux",
		LookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
		Run: func(name string, args ...string) (string, error) {
			calls = append(calls, strings.Join(args, " "))
			if args[0] == "rm" {
				return "\x1b[1mworktree has uncommitted changes\x1b[0m", errors.New("exit status 1")
			}
			return "", nil
		}}
	err := s.RemoveSpare(t.TempDir(), Spare{ID: "ddd44444", Name: "vigilante"})
	if err == nil || !strings.Contains(err.Error(), "worktree has uncommitted changes") || strings.Contains(err.Error(), "\x1b") {
		t.Fatalf("err %v", err)
	}
	if strings.Count(strings.Join(calls, "\n"), "rm ddd44444") != 2 {
		t.Errorf("calls %v", calls)
	}
	if err := s.RemoveSpare(t.TempDir(), Spare{ID: "not an id"}); err == nil {
		t.Error("a name that is not an id was passed to claude")
	}
}

// A session that works in a worktree Claude Code made says so: claude rm removes it too.
func TestSpareInWorktree(t *testing.T) {
	if !(Spare{Folder: "/w/repo/.claude/worktrees/fix"}).InWorktree() || (Spare{Folder: "/w/repo"}).InWorktree() {
		t.Error("InWorktree")
	}
}
