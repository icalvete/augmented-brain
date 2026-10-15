// Command augmented-brain sets up and runs a second brain for AI coding assistants.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/doctor"
	"github.com/icalvete/augmented-brain/internal/hook"
	"github.com/icalvete/augmented-brain/internal/mcpserver"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/version"
)

const usage = `augmented brain: a second brain for AI coding assistants.

Usage:
  augmented-brain init         set up augmented brain and your first space
  augmented-brain space        show which space this folder belongs to
  augmented-brain space add    add another space (work, personal…)
  augmented-brain check        check the notes of this folder's space and rebuild its index
  augmented-brain doctor       say what is missing, and how to fix it
  augmented-brain backup       save this folder's space to its private repository (--all: every space)
  augmented-brain backup setup URL   connect this folder's space to a private git repository
  augmented-brain restore URL  bring a space back from its backup, on a new machine
  augmented-brain mcp          run the MCP server (your assistant starts it, not you)
  augmented-brain hook EVENT   answer an assistant hook: session-start, pre-tool-use
  augmented-brain version      show the version
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(errOut, usage)
		return 2
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintln(out, version.Version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(out, usage)
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
	case "mcp":
		return cmdMCP(errOut)
	case "hook":
		return cmdHook(args[1:], in, out, errOut)
	case "backup":
		return cmdBackup(args[1:], out, errOut)
	case "restore":
		return cmdRestore(args[1:], out, errOut)
	default:
		fmt.Fprintf(errOut, "unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

// currentSpace loads the configuration and finds the space of the working folder.
func currentSpace(errOut io.Writer) (*config.Config, *config.Space, bool) {
	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return nil, nil, false
	}
	c, err := config.Load(dir)
	if errors.Is(err, config.ErrNotFound) {
		fmt.Fprintf(errOut, "%v.\nRun `augmented-brain init` to create one.\n", err)
		return nil, nil, false
	}
	if err != nil {
		fmt.Fprintln(errOut, err)
		return nil, nil, false
	}
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return nil, nil, false
	}
	s := c.SpaceFor(wd)
	if s == nil {
		fmt.Fprintf(errOut, "%s is not inside any space's work folder.\nSpaces: %v\n", wd, c.SpaceNames())
		return c, nil, false
	}
	return c, s, true
}

func cmdSpace(out, errOut io.Writer) int {
	_, s, ok := currentSpace(errOut)
	if !ok {
		return 1
	}
	fmt.Fprintf(out, "space:  %s\nmemory: %s\n", s.Name, s.Memory)
	return 0
}

func cmdCheck(out, errOut io.Writer) int {
	c, s, ok := currentSpace(errOut)
	if !ok {
		return 1
	}
	m, problems, err := memory.Load(s.Memory)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	for _, p := range problems {
		fmt.Fprintln(out, "  ✗", p.Error())
	}
	fmt.Fprintf(out, "space %s: %s, %s\n", s.Name,
		plural(len(m.Notes), "note"), plural(len(problems), "problem"))
	if len(problems) > 0 {
		return 1
	}
	// All notes are valid: bring the index up to date.
	if changed, err := m.WriteIndex(c.Language); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	} else if changed {
		fmt.Fprintf(out, "%s rebuilt\n", memory.IndexFile)
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
	checks := doctor.Run(doctor.Env{ConfigDir: dir, WorkDir: wd})
	marks := map[doctor.Status]string{doctor.OK: "✓", doctor.Warning: "!", doctor.Failed: "✗"}
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

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
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
		s := c.SpaceFor(wd)
		if s == nil {
			return nil, fmt.Errorf("%s belongs to no space (spaces: %v); open the assistant inside a space's work folder, or run `augmented-brain space add`", wd, c.SpaceNames())
		}
		return brain.Open(c, s)
	}
	lockFile := ""
	if dir, err := config.Dir(); err == nil {
		lockFile = filepath.Join(dir, ".lock")
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
		fmt.Fprintln(errOut, "usage: augmented-brain hook session-start|pre-tool-use")
		return 2
	}
	dir, _ := config.Dir()
	var err error
	switch args[0] {
	case "session-start":
		err = hook.SessionStart(in, out, dir)
	case "pre-tool-use":
		err = hook.PreToolUse(in, out, dir)
	default:
		fmt.Fprintf(errOut, "unknown hook %q\n", args[0])
		return 2
	}
	if err != nil {
		fmt.Fprintln(errOut, "augmented-brain hook:", err)
	}
	return 0
}
