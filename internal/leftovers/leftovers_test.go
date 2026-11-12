package leftovers

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/testenv"
)

func TestMain(m *testing.M) {
	testenv.CleanGit()
	os.Exit(m.Run())
}

const livePID = 4242

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "init.defaultBranch=main"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// setup makes a repository with an origin, worktrees of every kind Find must tell
// apart, Claude Code's file and sessions, and a scratch folder.
func setup(t *testing.T) (Env, string) {
	root := t.TempDir()
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	origin := filepath.Join(root, "origin.git")
	repo := filepath.Join(root, "repo")
	git(t, root, "init", "--bare", origin)
	git(t, root, "clone", origin, repo)
	write(t, filepath.Join(repo, "a.txt"), "a\n")
	git(t, repo, "add", "a.txt")
	git(t, repo, "commit", "-m", "a")
	git(t, repo, "push", "origin", "HEAD")
	wt := func(name string) string {
		p := filepath.Join(repo, ".claude", "worktrees", name)
		git(t, repo, "worktree", "add", "-b", "worktree-"+name, p)
		return p
	}
	wt("clean")
	write(t, filepath.Join(wt("changed"), "a.txt"), "changed\n")
	ahead := wt("ahead")
	write(t, filepath.Join(ahead, "b.txt"), "b\n")
	git(t, ahead, "add", "b.txt")
	git(t, ahead, "commit", "-m", "b")
	git(t, repo, "worktree", "lock", "--reason", "claude session dead (pid 999999 start 1)", wt("dead"))
	git(t, repo, "worktree", "lock", "--reason", "claude session live (pid 4242 start 1)", wt("live"))
	git(t, repo, "worktree", "lock", "--reason", "on a USB disk", wt("other"))
	wt("cwd")
	// The user's own branch: the worktree goes, the branch stays.
	mine := filepath.Join(repo, ".claude", "worktrees", "mine")
	git(t, repo, "worktree", "add", "-b", "feat/mine", mine)
	// Not under .claude/worktrees: never Claude Code's.
	git(t, repo, "worktree", "add", "-b", "outside", filepath.Join(root, "outside"))
	// A worktree whose folder is gone.
	gone := filepath.Join(root, "gone")
	git(t, repo, "worktree", "add", "-b", "gone", gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	claudeJSON := filepath.Join(root, "claude.json")
	projects := map[string]any{repo: map[string]any{}, filepath.Join(repo, ".claude", "worktrees", "clean"): map[string]any{}, filepath.Join(root, "nothing"): map[string]any{}}
	data, _ := json.Marshal(map[string]any{"projects": projects})
	write(t, claudeJSON, string(data))

	claudeDir := filepath.Join(root, "claude")
	const liveID = "11111111-2222-3333-4444-555555555555"
	write(t, filepath.Join(claudeDir, "sessions", "4242.json"), `{"pid":4242,"sessionId":"`+liveID+`","cwd":"`+filepath.ToSlash(filepath.Join(repo, ".claude", "worktrees", "cwd", "sub"))+`"}`)
	write(t, filepath.Join(claudeDir, "sessions", "77.json"), `{"pid":77,"sessionId":"99999999-2222-3333-4444-555555555555","cwd":"`+filepath.ToSlash(repo)+`"}`)

	tmp := filepath.Join(root, "tmp")
	old := time.Now().Add(-48 * time.Hour)
	for _, p := range []string{
		filepath.Join(tmp, "-proj", "aaaaaaaa-2222-3333-4444-555555555555", "scratchpad", "x.txt"),
		filepath.Join(tmp, "-proj", liveID, "scratchpad", "x.txt"),
		filepath.Join(tmp, "-proj", "bbbbbbbb-2222-3333-4444-555555555555", "scratchpad", "x.txt"),
		filepath.Join(tmp, "-other", "cccccccc-2222-3333-4444-555555555555", "tasks", "y.txt"),
		filepath.Join(tmp, "bash-edit-diff", "dddddddd-2222-3333-4444-555555555555", "z"),
		filepath.Join(tmp, "cache-break-state-aaaaaaaa-2222-3333-4444-555555555555.json"),
		filepath.Join(tmp, "cache-break-state-"+liveID+".json"),
	} {
		write(t, p, "0123456789")
		for d := p; d != tmp; d = filepath.Dir(d) {
			_ = os.Chtimes(d, old, old)
		}
	}
	// Touched an hour ago: its session may come back.
	recent := filepath.Join(tmp, "-proj", "bbbbbbbb-2222-3333-4444-555555555555", "scratchpad", "x.txt")
	hour := time.Now().Add(-time.Hour)
	_ = os.Chtimes(recent, hour, hour)

	return Env{ClaudeJSON: claudeJSON, ClaudeDir: claudeDir, TempRoot: tmp,
		Alive: func(pid int) bool { return pid == livePID }}, repo
}

func names(ws []Worktree) []string {
	var out []string
	for _, w := range ws {
		out = append(out, filepath.Base(w.Folder)+"="+w.Work)
	}
	slices.Sort(out)
	return out
}

func TestFindAndClean(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	env, repo := setup(t)
	r := Find(env)
	want := []string{"ahead=" + remote.KeptCommits, "changed=" + remote.KeptChanges, "clean=", "dead=", "mine="}
	if got := names(r.Worktrees); !slices.Equal(got, want) {
		t.Fatalf("worktrees: got %v, want %v", got, want)
	}
	for _, w := range r.Worktrees {
		if b := filepath.Base(w.Folder); (b == "mine") != (w.Branch == "") {
			t.Errorf("%s: branch %q", b, w.Branch)
		}
	}
	if len(r.Prunable) != 1 || filepath.Base(r.Prunable[0]) != "repo" {
		t.Errorf("prunable: %v", r.Prunable)
	}
	if !r.ScratchOK || len(r.Scratch) != 3 {
		t.Fatalf("scratch: ok %v, %v", r.ScratchOK, r.Scratch)
	}
	for _, s := range r.Scratch {
		if strings.Contains(s.Path, "bbbbbbbb") || strings.Contains(s.Path, "11111111") || strings.Contains(s.Path, "bash-edit-diff") {
			t.Errorf("scratch %s should stay", s.Path)
		}
	}
	if r.ScratchSize() != 30 {
		t.Errorf("size %d", r.ScratchSize())
	}

	d := Clean(env, r)
	if len(d.Errors) > 0 || len(d.Worktrees) != 3 || len(d.Pruned) != 1 || d.Scratch != 3 {
		t.Fatalf("clean: %+v", d)
	}
	for _, gone := range []string{"clean", "dead", "mine"} {
		if _, err := os.Stat(filepath.Join(repo, ".claude", "worktrees", gone)); !os.IsNotExist(err) {
			t.Errorf("%s still there", gone)
		}
	}
	for _, stays := range []string{"changed", "ahead", "live", "other", "cwd"} {
		if _, err := os.Stat(filepath.Join(repo, ".claude", "worktrees", stays)); err != nil {
			t.Errorf("%s removed", stays)
		}
	}
	branches := git(t, repo, "branch", "--format=%(refname:short)")
	for _, b := range []string{"worktree-clean", "worktree-dead"} {
		if strings.Contains(branches, b+"\n") || strings.HasSuffix(branches, b) {
			t.Errorf("branch %s stays", b)
		}
	}
	for _, b := range []string{"feat/mine", "gone", "worktree-ahead"} {
		if !strings.Contains(branches, b) {
			t.Errorf("branch %s removed", b)
		}
	}
	if strings.Contains(git(t, repo, "worktree", "list", "--porcelain"), "prunable") {
		t.Error("not pruned")
	}
	if _, err := os.Stat(filepath.Join(env.TempRoot, "-other")); !os.IsNotExist(err) {
		t.Error("the empty folder of -other stays")
	}
	if _, err := os.Stat(filepath.Join(env.TempRoot, "-proj", "11111111-2222-3333-4444-555555555555")); err != nil {
		t.Error("the live session's scratch folder was removed")
	}
	if !Find(env).Empty() {
		t.Errorf("left: %+v", Find(env))
	}
}

// Without the sessions running, no scratch folder is called left over.
func TestNoSessionsNoScratch(t *testing.T) {
	tmp := t.TempDir()
	write(t, filepath.Join(tmp, "-p", "aaaaaaaa-2222-3333-4444-555555555555", "x"), "x")
	r := Find(Env{ClaudeDir: filepath.Join(tmp, "none"), TempRoot: tmp, Now: time.Now().Add(72 * time.Hour)})
	if r.ScratchOK || len(r.Scratch) != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestHumanSize(t *testing.T) {
	for n, want := range map[int64]string{1200: "1 KB", 154_100_000: "154 MB", 2_100_000_000: "2.1 GB"} {
		if got := HumanSize(n); got != want {
			t.Errorf("%d: %q", n, got)
		}
	}
}
