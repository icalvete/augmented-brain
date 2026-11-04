// Command augmented-brain sets up and runs augmented brain: memory, method and
// a guard for AI coding assistants.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/doctor"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/format"
	"github.com/icalvete/augmented-brain/internal/hook"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/mcpserver"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/session"
	"github.com/icalvete/augmented-brain/internal/state"
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
	c, rest, ok := find(args)
	if !ok {
		l := userLanguage()
		fmt.Fprintf(errOut, "%s\n\n%s", i18n.T(l, "usage.unknown", args[0]), i18n.T(l, "usage"))
		return 2
	}
	return c.run(rest, in, out, errOut)
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

// parseArgs reads fs's flags wherever they are among args (Go's flag package stops at
// the first argument that is not a flag, and would leave the rest unread without a
// word) and returns the other arguments, up to max of them. More than max is an error,
// said in errOut: an argument is never ignored in silence.
func parseArgs(fs *flag.FlagSet, args []string, max int, errOut io.Writer) ([]string, bool) {
	var rest []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, false
		}
		if fs.NArg() == 0 {
			break
		}
		rest, args = append(rest, fs.Arg(0)), fs.Args()[1:]
	}
	if len(rest) > max {
		fmt.Fprintln(errOut, i18n.T(userLanguage(), "args.extra", strings.Join(rest[max:], " ")))
		return nil, false
	}
	return rest, true
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
	// It may move notes to a new format and rebuild indexes: not while a session writes.
	unlock, err := filelock.Lock(filepath.Join(c.Dir, backup.NotesLock))
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer unlock()
	if err := brain.MoveSecrets(c.Dir); err != nil {
		fmt.Fprintln(errOut, err)
	}
	if paths.Exposed(s.Memory) {
		if err := paths.Private(s.Memory); err != nil {
			fmt.Fprintln(errOut, err)
		} else {
			fmt.Fprintln(out, i18n.T(c.Language, "check.private", paths.Tilde(s.Memory)))
		}
	}
	if r, err := format.EnsureNotes(s.Memory, c.Dir, s.Name, time.Now()); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	} else if r.Migrated() {
		fmt.Fprintln(out, i18n.T(c.Language, "check.format", r.From, r.To, strings.Join(r.Done, "; "), paths.Tilde(r.Copy)))
	}
	shared, fine := checkShared(c, out, errOut)
	if shared == nil {
		return 1
	}
	m, problems, err := memory.Load(s.Memory, shared)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	for _, p := range problems {
		fmt.Fprintln(out, "  ✗", p.Error())
	}
	// Every file the backup takes, docs/ and the archive too, not only the notes.
	if found, err := backup.Secrets(s.Memory); err == nil {
		for _, f := range found {
			fmt.Fprintln(out, "  !", i18n.T(c.Language, "check.secret", f.Path, i18n.T(c.Language, "secret."+f.Kind)))
		}
	}
	if b, err := brain.Open(c, s); err == nil {
		for _, x := range b.Mismatches() {
			fmt.Fprintln(out, "  !", i18n.T(c.Language, "check.repo", x.Note, x.Repo, strings.Join(tildes(s.WorkFolders), ", ")))
		}
	}
	fmt.Fprintln(out, i18n.T(c.Language, "check.summary", s.Name,
		i18n.Count(c.Language, "notes", len(m.Notes)), i18n.Count(c.Language, "problems", len(problems))))
	if len(problems) > 0 || !fine {
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

// checkShared checks the shared notes, when there are any, as check does a space's, and
// returns them (nil when they cannot be read); fine is false when one is not valid.
func checkShared(c *config.Config, out, errOut io.Writer) (*memory.Memory, bool) {
	dir := c.SharedDir()
	if _, err := os.Stat(dir); err != nil {
		return &memory.Memory{Dir: dir, Notes: map[string]*memory.Note{}}, true
	}
	if r, err := format.EnsureNotes(dir, c.Dir, config.SharedFolder, time.Now()); err != nil {
		fmt.Fprintln(errOut, err)
		return nil, false
	} else if r.Migrated() {
		fmt.Fprintln(out, i18n.T(c.Language, "check.format", r.From, r.To, strings.Join(r.Done, "; "), paths.Tilde(r.Copy)))
	}
	m, problems, err := memory.LoadShared(dir)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return nil, false
	}
	for _, p := range problems {
		fmt.Fprintln(out, "  ✗", p.Error())
	}
	fmt.Fprintln(out, i18n.T(c.Language, "check.shared", paths.Tilde(dir),
		i18n.Count(c.Language, "notes", len(m.Notes)), i18n.Count(c.Language, "problems", len(problems))))
	if len(problems) > 0 {
		return m, false
	}
	if changed, err := m.WriteIndex(c.Language); err != nil {
		fmt.Fprintln(errOut, err)
		return nil, false
	} else if changed {
		fmt.Fprintln(out, i18n.T(c.Language, "check.rebuilt", paths.Tilde(filepath.Join(dir, memory.IndexFile))))
	}
	return m, true
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
		var newer *format.NewerError
		if errors.As(err, &newer) {
			return nil, err // it says to update; init would not help
		}
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
		// Noted, so a newer program can tell which sessions still run this one.
		defer state.RegisterServer(dir, version.Version)()
	}
	if err := mcpserver.New(open, lockFile).Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return 0
}

