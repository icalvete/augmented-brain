package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/version"
)

// commands is the only list of commands: run answers from it alone. A command run
// could answer without being in it would be missing from the usage text and the
// reference without a test noticing, so: every cmd… function of the package is in the
// list, run picks no command by name itself, and every name in the list finds its entry.
func TestCommandsAreTheOnlyList(t *testing.T) {
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	cmdFunc := regexp.MustCompile(`^cmd[A-Z]`)
	var funcs []string          // every cmd… function: a command's work
	listed := map[string]bool{} // the names the list's entries use
	names := map[string]bool{}  // every name and alias of a command
	for _, c := range commands {
		names[c.name] = true
		for _, a := range c.aliases {
			names[a] = true
		}
		for _, w := range strings.Fields(c.name) {
			names[w] = true
		}
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range file.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && cmdFunc.MatchString(d.Name.Name) {
					funcs = append(funcs, d.Name.Name)
				}
				if d.Name.Name != "run" || d.Recv != nil {
					continue
				}
				// run chooses through find, never by a command's name of its own.
				ast.Inspect(d.Body, func(n ast.Node) bool {
					switch n := n.(type) {
					case *ast.SwitchStmt:
						t.Errorf("%s: run has a switch: answer commands from the list `commands`", fset.Position(n.Pos()))
					case *ast.BasicLit:
						if s, err := strconv.Unquote(n.Value); err == nil && n.Kind == token.STRING && names[s] {
							t.Errorf("%s: run names the command %q: answer it from the list `commands`", fset.Position(n.Pos()), s)
						}
					}
					return true
				})
			case *ast.GenDecl:
				for _, sp := range d.Specs {
					v, ok := sp.(*ast.ValueSpec)
					if !ok || !slices.ContainsFunc(v.Names, func(n *ast.Ident) bool { return n.Name == "commands" }) {
						continue
					}
					ast.Inspect(v, func(n ast.Node) bool {
						if id, ok := n.(*ast.Ident); ok {
							listed[id.Name] = true
						}
						return true
					})
				}
			}
		}
	}
	if len(funcs) == 0 {
		t.Fatal("no cmd… functions found: is the test in the right folder?")
	}
	for _, f := range funcs {
		if !listed[f] {
			t.Errorf("%s answers a command that is not in the list `commands` (commands.go)", f)
		}
	}

	for _, c := range commands {
		words := strings.Fields(c.name)
		parent := slices.IndexFunc(commands, func(p command) bool { return p.name == words[0] })
		switch {
		case len(words) == 1 && c.run == nil:
			t.Errorf("%s: a command of one word needs its run", c.name)
		case parent < 0 || commands[parent].run == nil:
			t.Errorf("%s: no command %q to answer it", c.name, words[0])
		}
		got, rest, ok := find(append(words, "x"))
		want := c.name
		if c.run == nil {
			want = words[0]
		}
		if !ok || got.name != want || len(rest) != len(words)-len(strings.Fields(want))+1 {
			t.Errorf("find(%s x) = %q %v %v, want %q", c.name, got.name, rest, ok, want)
		}
		for _, a := range c.aliases {
			if got, _, ok := find([]string{a}); !ok || got.name != c.name {
				t.Errorf("find(%s) = %q %v, want %q", a, got.name, ok, c.name)
			}
		}
	}
	for _, args := range [][]string{{"nope"}, {"space add"}, {"add"}, {"setup"}} {
		if c, _, ok := find(args); ok {
			t.Errorf("find(%q) = %q, want none", args, c.name)
		}
	}
}

// version and help answer to their aliases; an unknown command says so, with the usage.
func TestRunAliasesAndUnknown(t *testing.T) {
	t.Setenv("AB_CONFIG_DIR", t.TempDir())
	t.Setenv("AB_LANGUAGE", "en")
	for _, a := range []string{"version", "--version", "-v"} {
		var out, errOut bytes.Buffer
		if code := run([]string{a}, nil, &out, &errOut); code != 0 || out.String() != version.Version+"\n" || errOut.Len() > 0 {
			t.Errorf("%s: %d %q %q", a, code, out.String(), errOut.String())
		}
	}
	for _, a := range []string{"help", "--help", "-h"} {
		var out, errOut bytes.Buffer
		if code := run([]string{a}, nil, &out, &errOut); code != 0 || out.String() != i18n.T("en", "usage") || errOut.Len() > 0 {
			t.Errorf("%s: %d %q %q", a, code, out.String(), errOut.String())
		}
	}
	var out, errOut bytes.Buffer
	want := i18n.T("en", "usage.unknown", "nope") + "\n\n" + i18n.T("en", "usage")
	if code := run([]string{"nope", "add"}, nil, &out, &errOut); code != 2 || out.Len() > 0 || errOut.String() != want {
		t.Errorf("unknown: %d %q %q", code, out.String(), errOut.String())
	}
}
