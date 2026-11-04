package hostmem

import (
	"slices"
	"testing"
)

// Fake is a process tree for tests.
type Fake struct {
	Procs map[int]Proc
	Sizes map[int]int64 // bytes
}

func (f Fake) Proc(pid int) (Proc, bool) { p, ok := f.Procs[pid]; return p, ok }
func (f Fake) RSS(pid int) (int64, bool) { n, ok := f.Sizes[pid]; return n, ok }
func (f Fake) PIDs() []int {
	var out []int
	for pid := range f.Procs {
		out = append(out, pid)
	}
	slices.Sort(out)
	return out
}

const mb = 1 << 20

// tree: VS Code's extension host 100 runs claude 200, which runs a shell 300 (the
// hook's parent); a terminal claude 500 under a shell 400; a shell 600 under nothing.
func tree() Fake {
	return Fake{
		Procs: map[int]Proc{
			1:   {PID: 1, Name: "systemd", Args: []string{"/sbin/init"}, Start: "1"},
			90:  {PID: 90, PPID: 1, Name: "code", Args: []string{"/usr/share/code/code"}, Start: "5"},
			100: {PID: 100, PPID: 90, Name: "code", Args: []string{"/proc/self/exe", "--type=utility", "--utility-sub-type=node.mojom.NodeService"}, Start: "10"},
			200: {PID: 200, PPID: 100, Name: "claude", Args: []string{"/home/u/.vscode/extensions/anthropic.claude-code-2.1/resources/native-binary/claude", "--output-format", "stream-json"}, Start: "20"},
			201: {PID: 201, PPID: 100, Name: "claude", Start: "21"},
			300: {PID: 300, PPID: 200, Name: "sh", Args: []string{"/bin/sh", "-c", "augmented-brain hook stop"}, Start: "30"},
			400: {PID: 400, PPID: 1, Name: "bash", Start: "40"},
			500: {PID: 500, PPID: 400, Name: "MainThread", Args: []string{"/usr/local/bin/claude"}, Start: "50"},
			600: {PID: 600, PPID: 1, Name: "bash", Start: "60"},
			700: {PID: 700, PPID: 1, Name: "Code Helper (Plu", Args: []string{"/Applications/Visual Studio Code.app/Contents/Frameworks/Code Helper (Plugin).app/Contents/MacOS/Code Helper (Plugin)"}, Start: "70"},
			710: {PID: 710, PPID: 700, Name: "claude", Start: "71"},
		},
		Sizes: map[int]int64{100: 3000 * mb, 200: 300 * mb, 201: 200 * mb, 500: 600 * mb, 700: 450 * mb},
	}
}

// In VS Code the host is the extension host above claude; in a terminal, claude itself.
func TestFind(t *testing.T) {
	f := tree()
	for _, c := range []struct {
		from, pid int
		kind      string
		rss       int64
	}{
		{300, 100, VSCode, 3000 * mb}, // through the hook's shell
		{200, 100, VSCode, 3000 * mb}, // claude ran the hook with no shell between
		{500, 500, Claude, 600 * mb},  // terminal: known by its command, not its name
		{710, 700, VSCode, 450 * mb},  // macOS
	} {
		h, ok := Find(f, c.from)
		if !ok || h.PID != c.pid || h.Kind != c.kind || h.RSS != c.rss {
			t.Errorf("from %d: %+v %v", c.from, h, ok)
		}
	}
	for _, from := range []int{600, 400, 999} {
		if h, ok := Find(f, from); ok {
			t.Errorf("from %d, with no claude above: %+v", from, h)
		}
	}
	// No size: nothing to say.
	delete(f.Sizes, 100)
	if _, ok := Find(f, 300); ok {
		t.Error("found a host whose size cannot be read")
	}
}

// Sessions of one window share their host, which is counted once.
func TestHosts(t *testing.T) {
	var got []int
	for _, h := range Hosts(tree()) {
		got = append(got, h.PID)
	}
	if !slices.Equal(got, []int{100, 500, 700}) {
		t.Errorf("hosts: %v", got)
	}
}

func TestDue(t *testing.T) {
	steps := []int64{2048, 4096, 8192}
	for _, c := range []struct {
		rss, warned, want int64
	}{
		{500 * mb, 0, 0},
		{2047 * mb, 0, 0},
		{2048 * mb, 0, 2048},
		{3000 * mb, 2048, 0},     // never the same step twice
		{4100 * mb, 2048, 4096},  // the next step
		{9000 * mb, 0, 8192},     // a jump: the highest step crossed, once
		{9000 * mb, 8192, 0},     // past the last step: quiet
		{1000 * mb, 4096, 0},     // shrank: nothing
		{16000 * mb, 4096, 8192}, // after 4 GB, 8 GB still comes
	} {
		if got := Due(c.rss, c.warned, steps); got != c.want {
			t.Errorf("Due(%d MB, %d) = %d, want %d", c.rss/mb, c.warned, got, c.want)
		}
	}
}

func TestSteps(t *testing.T) {
	env := func(v string) func(string) string { return func(string) string { return v } }
	if got := Steps(env("")); !slices.Equal(got, StepsMB) {
		t.Errorf("default: %v", got)
	}
	if got := Steps(env("300")); !slices.Equal(got, []int64{300, 600, 1200}) {
		t.Errorf("override: %v", got)
	}
	for _, bad := range []string{"0", "-5", "lots"} {
		if got := Steps(env(bad)); !slices.Equal(got, StepsMB) {
			t.Errorf("%q: %v", bad, got)
		}
	}
}

// A key names a process by pid and start: the same pid, started again, is another one.
func TestAlive(t *testing.T) {
	f := tree()
	if !Alive(f, Key(100, "10")) {
		t.Error("running host not alive")
	}
	if Alive(f, Key(100, "9")) {
		t.Error("a reused pid counted as the same host")
	}
	for _, k := range []string{Key(999, "1"), "junk", "x:1"} {
		if Alive(f, k) {
			t.Errorf("%q alive", k)
		}
	}
}

func TestGB(t *testing.T) {
	if got := GB(9400*mb, "en"); got != "9.2" {
		t.Errorf("en: %s", got)
	}
	if got := GB(2048*mb, "es"); got != "2,0" {
		t.Errorf("es: %s", got)
	}
}
