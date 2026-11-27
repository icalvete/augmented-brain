package panel

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"rsc.io/qr"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/runlimit"
	"github.com/icalvete/augmented-brain/internal/telegramprog"
)

// The Telegram page connects the person's own Telegram account: the application from
// my.telegram.org, the QR code to scan, the two-step password, and disconnecting. It
// runs augmented brain's Telegram program (package telegramprog) and never holds the
// notes while it waits for it. Every change is refused through panel remote: the login
// is the whole account, and it is done only on this computer.

// tgAccount is who the session is logged in as, as the program says it.
type tgAccount struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

// tgLogin is the login in progress, if any: one at a time, kept by the panel's process.
type tgLogin struct {
	mu       sync.Mutex
	cancel   context.CancelFunc
	stdin    io.WriteCloser
	qr       string // the tg://login address to draw
	password bool   // the program waits for the two-step password
	account  *tgAccount
	err      string
	running  bool
}

var login tgLogin

// Variables, for tests.
var (
	tgPath    = telegramprog.Path
	tgInstall = telegramprog.Install
	tgCall    = telegramprog.Call
	tgStart   = startLogin
	// tgTimer and tgUntimer set and remove the timer that looks for replies; tests
	// never touch the real systemd or launchd.
	tgTimer   = remote.SetTelegramTimer
	tgUntimer = remote.UnsetTelegramTimer
)

// startLogin runs the program's login and follows what it says.
func startLogin(path string) error {
	login.mu.Lock()
	defer login.mu.Unlock()
	if login.running {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	cmd := runlimit.Command(ctx, path, "login", "--json")
	cmd.Env = os.Environ()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return err
	}
	login.cancel, login.stdin = cancel, stdin
	login.qr, login.password, login.account, login.err, login.running = "", false, nil, "", true
	go func() {
		defer cancel()
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			var ev struct {
				QR       string     `json:"qr"`
				Password bool       `json:"password"`
				Account  *tgAccount `json:"account"`
				Error    string     `json:"error"`
			}
			if json.Unmarshal(sc.Bytes(), &ev) != nil {
				continue
			}
			login.mu.Lock()
			switch {
			case ev.QR != "":
				login.qr = ev.QR
			case ev.Password:
				login.password = true
			case ev.Account != nil:
				login.account, login.qr, login.password = ev.Account, "", false
				if dir, err := config.Dir(); err == nil {
					_ = tgTimer(dir) // doctor says when it is missing
				}
			case ev.Error != "":
				login.err, login.qr, login.password = ev.Error, "", false
			}
			login.mu.Unlock()
		}
		_ = cmd.Wait()
		login.mu.Lock()
		login.running = false
		_ = login.stdin.Close()
		login.mu.Unlock()
	}()
	return nil
}

// qrSVG draws text as a QR code in SVG, dark on light, with its quiet zone.
func qrSVG(text string) template.HTML {
	code, err := qr.Encode(text, qr.L)
	if err != nil {
		return ""
	}
	n := code.Size + 8
	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="qr" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="264" height="264" shape-rendering="crispEdges" role="img" aria-label="QR"><rect width="%d" height="%d" fill="#fff"/><path fill="#000" d="`, n, n, n, n)
	for y := 0; y < code.Size; y++ {
		for x := 0; x < code.Size; x++ {
			if code.Black(x, y) {
				fmt.Fprintf(&b, "M%d %dh1v1h-1z", x+4, y+4)
			}
		}
	}
	b.WriteString(`"/></svg>`)
	return template.HTML(b.String())
}

// telegram is the page: what is there now, and the next step.
func (s *Server) telegram(v *view, r *http.Request) (string, page, error) {
	v.unlock() // the program reaches Telegram: seconds, not with the notes held
	d := struct {
		Supported, Installed, App, Connected, Waiting, Password, Remote bool
		Account                                                         *tgAccount
		QR                                                              template.HTML
		Err                                                             string
	}{Supported: telegramprog.Supported(), Remote: s.viaRemote(r)}
	path := ""
	if d.Supported {
		path = tgPath()
		d.Installed = path != ""
	}
	login.mu.Lock()
	d.Waiting = login.running && login.account == nil
	d.Password = login.running && login.password
	if login.qr != "" && d.Waiting && !d.Password {
		d.QR = qrSVG(login.qr)
	}
	d.Err = login.err
	login.mu.Unlock()
	if d.Installed && !d.Waiting {
		var st struct {
			Connected bool       `json:"connected"`
			App       bool       `json:"app"`
			Account   *tgAccount `json:"account"`
		}
		if err := tgCall(r.Context(), path, []string{"status"}, "", &st); err != nil && !errors.Is(err, telegramprog.ErrNotConnected) {
			d.Err = err.Error()
		}
		d.Connected, d.Account = st.Connected, st.Account
		d.App = st.App || st.Connected
	}
	p := v.frame("telegram", v.t("telegram"))
	p.D = d
	return "telegram", p, nil
}

func (s *Server) telegramInstall(v *view, _ *http.Request) (string, error) {
	v.unlock()
	if tgPath() != "" {
		return "/telegram", nil
	}
	if _, err := tgInstall(); err != nil {
		return "", errors.New(v.t("telegram.installfailed", err))
	}
	return "/telegram", nil
}

func (s *Server) telegramApp(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "api_id", "api_hash"); err != nil {
		return "", err
	}
	v.unlock()
	path := tgPath()
	if path == "" {
		return "", errors.New(v.t("telegram.notinstalled"))
	}
	in, _ := json.Marshal(map[string]string{"api_id": field(r, "api_id"), "api_hash": field(r, "api_hash")})
	var out struct{}
	if err := tgCall(r.Context(), path, []string{"app"}, string(in), &out); err != nil {
		return "", err
	}
	return "/telegram", tgStart(path)
}

func (s *Server) telegramLogin(v *view, _ *http.Request) (string, error) {
	v.unlock()
	path := tgPath()
	if path == "" {
		return "", errors.New(v.t("telegram.notinstalled"))
	}
	return "/telegram", tgStart(path)
}

func (s *Server) telegramPassword(v *view, r *http.Request) (string, error) {
	// Not field: a password may begin or end with a space.
	p := r.PostForm.Get("password")
	if p == "" || strings.ContainsAny(p, "\r\n") {
		return "", errors.New(v.t("err.required", v.t("f.password")))
	}
	login.mu.Lock()
	defer login.mu.Unlock()
	if !login.running || !login.password {
		return "/telegram", nil
	}
	login.password = false
	_, err := io.WriteString(login.stdin, p+"\n")
	return "/telegram", err
}

func (s *Server) telegramCancel(_ *view, _ *http.Request) (string, error) {
	login.mu.Lock()
	defer login.mu.Unlock()
	if login.cancel != nil {
		login.cancel()
	}
	login.qr, login.password, login.err = "", false, ""
	return "/telegram", nil
}

func (s *Server) telegramDisconnect(v *view, r *http.Request) (string, error) {
	if field(r, "confirm") != "yes" {
		return "", errors.New(v.t("err.confirm"))
	}
	v.unlock()
	path := tgPath()
	if path == "" {
		return "/telegram", nil
	}
	args := []string{"disconnect"}
	if field(r, "forget") == "yes" {
		args = append(args, "--forget")
	}
	var out struct{}
	if err := tgCall(r.Context(), path, args, "", &out); err != nil {
		return "", err
	}
	_, _ = tgUntimer()
	login.mu.Lock()
	login.account, login.err = nil, ""
	login.mu.Unlock()
	return "/telegram", nil
}
