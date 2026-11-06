// Package portcheck reads the repository's tests for patterns that pass on Linux and
// fail on Windows, so they are caught before CI runs them there. Only tests use it.
//
//   - R1: a check of a file's permission bits (.Perm()) in a test function that never
//     looks at runtime.GOOS. Windows has no such modes: the check fails there.
//   - R2: JSON written by hand, a literal with {" or ":" joined to an expression that
//     looks like a path. A Windows path has backslashes JSON must escape: marshal it,
//     or write it with filepath.ToSlash.
//
// A line that is fine as it is says so with a `// windows-ok` comment.
package portcheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding is one pattern found: where, and which rule.
type Finding struct {
	File string
	Line int
	Rule string // R1 or R2
}

// pathish is what an expression is named when it holds a path.
var pathish = regexp.MustCompile(`(?i)(dir|path|file|folder|cfg|work|home|tmp)`)

// Scan reads every *_test.go under root, skipping hidden folders and testdata.
func Scan(root string) ([]Finding, error) {
	var out []Finding
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		f, err := File(filepath.ToSlash(rel), src)
		out = append(out, f...)
		return err
	})
	return out, err
}

// File checks the source of one test file, named name in what it finds.
func File(name string, src []byte) ([]Finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(src), "\n")
	ok := func(pos token.Pos) bool {
		n := fset.Position(pos).Line
		return n >= 1 && n <= len(lines) && strings.Contains(lines[n-1], "windows-ok")
	}
	var out []Finding
	add := func(pos token.Pos, rule string) {
		if !ok(pos) {
			out = append(out, Finding{name, fset.Position(pos).Line, rule})
		}
	}
	for _, decl := range file.Decls {
		fn, isFunc := decl.(*ast.FuncDecl)
		if !isFunc || fn.Body == nil {
			continue
		}
		looksAtOS := false
		var perms []token.Pos
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if id, isID := x.X.(*ast.Ident); isID && id.Name == "runtime" && x.Sel.Name == "GOOS" {
					looksAtOS = true
				}
			case *ast.CallExpr:
				if sel, isSel := x.Fun.(*ast.SelectorExpr); isSel && sel.Sel.Name == "Perm" && len(x.Args) == 0 {
					perms = append(perms, x.Pos())
				}
			case *ast.BinaryExpr:
				if x.Op == token.ADD && handJSON(x, src, fset) {
					add(x.Pos(), "R2")
					return false // its parts are part of the same join
				}
			}
			return true
		})
		if !looksAtOS {
			for _, p := range perms {
				add(p, "R1")
			}
		}
	}
	return out, nil
}

// handJSON says whether a join of strings writes JSON by hand around a path: a literal
// with {" or ":" in it, and a part that is not a literal, is named like a path, and is
// not made safe (filepath.ToSlash, json.Marshal, strconv.Quote).
func handJSON(b *ast.BinaryExpr, src []byte, fset *token.FileSet) bool {
	var parts []ast.Expr
	var flat func(e ast.Expr)
	flat = func(e ast.Expr) {
		if x, isBin := e.(*ast.BinaryExpr); isBin && x.Op == token.ADD {
			flat(x.X)
			flat(x.Y)
			return
		}
		parts = append(parts, e)
	}
	flat(b)
	json := false
	for _, p := range parts {
		if lit, isLit := p.(*ast.BasicLit); isLit && lit.Kind == token.STRING {
			if v := lit.Value; strings.Contains(v, `{"`) || strings.Contains(v, `":"`) || strings.Contains(v, `{\"`) || strings.Contains(v, `\":\"`) {
				json = true
			}
		}
	}
	if !json {
		return false
	}
	for _, p := range parts {
		if _, isLit := p.(*ast.BasicLit); isLit {
			continue
		}
		text := string(src[fset.Position(p.Pos()).Offset:fset.Position(p.End()).Offset])
		if strings.Contains(text, "ToSlash") || strings.Contains(text, "Marshal") || strings.Contains(text, "Quote") {
			continue
		}
		if pathish.MatchString(text) {
			return true
		}
	}
	return false
}