// hookEvent is a hook the plugin calls (`augmented-brain hook <name>`); part is the
// part of the session text session-start asks for, 0 for the others.
type hookEvent struct {
	name string
	run  func(in io.Reader, out io.Writer, dir string, part int) error
}

// hookEvents are every hook the command answers, in the order its usage names them.
var hookEvents = []hookEvent{
	{"session-start", func(in io.Reader, out io.Writer, dir string, part int) error {
		return hook.SessionStartPart(in, out, dir, part)
	}},
	{"pre-tool-use", func(in io.Reader, out io.Writer, dir string, _ int) error { return hook.PreToolUse(in, out, dir) }},
	{"stop", func(in io.Reader, out io.Writer, dir string, _ int) error { return hook.Stop(in, out, dir) }},
	{"post-compact", func(in io.Reader, _ io.Writer, dir string, _ int) error { return hook.PostCompact(in, dir) }},
	{"session-end", func(in io.Reader, _ io.Writer, dir string, _ int) error { return hook.SessionEnd(in, dir) }},
}

// cmdHook answers a Claude Code hook. A hook must never break the assistant, so it
// always exits 0 and reports problems on stderr.
func cmdHook(args []string, in io.Reader, out, errOut io.Writer) int {
	part := 0 // session-start N: part N of the session text
	if len(args) == 2 && args[0] == "session-start" {
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 {
			fmt.Fprintf(errOut, "usage: augmented-brain hook session-start [1-%d]\n", session.Parts)
			return 2
		}
		if n > session.Parts {
			// A plugin newer than this program asks for a part it does not have: an
			// empty one, so the session still starts with the others.
			fmt.Fprintln(out, `{"hookSpecificOutput":{"hookEventName":"SessionStart"}}`)
			return 0
		}
		part, args = n, args[:1]
	}
	if len(args) != 1 {
		names := make([]string, len(hookEvents))
		for i, e := range hookEvents {
			names[i] = e.name
		}
		fmt.Fprintln(errOut, "usage: augmented-brain hook "+strings.Join(names, "|"))
		return 2
	}
	dir, _ := config.Dir()
	i := slices.IndexFunc(hookEvents, func(e hookEvent) bool { return e.name == args[0] })
	if i < 0 {
		fmt.Fprintf(errOut, "unknown hook %q\n", args[0])
		return 2
	}
	if err := hookEvents[i].run(in, out, dir, part); err != nil {
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
