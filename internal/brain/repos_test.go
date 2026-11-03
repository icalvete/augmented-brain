package brain

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// A work folder reached through a link (as /var is /private/var on macOS, or a folder
// the user linked into their home) still holds its repositories, whichever way their
// path is written; a link inside a work folder that leads out of it does not.
func TestDescribeRepoThroughLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links need privileges on Windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	real := filepath.Join(home, "disk", "work")
	outside := filepath.Join(home, "elsewhere")
	for _, d := range []string{filepath.Join(real, "site", ".git"), filepath.Join(outside, ".git")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	work := filepath.Join(home, "work")
	if err := os.Symlink(real, work); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(real, "out")); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfgDir, setup.Options{Language: "en", Space: "work", WorkFolder: work,
		Memory: "~/brain", Today: "2026-10-15"}); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Open(c, c.Spaces["work"])
	if err != nil {
		t.Fatal(err)
	}
	b.Now = func() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC) }

	if _, err := b.DescribeRepo(filepath.Join(work, "site"), "The site"); err != nil {
		t.Fatalf("through the link: %v", err)
	}
	for _, p := range []string{filepath.Join(work, "site"), filepath.Join(real, "site")} {
		if got := b.RepoDescriptionOf(p); got != "The site" {
			t.Errorf("RepoDescriptionOf(%s) = %q", p, got)
		}
	}
	if _, err := b.DescribeRepo("site", "The site, by name"); err != nil {
		t.Fatal(err)
	}
	if got := b.RepoDescriptionOf(filepath.Join(work, "site")); got != "The site, by name" {
		t.Errorf("by name: %q", got)
	}
	if n := strings.Count(read(t, b, memory.PathFor("reference", RepoNoteID)), "site` —"); n != 1 {
		t.Errorf("the repository has %d lines, written two ways", n)
	}
	// "out" is a link in the work folder to a folder outside it.
	if p, ok := b.RepoPath("out"); !ok || paths.Within(p, real) {
		t.Errorf("RepoPath(out) = %q, %v: it should be where the link leads", p, ok)
	}
	for _, r := range []string{"out", filepath.Join(work, "out")} {
		if _, err := b.DescribeRepo(r, "not in the work folder"); err == nil || !strings.Contains(err.Error(), "outside") {
			t.Errorf("DescribeRepo(%s) through a link out: %v", r, err)
		}
	}
}

