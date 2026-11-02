// Command augmented-brain sets up and runs augmented brain: memory, method and
// a guard for AI coding assistants.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/doctor"
	"github.com/icalvete/augmented-brain/internal/format"
	"github.com/icalvete/augmented-brain/internal/hook"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/mcpserver"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(errOut, i18n.T(userLanguage(), "usage"))
		return 2
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintln(out, version.Version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(out, i18n.T(userLanguage(), "usage"))
		return 0
	case "init":
		return cmdInit(args[1:], in, out, errOut)
	case "space":
		if len(args) > 1 && args[1] == "add" {
			return cmdSpaceAdd(args[2:], in, out, errOut)
		}
		return cmdSpace(out, errOut)
	case "check":
		return cmdCheck(out, errOut)
	case "doctor":
		return cmdDoctor(out, errOut)
	case "import":
		return cmdImport(args[1:], out, errOut)
	case "mcp":
		return cmdMCP(errOut)
	case "hook":
		return cmdHook(args[1:], in, out, errOut)
	case "backup":
		return cmdBackup(args[1:], out, errOut)
	case "restore":
		return cmdRestore(args[1:], out, errOut)
	case "panel":
		return cmdPanel(args[1:], out, errOut)
	case "update":
		return cmdUpdate(args[1:], out, errOut)
	case "uninstall":
		return cmdUninstall(args[1:], in, out, errOut)
	default:
		l := userLanguage()
		fmt.Fprintf(errOut, "%s\n\n%s", i18n.T(l, "usage.unknown", args[0]), i18n.T(l, "usage"))
		return 2
	}
}

// userLanguage is the language of the configuration, or the system's when there is
// none yet.
func userLanguage() string {
	if dir, err := config.Dir(); err == nil {
		if c, err := config.Load(dir); err == nil {
			return c.Language
		}
	}
	return i18n.FromEnv()
}

// loadConfig loads the configuration, or says why it cannot (with what to run when
// there is none yet).
func loadConfig(errOut io.Writer) (*config.Config, bool) {
	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return nil, false
	}
	c, err := config.Load(dir)
	if errors.Is(err, config.ErrNotFound) {
		fmt.Fprintln(errOut, i18n.T(i18n.FromEnv(), "config.missing", err))
		return nil, false
	}
	if err != nil {
		fmt.Fprintln(errOut, err)
		return nil, false
	}
	return c, true
}

// currentSpace loads the configuration and finds the space of the working folder, or
// the default space when the folder is outside every space.
func currentSpace(errOut io.Writer) (*config.Config, *config.Space, bool) {
	c, ok := loadConfig(errOut)
	if !ok {
		return nil, nil, false
	}
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return nil, nil, false
	}
	// The same space the hooks and the MCP server use: the default space outside every
	// work folder.
	s, _ := c.SpaceAt(wd)
	if s == nil {
		fmt.Fprintln(errOut, i18n.T(c.Language, "space.none", wd, strings.Join(c.SpaceNames(), ", ")))
		return c, nil, false
	}
	return c, s, true
}

func cmdSpace(out, errOut io.Writer) int {
	c, s, ok := currentSpace(errOut)
	if !ok {
		return 1
	}
	fmt.Fprintln(out, i18n.T(c.Language, "space.show", s.Name, s.Memory))
	return 0
}

func cmdCheck(out, errOut io.Writer) int {
	c, s, ok := currentSpace(errOut)
	if !ok {
		return 1
	}
	if r, err := format.EnsureNotes(s.Memory, c.Dir, s.Name, time.Now()); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	} else if r.Migrated() {
		fmt.Fprintln(out, i18n.T(c.Language, "check.format", r.From, r.To, strings.Join(r.Done, "; "), paths.Tilde(r.Copy)))
	}
	m, problems, err := memory.Load(s.Memory)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	for _, p := range problems {
		fmt.Fprintln(out, "  ✗", p.Error())
	}
	if b, err := brain.Open(c, s); err == nil {
		for _, x := range b.Mismatches() {
			fmt.Fprintln(out, "  !", i18n.T(c.Language, "check.repo", x.Note, x.Repo, strings.Join(tildes(s.WorkFolders), ", ")))
		}
	}
	fmt.Fprintln(out, i18n.T(c.Language, "check.summary", s.Name,
		i18n.Count(c.Language, "notes", len(m.Notes)), i18n.Count(c.Language, "problems", len(problems))))
	if len(problems) > 0 {
		return 1
	}
	// All notes are valid: bring the index up to date.
	if changed, err := m.WriteIndex(c.Language); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	} else if changed {
		fmt.Fprintln(out, i18n.T(c.Language, "check.rebuilt", memory.IndexFile))
	}
	return 0
}

func cmdDoctor(out, errOut io.Writer) int {
	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	wd, _ := os.Getwd()
	checks := doctor.Run(doctor.Env{ConfigDir: dir, WorkDir: wd, Lang: i18n.FromEnv()})
	marks := map[doctor.Status]string{doctor.OK: "✓", doctor.Info: "·", doctor.Warning: "!", doctor.Failed: "✗"}
	for _, c := range checks {
		fmt.Fprintf(out, "%s %s\n", marks[c.Status], c.What)
		if c.Fix != "" {
			fmt.Fprintf(out, "    → %s\n", c.Fix)
		}
	}
	if doctor.Worst(checks) == doctor.Failed {
		return 1
	}
	return 0
}

// cmdMCP serves the tools over stdin and stdout. The assistant starts it in the folder
// it works in, which selects the space; the space is resolved again on every call.
func cmdMCP(errOut io.Writer) int {
	open := func() (*brain.Brain, error) {
		dir, err := config.Dir()
		if err != nil {
			return nil, err
		}
		c, err := config.Load(dir)
		if err != nil {
			return nil, fmt.Errorf("%w; run `augmented-brain init`", err)
		}
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		s, _ := c.SpaceAt(wd)
		if s == nil {
			return nil, fmt.Errorf("%s belongs to no space (spaces: %v); open the assistant inside a space's work folder, or run `augmented-brain space add`", wd, c.SpaceNames())
		}
		return brain.Open(c, s)
	}
	lockFile := ""
	if dir, err := config.Dir(); err == nil {
		lockFile = filepath.Join(dir, backup.NotesLock)
	}
	if err := mcpserver.New(open, lockFile).Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return 0
}

// cmdHook answers a Claude Code hook. A hook must never break the assistant, so it
// always exits 0 and reports problems on stderr.
func cmdHook(args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(errOut, "usage: augmented-brain hook session-start|pre-tool-use|stop|post-compact|session-end")
		return 2
	}
	dir, _ := config.Dir()
	var err error
	switch args[0] {
	case "session-start":
		err = hook.SessionStart(in, out, dir)
	case "pre-tool-use":
		err = hook.PreToolUse(in, out, dir)
	case "session-end":
		err = hook.SessionEnd(in, dir)
	case "stop":
		err = hook.Stop(in, out, dir)
	case "post-compact":
		err = hook.PostCompact(in, dir)
	default:
		fmt.Fprintf(errOut, "unknown hook %q\n", args[0])
		return 2
	}
	if err != nil {
		fmt.Fprintln(errOut, "augmented-brain hook:", err)
	}
	return 0
}

// tildes writes each path under the home folder with ~, as a person reads it.
func tildes(ps []string) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = paths.Tilde(p)
	}
	return out
}
