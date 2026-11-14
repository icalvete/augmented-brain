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
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/detach"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/tailnet"
)

// UnpublishLeftover takes back a Tailscale address of cfgDir's panel that outlived it
// (a panel that crashed, or a reboot: Tailscale brings back what was published), so
// that whatever takes that local port later is never reachable from the network. Only
// the port the panel recorded (tailnet.Published) is ever taken back: with none, it
// does nothing and never runs tailscale; the user's own `tailscale serve`, or another
// configuration's panel, stays. Tests replace it.
var UnpublishLeftover = func(cfgDir string) error {
	port := tailnet.Published(cfgDir)
	if port == 0 {
		return nil
	}
	if err := tailnet.New().Unserve(port); err != nil && !errors.Is(err, tailnet.ErrMissing) {
		return err
	}
	tailnet.Forget(cfgDir, port)
	return nil
}

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
	// Remote is set while the panel is published to the user's Tailscale network.
	Remote *Remote `json:"remote,omitempty"`
	// RemotePort is the panel's second socket, the one Tailscale publishes, set with
	// Remote: what comes there is from the network, whatever its Host says. A panel of
	// an earlier program has none (it told the network apart by the Host header).
	RemotePort int `json:"remote_port,omitempty"`
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

// netSessionOf is sessionOf for the network's socket (panel remote): another label,
// so neither socket takes the other's session.
func netSessionOf(token string) string { return mac(token, "net session")[:32] }

// proof is what the server answers to a ping with challenge c: only a server that
// knows the token can give it. The "ping" prefix keeps it apart from sessionOf.
func proof(token, c string) string { return mac(token, "ping\x00"+c) }

// URL is the address of the panel's pages, under /s/<session>/. It is the key to the
// panel: the command never prints it, it prints a ticket (TicketURL).
func (i Info) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/s/%s/", i.Port, sessionOf(i.Token))
}

// TicketURL is the address that spends ticket t and opens the panel.
func (i Info) TicketURL(t string) string {
	return fmt.Sprintf("http://127.0.0.1:%d/t/%s", i.Port, t)
}

// RemoteTicketURL is TicketURL through the user's Tailscale network, or "".
func (i Info) RemoteTicketURL(t string) string {
	if i.Remote == nil || i.Remote.Host == "" {
		return ""
	}
	return fmt.Sprintf("https://%s/t/%s", i.Remote.Host, t)
}

// Ticket asks the running panel i for a ticket that opens page (a path under the
// session, such as "today?s=work", or ""); remote makes it wait longer, for a phone.
// An earlier panel, without tickets, answers 404: the caller starts a new one.
func Ticket(i Info, page string, remote bool) (string, error) {
	form := url.Values{"page": {page}}
	if remote {
		form.Set("remote", "1")
	}
	req, err := http.NewRequest(http.MethodPost, i.origin()+"/ticket", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+i.Token)
	req.Header.Set("Origin", i.origin())
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	t := strings.TrimSpace(string(body))
	if resp.StatusCode != http.StatusOK || len(t) != 32 {
		return "", fmt.Errorf("the panel answered %s to /ticket", resp.Status)
	}
	return t, nil
}

func (i Info) origin() string { return fmt.Sprintf("http://127.0.0.1:%d", i.Port) }

// RemoteURL is the address to open from another device of the user's Tailscale
// network, or "" when the panel is not published there.
func (i Info) RemoteURL() string {
	if i.Remote == nil || i.Remote.Host == "" {
		return ""
	}
	return fmt.Sprintf("https://%s/s/%s/", i.Remote.Host, netSessionOf(i.Token))
}

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

// RemoteWait is how long the command waits for the panel to go remote (POST /remote):
// the panel asks `tailscale status` then, which may take up to 20 seconds when
// Tailscale has just woken up or the machine is busy.
var RemoteWait = 25 * time.Second