// mkRepo makes a folder inside the space's first work folder.
func mkRepo(t *testing.T, b *Brain, rel string) string {
	t.Helper()
	p := filepath.Join(b.Space.WorkFolders[0], filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Join(p, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// What a repository is goes in one reference of the space, a line each, and comes
// back from it; a new description replaces the old one, and an empty one takes it out.
func TestDescribeRepoRoundTrip(t *testing.T) {
	b := open(t, "en")
	site := mkRepo(t, b, "code/site")
	mkRepo(t, b, "code/shop")
	if where, err := b.DescribeRepo("code/site", "The public website"); err != nil || where != "~/work/code/site" {
		t.Fatalf("DescribeRepo = %q, %v", where, err)
	}
	if _, err := b.DescribeRepo(filepath.Join(b.Space.WorkFolders[0], "code", "shop"), "  The shop,\n  online  "); err != nil {
		t.Fatal(err)
	}
	path := memory.PathFor("reference", RepoNoteID)
	text := read(t, b, path)
	for _, want := range []string{"type: reference", "title: Repositories", "what it is``.\n\n- `~/work/code/site`", "- `~/work/code/site` — The public website\n", "- `~/work/code/shop` — The shop, online\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("the note lacks %q:\n%s", want, text)
		}
	}
	// It is a note like any other: a fresh load reads it, with no problem.
	c, err := config.Load(b.ConfigDir)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := Open(c, b.Space)
	if err != nil {
		t.Fatal(err)
	}
	if len(b2.Problems) != 0 {
		t.Errorf("problems: %v", b2.Problems)
	}
	if got := b2.RepoDescriptionOf(site); got != "The public website" {
		t.Errorf("RepoDescriptionOf = %q", got)
	}
	b2.Now = b.Now
	if _, err := b2.DescribeRepo("~/work/code/site", "The website, new"); err != nil {
		t.Fatal(err)
	}
	if got := b2.RepoDescriptionOf(site); got != "The website, new" {
		t.Errorf("after a change: %q", got)
	}
	if n := strings.Count(read(t, b2, path), "code/site`"); n != 1 {
		t.Errorf("the repository has %d lines", n)
	}
	if _, err := b2.DescribeRepo("code/site", ""); err != nil {
		t.Fatal(err)
	}
	if got := b2.RepoDescriptionOf(site); got != "" {
		t.Errorf("after taking it out: %q", got)
	}
	if !strings.Contains(read(t, b2, path), "code/shop` — The shop") {
		t.Error("taking one out took the other too")
	}
}

// A line written by hand, relative to a work folder, counts; other lines stay.
func TestRepoNoteByHand(t *testing.T) {
	b := open(t, "es")
	site := mkRepo(t, b, "site")
	note := "---\nid: ref-repositories\ntype: reference\ntitle: Repositorios\nsummary: mine\ncreated: 2026-10-01\nupdated: 2026-10-01\n---\nMy own words.\n\n- `site` - La web\n- not a line of the list\n"
	if err := os.MkdirAll(filepath.Join(b.Mem.Dir, "knowledge"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b.Mem.Dir, "knowledge", "ref-repositories.md"), []byte(note), 0o600); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(b.ConfigDir)
	now := b.Now
	b, err := Open(c, b.Space)
	if err != nil {
		t.Fatal(err)
	}
	b.Now = now
	if got := b.RepoDescriptionOf(site); got != "La web" {
		t.Errorf("RepoDescriptionOf = %q", got)
	}
	if _, err := b.DescribeRepo("site", "La web pública"); err != nil {
		t.Fatal(err)
	}
	text := read(t, b, "knowledge/ref-repositories.md")
	for _, want := range []string{"My own words.\n", "- not a line of the list\n", "- `~/work/site` — La web pública\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "`site` - La web\n") {
		t.Error("the old line stayed")
	}
}

// A secret is refused and nothing is written; so are folders outside the work
// folders, and a description longer than a line.
func TestDescribeRepoRefuses(t *testing.T) {
	b := open(t, "en")
	mkRepo(t, b, "site")
	key := "AK" + "IA" + "QWERTYUIOPASDFGH"
	if _, err := b.DescribeRepo("site", "deploys with "+key); err == nil || !strings.Contains(err.Error(), "an AWS access key") {
		t.Errorf("a key: %v", err)
	}
	if _, err := os.Stat(filepath.Join(b.Mem.Dir, "knowledge", "ref-repositories.md")); !os.IsNotExist(err) {
		t.Errorf("the refused description was written: %v", err)
	}
	if _, ok := b.Mem.Notes[RepoNoteID]; ok {
		t.Error("the refused note is in the index")
	}
	outside := filepath.Join(filepath.Dir(b.Space.WorkFolders[0]), "elsewhere")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := b.DescribeRepo(outside, "not mine"); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("outside the work folders: %v", err)
	}
	if _, err := b.DescribeRepo("missing", "nothing there"); err == nil {
		t.Error("a folder that does not exist was described")
	}
	if _, err := b.DescribeRepo("site", strings.Repeat("a", 301)); err == nil {
		t.Error("a description of 301 characters was saved")
	}
	if _, err := b.DescribeRepo("", "x"); err == nil {
		t.Error("no repository was accepted")
	}
}

func TestParseRepoLine(t *testing.T) {
	for line, want := range map[string][2]string{
		"- `~/a` — The a":   {"~/a", "The a"},
		"- `b/c` - The c  ": {"b/c", "The c"},
		"- `/x y` — z\r":    {"/x y", "z"},
	} {
		k, v, ok := parseRepoLine(line)
		if !ok || k != want[0] || v != want[1] {
			t.Errorf("parseRepoLine(%q) = %q, %q, %v", line, k, v, ok)
		}
	}
	for _, line := range []string{"- `` — empty", "- ~/a — no quotes", "text", "- `a`— no space", "## `a` — heading"} {
		if _, _, ok := parseRepoLine(line); ok {
			t.Errorf("parseRepoLine(%q) read a line", line)
		}
	}
}

// A description whose write fails leaves the note on disk and in the index as it was,
// and no half file behind.
func TestDescribeRepoFailedKeepsNote(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("needs a folder this user cannot write in")
	}
	b := open(t, "en")
	mkRepo(t, b, "site")
	if _, err := b.DescribeRepo("site", "first"); err != nil {
		t.Fatal(err)
	}
	path := memory.PathFor("reference", RepoNoteID)
	before := read(t, b, path)
	dir := filepath.Join(b.Mem.Dir, "knowledge")
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	if _, err := b.DescribeRepo("site", "second"); err == nil {
		t.Fatal("the write should have failed")
	}
	if read(t, b, path) != before {
		t.Error("the note changed after a failed write")
	}
	if !strings.Contains(b.Mem.Notes[RepoNoteID].Body, "— first") {
		t.Error("the index changed after a failed write")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("files left behind: %v", entries)
	}
}
