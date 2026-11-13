package guard

import (
	"bytes"
	"os"
	"slices"
	"strings"
)

// lineFiles are the files a command line writes or changes before a later command in
// it may run one of them as a script (echo '…' > x.sh && bash x.sh): the script is not
// on disk yet, or not as it will be, when the guard looks.
type lineFiles struct {
	// written holds what a file will hold when that is known from the line alone: what
	// echo, printf or a heredoc wrote into it (after what it already held).
	written map[string]string
	// touched are the files some other command writes or changes in ways the guard
	// cannot follow (cp, curl -o, a program's output): running one of them asks.
	touched map[string]bool
	// later are files a program's arguments name, kept as written until the guard needs
	// to know whether a file was touched (isTouched): most lines never run a script,
	// and resolving thousands of arguments (prettier --write a.ts b.ts …) is slow.
	later []laterTouch
	// resolved keeps what resolve said of each path in this line: every command of a
	// long line names files, and following links costs a system call per folder (slow
	// on Windows), so no path is resolved twice.
	resolved map[string]resolved
	// inSpace says the line, read with only the rules that hold outside the spaces,
	// reached into a space or went where the guard cannot follow: it is read again
	// with every rule. remotes are the remote addresses of the lists' repositories,
	// read once (remotesRead).
	inSpace     bool
	remotes     []string
	remotesRead bool
}

type resolved struct {
	path string
	err  error
}

// laterTouch is a file named after a cd the guard followed (or not): resolved there.
type laterTouch struct {
	cwd  string
	lost bool
	path string
}

// isTouched says whether the file k (a fileKey) was touched, resolving first the
// arguments kept for later.
func (p Policy) isTouched(k string) bool {
	for _, t := range p.files.later {
		q := p
		q.Cwd, q.lost = t.cwd, t.lost
		p.files.touched[q.fileKey(t.path)] = true
	}
	p.files.later = nil
	return p.files.touched[k]
}

// readOnly are programs that name a file without changing what it holds (the shells
// too: they read a script, and what it runs is checked on its own).
var readOnly = append([]string{"chmod", "chown", "chgrp", "ls", "cat", "head", "tail", "less", "more", "file",
	"stat", "test", "[", "wc", "grep", "shellcheck", "source", ".",
	"echo", "printf", "diff", "cmp", "md5sum", "sha256sum"}, shells...)

// fileKey is how a path is kept in lineFiles: resolved when the guard can, as written
// otherwise (after a cd it could not follow).
func (p Policy) fileKey(path string) string {
	if d, err := p.resolve(path); err == nil {
		return d
	}
	return "?" + path
}

// recordWrites notes the files command c writes: its output redirections (known text
// for echo, printf and cat or tee of a heredoc), the files tee writes, and any file
// named by a program that may change it.
func (p Policy) recordWrites(c command) {
	if p.files == nil {
		return
	}
	_, name, args := program(c.words)
	text, _, _ := output(c, "", false)
	known := true
	if name == "cat" && len(args) > 0 || text == "" && !(name == "echo" || name == "printf") {
		known = false
	}
	for _, f := range c.outputs {
		p.write(f, text, known)
	}
	if name == "tee" {
		for _, a := range args {
			if !strings.HasPrefix(a, "-") {
				p.write(a, c.stdin, c.stdin != "")
			}
		}
		return
	}
	if name == "" || slices.Contains(readOnly, name) {
		return
	}
	for _, a := range args {
		if a != "" && !strings.HasPrefix(a, "-") {
			p.files.later = append(p.files.later, laterTouch{p.Cwd, p.lost, strings.TrimPrefix(a, "of=")})
		}
		if _, v, ok := strings.Cut(a, "="); ok && strings.HasPrefix(a, "-") && v != "" {
			p.files.later = append(p.files.later, laterTouch{p.Cwd, p.lost, v}) // --output=x.sh
		}
	}
}

// write notes that path gets text (known) or something the guard cannot tell. What the
// file held before counts too, so an append is checked whole; an overwrite may then be
// checked against more than it holds, which only asks more.
func (p Policy) write(path, text string, known bool) {
	k := p.fileKey(path)
	if !known || p.isTouched(k) {
		p.files.touched[k] = true
		delete(p.files.written, k)
		return
	}
	before, ok := p.files.written[k]
	if !ok && !strings.HasPrefix(k, "?") {
		if info, err := os.Stat(k); err == nil {
			data, err := os.ReadFile(k)
			if err != nil || !info.Mode().IsRegular() || info.Size() > maxScript || bytes.IndexByte(data, 0) >= 0 {
				p.files.touched[k] = true
				return
			}
			before = string(data)
		}
	}
	p.files.written[k] = before + text
}