// slowClient is client for POST /remote.
var slowClient = &http.Client{Timeout: RemoteWait, CheckRedirect: client.CheckRedirect}

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
	return resp.StatusCode == http.StatusOK &&
		hmac.Equal([]byte(strings.TrimSpace(string(body))), []byte(proof(i.Token, c)))
}

// lock takes the panel lock of cfgDir.
func lock(cfgDir string) (func(), error) {
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return nil, err
	}
	return filelock.Lock(filepath.Join(cfgDir, lockFile))
}

// ours reports whether process pid is the panel's server that started at started: it
// runs this same program (by its path on Linux, by its name elsewhere), as `panel
// serve` (Linux and macOS; Windows does not tell), and it started before the panel
// wrote started (a process id used again belongs to a process that started later). A
// process that cannot be told is not ours: never end what may be another session's
// MCP server, a backup or a hook.
func ours(pid int, started time.Time) bool {
	if pid == os.Getpid() || !pidAlive(pid) || started.IsZero() {
		return false
	}
	args, since, err := procInfo(pid)
	if err != nil || since.IsZero() || since.After(started.Add(startSlack)) {
		return false
	}
	if args != nil && !servesPanel(args) {
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

// startSlack is how much later than the panel's start a process may seem to have
// started, for the clocks' rounding: /proc counts in ticks, ps in seconds.
const startSlack = 2 * time.Second

// servesPanel reports whether a command line is the panel's server: `panel serve`.
func servesPanel(args []string) bool {
	for k := 1; k+1 < len(args); k++ {
		if args[k] == "panel" && args[k+1] == "serve" {
			return true
		}
	}
	return false
}

// end stops the panel's process of i when it is ours: gently first, then by force.
func end(i Info) bool {
	pid := i.PID
	if !ours(pid, i.Started) {
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
	ended := end(i)
	_ = os.Remove(filepath.Join(cfgDir, InfoFile))
	if i.Remote != nil {
		_ = UnpublishLeftover(cfgDir)
	}
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
	return StartMode(cfgDir, self, false)
}

// StartMode is Start; with remote, the panel also answers through the user's Tailscale
// network (`self panel serve --remote`). A panel already running only on 127.0.0.1 is
// stopped and started again in remote mode; a remote one serves a local request as is.
func StartMode(cfgDir, self string, remote bool) (info Info, started bool, err error) {
	unlock, err := lock(cfgDir)
	if err != nil {
		return Info{}, false, err
	}
	defer unlock()
	return startModeLocked(cfgDir, self, remote)
}

// How GoRemote got the panel into remote mode.
const (
	WasRemote = "was-remote" // it already was
	Switched  = "switched"   // the running panel switched, keeping its key and open pages
	Restarted = "restarted"  // it had to be started again (an earlier panel): open pages need reloading
	Started   = "started"    // none was running
)

// GoRemote has the panel answer through the user's Tailscale network too. A panel
// running only on 127.0.0.1 is asked to switch in place (POST /remote), so the pages
// open in the browser go on working; one that cannot (an earlier program) is stopped
// and started again in remote mode, as before. how says which.
func GoRemote(cfgDir, self string) (info Info, how string, err error) {
	unlock, err := lock(cfgDir)
	if err != nil {
		return Info{}, "", err
	}
	defer unlock()
	if i, ok := ReadInfo(cfgDir); ok && Alive(i) {
		// A panel of an earlier program, remote without a socket of its own, tells the
		// network apart by the Host header: it is started again, as this one.
		if i.Remote != nil && i.RemotePort != 0 {
			return i, WasRemote, nil
		}
		if i.Remote == nil {
			err := ask(i, "/remote")
			if err == nil {
				if now, ok := ReadInfo(cfgDir); ok && now.PID == i.PID && now.Remote != nil && now.RemotePort != 0 {
					return now, Switched, nil
				}
			} else if !earlier(err) {
				// Tailscale did not answer in time, or refused: the panel stays as it
				// is, and so do the pages open in the browser.
				return Info{}, "", err
			}
		}
		info, _, err := startModeLocked(cfgDir, self, true)
		return info, Restarted, err
	}
	info, _, err = startModeLocked(cfgDir, self, true)
	return info, Started, err
}

// GoLocal takes the network back from a panel that GoRemote switched, when publishing
// its address failed: it goes on, on 127.0.0.1 only.
func GoLocal(cfgDir string) error {
	unlock, err := lock(cfgDir)
	if err != nil {
		return err
	}
	defer unlock()
	i, ok := ReadInfo(cfgDir)
	if !ok || !Alive(i) {
		return nil
	}
	return ask(i, "/local")
}

// answerError is a panel's answer other than 2xx.
type answerError struct {
	path   string
	code   int
	status string
	text   string
}

func (e *answerError) Error() string {
	if e.text != "" {
		return fmt.Sprintf("the panel answered %s to %s: %s", e.status, e.path, e.text)
	}
	return fmt.Sprintf("the panel answered %s to %s", e.status, e.path)
}

// earlier reports whether err is the answer of a panel that cannot do what was asked
// (an earlier program: no such path, or no way to publish), and not a failure of this
// one (Tailscale slow or stopped, a timeout).
func earlier(err error) bool {
	var a *answerError
	if !errors.As(err, &a) {
		return false
	}
	return a.code == http.StatusNotFound || a.code == http.StatusMethodNotAllowed || a.code == http.StatusNotImplemented
}

// ask posts to the running panel as the command, with its token. Any answer but 2xx
// is an *answerError: an earlier panel without that path answers 404, or 403.
func ask(i Info, path string) error {
	req, err := http.NewRequest(http.MethodPost, i.origin()+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+i.Token)
	req.Header.Set("Origin", i.origin())
	c := client
	if path == "/remote" {
		c = slowClient
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return &answerError{path: path, code: resp.StatusCode, status: resp.Status, text: strings.TrimSpace(string(body))}
	}
	return nil
}

func startModeLocked(cfgDir, self string, remote bool) (info Info, started bool, err error) {
	if i, ok := ReadInfo(cfgDir); ok {
		// A remote panel of an earlier program has no socket of its own for the
		// network (RemotePort): it is stopped and started again, as this one.
		if Alive(i) && (!remote || (i.Remote != nil && i.RemotePort != 0)) {
			return i, false, nil
		}
		if Alive(i) {
			if err := stopRunning(cfgDir, i); err != nil {
				return Info{}, false, err
			}
		} else {
			clearStale(cfgDir, i)
		}
	}
	// No panel answers now: nothing of ours should be published.
	_ = UnpublishLeftover(cfgDir)
	logPath := filepath.Join(cfgDir, logFile)
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return Info{}, false, err
	}
	defer log.Close()
	args := []string{"panel", "serve"}
	if remote {
		args = append(args, "--remote")
	}
	cmd := exec.Command(self, args...)
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
	if err := stopRunning(cfgDir, i); err != nil {
		return false, err
	}
	return true, nil
}

// stopRunning asks the live panel i to stop and waits for its process to end; the
// caller holds the panel lock.
func stopRunning(cfgDir string, i Info) error {
	req, err := http.NewRequest(http.MethodPost, i.origin()+"/stop", nil)
	if err != nil {
		return err
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
		return err
	}
	if err != nil || !waitExit(i.PID, stopWait) {
		if !end(i) && pidAlive(i.PID) {
			if err == nil {
				err = fmt.Errorf("the panel (process %d) did not stop", i.PID)
			}
			return err
		}
	}
	removeInfo(cfgDir, i.PID)
	return nil
}

// OpenBrowser opens url in the person's browser, without waiting for it. It fails
// where there is no desktop to open it on (an SSH session): then the URL is printed.
// On Linux the browser gets, when it can read it, a file of cfgDir that sends it on to
// url (redirectFile), not url itself: the arguments of a process are there for every
// user of the machine to read, and url holds a ticket that opens the panel.
func OpenBrowser(cfgDir, url string) error {
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
		target := url
		if browserReadsFiles(xdgQuery) {
			if f, err := redirectFile(cfgDir, url, time.Now()); err == nil {
				target = f
			}
		}
		cmd = exec.Command("xdg-open", target)
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

// openDir is the folder of the redirect files, in the configuration folder.
const openDir = "state/panel-open"

// redirectFile writes a page, readable only by its owner, that sends the browser on to
// url, and returns its path. Files left by earlier runs are removed once their ticket
// has expired: by then the browser has read them, or never will.
func redirectFile(cfgDir, url string, now time.Time) (string, error) {
	dir := filepath.Join(cfgDir, filepath.FromSlash(openDir))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if old, err := os.ReadDir(dir); err == nil {
		for _, e := range old {
			if fi, err := e.Info(); err == nil && now.Sub(fi.ModTime()) > TicketTTL {
				_ = os.Remove(filepath.Join(dir, e.Name()))
			}
		}
	}
	r, err := random()
	if err != nil {
		return "", err
	}
	f := filepath.Join(dir, "open-"+r[:16]+".html")
	u := html.EscapeString(url)
	page := `<!doctype html><meta charset="utf-8"><meta http-equiv="refresh" content="0;url=` + u +
		`"><title>augmented brain</title><a href="` + u + `">augmented brain</a>` + "\n"
	if err := paths.WriteAtomicQuick(f, []byte(page), 0o600); err != nil {
		return "", err
	}
	return f, nil
}

// xdgQuery runs one of the desktop's queries (xdg-settings, xdg-mime) and returns its
// answer, "" when it fails or takes over 2 seconds.
func xdgQuery(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// browserReadsFiles says whether xdg-open of a redirect file reaches the default
// browser and the browser can read it: HTML files open in the default browser, and it
// is not a snap or a flatpak, whose sandbox keeps them from the configuration folder
// (hidden in the home folder). When in doubt, no: the browser then gets the address.
func browserReadsFiles(query func(string, ...string) string) bool {
	browser := query("xdg-settings", "get", "default-web-browser")
	if !strings.HasSuffix(browser, ".desktop") || query("xdg-mime", "query", "default", "text/html") != browser {
		return false
	}
	f := desktopFile(browser)
	if f == "" || strings.Contains(f, "/snapd/") || strings.Contains(f, "/flatpak/") {
		return false
	}
	b, err := os.ReadFile(f)
	if err != nil {
		return false
	}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "Exec=") && (strings.Contains(l, "/snap/") || strings.Contains(l, "flatpak")) {
			return false
		}
	}
	return true
}

// desktopFile finds the .desktop file name in the folders the desktop reads them from.
func desktopFile(name string) string {
	if strings.ContainsAny(name, `/\`) {
		return ""
	}
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		if h, err := os.UserHomeDir(); err == nil {
			data = filepath.Join(h, ".local", "share")
		}
	}
	dirs := []string{data}
	sys := os.Getenv("XDG_DATA_DIRS")
	if sys == "" {
		sys = "/usr/local/share:/usr/share"
	}
	dirs = append(dirs, filepath.SplitList(sys)...)
	for _, d := range dirs {
		if d == "" {
			continue
		}
		f := filepath.Join(d, "applications", name)
		if _, err := os.Stat(f); err == nil {
			if r, err := filepath.EvalSymlinks(f); err == nil {
				return r
			}
			return f
		}
	}
	return ""
}

// sameHost reports whether host is the panel's own: 127.0.0.1 or localhost on its port.
// Anything else (a DNS name that points to 127.0.0.1, a rebinding attack) is refused.
func sameHost(host string, port int) bool {
	p := fmt.Sprintf(":%d", port)
	return strings.HasSuffix(host, p) && (strings.TrimSuffix(host, p) == "127.0.0.1" || strings.TrimSuffix(host, p) == "localhost")
}
