// Package tailnet talks to Tailscale, for the panel's remote mode: whether it is
// installed and connected, this machine's name and owner, and publishing the panel to
// the user's own Tailscale network (`tailscale serve`) and taking it back.
//
// It only ever touches its own entry: the HTTPS port Port, and only while it points at
// the local port the panel recorded (Record). Whatever else the user publishes stays as it is (never `serve reset`).
// Everything goes through Tailnet.Run, so tests use a fake and never run tailscale.
package tailnet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Port is the HTTPS port of the tailnet address the panel is published on. Not 443, to
// leave the usual one to whatever the user publishes.
const Port = 8443

// Errors the commands explain in the user's language.
var (
	// ErrMissing: no tailscale program.
	ErrMissing = errors.New("tailscale is not installed")
	// ErrStopped: installed, but not connected to a network.
	ErrStopped = errors.New("tailscale is not connected")
	// ErrNoHTTPS: the network has no HTTPS certificates turned on.
	ErrNoHTTPS = errors.New("HTTPS certificates are off in this Tailscale network")
	// ErrOperator: this user may not change Tailscale's settings without root.
	ErrOperator = errors.New("this user may not publish with tailscale serve")
	// ErrPortTaken: the user already publishes something else on Port.
	ErrPortTaken = errors.New("the tailscale port is taken by something else")
)

// Runner runs a command and returns what it printed on the standard output; a failing
// command is an error that holds its standard error.
type Runner func(name string, args ...string) (string, error)

// Tailnet is Tailscale on this machine.
type Tailnet struct {
	Run      Runner
	LookPath func(string) (string, error)
	GOOS     string
	// Stat tells whether a file exists: where the macOS app keeps its program.
	Stat func(string) error
}

// New is the real Tailscale.
func New() Tailnet {
	return Tailnet{Run: realRun, LookPath: exec.LookPath, GOOS: runtime.GOOS,
		Stat: func(p string) error { _, err := os.Stat(p); return err }}
}

func realRun(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(errOut.String()); msg != "" {
			err = fmt.Errorf("%w: %s", err, msg)
		}
		return out.String(), err
	}
	return out.String(), nil
}

// macApp is where the Tailscale app for macOS keeps its command-line program when it
// is not in the PATH.
const macApp = "/Applications/Tailscale.app/Contents/MacOS/Tailscale"

// program is the tailscale program to run, or ErrMissing.
func (t Tailnet) program() (string, error) {
	if p, err := t.LookPath("tailscale"); err == nil {
		return p, nil
	}
	if t.GOOS == "darwin" && t.Stat != nil && t.Stat(macApp) == nil {
		return macApp, nil
	}
	return "", ErrMissing
}

// Status is what the panel needs to know of this machine in the network.
type Status struct {
	// Name is this machine's name in the network, such as box.example.ts.net.
	Name string
	// Owner is the login of the account the machine belongs to: the only one the
	// remote panel lets in.
	Owner string
	// HTTPS is whether the network gives this machine certificates.
	HTTPS bool
}

type statusJSON struct {
	BackendState string
	Self         struct {
		DNSName string
		UserID  int64
	}
	User        map[string]struct{ LoginName string }
	CertDomains []string
}

// Status reads `tailscale status --json`. It fails with ErrMissing or ErrStopped.
func (t Tailnet) Status() (Status, error) {
	prog, err := t.program()
	if err != nil {
		return Status{}, err
	}
	out, err := t.Run(prog, "status", "--json")
	var s statusJSON
	if jerr := json.Unmarshal([]byte(out), &s); jerr != nil {
		if err != nil {
			return Status{}, fmt.Errorf("%w: %v", ErrStopped, err)
		}
		return Status{}, fmt.Errorf("tailscale status: %v", jerr)
	}
	if s.BackendState != "Running" {
		return Status{}, ErrStopped
	}
	st := Status{Name: strings.TrimSuffix(s.Self.DNSName, ".")}
	if u, ok := s.User[strconv.FormatInt(s.Self.UserID, 10)]; ok {
		st.Owner = u.LoginName
	}
	st.HTTPS = st.Name != "" && slices.Contains(s.CertDomains, st.Name)
	if st.Name == "" || st.Owner == "" {
		return Status{}, fmt.Errorf("tailscale status: no name or owner for this machine")
	}
	return st, nil
}

// Address is the panel's address in the network, without a path.
func (s Status) Address() string { return fmt.Sprintf("https://%s:%d", s.Name, Port) }

// Host is the Host header requests through the network carry.
func (s Status) Host() string { return fmt.Sprintf("%s:%d", s.Name, Port) }

type serveJSON struct {
	TCP map[string]struct{ HTTPS bool }
	Web map[string]struct {
		Handlers map[string]struct{ Proxy string }
	}
}

