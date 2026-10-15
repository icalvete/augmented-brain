package guard

import "strings"

// command is one simple command: its words, with quotes removed, and the text it
// reads from a heredoc or a here-string, if any.
type command struct {
	words []string
	stdin string
}

// parse splits a shell command line into simple commands. It understands quotes,
// backslashes, comments, the separators ; & | ( ) { } and newlines, heredocs and
// here-strings (kept as the command's stdin), and command and process substitutions
// ($(…), `…`, <(…), >(…)), which become commands of their own, also inside double
// quotes. It is not a full shell parser: it only has to find every program that would
// run and its arguments, which is all the guard needs.
func parse(line string) []command {
	var out []command
	var words []string
	var subs []string
	var pending []heredoc
	var cur strings.Builder
	inWord, hereString := false, false
	stdin := ""

	flushWord := func() {
		if !inWord {
			return
		}
		if hereString {
			stdin, hereString = cur.String(), false
		} else {
			words = append(words, cur.String())
		}
		cur.Reset()
		inWord = false
	}
	flushCmd := func() {
		flushWord()
		if len(words) > 0 || stdin != "" {
			out = append(out, command{words: words, stdin: stdin})
		}
		words, stdin = nil, ""
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
				switch {
				case rs[i] == '\\' && i+1 < len(rs) && strings.ContainsRune(`"\$`+"`", rs[i+1]):
					i++
					cur.WriteRune(rs[i])
				case rs[i] == '$' && i+1 < len(rs) && rs[i+1] == '(':
					var s string
					s, i = balanced(rs, i+2)
					subs = append(subs, s)
				case rs[i] == '`':
					var s string
					s, i = backquoted(rs, i+1)
					subs = append(subs, s)
				default:
					cur.WriteRune(rs[i])
				}
			}
		case (r == '$' || r == '<' || r == '>') && i+1 < len(rs) && rs[i+1] == '(':
			var s string
			s, i = balanced(rs, i+2)
			subs = append(subs, s)
			inWord = true
		case r == '`':
			var s string
			s, i = backquoted(rs, i+1)
			subs = append(subs, s)
			inWord = true
		case r == '<' && i+2 < len(rs) && rs[i+1] == '<' && rs[i+2] == '<':
			flushWord()
			hereString = true
			i += 2
		case r == '<' && i+1 < len(rs) && rs[i+1] == '<':
			flushWord()
			var h heredoc
			h, i = readDelimiter(rs, i+2)
			h.cmd = len(out) // the command being read now
			pending = append(pending, h)
		case r == '\n':
			flushCmd()
			for _, h := range pending {
				var body string
				body, i = readBody(rs, i, h)
				if h.cmd < len(out) {
					out[h.cmd].stdin += body
				}
			}
			pending = nil
		case r == ';' || r == '&' || r == '|' || r == '(' || r == ')':
			flushCmd()
		case r == ' ' || r == '\t':
			flushWord()
		case r == '#' && !inWord:
			for i+1 < len(rs) && rs[i+1] != '\n' {
				i++
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	flushCmd()
	for _, s := range subs {
		out = append(out, parse(s)...)
	}
	return out
}

// balanced returns the text up to the parenthesis that closes one opened just before
// i, and the index of that parenthesis.
func balanced(rs []rune, i int) (string, int) {
	depth, start := 1, i
	var quote rune
	for ; i < len(rs); i++ {
		switch r := rs[i]; {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
		case r == '(':
			depth++
		case r == ')':
			if depth--; depth == 0 {
				return string(rs[start:i]), i
			}
		}
	}
	return string(rs[start:]), len(rs) - 1
}

// backquoted returns the text up to the next unescaped backquote.
func backquoted(rs []rune, i int) (string, int) {
	start := i
	for ; i < len(rs); i++ {
		if rs[i] == '\\' {
			i++
			continue
		}
		if rs[i] == '`' {
			return string(rs[start:i]), i
		}
	}
	return string(rs[start:]), len(rs) - 1
}

type heredoc struct {
	delim     string
	stripTabs bool
	cmd       int // index of the command it belongs to
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

// readBody reads a heredoc body. i is the newline that ends the line holding "<<"; it
// returns the body and the index of the newline that ends the delimiter line.
func readBody(rs []rune, i int, h heredoc) (string, int) {
	var body strings.Builder
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
		body.WriteString(line + "\n")
	}
	if i >= len(rs) {
		i = len(rs) - 1
	}
	return body.String(), i
}

// reserved words of the shell: what follows them is the command.
var reserved = map[string]bool{
	"if": true, "then": true, "else": true, "elif": true, "fi": true, "do": true, "done": true,
	"while": true, "until": true, "!": true, "{": true, "}": true, "time": true, "coproc": true,
}

// wrappers run the command that follows them. The value lists the short options that
// take a value, so the value is not taken for the program.
var wrappers = map[string]string{
	"env": "uCS", "nohup": "", "exec": "a", "command": "", "nice": "n",
	"timeout": "ks", "xargs": "nIPLsdEa", "watch": "ndg", "stdbuf": "ioe", "builtin": "",
	"ionice": "cnp", "chrt": "", "taskset": "", "setsid": "", "unbuffer": "", "caffeinate": "",
}

// program strips variable assignments (NAME=value), reserved words and wrappers from
// the front of a command, and returns the program and its arguments.
func program(words []string) (env map[string]string, name string, args []string) {
	env = map[string]string{}
	i := 0
	for i < len(words) {
		w := words[i]
		if eq := strings.IndexByte(w, '='); eq > 0 && isName(w[:eq]) {
			env[w[:eq]] = w[eq+1:]
			i++
			continue
		}
		if reserved[w] {
			i++
			continue
		}
		opts, isWrapper := wrappers[base(w)]
		if !isWrapper {
			break
		}
		wrapper := base(w)
		i++
		for i < len(words) && strings.HasPrefix(words[i], "-") && words[i] != "-" {
			a := words[i]
			i++
			if a == "--" {
				break
			}
			if wrapper == "command" && (a == "-v" || a == "-V") {
				return env, "", nil // a lookup, nothing runs
			}
			if !strings.HasPrefix(a, "--") && len(a) == 2 && strings.ContainsRune(opts, rune(a[1])) && i < len(words) {
				i++ // the option's value
			}
		}
		if wrapper == "timeout" && i < len(words) {
			i++ // the duration
		}
		if wrapper == "taskset" && i < len(words) {
			i++ // the CPU mask
		}
		if wrapper == "env" { // env NAME=value … program
			for i < len(words) {
				if eq := strings.IndexByte(words[i], '='); eq > 0 && isName(words[i][:eq]) {
					env[words[i][:eq]] = words[i][eq+1:]
					i++
					continue
				}
				break
			}
		}
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

// base returns the program name without its path or a leading backslash.
func base(p string) string {
	p = strings.TrimPrefix(p, `\`)
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}
