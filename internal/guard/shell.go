package guard

import "strings"

// command is one simple command: its words, with quotes removed, and the text it
// reads from a heredoc or a here-string, if any.
type command struct {
	words []string
	stdin string
	piped bool // reads the output of the command before it (a | b)
}

// parse splits a shell command line into simple commands. It understands quotes,
// backslashes, comments, the separators ; & | ( ) { } and newlines, heredocs and
// here-strings (kept as the command's stdin), and command and process substitutions
// ($(…), `…`, <(…), >(…)), which become commands of their own, also inside double
// quotes. It is not a full shell parser: it only has to find every program that would
// run and its arguments, which is all the guard needs.
func parse(line string) []command {
	p := &parser{rs: []rune(line)}
	for ; p.i < len(p.rs); p.i++ {
		p.step()
	}
	p.endCommand()
	out := p.out
	for _, s := range p.subs {
		out = append(out, parse(s)...)
	}
	return out
}

// parser is what parse knows while it reads a line, one rune at a time.
type parser struct {
	rs  []rune
	i   int // the rune being read
	out []command

	words      []string // of the command being read
	stdin      string
	piped      bool // the command being read gets the previous one's output
	cur        strings.Builder
	inWord     bool
	hereString bool // the word being read is the text of a <<<, not an argument

	subs    []string  // substitutions, parsed once the line is done
	pending []heredoc // heredocs whose body starts on the next line
}

// step reads the rune at p.i, and the runes after it that go with it.
func (p *parser) step() {
	rs, i := p.rs, p.i
	r := rs[i]
	ahead := func(n int) rune { // the rune n places ahead, or 0 past the end
		if i+n < len(rs) {
			return rs[i+n]
		}
		return 0
	}
	switch {
	case r == '\\' && i+1 < len(rs):
		p.i++
		if rs[p.i] != '\n' {
			p.cur.WriteRune(rs[p.i])
			p.inWord = true
		}
	case r == '\'':
		p.singleQuoted()
	case r == '"':
		p.doubleQuoted()
	case (r == '$' || r == '<' || r == '>') && ahead(1) == '(':
		p.substitution(i + 2)
		p.inWord = true
	case r == '`':
		p.backquote(i + 1)
		p.inWord = true
	case r == '<' && ahead(1) == '<' && ahead(2) == '<':
		p.endWord()
		p.hereString = true
		p.i += 2
	case r == '<' && ahead(1) == '<':
		p.endWord()
		var h heredoc
		h, p.i = readDelimiter(rs, i+2)
		h.cmd = len(p.out) // the command being read now
		p.pending = append(p.pending, h)
	case r == '\n':
		p.endCommand()
		p.piped = false
		p.readHeredocs()
	case r == '|' && ahead(1) == '|': // ||: the next command does not read this one
		p.endCommand()
		p.piped = false
		p.i++
	case r == '|':
		p.endCommand()
		p.piped = true
		if ahead(1) == '&' { // |& pipes stderr too
			p.i++
		}
	case r == '(' && len(p.words) == 0 && !p.inWord: // … | (bash): the subshell still reads the pipe
		p.endCommand()
	case r == ';' || r == '&' || r == '(' || r == ')':
		p.endCommand()
		p.piped = false
	case r == ' ' || r == '\t':
		p.endWord()
	case r == '#' && !p.inWord: // a comment, up to the end of the line
		for p.i+1 < len(rs) && rs[p.i+1] != '\n' {
			p.i++
		}
	default:
		p.cur.WriteRune(r)
		p.inWord = true
	}
}

// singleQuoted reads '…', where every rune stands for itself.
func (p *parser) singleQuoted() {
	p.inWord = true
	for p.i++; p.i < len(p.rs) && p.rs[p.i] != '\''; p.i++ {
		p.cur.WriteRune(p.rs[p.i])
	}
}

// doubleQuoted reads "…", where a backslash escapes only " \ $ and `, and $(…) and `…`
// still run.
func (p *parser) doubleQuoted() {
	rs := p.rs
	p.inWord = true
	for p.i++; p.i < len(rs) && rs[p.i] != '"'; p.i++ {
		switch i := p.i; {
		case rs[i] == '\\' && i+1 < len(rs) && strings.ContainsRune(`"\$`+"`", rs[i+1]):
			p.i++
			p.cur.WriteRune(rs[p.i])
		case rs[i] == '$' && i+1 < len(rs) && rs[i+1] == '(':
			p.substitution(i + 2)
		case rs[i] == '`':
			p.backquote(i + 1)
		default:
			p.cur.WriteRune(rs[i])
		}
	}
}

// substitution keeps, to check later, the text of a $(…), <(…) or >(…) that starts at i.
func (p *parser) substitution(i int) {
	var s string
	s, p.i = balanced(p.rs, i)
	p.subs = append(p.subs, s)
}

// backquote keeps, to check later, the text of a `…` that starts at i.
func (p *parser) backquote(i int) {
	var s string
	s, p.i = backquoted(p.rs, i)
	p.subs = append(p.subs, s)
}

// readHeredocs reads, after the newline at p.i, the bodies of the heredocs opened on
// the line it ends, and gives each to its command as stdin.
func (p *parser) readHeredocs() {
	for _, h := range p.pending {
		var body string
		body, p.i = readBody(p.rs, p.i, h)
		if h.cmd < len(p.out) {
			p.out[h.cmd].stdin += body
		}
	}
	p.pending = nil
}

// endWord ends the word being read: an argument, or the text of a here-string.
func (p *parser) endWord() {
	if !p.inWord {
		return
	}
	if p.hereString {
		p.stdin, p.hereString = p.cur.String(), false
	} else {
		p.words = append(p.words, p.cur.String())
	}
	p.cur.Reset()
	p.inWord = false
}

// endCommand ends the command being read, if it has anything in it.
func (p *parser) endCommand() {
	p.endWord()
	if len(p.words) > 0 || p.stdin != "" {
		p.out = append(p.out, command{words: p.words, stdin: p.stdin, piped: p.piped})
		p.piped = false
	}
	p.words, p.stdin = nil, ""
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
	"ionice": "cnp", "chrt": "", "taskset": "", "setsid": "", "unbuffer": "", "caffeinate": "", "npx": "pc",
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