// Served returns what our port publishes: "" when nothing, else the proxy target of
// its root ("http://127.0.0.1:1234"), or "other" when it is something that is not a
// proxy to 127.0.0.1. A proxy to 127.0.0.1 is not ours for that alone: only the one
// to the port the panel recorded (Published) is.
func (t Tailnet) Served() (string, error) {
	prog, err := t.program()
	if err != nil {
		return "", err
	}
	out, err := t.Run(prog, "serve", "status", "--json")
	if err != nil {
		return "", fmt.Errorf("tailscale serve status: %v", err)
	}
	var s serveJSON
	if strings.TrimSpace(out) != "" {
		if err := json.Unmarshal([]byte(out), &s); err != nil {
			return "", fmt.Errorf("tailscale serve status: %v", err)
		}
	}
	port := strconv.Itoa(Port)
	if _, ok := s.TCP[port]; !ok {
		return "", nil
	}
	for host, web := range s.Web {
		if !strings.HasSuffix(host, ":"+port) {
			continue
		}
		if len(web.Handlers) == 1 {
			if h, ok := web.Handlers["/"]; ok && strings.HasPrefix(h.Proxy, "http://127.0.0.1:") {
				return h.Proxy, nil
			}
		}
	}
	return "other", nil
}

// Target is what Port publishes when it publishes the local port local.
func Target(local int) string { return fmt.Sprintf("http://127.0.0.1:%d", local) }

// Ours reports whether cur (what Served returned) is a proxy to one of the ports.
func Ours(cur string, ports ...int) bool {
	for _, p := range ports {
		if p > 0 && cur == Target(p) {
			return true
		}
	}
	return false
}

// Serve publishes http://127.0.0.1:<local> on the network's HTTPS port Port, in the
// background. An entry of ours already there (to one of ours: a panel that ended
// without taking it back) is replaced; anything else on Port, a proxy to another local
// port included, is ErrPortTaken and stays as it is.
func (t Tailnet) Serve(local int, ours ...int) error {
	prog, err := t.program()
	if err != nil {
		return err
	}
	cur, err := t.Served()
	if err != nil {
		return err
	}
	target := Target(local)
	if cur == target {
		return nil
	}
	if cur != "" && !Ours(cur, ours...) {
		return ErrPortTaken
	}
	if cur != "" {
		if _, err := t.Run(prog, "serve", "--https="+strconv.Itoa(Port), "off"); err != nil {
			return explain(err)
		}
	}
	_, err = t.Run(prog, "serve", "--bg", "--https="+strconv.Itoa(Port), target)
	return explain(err)
}

// Unserve takes our entry back: only Port, and only when it is a proxy to one of ours.
// With no port of ours it does nothing, and never runs tailscale.
func (t Tailnet) Unserve(ours ...int) error {
	if !slices.ContainsFunc(ours, func(p int) bool { return p > 0 }) {
		return nil
	}
	prog, err := t.program()
	if err != nil {
		return err
	}
	cur, err := t.Served()
	if err != nil || !Ours(cur, ours...) {
		return err
	}
	_, err = t.Run(prog, "serve", "--https="+strconv.Itoa(Port), "off")
	return explain(err)
}

// explain turns tailscale's refusals into the errors the commands explain.
func explain(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "access denied") || strings.Contains(msg, "--operator"):
		return ErrOperator
	case strings.Contains(msg, "https") && (strings.Contains(msg, "not enabled") || strings.Contains(msg, "disabled")):
		return ErrNoHTTPS
	}
	return err
}

// usedFile, in the configuration's state folder, says `panel remote` has worked once on
// this computer: only then does doctor look at Tailscale.
const usedFile = "panel-remote"

// MarkUsed records that the remote panel worked on this computer.
func MarkUsed(cfgDir string) error {
	dir := filepath.Join(cfgDir, "state")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, usedFile), []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600)
}

// publishedFile, in the configuration's state folder, holds the local port this
// configuration's panel published on Port: what is there is ours only when it points
// at exactly that port. Another configuration on the same machine, or the user's own
// `tailscale serve` to some other local port, is never taken back.
const publishedFile = "panel-published"

// Record notes that this configuration publishes local port port. It is written before
// publishing, so a command that dies halfway still knows what it may take back.
func Record(cfgDir string, port int) error {
	dir := filepath.Join(cfgDir, "state")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, publishedFile), []byte(strconv.Itoa(port)+"\n"), 0o600)
}

// Published is the local port Record noted, or 0.
func Published(cfgDir string) int {
	data, err := os.ReadFile(filepath.Join(cfgDir, "state", publishedFile))
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || n <= 0 || n > 65535 {
		return 0
	}
	return n
}

// Forget removes the note of Record, when port is still the one noted (0: any).
func Forget(cfgDir string, port int) {
	if port == 0 || Published(cfgDir) == port {
		_ = os.Remove(filepath.Join(cfgDir, "state", publishedFile))
	}
}

// Used reports whether the remote panel ever worked on this computer.
func Used(cfgDir string) bool {
	_, err := os.Stat(filepath.Join(cfgDir, "state", usedFile))
	return err == nil
}
