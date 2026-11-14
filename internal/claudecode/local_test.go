package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The program carries the plugin as it is in the repository: every file of plugin/ and
// the marketplace file, and nothing else.
func TestFilesAreTheRepositorys(t *testing.T) {
	root := filepath.Join("..", "..")
	var want []string
	want = append(want, ".claude-plugin/marketplace.json")
	filepath.WalkDir(filepath.Join(root, "plugin"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			r, _ := filepath.Rel(root, p)
			want = append(want, filepath.ToSlash(r))
		}
		return nil
	})
	if got := strings.Join(Files(), "\n"); got != strings.Join(want, "\n") {
		t.Errorf("carried:\n%s\nin the repository:\n%s", got, strings.Join(want, "\n"))
	}
	for _, f := range Files() {
		carried, err := Content(f, "x", "linux")
		if err != nil {
			t.Fatal(err)
		}
		disk, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			t.Fatal(err)
		}
		if f != "plugin/.claude-plugin/plugin.json" && string(carried) != string(disk) {
			t.Errorf("%s differs from the repository's", f)
		}
	}
}

func TestWriteLocal(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "marketplace")
	if err := WriteLocal(dir, "1.2.3-beta", "linux"); err != nil {
		t.Fatal(err)
	}
	var plugin map[string]any
	readJSON(t, filepath.Join(dir, "plugin", ".claude-plugin", "plugin.json"), &plugin)
	if plugin["version"] != "1.2.3-beta" || plugin["name"] != "brain" {
		t.Errorf("plugin.json = %v", plugin)
	}
	var market struct {
		Name    string
		Plugins []struct{ Name, Source string }
	}
	readJSON(t, filepath.Join(dir, ".claude-plugin", "marketplace.json"), &market)
	if market.Name != MarketplaceName || len(market.Plugins) != 1 ||
		"brain@"+market.Name != Plugin || market.Plugins[0].Source != "./plugin" {
		t.Errorf("marketplace.json = %+v", market)
	}
	mcp, _ := os.ReadFile(filepath.Join(dir, "plugin", ".mcp.json"))
	if !strings.Contains(string(mcp), `"PATH"`) {
		t.Errorf("on Linux .mcp.json keeps its PATH:\n%s", mcp)
	}

	// A file the program wrote and no longer carries goes; one put there by someone
	// else stays.
	os.WriteFile(filepath.Join(dir, "plugin", "commands", "mine.md"), []byte("x"), 0o644)
	os.MkdirAll(filepath.Join(dir, "plugin", "agents"), 0o755)
	os.WriteFile(filepath.Join(dir, "plugin", "agents", "old.md"), []byte("x"), 0o644)
	list, _ := os.ReadFile(filepath.Join(dir, writtenList))
	os.WriteFile(filepath.Join(dir, writtenList), append(list, "plugin/agents/old.md\n../../outside\n"...), 0o644)
	outside := filepath.Join(filepath.Dir(dir), "outside")
	os.WriteFile(outside, []byte("x"), 0o644)
	if err := WriteLocal(dir, "1.2.4-beta", "linux"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "plugin", "agents")); !os.IsNotExist(err) {
		t.Error("a file no longer carried, and its empty folder, are left")
	}
	for _, keep := range []string{filepath.Join(dir, "plugin", "commands", "mine.md"), outside} {
		if _, err := os.Stat(keep); err != nil {
			t.Errorf("%s was removed: %v", keep, err)
		}
	}

	// Uninstalling takes away only what the program wrote.
	if err := RemoveLocal(dir); err != nil {
		t.Fatal(err)
	}
	left := []string{}
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			left = append(left, p)
		}
		return nil
	})
	if len(left) != 1 || filepath.Base(left[0]) != "mine.md" {
		t.Errorf("left after RemoveLocal: %v", left)
	}
}

// On Windows the PATH of .mcp.json is left out: "${PATH}:…" joins with ":", which
// there breaks the last folder of the user's PATH.
func TestMCPOnWindows(t *testing.T) {
	b, err := Content("plugin/.mcp.json", "1", "windows")
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		MCPServers map[string]struct {
			Command string
			Args    []string
			Env     map[string]string
		}
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	s := m.MCPServers["augmented-brain"]
	if s.Command != "augmented-brain" || len(s.Args) != 1 || s.Args[0] != "mcp" || s.Env != nil {
		t.Errorf("on Windows: %s", b)
	}
}

