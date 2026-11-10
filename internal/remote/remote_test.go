package remote

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/doccheck"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// fake is a System whose commands are recorded, never run: systemctl and loginctl answer
// from the maps, and every path is inside one temporary folder.
type fake struct {
	t      *testing.T
	root   string
	calls  []string
	active map[string]string    // unit -> is-active answer
	linger string               // "yes", "no", or "missing" (no loginctl)
	failOn string               // a systemctl call that fails
	live   map[string]bool      // claude's background sessions that run
	named  map[string][2]string // id -> name and folder, for claude agents
	busy   map[string]int       // id -> how many more looks it is still answering
	bgOut  string               // what claude --bg prints
	ranIn  []string             // the folder and command of each RunIn
	sys    System
	cfgDir string
}

func newFake(t *testing.T) *fake {
	t.Helper()
	// Resolved, as the configuration resolves work folders: on macOS the temporary folder
	// is behind a link (/var is /private/var), on Windows it may be a short 8.3 name.
	root, err := paths.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", root)
	t.Setenv("AB_CONFIG_DIR", "")
	f := &fake{t: t, root: root, active: map[string]string{}, linger: "no", cfgDir: filepath.Join(root, ".config", "augmented-brain")}
	f.live = map[string]bool{}
	f.busy = map[string]int{}
	f.bgOut = "Starting background service…\nbackgrounded · ab12cd34 · x\n"
	f.sys = System{
		Run: f.run,
		RunIn: func(dir string, name string, args ...string) (string, error) {
			f.ranIn = append(f.ranIn, dir+" | "+filepath.Base(name)+" "+strings.Join(args, " "))
			if m := backgrounded.FindStringSubmatch(f.bgOut); m != nil {
				f.live[m[1]] = true
			}
			return f.bgOut, nil
		},
		LookPath: func(name string) (string, error) {
			if name == "claude" {
				return filepath.Join(root, ".local", "bin", "claude"), nil
			}
			return "", exec.ErrNotFound
		},
		Hostname:   func() (string, error) { return "box.local", nil },
		GOOS:       "linux",
		User:       "lucia",
		UnitDir:    filepath.Join(root, ".config", "systemd", "user"),
		ClaudeJSON: filepath.Join(root, ".claude.json"),
	}
	if _, err := setup.Init(f.cfgDir, setup.Options{Space: "personal", WorkFolder: filepath.Join(root, "p"), Memory: filepath.Join(root, "p", "notes")}); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.AddSpace(f.cfgDir, setup.Options{Space: "work", WorkFolder: filepath.Join(root, "w"), Memory: filepath.Join(root, "w", "notes")}); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *fake) run(name string, args ...string) (string, error) {
	call := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, call)
	switch {
	case filepath.Base(name) == "claude" && len(args) > 0 && args[0] == "agents":
		var list []map[string]string
		for id := range f.live {
			status := "idle"
			if f.busy[id] > 0 {
				f.busy[id]--
				status = "busy"
			}
			list = append(list, map[string]string{"id": id, "name": f.named[id][0], "cwd": f.named[id][1], "status": status})
		}
		data, _ := json.Marshal(list)
		return string(data), nil
	case filepath.Base(name) == "claude" && len(args) > 1 && (args[0] == "stop" || args[0] == "rm"):
		delete(f.live, args[1])
		return "", nil
	case name == "loginctl":
		switch f.linger {
		case "missing":
			return "", exec.ErrNotFound
		case "yes":
			return "Linger=yes\n", nil
		}
		return "Linger=no\n", nil
	case name == "systemctl" && f.failOn != "" && strings.Contains(call, f.failOn):
		return "", errors.New("boom")
	case name == "systemctl" && len(args) >= 3 && args[1] == "is-active":
		a := f.active[args[2]]
		if a == "" {
			a = "inactive"
		}
		if a != "active" {
			return a + "\n", errors.New("exit status 3")
		}
		return a + "\n", nil
	case name == "systemctl" && len(args) >= 3 && args[1] == "is-enabled":
		return "enabled\n", nil
	}
	return "", nil
}

