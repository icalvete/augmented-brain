package guard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// robustPolicy is a policy with every list filled in, under a throwaway home, so git
// aliases, ~ and rm's home check never read the real one.
func robustPolicy(t testing.TB) Policy {
	t.Helper()
	home := t.TempDir()
	if tt, ok := t.(*testing.T); ok {
		tt.Setenv("HOME", home)
	} else {
		os.Setenv("HOME", home)
	}
	work := filepath.Join(home, "work")
	return Policy{
		Cwd:                   work,
		Env:                   map[string]string{},
		GitAllowedRepos:       []string{filepath.Join(work, "allowed")},
		GitAskRepos:           []string{filepath.Join(work, "ask")},
		UntouchableRepos:      []string{filepath.Join(work, "untouch")},
		ProductionAWSProfiles: []string{"prod"},
		UntouchableHosts:      []string{"db01"},
		RootAsk:               []RootException{{Host: "web01", Until: time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)}},
		Hostname:              "thishost",
		Protected:             []string{filepath.Join(home, "cfg", "config.toml")},
		Hidden:                []string{filepath.Join(home, "cfg", "panel.json")},
		ConfigDirs:            []string{filepath.Join(home, "cfg"), "AB_CONFIG_DIR"},
		Lang:                  "en",
	}
}

// slowLimit is how long one input may take: the hook has 10 seconds, and the race
// detector makes everything several times slower.
func slowLimit() time.Duration {
	if raceEnabled {
		return 5 * time.Second
	}
	return time.Second
}

// Inputs that used to keep the guard busy for seconds, or forever: each answers within
// a second, and the nested ones ask.
func TestNoInputStallsTheGuard(t *testing.T) {
	p := robustPolicy(t)
	exports := func(n, cmds int) string {
		var b strings.Builder
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, "export V%d=1\n", i)
		}
		return b.String() + strings.Repeat("ls\n", cmds)
	}
	words := func(n int) string {
		var b strings.Builder
		b.WriteString("npx prettier --write")
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, " src/components/file%d.ts", i)
		}
		return b.String()
	}
	for _, c := range []struct {
		name string
		in   string
		want Verdict
	}{
		{"find -exec nested 26 times", strings.Repeat("find . -exec ", 26) + "ls", Ask},
		{"find -exec nested 40 times", strings.Repeat("find . -exec ", 40) + "ls", Ask},
		{"one find, -exec with no terminator", "find . " + strings.Repeat("-exec echo {} ", 40), Allow},
		{"git submodule foreach chain 10KB", strings.Repeat("git submodule foreach ", 460) + "ls", Ask},
		{"eval chain 50KB", strings.Repeat("eval ", 10000) + "ls", Ask},
		{"bash -c chain", strings.Repeat("bash -c ", 10000) + "ls", Allow}, // -c takes the word after it alone
		{"sudo -u chain", strings.Repeat("sudo -u app ", 5000) + "ls", Ask},
		{"ssh chain", strings.Repeat("ssh h ", 5000) + "ls", Ask},
		{"nested $(", "echo " + strings.Repeat("$(", 23000) + "date" + strings.Repeat(")", 23000), Ask},
		{"open $(", strings.Repeat("$(", 50000), Ask},
		{"open $((", strings.Repeat("$((", 33000), Ask},
		{"nested $((", "echo " + strings.Repeat("$((", 20000) + "1" + strings.Repeat("))", 20000), Ask},
		{"exports 5000 + 5000 commands", exports(5000, 5000), Allow},
		{"append to one file 15000 times", strings.Repeat("echo a >> f.sh;", 15000), Allow},
		{"prettier with 3000 files", words(3000), Allow},
		{"too long", "echo " + strings.Repeat("a", maxCommand), Ask},
	} {
		start := time.Now()
		d := p.Command(c.in)
		if el := time.Since(start); el > slowLimit() {
			t.Errorf("%s (%d bytes): %v", c.name, len(c.in), el)
		}
		if d.Verdict != c.want {
			t.Errorf("%s: %v (%s), want %v", c.name, d.Verdict, d.Reason, c.want)
		}
	}
}

