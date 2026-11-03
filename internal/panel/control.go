// Package panel is a small web page to see and edit a person's spaces: tasks, watch
// dates, the inbox, areas, knowledge, people and the profile. It runs on demand, on
// this machine only, and stops by itself when nobody uses it.
//
// `augmented-brain panel` starts the server in a detached process and returns at once
// (so it works from a terminal and from the assistant's shell tool, which waits for the
// command to end); `augmented-brain panel stop` stops it. The server listens on
// 127.0.0.1, on a free port. Its pages live under /s/<session>/, where the session
// comes from the random token of this run (in panel.json); the command itself sends
// the token in a header. Every write goes through package brain, under the same locks
// as the MCP server.
package panel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/detach"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// InfoFile, in the configuration folder, says where the running panel listens. It
// holds the token, so only the user can read it.
const InfoFile = "panel.json"

// lockFile, in the configuration folder, lets one `panel` or `panel stop` at a time
// look for the running panel and start or stop it, so two at once never start two.
const lockFile = "panel.lock"

// logFile, in the configuration folder, keeps what the detached server says when it
// fails to start.
const logFile = "panel.log"

// Idle is how long the server waits without a request before it stops by itself.
const Idle = 30 * time.Minute

// stopWait is how long Stop waits for the server's process to end before it kills it.
const stopWait = 10 * time.Second

// Info is the running panel: its process, its port and the token of this run.
type Info struct {
	PID     int       `json:"pid"`
	Port    int       `json:"port"`
	Token   string    `json:"token"`
	Started time.Time `json:"started"`
}

// mac is HMAC-SHA256 of msg under the token, in hex.
func mac(token, msg string) string {
	h := hmac.New(sha256.New, []byte(token))
	h.Write([]byte(msg))
	return hex.EncodeToString(h.Sum(nil))
}

// sessionOf is the part of the panel's address that lets a browser in: it comes from
// the token, so the command can print it, but it is not the token. No cookie carries
// it: a cookie of 127.0.0.1 would also go to every other local server, whatever its
// port.
func sessionOf(token string) string { return mac(token, "session")[:32] }

// proof is what the server answers to a ping with challenge c: only a server that
// knows the token can give it. The "ping" prefix keeps it apart from sessionOf.
func proof(token, c string) string { return mac(token, "ping\x00"+c) }

// URL is the address to open: the panel's pages live under /s/<session>/.
func (i Info) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/s/%s/", i.Port, sessionOf(i.Token))
}

func (i Info) origin() string { return fmt.Sprintf("http://127.0.0.1:%d", i.Port) }

// ReadInfo returns the panel the configuration folder knows about, if any. It may have
// stopped since: Alive tells.
func ReadInfo(cfgDir string) (Info, bool) {
	var i Info
	data, err := os.ReadFile(filepath.Join(cfgDir, InfoFile))
	if err != nil || json.Unmarshal(data, &i) != nil || i.Port == 0 || i.Token == "" {
		return Info{}, false
	}
	return i, true
}

// writeInfo writes the info file in one step: a new file of its own, renamed over the
// old one, so a reader never sees half of it and two writers never share a file.
func writeInfo(cfgDir string, i Info) error {
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(cfgDir, "panel-*.json.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp, 0o600)
	}
	if err == nil {
		err = paths.Rename(tmp, filepath.Join(cfgDir, InfoFile))
	}
	if err != nil {
		_ = os.Remove(tmp)
	}
	return err
}

// removeInfo deletes the info file when it still describes the process pid, so a
// server that stops never removes the file of a newer one.
func removeInfo(cfgDir string, pid int) {
	if i, ok := ReadInfo(cfgDir); ok && i.PID == pid {
		_ = paths.Remove(filepath.Join(cfgDir, InfoFile)) // Stop may be reading it right now
	}
}

var client = &http.Client{
	Timeout: 2 * time.Second,
	// The CLI talks to its own server only; never follow it anywhere.
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// Alive reports whether the panel in i answers on its port and proves it knows the
// token: the command sends a random challenge, and only the real server can answer
// HMAC-SHA256(token, challenge). The token itself is never sent, so whatever took the
// port of a panel that ended learns nothing from it.
func Alive(i Info) bool {
	c, err := random()
	if err != nil {
		return false
	}
	resp, err := client.Get(i.origin() + "/ping?c=" + c)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	return resp.StatusCode == http.StatusOK && resp.Header.Get("X-Augmented-Brain") == "panel" &&
		hmac.Equal([]byte(strings.TrimSpace(string(body))), []byte(proof(i.Token, c)))
}

// lock takes the panel lock of cfgDir.
func lock(cfgDir string) (func(), error) {
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return nil, err
	}
	return filelock.Lock(filepath.Join(cfgDir, lockFile))
}