func (f *fake) load() (*config.Config, Settings) {
	f.t.Helper()
	c, err := config.Load(f.cfgDir)
	if err != nil {
		f.t.Fatal(err)
	}
	set, err := LoadSettings(c, "box.local")
	if err != nil {
		f.t.Fatal(err)
	}
	return c, set
}

func (f *fake) trust(dir string) {
	f.t.Helper()
	// Marshalled, not concatenated: a Windows path has backslashes JSON must escape.
	body, err := json.Marshal(map[string]any{"projects": map[string]any{dir: map[string]bool{"hasTrustDialogAccepted": true}}})
	if err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(f.sys.ClaudeJSON, body, 0o600); err != nil {
		f.t.Fatal(err)
	}
}

// The unit's text, exactly: it is what systemd runs, so a change must be deliberate.
func TestUnitText(t *testing.T) {
	svc := Service{Space: "personal", Name: "box-personal", Folder: "/home/lucia/.config/augmented-brain/remote/personal", Unit: "augmented-brain-remote-personal.service"}
	want := `# Written by augmented-brain remote install; remove it with augmented-brain remote uninstall.
[Unit]
Description=augmented brain: Claude remote access, space personal
After=network-online.target
Wants=network-online.target

[Service]
WorkingDirectory=/home/lucia/.config/augmented-brain/remote/personal
Environment=PATH=/home/lucia/.local/bin:/usr/bin:/bin
ExecStart=/home/lucia/.local/bin/claude remote-control --name box-personal --permission-mode default
Restart=always
RestartSec=30
StandardOutput=null
StandardError=journal
SyslogIdentifier=augmented-brain-remote-personal
LogRateLimitIntervalSec=60
LogRateLimitBurst=200

[Install]
WantedBy=default.target
`
	if got := svc.UnitText("/home/lucia/.local/bin/claude", "default", "/home/lucia/.local/bin:/usr/bin:/bin"); got != want {
		t.Errorf("unit:\n%s\nwant:\n%s", got, want)
	}
}

// A path with a space, a % or a $ must reach systemd as written.
func TestUnitTextQuotesPaths(t *testing.T) {
	svc := Service{Space: "work", Name: "x-work", Folder: "/home/a b/100%/remote/work"}
	got := svc.UnitText(`/opt/my claude/$bin/claude`, "plan", `/opt/my claude/$bin:/usr/bin`)
	for _, line := range []string{
		`Environment="PATH=/opt/my claude/$bin:/usr/bin"`,
		"WorkingDirectory=/home/a b/100%%/remote/work",
		`ExecStart="/opt/my claude/$$bin/claude" remote-control --name x-work --permission-mode plan`,
	} {
		if !strings.Contains(got, line+"\n") {
			t.Errorf("missing %q in:\n%s", line, got)
		}
	}
}

// The PATH of a unit: claude's folder and this program's first, then the system's, each
// once, so a remote session finds both after a reboot (9-nov: it found neither).
func TestUnitPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("systemd paths") // windows-ok
	}
	s := System{Self: "/home/lucia/.local/bin/augmented-brain"}
	if got, want := s.UnitPath("/home/lucia/.local/bin/claude"), "/home/lucia/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"; got != want {
		t.Errorf("UnitPath = %q, want %q", got, want)
	}
	s.Self = "/opt/ab/augmented-brain"
	if got := s.UnitPath("/usr/bin/claude"); !strings.HasPrefix(got, "/usr/bin:/opt/ab:/usr/local/sbin") {
		t.Errorf("UnitPath = %q", got)
	}
}

// claude out of the PATH is found where its installer puts it.
func TestFindClaudeOutsidePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bits") // windows-ok
	}
	home := t.TempDir()
	s := System{Home: home, LookPath: func(string) (string, error) { return "", exec.ErrNotFound }}
	if _, err := s.FindClaude(); err == nil {
		t.Fatal("found a claude that is not there")
	}
	bin := filepath.Join(home, ".local", "bin", "claude")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := s.FindClaude(); err != nil || got != bin {
		t.Errorf("FindClaude = %q, %v; want %q", got, err, bin)
	}
}

