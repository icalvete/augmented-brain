// Package hook answers Claude Code's hooks: SessionStart loads the space of the folder
// the session starts in, and PreToolUse runs the guard before a command or a write.
//
// A hook must never break the assistant: when its input or the configuration cannot be
// read, it falls back to the general rules (and lets everything else through).
package hook

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/guard"
	"github.com/icalvete/augmented-brain/internal/session"
	"github.com/icalvete/augmented-brain/internal/update"
	"github.com/icalvete/augmented-brain/internal/version"
)

// LatestVersion asks for the latest published version; tests replace it.
var LatestVersion = update.Fetch

type input struct {
	Cwd       string         `json:"cwd"`
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
	// Common to every hook.
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	// SessionStart: startup, resume, clear, compact or fork.
	Source string `json:"source"`
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
}

func read(in io.Reader) input {
	var i input
	_ = json.NewDecoder(in).Decode(&i)
	if i.Cwd == "" {
		i.Cwd, _ = os.Getwd()
	}
	return i
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

// SessionStart adds the space's profile, rules and index to the session.
func SessionStart(in io.Reader, out io.Writer, cfgDir string) error {
	i := read(in)
	c, s, err := load(cfgDir, i.Cwd)
	var text string
	switch {
	case errors.Is(err, config.ErrNotFound):
		text = "augmented brain is installed but not set up yet: no memory is loaded. The user can run `augmented-brain init`."
	case err != nil:
		text = "augmented brain cannot read its configuration, so no memory is loaded, and until it is fixed the guard asks the user before every aws, ssh, scp, rsync, sftp and mosh command (it cannot tell production profiles and untouchable hosts). Tell the user: " + err.Error() + ". `augmented-brain doctor` shows the problem."
	case s == nil:
		text = session.NoSpace(c, i.Cwd)
	default:
		text = session.ContextWith(c, s, startNote(c, s, i, cfgDir))
	}
	return write(out, output{specific{HookEventName: "SessionStart", AdditionalContext: text}})
}

// startNote is what this session must hear first, on top of the space's text: the
// default space was loaded, the backup failed, a new version is out, the notes moved
// to a new format, a session to recover, the guard's lists to ask for again. It also
// does the work behind some of them (the migration, recording the session).
func startNote(c *config.Config, s *config.Space, i input, cfgDir string) string {
	var note string
	if _, isDefault := c.SpaceAt(i.Cwd); isDefault {
		note = session.DefaultNote(c, s, i.Cwd)
	}
	note += backupNote(c, s, cfgDir)
	if latest := update.Available(cfgDir, version.Version, time.Now(), LatestVersion); latest != "" {
		note += session.UpdateNote(c.Language, version.Version, latest)
	}
	note += migrate(c, s, cfgDir)
	note += sessionStarted(c, s, i, cfgDir)
	note += askSafety(c, cfgDir)
	return note
}

// backupNote says when the space's last backup failed or left files out.
func backupNote(c *config.Config, s *config.Space, cfgDir string) string {
	st, ok := backup.LastStatus(cfgDir, s.Name)
	switch {
	case !ok || s.Backup == "":
		return ""
	case !st.OK:
		return session.BackupFailedNote(c.Language, s.Name, st.Time, st.Error)
	case st.Warning != "":
		return session.BackupWarningNote(c.Language, s.Name, st.Time, st.Warning)
	}
	return ""
}

// PreToolUse runs the guard on a shell command (Bash, or Monitor, which runs a command
// in the background and is checked the same way) or on the text of a file write, and
// blocks PowerShell commands, which the guard cannot read. A Monitor that opens a
// WebSocket has no command and runs no program, so it goes through.
func PreToolUse(in io.Reader, out io.Writer, cfgDir string) error {
	i := read(in)
	p := policy(cfgDir, i.Cwd)
	var d guard.Decision
	switch i.ToolName {
	case "Bash", "Monitor":
		cmd, _ := i.ToolInput["command"].(string)
		d = p.Command(cmd)
	case "PowerShell":
		d = guard.PowerShell()
	case "Write", "Edit", "MultiEdit", "NotebookEdit":
		d = guard.Write(written(i.ToolInput))
	default:
		return nil
	}
	decision := map[guard.Verdict]string{guard.Deny: "deny", guard.Ask: "ask"}[d.Verdict]
	if decision == "" {
		return nil
	}
	return write(out, output{specific{HookEventName: "PreToolUse",
		PermissionDecision: decision, PermissionDecisionReason: "augmented brain: " + d.Reason}})
}

// Getenv reads the hook's environment, which the assistant's commands inherit; tests
// replace it.
var Getenv = os.Getenv

// policy is what the guard applies to a command run in dir: the production profiles and
// untouchable hosts of the whole machine (every space's lists too), the repositories
// every space lets the assistant commit in, and the AWS profile the command inherits.
// When the configuration exists but cannot be read, the guard is told, so it asks
// before every command those lists would cover instead of letting them through unseen.
func policy(cfgDir, dir string) guard.Policy {
	p := guard.Policy{Cwd: dir, Env: map[string]string{}}
	for _, k := range []string{"AWS_PROFILE", "AWS_DEFAULT_PROFILE"} {
		if v := Getenv(k); v != "" {
			p.Env[k] = v
		}
	}
	c, _, err := load(cfgDir, dir)
	switch {
	case errors.Is(err, config.ErrNotFound):
	case err != nil:
		p.ConfigProblem = err.Error()
	default:
		p.ProductionAWSProfiles, p.UntouchableHosts = c.GuardLists()
		p.GitAllowedRepos, p.GitAskRepos = c.GitLists()
	}
	return p
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