// ours reports whether process pid runs this same program: by its path on Linux, by
// its name elsewhere (best effort). A process that cannot be told is not ours.
func ours(pid int) bool {
	if pid == os.Getpid() || !pidAlive(pid) {
		return false
	}
	self, err := os.Executable()
	if err != nil {
		return false
	}
	if r, err := filepath.EvalSymlinks(self); err == nil {
		self = r
	}
	exe, err := exePath(pid)
	if err != nil || exe == "" {
		return false
	}
	if exactExe {
		return exe == self
	}
	trim := func(p string) string { return strings.TrimSuffix(strings.ToLower(filepath.Base(p)), ".exe") }
	return trim(exe) == trim(self)
}

// waitExit waits up to d for process pid to end, and reports whether it did.
func waitExit(pid int, d time.Duration) bool {
	for end := time.Now().Add(d); ; {
		if !pidAlive(pid) {
			return true
		}
		if time.Now().After(end) {
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// end stops process pid when it is ours: gently first, then by force.
func end(pid int) bool {
	if !ours(pid) {
		return false
	}
	_ = terminate(pid)
	if waitExit(pid, 5*time.Second) {
		return true
	}
	_ = kill(pid)
	return waitExit(pid, 5*time.Second)
}

// clearStale forgets a panel that does not answer: it ends its process when that is
// still this program (a server that hangs), and removes the info file.
func clearStale(cfgDir string, i Info) bool {
	ended := end(i.PID)
	_ = os.Remove(filepath.Join(cfgDir, InfoFile))
	return ended
}

// home is the folder the detached server runs in: one that is never removed while it
// runs (not the configuration folder, which `uninstall --config` removes).
func home() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return os.TempDir()
}

// Start returns the running panel, or starts one: the program self, in a detached
// process (`self panel serve`), and waits until it answers. started says which. The
// panel lock is held the whole time, so several at once start one server only, and
// all of them get its address.
func Start(cfgDir, self string) (info Info, started bool, err error) {
	unlock, err := lock(cfgDir)
	if err != nil {
		return Info{}, false, err
	}
	defer unlock()
	if i, ok := ReadInfo(cfgDir); ok {
		if Alive(i) {
			return i, false, nil
		}
		clearStale(cfgDir, i)
	}
	logPath := filepath.Join(cfgDir, logFile)
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return Info{}, false, err
	}
	defer log.Close()
	cmd := exec.Command(self, "panel", "serve")
	// Nothing of the caller's terminal or pipes: the assistant's shell tool waits until
	// every copy of its pipes is closed.
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, log, log
	cmd.Dir = home()
	detach.Cmd(cmd)
	if cmd, err = detach.Start(cmd); err != nil {
		return Info{}, false, err
	}
	pid := cmd.Process.Pid
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-exited:
			return Info{}, false, fmt.Errorf("the panel stopped as it started (%v); see %s", err, logPath)
		case <-time.After(50 * time.Millisecond):
		}
		if i, ok := ReadInfo(cfgDir); ok && i.PID == pid && Alive(i) {
			return i, true, nil
		}
	}
	return Info{}, false, fmt.Errorf("the panel did not answer within 10 seconds; see %s", logPath)
}

// Stop stops the running panel and waits until its process has ended. It asks the
// server first; a server that does not answer, but whose process is still this
// program, is ended by its process id. It reports false when none was running.
func Stop(cfgDir string) (bool, error) {
	unlock, err := lock(cfgDir)
	if err != nil {
		return false, err
	}
	defer unlock()
	i, ok := ReadInfo(cfgDir)
	if !ok {
		return false, nil
	}
	if !Alive(i) {
		return clearStale(cfgDir, i), nil
	}
	req, err := http.NewRequest(http.MethodPost, i.origin()+"/stop", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+i.Token)
	req.Header.Set("Origin", i.origin())
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("the panel did not stop: %s", resp.Status)
		}
	}
	if i.PID == os.Getpid() {
		// A server in this same process (the tests): it removes its file as it goes.
		for k := 0; k < 100; k++ {
			if _, ok := ReadInfo(cfgDir); !ok {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		return err == nil, err
	}
	if err != nil || !waitExit(i.PID, stopWait) {
		if !end(i.PID) && pidAlive(i.PID) {
			if err == nil {
				err = fmt.Errorf("the panel (process %d) did not stop", i.PID)
			}
			return false, err
		}
	}
	removeInfo(cfgDir, i.PID)
	return true, nil
}

// OpenBrowser opens url in the person's browser, without waiting for it. It fails
// where there is no desktop to open it on (an SSH session): then the URL is printed.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		// rundll32 takes the URL as one argument, untouched; `cmd /c start` would split it at &.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			return errors.New("no desktop session")
		}
		if _, err := exec.LookPath("xdg-open"); err != nil {
			return err
		}
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	detach.Cmd(cmd)
	cmd, err := detach.Start(cmd)
	if err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// sameHost reports whether host is the panel's own: 127.0.0.1 or localhost on its port.
// Anything else (a DNS name that points to 127.0.0.1, a rebinding attack) is refused.
func sameHost(host string, port int) bool {
	p := fmt.Sprintf(":%d", port)
	return strings.HasSuffix(host, p) && (strings.TrimSuffix(host, p) == "127.0.0.1" || strings.TrimSuffix(host, p) == "localhost")
}