func TestSettings(t *testing.T) {
	f := newFake(t)
	c, _ := f.load()
	// Defaults: nothing enabled, the default space, the host's name, mode default.
	set, err := LoadSettings(c, "Box Name!.local")
	if err != nil {
		t.Fatal(err)
	}
	if set.Enabled || !slices.Equal(set.Spaces, []string{c.DefaultSpace}) || set.NamePrefix != "Box-Name-.local" || set.PermissionMode != "default" {
		t.Errorf("defaults = %+v", set)
	}
	write := func(text string) error {
		if err := os.WriteFile(filepath.Join(f.cfgDir, SettingsFile), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := LoadSettings(c, "box")
		return err
	}
	if err := write("enabled = true\nspaces = [\"personal\", \"work\"]\nname_prefix = \"casa\"\npermission_mode = \"plan\"\n"); err != nil {
		t.Fatal(err)
	}
	set, _ = LoadSettings(c, "box")
	if !set.Enabled || len(set.Spaces) != 2 || set.NamePrefix != "casa" || set.PermissionMode != "plan" {
		t.Errorf("read = %+v", set)
	}
	for _, bad := range []struct{ text, want string }{
		{`permission_mode = "yolo"`, "permission_mode"},
		{`name_prefix = "a b; rm"`, "name_prefix"},
		{`spaces = ["nowhere"]`, "nowhere"},
		{`colour = "red"`, "colour"},
	} {
		if err := write(bad.text); err == nil || !strings.Contains(err.Error(), bad.want) {
			t.Errorf("%s: error = %v", bad.text, err)
		}
	}
	for _, mode := range PermissionModes {
		if err := write("permission_mode = \"" + mode + "\"\n"); err != nil {
			t.Errorf("%s refused: %v", mode, err)
		}
	}
}

// Install starts each service in the space's first work folder (no folder of its own,
// config.toml untouched), writes the unit, reloads systemd and enables the service;
// running it again changes nothing.
func TestInstallIsRepeatable(t *testing.T) {
	f := newFake(t)
	c, set := f.load()
	set.Spaces = []string{"personal", "work"}
	before0, _ := os.ReadFile(filepath.Join(f.cfgDir, config.FileName))
	res, err := f.sys.Install(c, set)
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{res.Services[0].Folder, res.Services[1].Folder}; !slices.Equal(got, []string{filepath.Join(f.root, "p"), filepath.Join(f.root, "w")}) {
		t.Errorf("folders = %q, want each space's first work folder", got)
	}
	if _, err := os.Stat(filepath.Join(f.cfgDir, "remote")); err == nil {
		t.Error("a space with work folders needs no folder of its own")
	}
	if after0, _ := os.ReadFile(filepath.Join(f.cfgDir, config.FileName)); string(after0) != string(before0) {
		t.Errorf("config.toml changed:\n%s", after0)
	}
	unit, err := os.ReadFile(filepath.Join(f.sys.UnitDir, "augmented-brain-remote-personal.service"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(unit), "ExecStart="+unitArg(filepath.Join(f.root, ".local", "bin", "claude"), true)+" remote-control --name box.local-personal --permission-mode default\n") {
		t.Errorf("unit:\n%s", unit)
	}
	wantCalls := []string{
		"systemctl --user show-environment",
		"systemctl --user daemon-reload",
		"systemctl --user enable --now augmented-brain-remote-personal.service",
		"systemctl --user enable --now augmented-brain-remote-work.service",
	}
	if got := filterCalls(f.calls, "systemctl"); !slices.Equal(got, wantCalls) {
		t.Errorf("systemctl calls = %q", got)
	}
	if len(res.Added) != 0 || len(res.Untrusted) != 2 || res.Linger != LingerOff {
		t.Errorf("result = %+v", res)
	}
	if !strings.Contains(f.sys.LingerCommand(), "sudo loginctl enable-linger lucia") {
		t.Errorf("linger command = %q", f.sys.LingerCommand())
	}
	for _, call := range f.calls {
		if strings.Contains(call, "enable-linger") {
			t.Errorf("install ran %q: that is for root", call)
		}
	}
	if data, _ := os.ReadFile(filepath.Join(f.cfgDir, SettingsFile)); !strings.Contains(string(data), "enabled = true") {
		t.Errorf("remote.toml:\n%s", data)
	}

	// Again: same files, same work_folders, no restart.
	before, _ := os.ReadFile(filepath.Join(f.cfgDir, config.FileName))
	f.calls = nil
	c, set = f.load()
	res, err = f.sys.Install(c, set)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(f.cfgDir, config.FileName))
	if string(before) != string(after) {
		t.Errorf("the second install changed config.toml:\n%s\n---\n%s", before, after)
	}
	if len(res.Added) != 0 {
		t.Errorf("added again: %+v", res.Added)
	}
	for _, call := range f.calls {
		if strings.Contains(call, "restart") {
			t.Errorf("restarted with nothing changed: %q", call)
		}
	}

	// Another claude: the unit changes and the running service is restarted.
	f.sys.LookPath = func(string) (string, error) { return "/usr/local/bin/claude", nil }
	f.calls = nil
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(f.calls, "systemctl --user restart augmented-brain-remote-personal.service") {
		t.Errorf("no restart after the unit changed: %q", f.calls)
	}
}

// A space with no work folder gets a folder of its own, made a work folder of the space.
func TestInstallOwnFolder(t *testing.T) {
	f := newFake(t)
	if err := config.SetSpaceList(f.cfgDir, "work", "work_folders", nil); err != nil {
		t.Fatal(err)
	}
	c, set := f.load()
	set.Spaces = []string{"personal", "work"}
	res, err := f.sys.Install(c, set)
	if err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(f.cfgDir, "remote", "work")
	if st, err := os.Stat(folder); err != nil || !st.IsDir() || (runtime.GOOS != "windows" && st.Mode().Perm() != 0o700) { // Windows has no modes
		t.Errorf("folder: %v %v", st, err)
	}
	if len(res.Added) != 1 || res.Added[0].Space != "work" || !res.Services[1].Own || res.Services[0].Own {
		t.Errorf("result = %+v", res)
	}
	c2, _ := f.load()
	if s, isDefault := c2.SpaceAt(folder); s == nil || s.Name != "work" || isDefault {
		t.Errorf("a session in %s must belong to work, got %v (default %v)", folder, s, isDefault)
	}
	if _, err := os.Stat(filepath.Join(f.cfgDir, "remote", "personal")); err == nil {
		t.Error("personal has a work folder: no folder of its own")
	}
}

// folders in remote.toml picks another folder, inside a work folder of the space.
func TestFoldersSetting(t *testing.T) {
	f := newFake(t)
	sub := filepath.Join(f.root, "p", "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(text string) {
		if err := os.WriteFile(filepath.Join(f.cfgDir, SettingsFile), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("folders = { personal = \"~/p/sub\" }\n")
	c, set := f.load()
	if got := Services(c, set)[0].Folder; got != sub {
		t.Errorf("folder = %q, want %q", got, sub)
	}
	write("folders = { personal = \"~/p/missing\" }\n")
	c, set = f.load()
	if _, err := f.sys.Install(c, set); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Errorf("a missing folder: %v", err)
	}
	for _, bad := range []struct{ text, want string }{
		{"folders = { personal = \"~/w\" }\n", "work_folders"},
		{"folders = { nowhere = \"~/p\" }\n", "nowhere"},
	} {
		write(bad.text)
		if _, err := LoadSettings(c, "box"); err == nil || !strings.Contains(err.Error(), bad.want) {
			t.Errorf("%s: error = %v", bad.text, err)
		}
	}
}

func filterCalls(calls []string, prefix string) []string {
	var out []string
	for _, c := range calls {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}

// Only the configuration folder, systemd's user folder and (read only) Claude's file are
// involved: a tree walk of the fake home shows nothing else was written.
func TestInstallWritesOnlyWhereItShould(t *testing.T) {
	f := newFake(t)
	c, set := f.load()
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	err := filepath.WalkDir(f.root, func(p string, d os.DirEntry, err error) error {
		rel, _ := filepath.Rel(f.root, p)
		top := strings.Split(rel, string(filepath.Separator))
		ok := rel == "." || top[0] == ".config" || top[0] == "p" || top[0] == "w"
		if !ok {
			t.Errorf("unexpected path: %s", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f.sys.ClaudeJSON); err == nil {
		t.Error("Claude's own file was written")
	}
}

func TestInstallErrors(t *testing.T) {
	f := newFake(t)
	c, set := f.load()
	f.sys.LookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	if _, err := f.sys.Install(c, set); err == nil || !strings.Contains(err.Error(), "claude") {
		t.Errorf("no claude: %v", err)
	}
	if _, err := os.Stat(f.sys.UnitDir); err == nil {
		t.Error("it wrote before knowing it could")
	}
	// No systemd user manager (a container, WSL without systemd): nothing written, and
	// remote access not left on.
	f = newFake(t)
	c, set = f.load()
	f.failOn = "show-environment"
	if _, err := f.sys.Install(c, set); err == nil || !strings.Contains(err.Error(), "systemd") {
		t.Errorf("no systemd: %v", err)
	}
	if _, err := os.Stat(f.sys.UnitDir); err == nil {
		t.Error("units written without systemd")
	}
	if got, _ := LoadSettings(c, "box.local"); got.Enabled {
		t.Error("enabled without systemd")
	}
	f = newFake(t)
	c, set = f.load()
	f.failOn = "enable --now"
	if _, err := f.sys.Install(c, set); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("systemctl failure: %v", err)
	}
	if got, _ := LoadSettings(c, "box.local"); got.Enabled {
		t.Error("enabled after systemd refused the services")
	}
	set.Spaces = nil
	if _, err := f.sys.Install(c, set); err == nil {
		t.Error("no spaces must be an error")
	}
	f.sys.GOOS = "freebsd"
	if _, err := f.sys.Install(c, set); !errors.Is(err, ErrUnsupported) {
		t.Errorf("freebsd: %v", err)
	}
	if _, err := f.sys.Uninstall(c); !errors.Is(err, ErrUnsupported) {
		t.Errorf("freebsd uninstall: %v", err)
	}
	if _, err := f.sys.Status(c, set); !errors.Is(err, ErrUnsupported) {
		t.Errorf("freebsd status: %v", err)
	}
}

// Trust is read from Claude's file, for the folder or a parent, and an absent file means
// not trusted.
func TestTrust(t *testing.T) {
	f := newFake(t)
	c, set := f.load()
	st, err := f.sys.Status(c, set)
	if err != nil || st.Spaces[0].Trusted {
		t.Fatalf("no claude.json: %+v %v", st, err)
	}
	folder := filepath.Join(f.root, "p")
	f.trust(folder)
	if st, _ = f.sys.Status(c, set); !st.Spaces[0].Trusted {
		t.Error("the folder itself is trusted")
	}
	f.trust(f.root)
	if st, _ = f.sys.Status(c, set); !st.Spaces[0].Trusted {
		t.Error("a parent is trusted")
	}
	f.trust(filepath.Join(f.root, "elsewhere"))
	if st, _ = f.sys.Status(c, set); st.Spaces[0].Trusted {
		t.Error("another folder is trusted")
	}
	os.WriteFile(f.sys.ClaudeJSON, []byte("not json"), 0o600)
	if st, _ = f.sys.Status(c, set); st.Spaces[0].Trusted {
		t.Error("a broken file trusts nothing")
	}
}

func TestStatus(t *testing.T) {
	f := newFake(t)
	c, set := f.load()
	st, _ := f.sys.Status(c, set)
	if x := st.Spaces[0]; x.Installed || x.Name != "box.local-personal" {
		t.Errorf("before install: %+v", x)
	}
	if lines := f.sys.StatusLines("en", st); !strings.Contains(strings.Join(lines, "\n"), "remote install") {
		t.Errorf("a missing service must point at install: %q", lines)
	}
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	f.active["augmented-brain-remote-personal.service"] = "active"
	f.trust(filepath.Join(f.root, "p"))
	f.linger = "yes"
	st, _ = f.sys.Status(c, set)
	x := st.Spaces[0]
	if !x.Installed || !x.Running() || x.Enabled != "enabled" || !x.Trusted || st.Linger != LingerOn {
		t.Errorf("healthy: %+v linger %v", x, st.Linger)
	}
	if adv := f.sys.Advice("en", x, st.Linger); len(adv) != 0 {
		t.Errorf("nothing to do, got %q", adv)
	}
	// Stopped, untrusted, no linger: three things to say, in Spanish too.
	f.active["augmented-brain-remote-personal.service"] = "failed"
	f.trust("/nowhere")
	f.linger = "no"
	st, _ = f.sys.Status(c, set)
	text := strings.Join(f.sys.StatusLines("es", st), "\n")
	for _, want := range []string{"ha fallado", "cd " + filepath.Join(f.root, "p") + " && claude", "sudo loginctl enable-linger lucia", "box.local-personal"} {
		if !strings.Contains(text, want) {
			t.Errorf("status lacks %q:\n%s", want, text)
		}
	}
	what, fix, ok := f.sys.DoctorLine("en", st.Spaces[0], st.Linger)
	if ok || !strings.Contains(what, "personal") || !strings.Contains(fix, "linger") {
		t.Errorf("doctor line: %q %q %v", what, fix, ok)
	}
	f.linger = "missing"
	if st, _ = f.sys.Status(c, set); st.Linger != LingerUnknown {
		t.Errorf("no loginctl: %v", st.Linger)
	}
}

// A unit written by an earlier program (errors to null, 7-nov) is told apart, and the
// fix is to install again, which rewrites and restarts it; then all is well.
func TestOutdatedUnit(t *testing.T) {
	f := newFake(t)
	c, set := f.load()
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	f.active["augmented-brain-remote-personal.service"] = "active"
	f.trust(filepath.Join(f.root, "p"))
	f.linger = "yes"
	unit := filepath.Join(f.sys.UnitDir, "augmented-brain-remote-personal.service")
	data, _ := os.ReadFile(unit)
	old := strings.Replace(string(data), "StandardError=journal\n", "StandardError=null\n", 1)
	if err := os.WriteFile(unit, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	st, _ := f.sys.Status(c, set)
	if !st.Spaces[0].Outdated {
		t.Fatal("an old unit was not told apart")
	}
	_, fix, ok := f.sys.DoctorLine("en", st.Spaces[0], st.Linger)
	if ok || !strings.Contains(fix, "remote install") {
		t.Fatalf("doctor: %q %v", fix, ok)
	}
	f.calls = nil
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	if got := filterCalls(f.calls, "systemctl"); !slices.Contains(got, "systemctl --user restart augmented-brain-remote-personal.service") {
		t.Fatalf("install did not restart the rewritten unit: %v", got)
	}
	st, _ = f.sys.Status(c, set)
	if _, _, ok := f.sys.DoctorLine("en", st.Spaces[0], st.Linger); !ok || st.Spaces[0].Outdated {
		t.Fatalf("still outdated after install: %+v", st.Spaces[0])
	}
	// A unit with no PATH (before 9-nov) is outdated too: its sessions find nothing.
	data, _ = os.ReadFile(unit)
	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "Environment=") {
			kept = append(kept, line)
		}
	}
	if err := os.WriteFile(unit, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if st, _ = f.sys.Status(c, set); !st.Spaces[0].Outdated {
		t.Fatal("a unit with no PATH was not told apart")
	}
}

// Uninstall stops and removes our units and nothing else: a file that is not ours, the
// folder and the work_folders stay.
func TestUninstall(t *testing.T) {
	f := newFake(t)
	c, set := f.load()
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(f.sys.UnitDir, "augmented-brain-remote-mine.service")
	if err := os.WriteFile(foreign, []byte("[Service]\nExecStart=/bin/true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(f.sys.UnitDir, "ssh-agent.service")
	os.WriteFile(other, []byte("x"), 0o644)
	f.calls = nil
	removed, err := f.sys.Uninstall(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(removed, []string{"augmented-brain-remote-personal.service"}) {
		t.Errorf("removed = %v", removed)
	}
	if _, err := os.Stat(filepath.Join(f.sys.UnitDir, "augmented-brain-remote-personal.service")); err == nil {
		t.Error("the unit is still there")
	}
	for _, p := range []string{foreign, other, filepath.Join(f.root, "p")} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s was removed", p)
		}
	}
	want := []string{"systemctl --user disable --now augmented-brain-remote-personal.service", "systemctl --user daemon-reload"}
	if got := filterCalls(f.calls, "systemctl"); !slices.Equal(got, want) {
		t.Errorf("calls = %q", got)
	}
	c2, set2 := f.load()
	if set2.Enabled {
		t.Error("still enabled")
	}
	if s := c2.SpaceFor(filepath.Join(f.root, "p")); s == nil || s.Name != "personal" {
		t.Error("the work_folders must stay")
	}
	// Again: nothing to do, and no error.
	if removed, err := f.sys.Uninstall(c2); err != nil || len(removed) != 0 {
		t.Errorf("second uninstall: %v %v", removed, err)
	}
	// A service that will not stop keeps its file.
	if _, err := f.sys.Install(c2, set); err != nil {
		t.Fatal(err)
	}
	f.failOn = "disable"
	f.active["augmented-brain-remote-personal.service"] = "active"
	if _, err := f.sys.Uninstall(c2); err == nil {
		t.Error("a service that did not stop must be an error")
	}
	if _, err := os.Stat(filepath.Join(f.sys.UnitDir, "augmented-brain-remote-personal.service")); err != nil {
		t.Error("its unit must stay")
	}
}

// An earlier program refuses a setting of config.toml it does not know, so remote access
// never adds one: the file after install loads as before, and has no [remote] table.
func TestConfigTomlIsUntouchedByTheSettings(t *testing.T) {
	f := newFake(t)
	c, set := f.load()
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(f.cfgDir, config.FileName))
	if strings.Contains(string(data), "remote") && strings.Contains(string(data), "[remote") {
		t.Errorf("config.toml has a remote table:\n%s", data)
	}
	if _, err := config.Load(f.cfgDir); err != nil {
		t.Fatal(err)
	}
}

// The configuration reference gives every setting of remote.toml in backticks, in each
// language, under its own heading.
func TestSettingsDocumented(t *testing.T) {
	dir, err := doccheck.Dir()
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range doccheck.Files("configuration") {
		p, err := doccheck.Read(dir, file)
		if err != nil {
			t.Errorf("%s: %v", file, err)
			continue
		}
		var code []string
		for _, s := range p.Sections {
			if s.Heading == "`"+SettingsFile+"`" {
				code = s.Code
			}
		}
		if code == nil {
			t.Errorf("%s: no section \"## `%s`\"", file, SettingsFile)
		}
		for _, name := range []string{"enabled", "spaces", "name_prefix", "permission_mode", "folders"} {
			if !doccheck.Mentions(code, name) {
				t.Errorf("%s: the remote.toml section lacks `%s`", file, name)
			}
		}
		for _, mode := range PermissionModes {
			if !doccheck.Mentions(code, mode) {
				t.Errorf("%s: the remote.toml section lacks the mode `%s`", file, mode)
			}
		}
	}
}
