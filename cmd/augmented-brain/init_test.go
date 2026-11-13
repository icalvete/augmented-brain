package main

import (
	"bytes"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
)

// emptyHome is a fake home with nothing set up yet, and no claude on the PATH, and
// returns it with its configuration folder.
func emptyHome(t *testing.T) (string, string) {
	t.Helper()
	h, _ := filepath.EvalSymlinks(t.TempDir()) // macOS: /var is /private/var
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("PATH", t.TempDir())
	cfg := filepath.Join(h, "cfg")
	t.Setenv("AB_CONFIG_DIR", cfg)
	for _, k := range []string{"AB_DESKTOP", "AB_LAUNCH", "AB_NEW_TERMINAL"} {
		t.Setenv(k, "")
	}
	return h, cfg
}

func runInit(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(append([]string{"init"}, args...), strings.NewReader(""), &out, &errOut)
	return code, out.String(), errOut.String()
}

// shell is how the closing message writes a command on this system.
func shell(s string) string {
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(s, " && claude", "; claude")
	}
	return s
}

// With no flags, init asks nothing and makes two spaces in ~/augmented-brain, personal
// being the default one.
func TestInitDefaultSpaces(t *testing.T) {
	h, cfg := emptyHome(t)
	code, out, errOut := runInit(t, "--yes", "--language", "en", "--no-plugin")
	if code != 0 {
		t.Fatalf("init: %d\n%s%s", code, out, errOut)
	}
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if c.Language != "en" || c.DefaultSpace != "personal" || !slices.Equal(c.SpaceNames(), []string{"personal", "work"}) {
		t.Errorf("configuration: language %q, default %q, spaces %v", c.Language, c.DefaultSpace, c.SpaceNames())
	}
	for _, sp := range []string{"work", "personal"} {
		if !exists(filepath.Join(h, "augmented-brain", sp, "notes", "tasks")) {
			t.Errorf("no notes folder for %s", sp)
		}
	}
	want := "\naugmented brain is ready. Your spaces:\n" +
		"  work       ~/augmented-brain/work\n" +
		"  personal   ~/augmented-brain/personal\n" +
		shell("\nNow open Claude in your work space:\n  cd ~/augmented-brain/work && claude\n") +
		"The first time, it will ask who you are and how you like to work.\n"
	if out != want {
		t.Errorf("output:\n%s\nwant:\n%s", out, want)
	}
}

// The installer chooses the closing message through the environment; the desktop app
// comes first.
func TestInitClosingMessage(t *testing.T) {
	for _, c := range []struct {
		env  []string
		want string
	}{
		{[]string{"AB_NEW_TERMINAL"}, "Now open a new terminal (so it finds what was just installed) and type:\n  cd ~/augmented-brain/work && claude\n"},
		{[]string{"AB_LAUNCH", "AB_NEW_TERMINAL"}, "From now on, to work: open a new terminal and type\n  cd ~/augmented-brain/work && claude\n"},
		{[]string{"AB_DESKTOP", "AB_LAUNCH"}, "pick the folder ~/augmented-brain/work with Choose….\n(Or, in a new terminal: cd ~/augmented-brain/work && claude)\n"},
	} {
		emptyHome(t)
		for _, k := range c.env {
			t.Setenv(k, "1")
		}
		code, out, errOut := runInit(t, "--yes", "--language", "en", "--no-plugin")
		if code != 0 || !strings.Contains(out, shell(c.want)) {
			t.Errorf("%v: %d\n%s%s", c.env, code, out, errOut)
		}
	}
}

