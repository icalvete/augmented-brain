package guard

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strconv"
	"testing"
)

// A wrapper is stripped by program before simple sees the command, so a name that is
// both a wrapper and a case of simple's switch has a rule that never runs. That is how
// doas went through as if it were the command after it.
func TestWrappersHaveNoRule(t *testing.T) {
	f, err := goparser.ParseFile(token.NewFileSet(), "guard.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]bool{}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "simple" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if cc, ok := n.(*ast.CaseClause); ok {
				for _, e := range cc.List {
					if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						if s, err := strconv.Unquote(lit.Value); err == nil {
							cases[s] = true
						}
					}
				}
			}
			return true
		})
	}
	if !cases["sudo"] || !cases["doas"] {
		t.Fatal("the cases of simple were not found: has it been renamed or moved?")
	}
	for name := range wrappers {
		if cases[name] {
			t.Errorf("%q is a wrapper and has a rule of its own in simple: program strips it, so the rule never runs", name)
		}
	}
}

func TestDoasIsRoot(t *testing.T) {
	for cmd, want := range map[string]Verdict{
		"doas id":                 Deny,
		"doas -u root id":         Deny,
		"doas -uroot id":          Deny,
		"doas -n apk add jq":      Deny,
		"doas -s":                 Deny,
		"doas sh -c 'id'":         Deny,
		"/usr/bin/doas id":        Deny,
		"env doas id":             Deny,
		"doas -u deploy id":       Allow,
		"doas -u deploy doas id":  Deny, // chained back to root
		"doas -u deploy git push": Deny, // what runs as another user is still checked
		"doas -L":                 Allow,
	} {
		if got := (Policy{}).Command(cmd).Verdict; got != want {
			t.Errorf("%s: got %v, want %v", cmd, got, want)
		}
	}
}
