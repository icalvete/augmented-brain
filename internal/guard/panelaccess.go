package guard

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// The panel's address carries a key derived from panel.json's token, and the pages that
// change the guard's lists work with nothing else but that address and the page's own
// form. An assistant that read panel.json, or called the panel from a command, could
// change the lists the guard answers to. So both are denied: reading the file with a
// tool or a command, and a command that names a panel page (http://127.0.0.1:<port>/s/<key>/).
// It is a speed bump for a well-meaning assistant, not a wall: a script written in line
// can still do anything the user's account can.

var panelURL = regexp.MustCompile(`(?i)(127\.0\.0\.1|localhost|\[::1\]|0\.0\.0\.0)(:\d+)?/s/[0-9a-f]{32}`)

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

// panelAccess reports whether a command line reaches for the panel's token file or
// calls one of the panel's pages.
func (p Policy) panelAccess(line string) bool {
	if len(p.Hidden) == 0 {
		return false
	}
	if panelURL.MatchString(line) || panelURL.MatchString(unescape(line)) {
		return true
	}
	if !strings.Contains(line, "panel.") && !strings.Contains(line, "panel*") && !strings.Contains(line, "panel?") && !strings.Contains(line, "*") {
		return false
	}
	words := strings.FieldsFunc(line, func(r rune) bool {
		return strings.ContainsRune(" \t\r\n\"'`;|&<>()={}", r)
	})
	for _, w := range words {
		base := filepath.Base(w)
		hit := base == panelFile
		if !hit && strings.ContainsAny(base, "*?[") {
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
		// A name with no folder, after a cd the guard did not follow, or one built from
		// a variable: if the line also names the configuration folder, deny.
		if strings.Contains(w, "$") || !strings.ContainsAny(w, `/\`) {
			for _, d := range p.ConfigDirs {
				if d != "" && strings.Contains(line, d) {
					return true
				}
			}
		}
	}
	return false
}

func unescape(s string) string {
	if u, err := url.PathUnescape(s); err == nil {
		return u
	}
	return s
}
