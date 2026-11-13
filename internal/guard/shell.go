package guard

import (
	"slices"
	"strings"
)

// command is one simple command: its words, with quotes removed, and the text it
// reads from a heredoc or a here-string, if any.
type command struct {
	words     []string
	stdin     string
	piped     bool     // reads the output of the command before it (a | b)
	redirects []string // the files of its redirections (> f, 2>>f, < f), not arguments
	outputs   []string // those of them it writes to (> f, >> f, <> f)
	// tooDeep stands for substitutions nested deeper than maxNest, never read: the
	// guard asks the user about them.
	tooDeep bool
}

// maxNest is how many $(…) inside one another parse reads; deeper ones become a
// command marked tooDeep. It keeps a line of thousands of $( from taking seconds.
const maxNest = 8

// parse splits a shell command line into simple commands. It understands quotes,
// backslashes, comments, the separators ; & | ( ) { } and newlines, heredocs and
// here-strings (kept as the command's stdin), and command and process substitutions
// ($(…), `…`, <(…), >(…)), which become commands of their own, also inside double
// quotes. It is not a full shell parser: it only has to find every program that would
// run and its arguments, which is all the guard needs.
func parse(line string) []command {
	return parseAt(line, 0)
}

// parseAt is parse of a line found inside level substitutions.
func parseAt(line string, level int) []command {
	p := &parser{rs: []rune(line), level: level}
	for ; p.i < len(p.rs); p.i++ {
		p.step()
	}
	p.endCommand()
	out := p.out
	for _, s := range p.subs {
		if level >= maxNest {
			return append(out, command{tooDeep: true})
		}
		out = append(out, parseAt(s, level+1)...)
	}
	return out
}

