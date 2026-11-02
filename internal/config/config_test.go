package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func write(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAndSpaceFor(t *testing.T) {
	home, _ := filepath.EvalSymlinks(t.TempDir()) // macOS: /var is /private/var
	t.Setenv("HOME", home)
	for _, d := range []string{"work", "work/clients", "personal"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfgDir := t.TempDir()
	write(t, cfgDir, `
language = "es"

[spaces.work]
memory = "~/brains/work"
work_folders = ["~/work"]
production_aws_profiles = ["prod"]

[spaces.clients]
memory = "~/brains/clients"
work_folders = ["~/work/clients"]

[spaces.personal]
memory = "~/brains/personal"
work_folders = ["~/personal"]
`)
	c, err := Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Language != "es" {
		t.Errorf("language = %q", c.Language)
	}
	if got := c.Spaces["work"].Memory; got != filepath.Join(home, "brains/work") {
		t.Errorf("memory not expanded: %q", got)
	}

	cases := map[string]string{
		"work":              "work",
		"work/project/src":  "work",
		"work/clients":      "clients", // the deepest folder wins
		"work/clients/acme": "clients",
		"personal":          "personal",
		"workshop":          "", // a name that only starts like a work folder
		"elsewhere":         "",
	}
	for rel, want := range cases {
		s := c.SpaceFor(filepath.Join(home, rel))
		got := ""
		if s != nil {
			got = s.Name
		}
		if got != want {
			t.Errorf("SpaceFor(%s) = %q, want %q", rel, got, want)
		}
	}
}

func TestLoadErrors(t *testing.T) {
	cases := map[string]string{
		"": "no configuration found",
		`language = "fr"` + "\n[spaces.a]\nmemory = \"/m\"": "language",
		`language = "en"`:                            "no spaces",
		"[spaces.a]\nwork_folders = [\"/w\"]":        "memory folder is missing",
		"[spaces.a]\nmemory = \"/m\"\nbakup = \"x\"": "unknown settings",
	}
	for body, want := range cases {
		dir := t.TempDir()
		if body != "" {
			write(t, dir, body)
		}
		_, err := Load(dir)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("config %q: error %v, want it to mention %q", body, err, want)
		}
		if body == "" && !errors.Is(err, ErrNotFound) {
			t.Errorf("missing file: want ErrNotFound, got %v", err)
		}
	}
}

func TestDir(t *testing.T) {
	t.Setenv("AB_CONFIG_DIR", "/tmp/ab-test")
	if d, _ := Dir(); d != "/tmp/ab-test" {
		t.Errorf("AB_CONFIG_DIR ignored: %q", d)
	}
	t.Setenv("AB_CONFIG_DIR", "")
	if runtime.GOOS == "windows" {
		return // Windows keeps it in AppData, its own user configuration folder
	}
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	if d, _ := Dir(); d != "/tmp/xdg/augmented-brain" {
		t.Errorf("XDG_CONFIG_HOME ignored: %q", d)
	}
}

// Found in the review before 0.1: adding a space or a backup must keep the user's
// comments, and a symlinked config.toml must stay a symlink.
func TestEditsKeepTheFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	real := filepath.Join(home, "dotfiles", "config.toml")
	dir := filepath.Join(home, "cfg")
	for _, d := range []string{filepath.Dir(real), dir, filepath.Join(home, "work"), filepath.Join(home, "other")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	original := "# my notes about this file\nlanguage = \"en\"\n\n[spaces.work]\nmemory = \"~/brain\"  # where it lives\nwork_folders = [\"~/work\"]\n"
	if err := os.WriteFile(real, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(dir, FileName)); err != nil {
		t.Fatal(err)
	}

	if err := AddSpace(dir, &Space{Name: "other", Memory: filepath.Join(home, "m2"), WorkFolders: []string{filepath.Join(home, "other")}}); err != nil {
		t.Fatal(err)
	}
	if err := SetSpaceKey(dir, "work", "backup", "git@example.com:me/brain.git"); err != nil {
		t.Fatal(err)
	}
	if err := SetSpaceKey(dir, "work", "backup", "git@example.com:me/brain2.git"); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(filepath.Join(dir, FileName)); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("the symlink was replaced by a file")
	}
	data, _ := os.ReadFile(real)
	text := string(data)
	for _, want := range []string{"# my notes about this file", "# where it lives", `memory = "~/m2"`, `backup = "git@example.com:me/brain2.git"`} {
		if !strings.Contains(text, want) {
			t.Errorf("file lacks %q:\n%s", want, text)
		}
	}
	if strings.Count(text, "backup =") != 1 {
		t.Errorf("backup set twice:\n%s", text)
	}
	c, err := Load(dir)
	if err != nil || len(c.Spaces) != 2 || c.Spaces["work"].Backup != "git@example.com:me/brain2.git" {
		t.Fatalf("config after edits: %+v %v", c, err)
	}
	if err := SetSpaceKey(dir, "nope", "backup", "x"); err == nil {
		t.Error("a missing space was accepted")
	}

	// Lists: set, then replaced in place.
	if err := SetSpaceList(dir, "work", "untouchable_hosts", []string{"db01"}); err != nil {
		t.Fatal(err)
	}
	if err := SetSpaceList(dir, "work", "untouchable_hosts", []string{"db01", "db02"}); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(real)
	if n := strings.Count(string(data), "untouchable_hosts ="); n != 1 {
		t.Errorf("untouchable_hosts set %d times:\n%s", n, data)
	}
	if c, err := Load(dir); err != nil || strings.Join(c.Spaces["work"].UntouchableHosts, ",") != "db01,db02" {
		t.Errorf("hosts after edits: %+v %v", c.Spaces["work"], err)
	}
}