// The folder the installer uses comes before the inherited PATH, so a project's own
// bin/augmented-brain cannot stand in for the guard.
func TestInstallFolderFirst(t *testing.T) {
	hooks, err := Content("plugin/hooks/hooks.json", "1", "linux")
	if err != nil {
		t.Fatal(err)
	}
	var h struct {
		Hooks map[string][]struct {
			Hooks []struct{ Command string }
		}
	}
	if err := json.Unmarshal(hooks, &h); err != nil {
		t.Fatal(err)
	}
	first := regexp.MustCompile(`^PATH="\$HOME/\.local/bin:\$PATH" augmented-brain hook `)
	n := 0
	for event, groups := range h.Hooks {
		for _, g := range groups {
			for _, c := range g.Hooks {
				n++
				if !first.MatchString(c.Command) {
					t.Errorf("%s: %s", event, c.Command)
				}
			}
		}
	}
	if n == 0 {
		t.Error("no hooks read")
	}
	mcp, _ := Content("plugin/.mcp.json", "1", "linux")
	if !strings.Contains(string(mcp), `"PATH": "${HOME:-}/.local/bin:${PATH}"`) {
		t.Errorf(".mcp.json:\n%s", mcp)
	}
}

func readJSON(t *testing.T, p string, v any) {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%s: %v", p, err)
	}
}

// The help skill carries every guide in both languages (scripts/build-guides.py writes
// them), each link between them leads to a file the plugin carries, and what Claude
// sees of the skill before using it fits in the 1,536 characters Claude Code keeps.
func TestHelpSkillCarriesTheGuides(t *testing.T) {
	carried := map[string]bool{}
	for _, f := range Files() {
		carried[f] = true
	}
	const dir = "plugin/skills/help/"
	for _, f := range []string{dir + "SKILL.md", dir + "guides/index.md"} {
		if !carried[f] {
			t.Errorf("the plugin does not carry %s", f)
		}
	}
	guides, _ := filepath.Glob(filepath.Join("..", "..", "docs", "guide", "*.md"))
	if len(guides) < 10 {
		t.Fatalf("found %d guides", len(guides))
	}
	for _, g := range guides {
		name := filepath.Base(g)
		lang := "en"
		if strings.HasSuffix(name, ".es.md") {
			lang, name = "es", strings.TrimSuffix(name, ".es.md")+".md"
		}
		if f := dir + "guides/" + lang + "/" + name; !carried[f] {
			t.Errorf("the help skill does not carry %s: run scripts/build-guides.py", f)
		}
	}
	link := regexp.MustCompile(`\]\(([^)\s#]+)`)
	for f := range carried {
		if !strings.HasPrefix(f, dir+"guides/") {
			continue
		}
		b, err := Content(f, "x", "linux")
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range link.FindAllStringSubmatch(string(b), -1) {
			to := m[1]
			if strings.Contains(to, "://") || strings.HasPrefix(to, "mailto:") {
				continue
			}
			if target := filepath.ToSlash(filepath.Join(filepath.Dir(f), to)); !carried[target] {
				t.Errorf("%s links to %s, which the plugin does not carry", f, to)
			}
		}
	}
	skill, err := Content(dir+"SKILL.md", "x", "linux")
	if err != nil {
		t.Fatal(err)
	}
	head := strings.SplitN(string(skill), "---", 3)
	if len(head) != 3 {
		t.Fatal("SKILL.md has no header")
	}
	seen := 0
	for _, line := range strings.Split(head[1], "\n") {
		for _, key := range []string{"description: ", "when_to_use: "} {
			if v, ok := strings.CutPrefix(line, key); ok {
				seen += len([]rune(v))
			}
		}
	}
	if seen == 0 || seen > 1536 {
		t.Errorf("SKILL.md's description and when_to_use take %d characters; Claude Code keeps 1,536", seen)
	}
}
