package guard

import "strings"

// commands splits a shell command line into simple commands, each a list of words with
// the quotes removed. It understands quotes, backslashes and the separators ; & | ( )
// and newlines. It is not a full shell parser: it only has to find the program and
// the arguments of every command, which is all the guard needs.
func commands(line string) [][]string {
	var cmds [][]string
	var words []string
	var pending []heredoc
	var cur strings.Builder
	inWord := false
	flushWord := func() {
		if inWord {
			words = append(words, cur.String())
			cur.Reset()
			inWord = false
		}
	}
	flushCmd := func() {
		flushWord()
		if len(words) > 0 {
			cmds = append(cmds, words)
			words = nil
		}
	}
	rs := []rune(line)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == '\\' && i+1 < len(rs):
			i++
			if rs[i] != '\n' {
				cur.WriteRune(rs[i])
				inWord = true
			}
		case r == '\'':
			inWord = true
			for i++; i < len(rs) && rs[i] != '\''; i++ {
				cur.WriteRune(rs[i])
			}
		case r == '"':
			inWord = true
			for i++; i < len(rs) && rs[i] != '"'; i++ {
				if rs[i] == '\\' && i+1 < len(rs) && strings.ContainsRune(`"\$`+"`", rs[i+1]) {
					i++
				}
				cur.WriteRune(rs[i])
			}
		case r == '`' || r == '$' && i+1 < len(rs) && rs[i+1] == '(':
			// A command substitution is a command of its own: cut here.
			flushCmd()
			if r == '$' {
				i++
			}
		case r == '<' && i+2 < len(rs) && rs[i+1] == '<' && rs[i+2] != '<':
			// A heredoc: its body, from the next line to the delimiter line, is data and
			// not commands. The rest of this line is still read as commands.
			var h heredoc
			h, i = readDelimiter(rs, i+2)
			pending = append(pending, h)
		case r == '\n':
			flushCmd()
			for _, h := range pending {
				i = skipBody(rs, i, h)
			}
			pending = nil
		case r == ';' || r == '&' || r == '|' || r == '(' || r == ')':
			flushCmd()
		case r == ' ' || r == '\t':
			flushWord()
		case r == '#' && !inWord:
			// a comment runs to the end of the line
			for i < len(rs) && rs[i] != '\n' {
				i++
			}
			flushCmd()
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	flushCmd()
	return cmds
}

type heredoc struct {
	delim     string
	stripTabs bool
}

// readDelimiter reads a heredoc delimiter that starts at i (just after "<<") and
// returns it with the index of its last rune.
func readDelimiter(rs []rune, i int) (heredoc, int) {
	var h heredoc
	if i < len(rs) && rs[i] == '-' {
		h.stripTabs = true
		i++
	}
	for i < len(rs) && (rs[i] == ' ' || rs[i] == '\t') {
		i++
	}
	var d strings.Builder
	for ; i < len(rs) && !strings.ContainsRune(" \t\n;&|<>()", rs[i]); i++ {
		if rs[i] != '\'' && rs[i] != '"' && rs[i] != '\\' {
			d.WriteRune(rs[i])
		}
	}
	h.delim = d.String()
	return h, i - 1
}

// skipBody skips a heredoc body. i is the newline that ends the line holding "<<"; it
// returns the index of the newline that ends the delimiter line (or the last rune).
func skipBody(rs []rune, i int, h heredoc) int {
	for i < len(rs)-1 {
		start := i + 1
		end := start
		for end < len(rs) && rs[end] != '\n' {
			end++
		}
		line := string(rs[start:end])
		if h.stripTabs {
			line = strings.TrimLeft(line, "\t")
		}
		i = end
		if line == h.delim {
			break
		}
	}
	if i >= len(rs) {
		i = len(rs) - 1
	}
	return i
}

// wrappers run the command that follows them.
var wrappers = map[string]bool{
	"env": true, "nohup": true, "time": true, "exec": true, "command": true, "nice": true,
	"timeout": true, "xargs": true, "watch": true, "stdbuf": true,
}

// program strips variable assignments (NAME=value) and wrappers from the front of a
// command, and returns them apart from the program and its arguments.
func program(words []string) (env map[string]string, name string, args []string) {
	env = map[string]string{}
	i := 0
	for ; i < len(words); i++ {
		w := words[i]
		if eq := strings.IndexByte(w, '='); eq > 0 && isName(w[:eq]) {
			env[w[:eq]] = w[eq+1:]
			continue
		}
		if wrappers[base(w)] {
			// skip the wrapper's own options and, for timeout, its duration
			for i+1 < len(words) && strings.HasPrefix(words[i+1], "-") {
				i++
			}
			if base(w) == "timeout" && i+1 < len(words) {
				i++
			}
			continue
		}
		break
	}
	if i >= len(words) {
		return env, "", nil
	}
	return env, base(words[i]), words[i+1:]
}

func isName(s string) bool {
	for i, r := range s {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return s != ""
}

// base returns the program name without its path: /usr/bin/sudo -> sudo.
func base(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}
