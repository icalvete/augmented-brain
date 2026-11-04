// Package hostmem watches the memory of the program that hosts a Claude session: VS
// Code's extension host when the session runs in VS Code (it starts the claude process
// and holds every session of its window), claude itself in a terminal.
//
// Such a host normally stays near half a gigabyte for days. Now and then one grows
// without stopping, by hundreds of megabytes a minute, until the system kills it, and
// every session of the window with it. The size of the conversation does not tell it
// coming; the host's resident memory does. The Stop hook looks at it after each answer
// and has the assistant warn the user at a few steps, once each.
//
// Everything here is quick and never fails: a process that cannot be read is skipped.
// On Linux it reads /proc (no program is run); on macOS it asks the kernel, and runs
// `ps` once for the size; elsewhere (Windows) it does nothing.
package hostmem

import (
	"path/filepath"
	"strconv"
	"strings"
)

// Proc is one process, as much of it as the checks need.
type Proc struct {
	PID, PPID int
	Name      string   // the system's short name for it (Linux keeps 15 characters)
	Args      []string // its command line
	// Start is when it started, as the system tells it. With the pid it names the
	// process: a pid reused by a later process has another start.
	Start string
}

// System reads processes: the real one (Local) or a fake in tests.
type System interface {
	// Proc reads one process; false when it is gone or cannot be read.
	Proc(pid int) (Proc, bool)
	// RSS is the process's resident memory, in bytes; false when it cannot be read.
	RSS(pid int) (int64, bool)
	// PIDs lists the processes this user can see (for doctor, which looks at them all).
	PIDs() []int
}

// Local is this machine; nil where memory is not watched (Windows). Tests set it.
var Local = local()

// Kinds of host.
const (
	VSCode = "vscode" // VS Code's extension host (or that of an editor built the same way)
	Claude = "claude" // claude itself: a terminal session
)

// Host is the process whose memory is watched for a Claude session.
type Host struct {
	Proc
	Kind string
	RSS  int64 // bytes
}

// Key names the host among every process this machine ran: pid and start.
func (h Host) Key() string { return Key(h.PID, h.Start) }

// Key is the name of a process for the marks: its pid and when it started.
func Key(pid int, start string) string { return strconv.Itoa(pid) + ":" + start }

// maxUp is how many processes up from the hook to look for claude: Claude Code runs the
// hook through a shell, which may or may not be a process of its own.
const maxUp = 5

// isClaude says whether p is Claude Code.
func isClaude(p Proc) bool {
	if p.Name == "claude" {
		return true
	}
	return len(p.Args) > 0 && strings.TrimSuffix(filepath.Base(p.Args[0]), ".exe") == "claude"
}

// isExtensionHost says whether p is VS Code's extension host. Since 2024 VS Code runs it
// as an Electron utility process ("code … --type=utility
// --utility-sub-type=node.mojom.NodeService" on Linux, "Code Helper (Plugin)" on macOS);
// before, it was "--type=extensionHost". Editors built on VS Code (Cursor, VSCodium) run
// it the same way.
func isExtensionHost(p Proc) bool {
	for _, a := range p.Args {
		if strings.Contains(a, "node.mojom.NodeService") || a == "--type=extensionHost" {
			return true
		}
	}
	return strings.HasPrefix(p.Name, "Code Helper (Plu") ||
		(len(p.Args) > 0 && strings.Contains(p.Args[0], "Code Helper (Plugin)"))
}

// Find walks up from pid (the hook's parent) to the claude process that runs the hook,
// and returns its host with its memory: claude's parent when that is VS Code's extension
// host, claude itself otherwise. False when there is no claude a few levels up, or the
// size cannot be read.
func Find(sys System, pid int) (Host, bool) {
	for range maxUp {
		p, ok := sys.Proc(pid)
		if !ok {
			return Host{}, false
		}
		if isClaude(p) {
			return measure(sys, hostOf(sys, p))
		}
		if p.PPID <= 1 || p.PPID == pid {
			return Host{}, false
		}
		pid = p.PPID
	}
	return Host{}, false
}

// hostOf is the host of a claude process, not yet measured.
func hostOf(sys System, claude Proc) Host {
	if parent, ok := sys.Proc(claude.PPID); ok && isExtensionHost(parent) {
		return Host{Proc: parent, Kind: VSCode}
	}
	return Host{Proc: claude, Kind: Claude}
}

func measure(sys System, h Host) (Host, bool) {
	rss, ok := sys.RSS(h.PID)
	h.RSS = rss
	return h, ok
}

// Hosts returns the host of every Claude session running on this machine, each once
// (the sessions of one VS Code window share its host), with its memory.
func Hosts(sys System) []Host {
	var out []Host
	seen := map[string]bool{}
	for _, pid := range sys.PIDs() {
		p, ok := sys.Proc(pid)
		if !ok || !isClaude(p) {
			continue
		}
		h := hostOf(sys, p)
		if seen[h.Key()] {
			continue
		}
		seen[h.Key()] = true
		if h, ok := measure(sys, h); ok {
			out = append(out, h)
		}
	}
	return out
}

// Alive says whether the process a key names still runs: same pid, same start.
func Alive(sys System, key string) bool {
	pidText, start, ok := strings.Cut(key, ":")
	pid, err := strconv.Atoi(pidText)
	if !ok || err != nil {
		return false
	}
	p, ok := sys.Proc(pid)
	return ok && p.Start == start
}

// StepsMB are the sizes, in megabytes, at which the user is warned: normal is about
// 512, so the first step is four times that.
var StepsMB = []int64{2048, 4096, 8192}

// Steps are the warning steps, in megabytes. AB_MEMORY_WARN_MB (read through getenv) is
// a test aid, not a setting: it moves the first step, and the others follow at twice
// and four times it, so the warning can be tried on a host of normal size.
func Steps(getenv func(string) string) []int64 {
	if n, err := strconv.ParseInt(getenv("AB_MEMORY_WARN_MB"), 10, 64); err == nil && n > 0 {
		return []int64{n, 2 * n, 4 * n}
	}
	return StepsMB
}

// Due is the step (in megabytes) to warn about for a host of rss bytes, which was last
// warned about at step warned (0 for never); 0 means nothing to say. A host that jumps
// over a step is warned about the highest one it crossed, and never about a step twice.
func Due(rss, warned int64, steps []int64) int64 {
	var due int64
	for _, s := range steps {
		if rss >= s<<20 {
			due = s
		}
	}
	if due > warned {
		return due
	}
	return 0
}

// GB is a size in bytes as gigabytes with one decimal, as people read it in lang (a
// decimal comma in Spanish).
func GB(bytes int64, lang string) string {
	s := strconv.FormatFloat(float64(bytes)/(1<<30), 'f', 1, 64)
	if lang == "es" {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s
}
