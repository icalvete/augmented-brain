package codex

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeCodex is Codex as far as Install, Uninstall and Check see it: its plugin command
// line and its app server, over one state.
type fakeCodex struct {
	marketplace bool
	plugin      bool
	hashes      map[string]string // the plugin's hooks: key → current hash
	state       map[string]string // hooks.state: key → trusted hash
	tty         any               // features.unified_exec_tty; nil when not set
	calls       []string
	failAdd     bool
}

func newFake() *fakeCodex {
	return &fakeCodex{state: map[string]string{"other:x": "sha256:theirs"}}
}

func (f *fakeCodex) run(name string, args ...string) (string, error) {
	cmd := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, cmd)
	switch {
	case strings.HasPrefix(cmd, "codex plugin marketplace add "):
		f.marketplace = true
	case cmd == "codex plugin add "+Plugin:
		if f.failAdd || !f.marketplace {
			return "error: plugin not found", errors.New("exit status 1")
		}
		f.plugin = true
		f.hashes = map[string]string{Plugin + ":hooks/hooks.json:pre_tool_use:0:0": "sha256:guard"}
		for n := range HookCount - 1 {
			f.hashes[fmt.Sprintf("%s:hooks/hooks.json:session_start:0:%d", Plugin, n)] = fmt.Sprintf("sha256:start%d", n)
		}
	case cmd == "codex plugin remove "+Plugin:
		if !f.plugin {
			return "plugin `brain@augmented-brain` is not installed", errors.New("exit status 1")
		}
		f.plugin, f.hashes = false, nil
	case cmd == "codex plugin marketplace remove "+MarketplaceName:
		if !f.marketplace {
			return "marketplace `augmented-brain` is not configured", errors.New("exit status 1")
		}
		f.marketplace = false
	default:
		return "unknown command", errors.New("exit status 2")
	}
	return "ok", nil
}

func (f *fakeCodex) serve() (Server, error) { return f, nil }
func (f *fakeCodex) Close() error           { return nil }

func (f *fakeCodex) Call(method string, params, result any) error {
	f.calls = append(f.calls, "rpc "+method)
	var p map[string]any
	b, _ := json.Marshal(params)
	json.Unmarshal(b, &p)
	var res any
	switch method {
	case "plugin/list":
		res = map[string]any{"marketplaces": []any{map[string]any{"plugins": []any{
			map[string]any{"id": Plugin, "installed": f.plugin, "enabled": f.plugin}}}}}
	case "hooks/list":
		var hooks []any
		for k, h := range f.hashes {
			trust := "untrusted"
			if t, ok := f.state[k]; ok {
				trust = "modified"
				if t == h {
					trust = "trusted"
				}
			}
			hooks = append(hooks, map[string]any{"key": k, "pluginId": Plugin, "currentHash": h, "trustStatus": trust})
		}
		hooks = append(hooks, map[string]any{"key": "/home/x/.codex/hooks.json:pre_tool_use:0:0", "currentHash": "sha256:mine", "trustStatus": "trusted"})
		res = map[string]any{"data": []any{map[string]any{"hooks": hooks}}}
	case "config/read":
		state := map[string]any{}
		for k, h := range f.state {
			state[k] = map[string]any{"trusted_hash": h}
		}
		features := map[string]any{}
		if f.tty != nil {
			features["unified_exec_tty"] = f.tty
		}
		res = map[string]any{"config": map[string]any{"features": features, "hooks": map[string]any{"state": state}}}
	case "config/batchWrite":
		for _, e := range p["edits"].([]any) {
			edit := e.(map[string]any)
			key := edit["keyPath"].(string)
			switch {
			case key == "features.unified_exec_tty":
				f.tty = edit["value"]
			case strings.HasPrefix(key, `hooks.state."`):
				var hook string
				if err := json.Unmarshal([]byte(strings.TrimPrefix(key, "hooks.state.")), &hook); err != nil {
					return err
				}
				if edit["value"] == nil {
					delete(f.state, hook)
				} else {
					f.state[hook] = edit["value"].(map[string]any)["trusted_hash"].(string)
				}
			default:
				return fmt.Errorf("unexpected key %q", key)
			}
		}
	default:
		return fmt.Errorf("unknown method %s", method)
	}
	if result == nil {
		return nil
	}
	b, _ = json.Marshal(res)
	return json.Unmarshal(b, result)
}

func testEnv(t *testing.T, f *fakeCodex) Env {
	return Env{Run: f.run, Serve: f.serve, Dir: filepath.Join(t.TempDir(), "codex-marketplace"),
		Exe: "/opt/ab dir/augmented-brain", Version: "9.9.9", GOOS: "linux", Cwd: t.TempDir()}
}

