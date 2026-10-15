// Command augmented-brain sets up and runs a second brain for AI coding assistants.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/doctor"
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