// A command run in a container at the deepest level is asked about, not let through.
func TestDepthLimitAsksEverywhere(t *testing.T) {
	p := robustPolicy(t)
	for _, cmd := range []string{
		"ssh a ssh b ssh c ssh d docker exec web id",
		"ssh a ssh b ssh c ssh d kubectl exec pod -- id",
		"sudo -u a sudo -u b sudo -u c sudo -u d sudo -u e id",
		"find . -exec find . -exec find . -exec find . -exec find . -exec ls ;",
	} {
		if d := p.Command(cmd); d.Verdict != Ask || !strings.Contains(d.Reason, "nested too deep") {
			t.Errorf("%s: %v (%s), want the depth question", cmd, d.Verdict, d.Reason)
		}
	}
	// Each -exec is still read: the command after the first one's terminator too.
	if d := p.Command(`find . -exec ls \; -exec sudo id \;`); d.Verdict != Deny {
		t.Errorf("second -exec: %v, want Deny", d.Verdict)
	}
}

// A git alias that runs itself ends in a question, quickly, instead of exhausting the
// stack.
func TestRecursiveGitAlias(t *testing.T) {
	p := robustPolicy(t)
	home := os.Getenv("HOME")
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[alias]\n\tagain = !git again\n\tloop = loop\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{"git again", "git loop"} {
		start := time.Now()
		d := p.Command(cmd)
		if el := time.Since(start); el > slowLimit() {
			t.Errorf("%s: %v", cmd, el)
		}
		if cmd == "git again" && d.Verdict != Ask {
			t.Errorf("%s: %v (%s), want Ask", cmd, d.Verdict, d.Reason)
		}
	}
}

// Only the variables the guard reads are carried after export, and those still are.
func TestExportCarriesWhatTheGuardReads(t *testing.T) {
	p := robustPolicy(t)
	if d := p.Command("export AWS_PROFILE=prod; aws s3 ls"); d.Verdict != Ask {
		t.Errorf("exported production profile: %v", d.Verdict)
	}
	if d := p.Command("export FOO=1; ls"); d.Verdict != Allow {
		t.Errorf("other variable: %v", d.Verdict)
	}
}

// Time grows with the length of the line, not with its square, for every shape.
// The short line is parsed 10 times in a row and the long one, 10 times longer, once:
// linear, both take about as long; quadratic, the long one takes about 10 times as
// long. Timing work of the same length, in turns, keeps a busy machine (the pre-push
// hook with other builds running) from stalling one side only, as it did when a 2 ms
// parse was set against a 60 ms one. A shape fails only when every one of three tries
// is over the limit: a quadratic parse is, every time.
func TestParseScalesLinearly(t *testing.T) {
	shapes := map[string]func(n int) string{
		"nested_subst": func(n int) string { return "echo " + strings.Repeat("$(", n/3) + "date" + strings.Repeat(")", n/3) },
		"open_subst":   func(n int) string { return strings.Repeat("$(", n/2) },
		"open_arith":   func(n int) string { return strings.Repeat("$((", n/3) },
		"nested_arith": func(n int) string { return "echo " + strings.Repeat("$((", n/5) + "1" + strings.Repeat("))", n/5) },
		"flat_subst":   func(n int) string { return "echo " + strings.Repeat("$(date) ", n/8) },
		"open_dquote":  func(n int) string { return "echo \"" + strings.Repeat("$(a b ", n/6) },
	}
	for name, build := range shapes {
		short, long := build(10<<10), build(100<<10)
		var tries []string
		for range 3 {
			time10, time100 := timeParse(short, long)
			// 10 times the text: far less than 10 times the time of 10 short ones (a floor for noise).
			if time100 <= 3*time10 || time100 <= 50*time.Millisecond {
				tries = nil
				break
			}
			tries = append(tries, fmt.Sprintf("10 KB ten times in %v, 100 KB in %v", time10, time100))
		}
		if tries != nil {
			t.Errorf("%s: %s", name, strings.Join(tries, "; "))
		}
	}
}

