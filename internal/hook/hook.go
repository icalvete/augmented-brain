// Package hook answers Claude Code's hooks: SessionStart loads the space of the folder
// the session starts in, and PreToolUse runs the guard before a command or a write.
//
// A hook must never break the assistant: when its input or the configuration cannot be
// read, it falls back to the general rules (and lets everything else through).
package hook

import (
	"encoding/json"
	"errors"
	"github.com/icalvete/augmented-brain/internal/confighistory"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/guard"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/session"
	"github.com/icalvete/augmented-brain/internal/state"
	"github.com/icalvete/augmented-brain/internal/update"
	"github.com/icalvete/augmented-brain/internal/version"
)

// LatestVersion asks for the latest published version; tests replace it.
var LatestVersion = update.Fetch

type input struct {
	Cwd       string         `json:"cwd"`
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
	// PreToolUse: the session's permission mode (default, plan, acceptEdits, auto,
	// dontAsk, bypassPermissions), when Claude Code sends it.
	PermissionMode string `json:"permission_mode"`
	// Common to every hook.
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	// SessionStart: startup, resume, clear, compact or fork.
	Source string `json:"source"`
	// SessionEnd: why it ended (clear, logout, prompt_input_exit, other...).
	Reason string `json:"reason"`
	// SessionStart: the model, when Claude Code sends it (it may not).
	Model string `json:"model"`
	// Stop: true while the assistant goes on because of a Stop hook.
	StopHookActive bool `json:"stop_hook_active"`
	// PostCompact: manual or auto, and the summary Claude Code wrote.
	Trigger        string `json:"trigger"`
	CompactSummary string `json:"compact_summary"`
}