// parser is what parse knows while it reads a line, one rune at a time.
type parser struct {
	rs    []rune
	i     int // the rune being read
	out   []command
	level int // how many substitutions the line is inside

	words      []string // of the command being read
	stdin      string
	piped      bool // the command being read gets the previous one's output
	cur        strings.Builder
	inWord     bool
	hereString bool // the word being read is the text of a <<<, not an argument
	redirect   bool // the word being read is the file of a redirection
	redirectTo bool // and the redirection writes to it
	redirects  []string
	outputs    []string

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
	case r == '$' && ahead(1) == '(' && p.arithmetic(i):
		p.cur.WriteRune(subst)
		p.inWord = true
	case (r == '$' || r == '<' || r == '>') && ahead(1) == '(':
		p.substitution(i + 2)
		p.cur.WriteRune(subst)
		p.inWord = true
	case r == '`':
		p.backquote(i + 1)
		p.cur.WriteRune(subst)
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
	case r == '<' || r == '>' || r == '&' && ahead(1) == '>':
		p.redirection()
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

// redirection reads an operator such as >, >>, 2>, &>, >&2, <, <> or >| at p.i. It is
// never part of a word: in "git status>log" the word is status. A number just before
// it is the file descriptor, not an argument; the word after it is the file, kept apart.
func (p *parser) redirection() {
	if p.inWord && isDigits(p.cur.String()) {
		p.cur.Reset()
		p.inWord = false
	}
	p.endWord()
	writes := p.rs[p.i] == '>'
	for p.i+1 < len(p.rs) && strings.ContainsRune("<>&|", p.rs[p.i+1]) && !(p.rs[p.i+1] == '|' && p.rs[p.i] != '>') {
		p.i++
		writes = writes || p.rs[p.i] == '>'
	}
	p.redirect, p.redirectTo = true, writes
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
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
		case rs[i] == '$' && i+1 < len(rs) && rs[i+1] == '(' && p.arithmetic(i):
			p.cur.WriteRune(subst)
		case rs[i] == '$' && i+1 < len(rs) && rs[i+1] == '(':
			p.substitution(i + 2)
			p.cur.WriteRune(subst)
		case rs[i] == '`':
			p.backquote(i + 1)
			p.cur.WriteRune(subst)
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

// arithmetic reads, when the $ at i opens an arithmetic expansion $((…)), the whole of
// it: only the command substitutions inside run, and they are kept to check later. What
// else is there is numbers and names, not programs: read as $( (…) ), the expression
// in "$(( $(date +%s)-1 ))" would be a program the guard cannot know. It says false,
// and reads nothing, when the text at i is not one: $( (cmd) ), or a $(( never closed.
func (p *parser) arithmetic(i int) bool {
	subs, end, ok := arithmeticAt(p.rs, i, p.level)
	if !ok {
		return false
	}
	p.subs = append(p.subs, subs...)
	p.i = end
	return true
}

// arithmeticAt is arithmetic's reading of rs: the substitutions in the $((…)) at i and
// the index of its last parenthesis. Like bash, it is one only when the parenthesis that
// closes the inner one is the outer's closing too: "$((a) )" is a subshell.
// level is how deep it is already: past maxNest it says false, and the text is read
// as a substitution, which parse does not follow that deep either.
func arithmeticAt(rs []rune, i, level int) (subs []string, end int, ok bool) {
	if level > maxNest || i+2 >= len(rs) || rs[i] != '$' || rs[i+1] != '(' || rs[i+2] != '(' {
		return nil, 0, false
	}
	_, inner := balanced(rs, i+3)
	if inner+1 >= len(rs) || rs[inner] != ')' || rs[inner+1] != ')' {
		return nil, 0, false
	}
	for j := i + 3; j < inner; j++ {
		switch {
		case rs[j] == '$' && j+1 < inner && rs[j+1] == '(':
			if more, e, ok := arithmeticAt(rs, j, level+1); ok {
				subs, j = append(subs, more...), e
				continue
			}
			var s string
			s, j = balanced(rs, j+2)
			subs = append(subs, s)
		case rs[j] == '`':
			var s string
			s, j = backquoted(rs, j+1)
			subs = append(subs, s)
		}
	}
	return subs, inner + 1, true
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
	switch {
	case p.hereString:
		p.stdin, p.hereString = p.cur.String(), false
	case p.redirect:
		p.redirects, p.redirect = append(p.redirects, p.cur.String()), false
		if p.redirectTo {
			p.outputs = append(p.outputs, p.cur.String())
		}
	default:
		p.words = append(p.words, p.cur.String())
	}
	p.cur.Reset()
	p.inWord = false
}

// endCommand ends the command being read, if it has anything in it.
func (p *parser) endCommand() {
	p.endWord()
	p.redirect = false
	if len(p.words) > 0 || p.stdin != "" || len(p.redirects) > 0 {
		p.out = append(p.out, command{words: p.words, stdin: p.stdin, piped: p.piped, redirects: p.redirects, outputs: p.outputs})
		p.piped = false
	}
	p.words, p.stdin, p.redirects, p.outputs = nil, "", nil, nil
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
// take a value, so the value is not taken for the program. A name here never reaches
// the switch in simple: one with a rule of its own (sudo, doas) must not be listed
// (TestWrappersHaveNoRule).
var wrappers = map[string]string{
	"env": "uCS", "nohup": "", "exec": "a", "command": "", "nice": "n",
	"timeout": "ks", "xargs": "nIPLsdEa", "watch": "ndg", "stdbuf": "ioe", "builtin": "",
	"ionice": "cnp", "chrt": "p", "taskset": "", "setsid": "", "unbuffer": "", "caffeinate": "", "npx": "pc",
	"time": "fo", "flock": "wEc", "busybox": "", "sshpass": "pfdP", "proxychains": "f", "torsocks": "",
}

// wrapperLong lists, for each wrapper, the long options that take a value. Written
// apart (--signal KILL), the value is the next word, and must not be taken for the
// program; written --signal=KILL, it is one word and needs no entry.
var wrapperLong = map[string][]string{
	"env":     {"--unset", "--chdir"},
	"nice":    {"--adjustment"},
	"timeout": {"--signal", "--kill-after"},
	"xargs": {"--arg-file", "--delimiter", "--max-args", "--max-procs", "--max-chars",
		"--process-slot-var"},
	"watch":  {"--interval", "--equexit"},
	"stdbuf": {"--input", "--output", "--error"},
	"ionice": {"--class", "--classdata", "--pid", "--pgid", "--uid"},
	"chrt":   {"--sched-runtime", "--sched-period", "--sched-deadline"},
	"npx":    {"--package", "--call"},
	"time":   {"--format", "--output"},
	"flock":  {"--wait", "--timeout", "--conflict-exit-code"},
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
			if w == "time" && i < len(words) && words[i] == "-p" {
				i++ // time -p: the POSIX output format
			}
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
			if wrapper == "env" { // env -S 'program args': the string is split into words
				if split, ok := envSplit(a, words, i); ok {
					words = split
					continue
				}
			}
			if wrapper == "command" && (a == "-v" || a == "-V") {
				return env, "", nil // a lookup, nothing runs
			}
			short := !strings.HasPrefix(a, "--") && len(a) == 2 && strings.ContainsRune(opts, rune(a[1]))
			if (short || slices.Contains(wrapperLong[wrapper], a)) && i < len(words) {
				i++ // the option's value
			}
		}
		if wrapper == "timeout" && i < len(words) {
			i++ // the duration
		}
		if (wrapper == "taskset" || wrapper == "flock") && i < len(words) {
			i++ // the CPU mask, the lock file
		}
		if wrapper == "chrt" && i < len(words) {
			i++ // the priority
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

// envSplit expands env's -S (--split-string) option a, whose word is just before i, in
// words: its value becomes words of their own where it stood.
func envSplit(a string, words []string, i int) ([]string, bool) {
	var value string
	rest := i
	switch {
	case a == "-S" || a == "--split-string":
		if i >= len(words) {
			return nil, false
		}
		value, rest = words[i], i+1
	case strings.HasPrefix(a, "--split-string="):
		value = strings.TrimPrefix(a, "--split-string=")
	case strings.HasPrefix(a, "-S") && !strings.HasPrefix(a, "--"):
		value = a[2:]
	default:
		return nil, false
	}
	out := append(append(append([]string{}, words[:i]...), strings.Fields(value)...), words[rest:]...)
	return out, true
}

func isName(s string) bool {
	for i, r := range s {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return s != ""
}

// base returns the program name without its path, a leading backslash or a Windows
// .exe, in lower case: on Windows GIT.exe is git.
func base(p string) string {
	p = strings.TrimPrefix(p, `\`)
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		p = p[i+1:]
	}
	p = strings.ToLower(p)
	return strings.TrimSuffix(p, ".exe")
}
