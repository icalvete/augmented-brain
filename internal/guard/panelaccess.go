package guard

import (
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// The panel's pages live under /s/<session>/, and a browser gets there through a ticket
// of one use, /t/<ticket>, that `augmented-brain panel` asks the running panel for. The
// pages that change the guard's lists work with nothing else but the session and the
// page's own form. An assistant that read panel.json, or called the panel from a
// command, could change the lists the guard answers to. So both are denied: reading the
// file with a tool or a command, and a command that names a panel address, whatever the
// host is written as (127.1, 2130706433, a variable: curl sends Host: 127.0.0.1 for all
// of them), or searches for one (browser history, the process list).
//
// It is a speed bump for a well-meaning assistant, not a wall: a script written in line
// can still do anything the user's account can. The wall is elsewhere: the command
// never prints the session, a ticket works once, and a change to the lists made in the
// panel reaches the user through a hook the assistant does not control.

// panelPath is a panel address: a session or a ticket, 32 hex digits, after any host.
var panelPath = regexp.MustCompile(`(?i)/[st]/[0-9a-f]{32}`)

// hex32 is a run of exactly 32 hex digits: a session or a ticket kept in a variable.
var hex32 = regexp.MustCompile(`(?i)(^|[^0-9a-z])[0-9a-f]{32}([^0-9a-z]|$)`)

// panelSearch is a pattern that looks for an address (/s/% in SQL, /s/[0-9a-f]* or
// /s/.* in grep), and loopback the address the panel listens on, which must be near.
var (
	panelSearch = regexp.MustCompile(`(?i)/[st]/(%|\[|\.\*|\*)`)
	loopback    = regexp.MustCompile(`(?i)127\.|localhost|\[::1\]`)
)

// panelFile is the name of the file that holds the panel's token.
const panelFile = "panel.json"

// panelDeny is the answer to anything that reaches for the panel's token or pages.
func (p Policy) panelDeny() Decision {
	return Decision{Deny, i18n.T(p.Lang, "guard.panel")}
}

// ReadFile checks a tool that reads the file or folder at path (Read, Grep): the
// panel's token file is never read, nor a folder that holds it, which a search would
// go through.
func (p Policy) ReadFile(path string) Decision {
	if path == "" || len(p.Hidden) == 0 {
		return allow
	}
	d, err := p.resolve(path)
	if err != nil {
		return allow
	}
	for _, h := range p.Hidden {
		if paths.Within(h, d) {
			return p.panelDeny()
		}
	}
	return allow
}

// panelAddress reports whether a command line names a panel address or searches for
// one. Quotes and backslashes are taken out first (a key split by "" is still the key),
// %xx is decoded and /./ folded, as curl and a browser would.
func panelAddress(line string) bool {
	flat := strings.NewReplacer(`"`, "", "'", "", `\`, "").Replace(line)
	flat = unescape(flat)
	for strings.Contains(flat, "/./") {
		flat = strings.ReplaceAll(flat, "/./", "/")
	}
	if panelPath.MatchString(flat) {
		return true
	}
	// /s/$K with K=<32 hex>, or a format string filled in from elsewhere in the line.
	if (strings.Contains(flat, "/s/") || strings.Contains(flat, "/t/")) && hex32.MatchString(flat) {
		return true
	}
	return panelSearch.MatchString(flat) && loopback.MatchString(flat)
}

// panelAccess reports whether a command line reaches for the panel's token file or
// calls one of the panel's pages.
func (p Policy) panelAccess(line string) bool {
	if len(p.Hidden) == 0 {
		return false
	}
	if panelAddress(line) {
		return true
	}
	flat := strings.NewReplacer(`"`, "", "'", "", `\`, "").Replace(line) // pan""el.json is panel.json
	if !strings.Contains(flat, "panel") && !strings.ContainsAny(flat, "*?[") {
		return false
	}
	inConfig := p.runsInConfig(line)
	for _, sw := range shellWords(line) {
		w := sw.text
		base := filepath.Base(w)
		// pan\el.json is panel.json in the shell; a backslash is kept in the word, since
		// on Windows it separates folders.
		hit := base == panelFile || strings.ReplaceAll(base, `\`, "") == panelFile
		if !hit && sw.glob && strings.ContainsAny(base, "*?[") {
			ok, err := filepath.Match(base, panelFile)
			hit = err == nil && ok
		}
		if !hit {
			continue
		}
		if r, err := p.resolve(w); err == nil {
			for _, h := range p.Hidden {
				if paths.Same(r, h) {
					return true
				}
			}
		}
		if base != panelFile && strings.ContainsAny(w, `/\`) { // a pattern: its folder is the file's
			if r, err := p.resolve(filepath.Dir(w)); err == nil {
				for _, h := range p.Hidden {
					if paths.Same(r, filepath.Dir(h)) {
						return true
					}
				}
			}
		}
		if sw.glob && p.patternHidden(w) { // ~/.config/augmented-b*/pan*: the whole path, part by part
			return true
		}
		// A name built from a variable ($AB_CONFIG_DIR/panel.json): if the line also
		// names the configuration folder, deny.
		if strings.Contains(w, "$") {
			for _, d := range p.ConfigDirs {
				if d != "" && strings.Contains(line, d) {
					return true
				}
			}
		}
		// A name with no folder counts only where the command runs: the configuration
		// folder, or a cd to it in the line. A line that merely names the folder
		// (AB_CONFIG_DIR=/tmp/x augmented-brain doctor && ls *) is not reading it.
		if !strings.ContainsAny(w, `/\`) && inConfig {
			return true
		}
	}
	return false
}

// runsInConfig reports whether a command line runs in the folder that holds the panel's
// token: the session's folder is that folder, or the line changes to it (cd, pushd).
func (p Policy) runsInConfig(line string) bool {
	if p.Cwd != "" {
		if r, err := paths.Resolve(p.Cwd); err == nil {
			for _, h := range p.Hidden {
				if paths.Same(r, filepath.Dir(h)) {
					return true
				}
			}
		}
	}
	if !strings.Contains(line, "cd") {
		return false
	}
	for _, d := range p.ConfigDirs {
		if d != "" && strings.Contains(line, d) {
			return true
		}
	}
	return false
}

// shellWord is a word of a command line, and whether a pattern character in it is
// outside quotes, where the shell would expand it.
type shellWord struct {
	text string
	glob bool
}

// shellWords splits a command line into words the way the shell reads quotes: a quoted
// "*" is text, not a pattern. The separators are blanks and the shell's operators.
func shellWords(line string) []shellWord {
	var out []shellWord
	var cur strings.Builder
	glob, quote := false, rune(0)
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, shellWord{cur.String(), glob})
		}
		cur.Reset()
		glob = false
	}
	for _, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
		case strings.ContainsRune(" \t\r\n`;|&<>()={}", r):
			flush()
		default:
			if strings.ContainsRune("*?[", r) {
				glob = true
			}
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

func unescape(s string) string {
	if u, err := url.PathUnescape(s); err == nil {
		return u
	}
	return s
}

// patternHidden says whether a pattern with a folder (~/.config/*/panel.json) would
// match the panel's token file: it is made absolute as the shell would, and matched
// against the file's path part by part.
func (p Policy) patternHidden(w string) bool {
	if !strings.ContainsAny(w, `/\`) {
		return false
	}
	pattern := w
	if pattern == "~" || strings.HasPrefix(pattern, "~/") {
		home, err := paths.Home()
		if err != nil {
			return false
		}
		pattern = filepath.Join(home, pattern[1:])
	} else if !filepath.IsAbs(pattern) {
		if p.Cwd == "" || p.lost {
			return false
		}
		pattern = filepath.Join(p.Cwd, pattern)
	}
	for _, h := range p.Hidden {
		if ok, err := filepath.Match(filepath.ToSlash(pattern), filepath.ToSlash(h)); err == nil && ok {
			return true
		}
	}
	return false
}

// copiers are the programs that, given a folder, read every file in it: a copy, an
// archive, a search.
var copiers = []string{"cp", "rsync", "tar", "zip", "7z", "7za", "scp", "rclone", "rg", "ag", "cpio", "pax"}

// copiesHidden says whether a command copies, archives or searches the folder that
// holds the panel's token file, or a folder that holds that one inside the home
// folder (cp -r ~/.config /tmp/c): the copy can then be read where the guard does not
// look. grep counts when recursive; ls and cat of other files in the folder do not.
func (p Policy) copiesHidden(words []string) bool {
	if len(p.Hidden) == 0 {
		return false
	}
	_, name, args := program(words)
	recursive := slices.Contains(copiers, name)
	if name == "grep" || name == "egrep" || name == "fgrep" {
		recursive = slices.ContainsFunc(args, func(a string) bool {
			return a == "--recursive" || a == "--dereference-recursive" ||
				len(a) > 1 && a[0] == '-' && a[1] != '-' && strings.ContainsAny(a, "rR")
		})
	}
	if !recursive {
		return false
	}
	home, _ := paths.Home()
	if h, err := paths.Resolve(home); err == nil {
		home = h
	}
	for _, a := range args {
		a = writtenArg(a)
		if a == "" || strings.HasPrefix(a, "-") {
			continue
		}
		d, err := p.resolve(a)
		if err != nil {
			continue
		}
		for _, h := range p.Hidden {
			dir := filepath.Dir(h)
			if paths.Same(d, dir) || paths.Within(dir, d) && home != "" && !paths.Same(d, home) && paths.Within(d, home) {
				return true
			}
		}
	}
	return false
}