// timeParse is the best of five rounds of parsing short 10 times and long once, timed in turns.
func timeParse(short, long string) (best10, best100 time.Duration) {
	best10, best100 = time.Hour, time.Hour
	for range 5 {
		start := time.Now()
		for range 10 {
			parse(short)
		}
		best10 = min(best10, time.Since(start))
		start = time.Now()
		parse(long)
		best100 = min(best100, time.Since(start))
	}
	return best10, best100
}

// fuzzSeeds are odd inputs for the fuzz tests: every shape that once stalled the guard
// or made it fail.
var fuzzSeeds = []string{
	"", " ", "\n", "\x00", ";", "|", "&&", "(", ")", "((", "{", "'", "\"", "`", "\\", "$", "$(", "$((", "$(((",
	"<(", ">(", "<<", "<<<", "<<EOF\nbody", "<<EOF\nbody\nEOF", "echo 'abc", "echo \"abc", "echo $(abc",
	"echo $((1+2)", "echo $(( $(( $(( 1 )) )) ))", "a\x00b", "echo \xff\xfe", "git push\r", "￼ x",
	"env -S ''", "sudo --", "ssh -J", "git -C -C -C", "git submodule foreach", "git rebase -x", "git bisect run",
	"find . -exec ;", "find . -exec find . -exec find . -exec ls", "rm -rf ${", "bash -c ''", "eval ''",
	"docker exec", "kubectl exec --", "export A", "echo > > >", "a=$(b=$(c=$(d)))", "x=`y=\\`z\\``",
	strings.Repeat("$(", 200), strings.Repeat("$((", 200), strings.Repeat("find . -exec ", 30) + "ls",
	strings.Repeat("sudo ", 200) + "id", strings.Repeat("eval ", 200) + "ls",
}

func FuzzParse(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, line string) {
		start := time.Now()
		for _, c := range parse(line) {
			program(c.words)
			output(c, "", false)
		}
		if el := time.Since(start); el > 2*slowLimit() {
			t.Fatalf("parse took %v for %d bytes", el, len(line))
		}
	})
}

func FuzzCommand(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add(s)
	}
	p := robustPolicy(f)
	f.Fuzz(func(t *testing.T, line string) {
		start := time.Now()
		d := p.Command(line)
		if d.Verdict < Allow || d.Verdict > Deny {
			t.Fatalf("verdict %d", d.Verdict)
		}
		if d.Verdict != Allow && d.Reason == "" {
			t.Fatalf("verdict %d with no reason for %q", d.Verdict, line)
		}
		_ = p.WriteFile(line, line)
		_ = p.ReadFile(line)
		if el := time.Since(start); el > 2*slowLimit() {
			t.Fatalf("Command took %v for %d bytes", el, len(line))
		}
	})
}

// A shell script the guard cannot read, or too large to, asks instead of passing
// unchecked; a large program run by its path (a build's output) still passes.
func TestUnreadableScriptAsks(t *testing.T) {
	p := robustPolicy(t)
	dir := t.TempDir()
	p.Cwd = dir
	big := filepath.Join(dir, "big.sh")
	if err := os.WriteFile(big, []byte(strings.Repeat("echo hello\n", 7000)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin", "tool"), []byte(strings.Repeat("x", maxScript+1)), 0o755); err != nil {
		t.Fatal(err)
	}
	for cmd, want := range map[string]Verdict{
		"bash big.sh":   Ask,
		"./big.sh":      Ask,
		"source big.sh": Ask,
		"./bin/tool":    Allow,
		"bash gone.sh":  Allow, // nothing there to run
	} {
		if d := p.Command(cmd); d.Verdict != want {
			t.Errorf("%s: %v (%s), want %v", cmd, d.Verdict, d.Reason, want)
		}
	}
}