func TestInstallTrustsAndRepeats(t *testing.T) {
	f := newFake()
	e := testEnv(t, f)
	if st := Check(e); !st.NotInstalled() || st.OK() {
		t.Fatalf("before install: %+v", st)
	}
	for i := range 2 {
		if err := Install(e); err != nil {
			t.Fatalf("install %d: %v", i+1, err)
		}
		st := Check(e)
		if !st.OK() || st.Trusted != HookCount {
			t.Fatalf("install %d: %+v", i+1, st)
		}
	}
	if f.tty != false {
		t.Errorf("unified_exec_tty = %v, want false", f.tty)
	}
	if f.state["other:x"] != "sha256:theirs" || len(f.state) != HookCount+1 {
		t.Errorf("trust entries: %v", f.state)
	}
	if b, err := os.ReadFile(filepath.Join(e.Dir, "brain", ".codex-plugin", "plugin.json")); err != nil || !strings.Contains(string(b), `"9.9.9"`) {
		t.Errorf("plugin.json: %s %v", b, err)
	}
}

func TestNewVersionIsSeenAndInstallFixesIt(t *testing.T) {
	f := newFake()
	e := testEnv(t, f)
	if err := Install(e); err != nil {
		t.Fatal(err)
	}
	f.hashes[Plugin+":hooks/hooks.json:pre_tool_use:0:0"] = "sha256:guard-v2" // a new program, not installed again
	st := Check(e)
	if st.OK() || !st.Modified {
		t.Fatalf("a changed hook must show: %+v", st)
	}
	if p := Problems("en", st); len(p) != 1 || !strings.Contains(p[0], "changed") {
		t.Errorf("problems: %q", p)
	}
	if err := Install(e); err != nil || !Check(e).OK() {
		t.Fatalf("install again: %v %+v", err, Check(e))
	}
}

func TestUninstallLeavesTheRest(t *testing.T) {
	f := newFake()
	e := testEnv(t, f)
	if err := Install(e); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(e.Dir, "notes.txt")
	os.WriteFile(foreign, []byte("mine"), 0o644)
	for i := range 2 {
		if err := Uninstall(e); err != nil {
			t.Fatalf("uninstall %d: %v", i+1, err)
		}
	}
	if f.plugin || f.marketplace || len(f.state) != 1 || f.state["other:x"] == "" {
		t.Errorf("after uninstall: plugin %v marketplace %v state %v", f.plugin, f.marketplace, f.state)
	}
	if f.tty != false {
		t.Errorf("unified_exec_tty must stay off: %v", f.tty)
	}
	if _, err := os.Stat(filepath.Join(e.Dir, "brain")); !os.IsNotExist(err) {
		t.Errorf("the plugin's files are still there: %v", err)
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Errorf("a file the program did not write was removed: %v", err)
	}
	if st := Check(e); !st.NotInstalled() {
		t.Errorf("after uninstall: %+v", st)
	}
}

func TestInstallFailsLoudly(t *testing.T) {
	f := newFake()
	f.failAdd = true
	if err := Install(testEnv(t, f)); err == nil || !strings.Contains(err.Error(), "adding the plugin") {
		t.Errorf("a failed plugin add must say so: %v", err)
	}
}

func TestHooks(t *testing.T) {
	fs, err := files("/opt/ab dir/augmented-brain", "1.0.0", "linux")
	if err != nil {
		t.Fatal(err)
	}
	var h struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
				Timeout int    `json:"timeout"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(fs["brain/hooks/hooks.json"], &h); err != nil {
		t.Fatal(err)
	}
	start := h.Hooks["SessionStart"][0].Hooks
	if len(start) != HookCount-1 {
		t.Fatalf("%d start hooks", len(start))
	}
	for i, s := range start {
		if want := fmt.Sprintf("'/opt/ab dir/augmented-brain' hook session-start %d codex", i+1); s.Command != want {
			t.Errorf("start %d: %q, want %q", i+1, s.Command, want)
		}
	}
	guard := h.Hooks["PreToolUse"][0].Hooks[0].Command
	if !strings.HasPrefix(guard, "'/opt/ab dir/augmented-brain' hook pre-tool-use codex; s=$?; [ $s -eq 127 ]") ||
		!strings.Contains(guard, "exit 2") {
		t.Errorf("guard: %q", guard)
	}
	if !strings.Contains(string(fs["brain/.mcp.json"]), `"AB_MCP_TASKS_ONLY": "1"`) {
		t.Errorf(".mcp.json: %s", fs["brain/.mcp.json"])
	}
	win, _ := files(`C:\ab\augmented-brain.exe`, "1.0.0", "windows")
	if !strings.Contains(string(win["brain/hooks/hooks.json"]), `"augmented-brain hook pre-tool-use codex"`) {
		t.Errorf("windows hooks: %s", win["brain/hooks/hooks.json"])
	}
}

func TestStateKeyQuotes(t *testing.T) {
	if got := stateKey("brain@augmented-brain:hooks/hooks.json:pre_tool_use:0:0"); got != `hooks.state."brain@augmented-brain:hooks/hooks.json:pre_tool_use:0:0"` {
		t.Errorf("stateKey: %s", got)
	}
}