// --space makes that one space without asking: its notes default to those under
// ~/augmented-brain, and there is no default space.
func TestInitOneSpace(t *testing.T) {
	h, cfg := emptyHome(t)
	work := filepath.Join(h, "lab")
	code, out, errOut := runInit(t, "--yes", "--language", "es", "--space", "lab", "--work", work)
	if code != 0 {
		t.Fatalf("init: %d\n%s%s", code, out, errOut)
	}
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if c.Language != "es" || c.DefaultSpace != "" || !slices.Equal(c.SpaceNames(), []string{"lab"}) {
		t.Errorf("configuration: language %q, default %q, spaces %v", c.Language, c.DefaultSpace, c.SpaceNames())
	}
	if !exists(filepath.Join(h, "augmented-brain", "lab", "notas", "tasks")) {
		t.Error("no notes folder")
	}
	if strings.Contains(out, "Creado:") {
		t.Errorf("without --custom, init lists what it created:\n%s", out)
	}
	for _, want := range []string{
		// No claude on the PATH: how to add the plugin later.
		"No encuentro Claude Code. Instálalo y después añade el plugin con:\n  augmented-brain update --plugin\n",
		"\naugmented brain está listo. Tus espacios:\n  lab        " + work + "\n",
		shell("cd " + work + " && claude\n"),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestInitRefusesAnUnknownLanguage(t *testing.T) {
	_, cfg := emptyHome(t)
	t.Setenv("AB_LANGUAGE", "en")
	code, _, errOut := runInit(t, "--yes", "--language", "fr", "--no-plugin")
	if code != 2 || !strings.Contains(errOut, `language "fr": use en or es`) {
		t.Errorf("init --language fr: %d %s", code, errOut)
	}
	if exists(cfg) {
		t.Error("a configuration was written")
	}
}

// space add --yes takes the name and the default folders, under ~/augmented-brain.
func TestSpaceAdd(t *testing.T) {
	h, cfg := home(t)
	var out, errOut bytes.Buffer
	if code := run([]string{"space", "add", "lab", "--yes"}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("space add: %d\n%s%s", code, out.String(), errOut.String())
	}
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	sp, ok := c.Spaces["lab"]
	if !ok {
		t.Fatalf("no space lab: %v", c.SpaceNames())
	}
	work := filepath.Join(h, "augmented-brain", "lab")
	if !slices.Equal(sp.WorkFolders, []string{work}) || sp.Memory != filepath.Join(work, "notas") {
		t.Errorf("space lab: %+v", sp)
	}
	if !exists(filepath.Join(work, "notas", "tasks")) {
		t.Error("no notes folder")
	}
	if want := "\nCreado:\n  + " + work; !strings.Contains(out.String(), want) ||
		!strings.HasSuffix(out.String(), "\nEspacio \"lab\" añadido.\n") {
		t.Errorf("output:\n%s", out.String())
	}
	// The same name again is refused.
	out.Reset()
	if code := run([]string{"space", "add", "lab", "--yes"}, strings.NewReader(""), &out, &errOut); code != 1 {
		t.Errorf("space add of an existing space: %d", code)
	}
	// The language is the machine's, set by init: space add has no --language.
	errOut.Reset()
	if code := run([]string{"space", "add", "other", "--yes", "--language", "en"}, strings.NewReader(""), &out, &errOut); code != 2 ||
		!strings.Contains(errOut.String(), "flag provided but not defined: -language") {
		t.Errorf("space add --language: %d %s", code, errOut.String())
	}
	if c, _ := config.Load(cfg); c.Spaces["other"] != nil {
		t.Error("space add --language added the space")
	}
}

// --custom asks for the language and one space; an empty answer takes the default, and
// it lists what it created.
func TestInitCustomAsks(t *testing.T) {
	h, cfg := emptyHome(t)
	var out, errOut bytes.Buffer
	answers := strings.NewReader("en\nlab\n\n\n")
	if code := run([]string{"init", "--custom", "--no-plugin"}, answers, &out, &errOut); code != 0 {
		t.Fatalf("init --custom: %d\n%s%s", code, out.String(), errOut.String())
	}
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if c.Language != "en" || !slices.Equal(c.SpaceNames(), []string{"lab"}) {
		t.Errorf("configuration: language %q, spaces %v", c.Language, c.SpaceNames())
	}
	for _, want := range []string{
		"Language / Idioma (en, es) [",
		"Space name (one part of your life: work, personal…) [work]: ",
		"Work folder: opening your assistant here selects this space [~/augmented-brain/lab]: ",
		"Memory folder: where this space's notes will live [~/augmented-brain/lab/notes]: ",
		"\nCreated:\n  + " + filepath.Join(h, "augmented-brain", "lab"),
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
}

// Without the plugin, init still finishes, but its last words say what is missing.
func TestInitWarnsWithoutThePlugin(t *testing.T) {
	emptyHome(t) // no claude and no git on the PATH
	t.Setenv("AB_LANGUAGE", "en")
	code, out, _ := runInit(t, "--yes", "--language", "en")
	// The plugin comes from the program's own folder now: git is no longer needed.
	if code != 0 || !strings.Contains(out, "the Claude Code plugin is not installed") || strings.Contains(out, "git") {
		t.Errorf("init without the plugin: %d\n%s", code, out)
	}
	emptyHome(t)
	code, out, _ = runInit(t, "--yes", "--language", "en", "--no-plugin")
	if code != 0 || strings.Contains(out, "plugin is not installed") {
		t.Errorf("--no-plugin was asked for, yet init warns:\n%s", out)
	}
}

// The panel opens on the space of the folder it is asked for in.
func TestPanelAddress(t *testing.T) {
	h, cfg := home(t)
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := panelPage(c, filepath.Join(h, "augmented-brain", "trabajo", "web")); got != "today?s=trabajo" {
		t.Errorf("in trabajo: %s", got)
	}
	if got := panelPage(c, t.TempDir()); got != "" {
		t.Errorf("outside every space: %s", got)
	}
}