// The default space serves folders outside every space; a default that is not a space
// is refused.
func TestDefaultSpace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, "cfg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := "language = \"es\"\n\n[spaces.trabajo]\nmemory = \"~/ab/trabajo/notas\"\nwork_folders = [\"~/ab/trabajo\"]\n\n[spaces.personal]\nmemory = \"~/ab/personal/notas\"\nwork_folders = [\"~/ab/personal\"]\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	c, _ := Load(dir)
	if s, d := c.SpaceAt(filepath.Join(home, "elsewhere")); s != nil || d {
		t.Errorf("no default yet: %v %v", s, d)
	}
	if err := SetDefaultSpace(dir, "personal"); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s, d := c.SpaceAt(filepath.Join(home, "elsewhere")); s == nil || s.Name != "personal" || !d {
		t.Errorf("outside: %v %v", s, d)
	}
	if s, d := c.SpaceAt(filepath.Join(home, "ab", "trabajo", "web")); s == nil || s.Name != "trabajo" || d {
		t.Errorf("inside trabajo: %v %v", s, d)
	}
	if err := SetDefaultSpace(dir, "nope"); err == nil || !strings.Contains(err.Error(), "not one of the spaces") {
		t.Errorf("a default that is not a space: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, FileName))
	if n := strings.Count(string(data), "default_space"); n != 1 {
		t.Errorf("default_space written %d times:\n%s", n, data)
	}
}

// The guard lists are machine-wide, and older per-space lists still count.
func TestGuardListsAndSetTopList(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, `# my comment
language = "en"

[spaces.work]
memory = "/m/work"
production_aws_profiles = ["prod", "shared"]

[spaces.personal]
memory = "/m/personal"
untouchable_hosts = ["nas"]
`)
	if err := SetTopList(dir, "production_aws_profiles", []string{"shared", "live"}); err != nil {
		t.Fatal(err)
	}
	if err := SetTopList(dir, "untouchable_hosts", []string{"db01"}); err != nil {
		t.Fatal(err)
	}
	if err := SetTopList(dir, "untouchable_hosts", []string{"db01", "db02"}); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	profiles, hosts := c.GuardLists()
	if strings.Join(profiles, ",") != "shared,live,prod" || strings.Join(hosts, ",") != "db01,db02,nas" {
		t.Errorf("GuardLists = %v %v", profiles, hosts)
	}
	data, _ := os.ReadFile(filepath.Join(dir, FileName))
	if !strings.HasPrefix(string(data), "# my comment\n") || strings.Count(string(data), "untouchable_hosts = [\"db01\"") != 1 {
		t.Errorf("file:\n%s", data)
	}
	if err := RemoveSpaceKey(dir, "work", "production_aws_profiles"); err != nil {
		t.Fatal(err)
	}
	if c, _ = Load(dir); len(c.Spaces["work"].ProductionAWSProfiles) != 0 {
		t.Error("the space's list is still there")
	}
	// Saved and read back, the machine-wide lists stay.
	c.Dir = t.TempDir()
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	back, err := Load(c.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(back.UntouchableHosts, ",") != "db01,db02" {
		t.Errorf("after Save: %v", back.UntouchableHosts)
	}
}

// Format 1 is never written (older releases refuse settings they do not know); a later
// one is, as the first line, and read back.
func TestFormat(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "language = \"en\"\n\n[spaces.work]\nmemory = \"/tmp/m\"\nwork_folders = [\"/tmp/w\"]\n")
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.FormatVersion() != 1 {
		t.Errorf("no format: %d, want 1", c.FormatVersion())
	}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	if err := SetFormat(dir, 1); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, FileName)); strings.Contains("\n"+string(data), "\nformat") {
		t.Errorf("format written:\n%s", data)
	}
	if err := SetFormat(dir, 2); err != nil {
		t.Fatal(err)
	}
	if c, err := Load(dir); err != nil || c.FormatVersion() != 2 {
		t.Errorf("after SetFormat(2): %v %v", c, err)
	} else if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	if c, err := Load(dir); err != nil || c.FormatVersion() != 2 {
		t.Errorf("Save lost the format: %v %v", c, err)
	}
	if err := SetFormat(dir, 3); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, FileName)); strings.Count(string(data), "format =") != 1 {
		t.Errorf("format written twice:\n%s", data)
	}
}

// git_ask_repos is read like git_allowed_repos, and an entry with a * keeps it.
func TestGitAskRepos(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	write(t, dir, "language = \"en\"\n\n[spaces.work]\nmemory = \"~/m\"\nwork_folders = [\"~/w\"]\n"+
		"git_allowed_repos = [\"~/w/app-*\"]\ngit_ask_repos = [\"~/w/infra\"]\n")
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := c.Spaces["work"]
	if len(s.GitAskRepos) != 1 || !filepath.IsAbs(s.GitAskRepos[0]) || filepath.Base(s.GitAllowedRepos[0]) != "app-*" {
		t.Fatalf("read: %v %v", s.GitAllowedRepos, s.GitAskRepos)
	}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, FileName))
	if !strings.Contains(string(data), `git_ask_repos = ["~/w/infra"]`) || !strings.Contains(string(data), `git_allowed_repos = ["~/w/app-*"]`) {
		t.Errorf("saved:\n%s", data)
	}
}