type specific struct {
	HookEventName            string `json:"hookEventName"`
	AdditionalContext        string `json:"additionalContext,omitempty"`
	PermissionDecision       string `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
}

type output struct {
	HookSpecificOutput specific `json:"hookSpecificOutput"`
	// SystemMessage is shown to the user by Claude Code itself, not by the assistant.
	SystemMessage string `json:"systemMessage,omitempty"`
}

// userNotice is what the user must hear about the guard's lists (state.TakeGuardNotice),
// about what a tool saved for every session (state.TakeToolNotice) and about the
// sessions the usage limit stopped (state.TakeQuotaNotice), as a message
// Claude Code shows them directly, or "". Taking it removes it.
func userNotice(lang, cfgDir string) string {
	var parts []string
	if n := state.TakeGuardNotice(cfgDir); n != "" {
		parts = append(parts, i18n.T(lang, "hook.guardnotice", n))
	}
	if n := state.TakeToolNotice(cfgDir); n != "" {
		parts = append(parts, i18n.T(lang, "hook.toolnotice", n))
	}
	if n := state.TakeQuotaNotice(cfgDir); n != "" {
		parts = append(parts, i18n.T(lang, "hook.quotanotice", n))
	}
	return strings.Join(parts, "\n\n")
}

func read(in io.Reader) input {
	i, _ := readChecked(in)
	return i
}

// readChecked is read, saying whether the input could be decoded: the guard refuses a
// tool call it cannot read rather than let it through.
func readChecked(in io.Reader) (input, bool) {
	var i input
	err := json.NewDecoder(in).Decode(&i)
	if i.Cwd == "" {
		i.Cwd, _ = os.Getwd()
	}
	return i, err == nil
}

func write(out io.Writer, o output) error {
	return json.NewEncoder(out).Encode(o)
}

// load reads the configuration and the space of dir. Both may be nil; err says why
// the configuration could not be read.
func load(cfgDir, dir string) (*config.Config, *config.Space, error) {
	c, err := config.Load(cfgDir)
	if err != nil {
		return nil, nil, err
	}
	s, _ := c.SpaceAt(dir)
	return c, s, nil
}

// SessionStart adds the space's profile, rules and index to the session, in one text
// within session.Budget (plugins before 0.5.5 call it so).
func SessionStart(in io.Reader, out io.Writer, cfgDir string) error {
	return SessionStartPart(in, out, cfgDir, 0)
}

// SessionStartPart adds part n of the session text (1 to session.Parts; 0 for all of
// it in one). The plugin runs one hook per part, at the same time: only part 1 says
// what is wrong and does the work behind the notes (a migration, recording the session).
func SessionStartPart(in io.Reader, out io.Writer, cfgDir string, n int) error {
	i := read(in)
	c, s, err := load(cfgDir, i.Cwd)
	if n > 1 && (err != nil || s == nil) {
		return write(out, output{HookSpecificOutput: specific{HookEventName: "SessionStart"}})
	}
	var text string
	switch {
	case errors.Is(err, config.ErrNotFound):
		text = "augmented brain is installed but not set up yet: no memory is loaded. The user can run `augmented-brain init`."
	case err != nil:
		text = "augmented brain cannot read its configuration, so no memory is loaded, and until it is fixed the guard asks the user before every aws, ssh, scp, rsync, sftp and mosh command (it cannot tell production profiles and untouchable hosts). Tell the user: " + err.Error() + ". `augmented-brain doctor` shows the problem."
	case s == nil:
		text = session.NoSpace(c, i.Cwd)
	case n > 1:
		// Part 1 may be moving the notes to a new format: wait for it, briefly.
		if unlock, err := filelock.LockWithin(filepath.Join(cfgDir, backup.NotesLock), LockWait); err == nil {
			unlock()
		}
		text = session.Part(c, s, "", n)
	case n == 1:
		task := launchedTask(cfgDir, i)
		note := startNote(c, s, i, cfgDir, task)
		freshBaseline(c, s, i, cfgDir)
		text = session.PartFor(c, s, note, task, 1)
	default:
		task := launchedTask(cfgDir, i)
		note := startNote(c, s, i, cfgDir, task)
		freshBaseline(c, s, i, cfgDir)
		text = session.ContextFor(c, s, note, task)
	}
	o := output{HookSpecificOutput: specific{HookEventName: "SessionStart", AdditionalContext: text}}
	if n <= 1 && c != nil {
		o.SystemMessage = userNotice(c.Language, cfgDir)
	}
	return write(out, o)
}

// startNote is what this session must hear first, on top of the space's text: the
// default space was loaded, the backup failed, a new version is out, the notes moved
// to a new format, a session to recover, the guard's lists to ask for again. It also
// does the work behind some of them (the migration, recording the session). A session
// opened for a task (task not "") is not offered other sessions to recover, nor told
// of other sessions to restart: those are for the user's other sessions.
func startNote(c *config.Config, s *config.Space, i input, cfgDir, task string) string {
	var note string
	if _, isDefault := c.SpaceAt(i.Cwd); isDefault {
		note = session.DefaultNote(c, s, i.Cwd)
	}
	note += backupNote(c, s, cfgDir)
	if latest := update.Available(cfgDir, version.Version, time.Now(), LatestVersion); latest != "" {
		note += session.UpdateNote(c.Language, version.Version, latest)
	}
	note += migrate(c, s, cfgDir)
	note += sessionStarted(c, s, i, cfgDir, task == "")
	note += modelStartNote(c.Language, cfgDir, i)
	note += askSafety(c, cfgDir)
	note += rootNote(c)
	note += oldServersNote(c, task == "")
	note += guardSealNote(c, cfgDir, i.Source)
	note += unknownSettingsNote(c, cfgDir)
	note += launcherNote(c)
	// A change made by hand since the last version becomes one: it can be undone too.
	_, _ = confighistory.Record(cfgDir, confighistory.ByOutside, "")
	return note
}

// launchedTask is the task this session was opened for with session start, or "".
func launchedTask(cfgDir string, i input) string {
	return remote.LaunchedTask(cfgDir, i.SessionID, Getenv("CLAUDE_JOB_DIR"), i.Cwd)
}

// backupNote says when the space's last backup failed or left files out.
func backupNote(c *config.Config, s *config.Space, cfgDir string) string {
	if s.Backup == "" {
		return ""
	}
	now := time.Now()
	if began, cut := backup.CutShort(cfgDir, s.Name, now); cut {
		return session.BackupCutShortNote(c.Language, s.Name, began)
	}
	st, ok := backup.LastStatus(cfgDir, s.Name)
	switch {
	case !ok:
		return ""
	case !st.OK:
		return session.BackupFailedNote(c.Language, s.Name, st.Time, st.Error)
	case st.Warning != "":
		return session.BackupWarningNote(c.Language, s.Name, st.Time, st.Warning)
	case st.Behind(s.Memory, now):
		return session.BackupBehindNote(c.Language, s.Name, st.Time)
	}
	return ""
}

// PreToolUse runs the guard on a shell command (Bash, or Monitor, which runs a command
// in the background and is checked the same way) or on the text of a file write, and
// blocks PowerShell commands, which the guard cannot read. A Monitor that opens a
// WebSocket has no command and runs no program, so it goes through. A tool that saves
// what every session answers to (memoryTools) asks first when the user wants that.
func PreToolUse(in io.Reader, out io.Writer, cfgDir string) error {
	i, ok := readChecked(in)
	if tool := memoryTool(i.ToolName); ok && tool != "" {
		decision, reason := trustDecision(cfgDir, language(cfgDir), i, tool)
		if decision == "" {
			return nil
		}
		return write(out, output{HookSpecificOutput: specific{HookEventName: "PreToolUse",
			PermissionDecision: decision, PermissionDecisionReason: "augmented brain: " + reason}})
	}
	p := policy(cfgDir, i.Cwd)
	// In bypassPermissions Claude Code may skip the dialog an "ask" opens, so what must
	// never go without the user's yes is refused there instead.
	p.NoPrompt = i.PermissionMode == "bypassPermissions"
	var d guard.Decision
	switch {
	case !ok:
		d = guard.Decision{Verdict: guard.Deny, Reason: i18n.T(p.Lang, "guard.unreadable")}
	default:
		d = toolDecision(p, i)
	}
	decision := map[guard.Verdict]string{guard.Deny: "deny", guard.Ask: "ask"}[d.Verdict]
	if decision == "" {
		return nil
	}
	return write(out, output{HookSpecificOutput: specific{HookEventName: "PreToolUse",
		PermissionDecision: decision, PermissionDecisionReason: "augmented brain: " + d.Reason}})
}

// toolCases are the tools toolDecision checks; the plugin's PreToolUse matcher names
// the same ones (a test compares them).
var toolCases = []string{"Bash", "Monitor", "PowerShell", "Read", "Grep", "Write", "Edit", "MultiEdit", "NotebookEdit"}

// toolDecision is the guard's answer to one tool call. A call whose command or path is
// there but is not text, or a Bash call with no command, is refused: a new shape of
// call from Claude Code must not switch the guard off without anyone noticing.
func toolDecision(p guard.Policy, i input) guard.Decision {
	unreadable := guard.Decision{Verdict: guard.Deny, Reason: i18n.T(p.Lang, "guard.unreadable")}
	// text is the field key of the call: ok is false when it is there but not text.
	text := func(key string) (string, bool) {
		v, there := i.ToolInput[key]
		if !there {
			return "", true
		}
		s, isText := v.(string)
		return s, isText
	}
	switch i.ToolName {
	case "Bash", "Monitor":
		cmd, ok := text("command")
		if _, there := i.ToolInput["command"]; !ok || !there && i.ToolName == "Bash" {
			return unreadable // a Monitor with no command watches a WebSocket: nothing runs
		}
		return p.Command(cmd)
	case "PowerShell":
		return guard.PowerShell(p.Lang)
	case "Read", "Grep":
		key := map[string]string{"Read": "file_path", "Grep": "path"}[i.ToolName]
		path, ok := text(key)
		if !ok {
			return unreadable
		}
		return p.ReadFile(path)
	case "Write", "Edit", "MultiEdit", "NotebookEdit":
		path, ok := text("file_path")
		notebook, ok2 := text("notebook_path")
		if !ok || !ok2 {
			return unreadable
		}
		if path == "" {
			path = notebook
		}
		return p.WriteFile(path, written(i.ToolInput))
	}
	return guard.Decision{}
}

// Getenv reads the hook's environment, which the assistant's commands inherit; tests
// replace it.
var Getenv = os.Getenv

// policy is what the guard applies to a command run in dir: the production profiles and
// untouchable hosts of the whole machine (every space's lists too), the repositories
// every space lets the assistant commit in, the AWS profile the command inherits, and
// what else it inherits that changes which repository gh or glab work on.
// When the configuration exists but cannot be read, the guard is told, so it asks
// before every command those lists would cover instead of letting them through unseen.
func policy(cfgDir, dir string) guard.Policy {
	// The configuration's language when it loads; the system's when it does not.
	p := guard.Policy{Cwd: dir, Env: map[string]string{}, Lang: i18n.FromEnv()}
	for _, k := range []string{"AWS_PROFILE", "AWS_DEFAULT_PROFILE",
		"GH_REPO", "GH_HOST", "GITLAB_REPO", "GITLAB_HOST", "GIT_DIR", "GIT_WORK_TREE"} {
		if v := Getenv(k); v != "" {
			p.Env[k] = v
		}
	}
	c, _, err := load(cfgDir, dir)
	switch {
	case errors.Is(err, config.ErrNotFound) && !leftovers(cfgDir):
		// Not set up yet: no lists to apply.
	case err != nil:
		p.ConfigProblem = err.Error()
	default:
		p.ProductionAWSProfiles, p.UntouchableHosts = c.GuardLists()
		p.GitAllowedRepos, p.GitAskRepos = c.GitLists()
		roots, _ := c.RootExceptions()
		for _, r := range roots {
			p.RootAsk = append(p.RootAsk, guard.RootException{Host: r.Host, Until: r.Until})
		}
		p.Hostname, _ = os.Hostname()
		p.Lang = c.Language
		if g, err := config.LoadGuard(cfgDir); err != nil {
			// The untouchable repositories are unknown, and so is whether the rest
			// was left as the user meant: everything the lists cover asks.
			p.ConfigProblem = err.Error()
			p.GuardProblem = err.Error()
		} else {
			p.UntouchableRepos = g.UntouchableRepos
			seal, err := state.ReadSeal(cfgDir)
			switch {
			case err != nil:
				p.ConfigChanged = true
				p.SealProblem = err.Error()
			case seal != "" && seal != c.GuardFingerprint(g):
				p.ConfigChanged = true
			}
		}
	}
	p.Protected = protectedFiles(cfgDir, dir)
	p.ProtectedDirs = protectedDirs()
	p.Hidden, p.ConfigDirs = hiddenFiles(cfgDir)
	return p
}

// leftovers says whether cfgDir holds what only a set-up configuration writes (the
// guard's lists, its state, its history) though config.toml is gone: that is a broken
// configuration, not one never set up, and the guard asks instead of applying nothing.
func leftovers(cfgDir string) bool {
	for _, f := range []string{config.GuardFile, "state", "history"} {
		if _, err := os.Stat(filepath.Join(cfgDir, f)); err == nil {
			return true
		}
	}
	return false
}

// written collects the text a write would put in the file: every string in the tool's
// input except the path and the text being replaced.
func written(v any) string {
	var parts []string
	var walk func(key string, v any)
	walk = func(key string, v any) {
		switch x := v.(type) {
		case string:
			if key != "file_path" && key != "notebook_path" && key != "old_string" {
				parts = append(parts, x)
			}
		case map[string]any:
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				walk(k, x[k])
			}
		case []any:
			for _, e := range x {
				walk(key, e)
			}
		}
	}
	walk("", v)
	out := ""
	for _, p := range parts {
		out += p + "\n"
	}
	return out
}

// protectedFiles are the files the assistant does not change on its own: the guard's
// configuration, and Claude Code's settings (yours and the project's), where a line can
// switch the plugin, and so the guard, off.
func protectedFiles(cfgDir, dir string) []string {
	var list []string
	if cfgDir != "" { // unknown: never a file of the current folder in its place
		list = []string{filepath.Join(cfgDir, config.FileName), filepath.Join(cfgDir, config.GuardFile),
			filepath.Join(cfgDir, config.SecretsFile), filepath.Join(cfgDir, "remote.toml"),
			filepath.Join(cfgDir, config.SessionFile)}
		// The record of the lists the user last saw: written by hand, it would
		// accept lists changed outside the panel.
		list = append(list, filepath.Join(cfgDir, state.Dir, state.SealFile))
		// The plugin's marketplace: Claude Code copies the plugin from it.
		for _, f := range claudecode.Files() {
			list = append(list, filepath.Join(claudecode.LocalDir(cfgDir), filepath.FromSlash(f)))
		}
	}
	if home, err := paths.Home(); err == nil {
		for _, f := range []string{"settings.json", "settings.local.json"} {
			list = append(list, filepath.Join(home, ".claude", f))
		}
		// Which plugins Claude Code loads, and from where.
		for _, f := range []string{"installed_plugins.json", "known_marketplaces.json"} {
			list = append(list, filepath.Join(home, ".claude", "plugins", f))
		}
		// The program the hooks run: they put ~/.local/bin first in their PATH.
		list = append(list, filepath.Join(home, ".local", "bin", programName()))
	}
	if exe, err := os.Executable(); err == nil {
		list = append(list, exe) // the program running this hook, wherever it is
	}
	for _, f := range []string{"settings.json", "settings.local.json"} {
		list = append(list, filepath.Join(dir, ".claude", f))
	}
	for i, f := range list {
		if r, err := paths.Resolve(f); err == nil {
			list[i] = r
		}
	}
	return list
}

// protectedDirs are the folders whose every file is protected: Claude Code's copy of
// the plugin, one folder per version, where it reads the hooks that are the guard.
func protectedDirs() []string {
	home, err := paths.Home()
	if err != nil {
		return nil
	}
	d := filepath.Join(home, ".claude", "plugins", "cache", claudecode.MarketplaceName)
	if r, err := paths.Resolve(d); err == nil {
		d = r
	}
	return []string{d}
}

// programName is the program's file name on this system.
func programName() string {
	if runtime.GOOS == "windows" {
		return "augmented-brain.exe"
	}
	return "augmented-brain"
}

// hiddenFiles are the files the assistant never reads (the panel's token file) and the
// ways a command can write the configuration folder.
func hiddenFiles(cfgDir string) (hidden, dirs []string) {
	if cfgDir == "" {
		return nil, []string{".config/augmented-brain", "AB_CONFIG_DIR"}
	}
	f := filepath.Join(cfgDir, "panel.json")
	if r, err := paths.Resolve(f); err == nil {
		f = r
	}
	dirs = []string{cfgDir, paths.Tilde(cfgDir), ".config/augmented-brain", "AB_CONFIG_DIR"}
	return []string{f}, dirs
}

// unknownSettingsNote, at the start of a session, has the assistant tell the user about
// settings of guard.toml or session.toml this program ignores: a misspelled one does
// nothing, and nothing else would say so. Every session, until the file is corrected.
func unknownSettingsNote(c *config.Config, cfgDir string) string {
	var note string
	for _, f := range config.UnknownByFile(c.Language, config.UnknownSettings(cfgDir)) {
		note += "\n" + i18n.T(c.Language, "hook.unknownsettings", f[0], f[1]) + " " + session.TellUserLine(c.Language) + ", " +
			i18n.T(c.Language, "hook.unknownsettings.fix", paths.Tilde(filepath.Join(cfgDir, f[0]))) + "\n"
	}
	return note
}

// guardSealNote compares the guard's lists with the ones the user last saw and says so
// when they differ. Only a session the user starts (source startup, or none from an
// earlier Claude Code) takes the current lists as the ones the user knows from now on,
// and then the user hears it directly, not through the assistant. A compaction, /clear,
// a resume or a fork is not the user looking: the lists stay unsealed and the guard
// keeps asking before a commit, push or merge until the user accepts them in the panel.
func guardSealNote(c *config.Config, cfgDir, source string) string {
	g, err := config.LoadGuard(cfgDir)
	if err != nil {
		return ""
	}
	cur := c.GuardFingerprint(g)
	old, err := state.ReadSeal(cfgDir)
	if err != nil {
		return "\n" + i18n.T(c.Language, "hook.seal.unread", err) + " " + session.TellUserLine(c.Language) + ".\n"
	}
	if old == cur {
		return ""
	}
	if old == "" {
		_ = state.WriteSeal(cfgDir, cur) // the first time: nothing to compare
		return ""
	}
	if source != "" && source != "startup" {
		return "\n" + i18n.T(c.Language, "hook.seal.kept") + " " + session.TellUserLine(c.Language) + ".\n"
	}
	if err := state.WriteSeal(cfgDir, cur); err != nil {
		return "\n" + i18n.T(c.Language, "hook.seal.unwritten", err) + " " + session.TellUserLine(c.Language) + ".\n"
	}
	_ = state.AddGuardNotice(cfgDir, i18n.T(c.Language, "hook.seal.startnotice"), time.Now())
	return "\n" + i18n.T(c.Language, "hook.seal.taken") + " " + session.TellUserLine(c.Language) + ".\n"
}

// GOOS is the system the hook runs on; tests change it.
var GOOS = runtime.GOOS

// launcherNote tells every session that `augmented-brain session start` exists and how
// to use it, so it never tells the user there is no way to open a session for a task.
// Only on Linux: the launcher is Linux only.
func launcherNote(c *config.Config) string {
	if GOOS != "linux" {
		return ""
	}
	return "\n" + i18n.T(c.Language, "hook.launcher") + "\n"
}
