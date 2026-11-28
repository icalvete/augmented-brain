package panel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/telegramprog"
)

// The Telegram page takes a person from nothing to connected: download the program,
// save the application, scan the QR code, type the two-step password, disconnect.
// Through panel remote it only shows; every change is refused.
func TestTelegramPage(t *testing.T) {
	if !telegramprog.Supported() {
		t.Skip("Telegram is not supported here")
	}
	s, _ := fixture(t, "en")
	oldPath, oldInstall, oldCall, oldStart := tgPath, tgInstall, tgCall, tgStart
	t.Cleanup(func() {
		tgPath, tgInstall, tgCall, tgStart = oldPath, oldInstall, oldCall, oldStart
		login = tgLogin{}
	})
	installed, app, connected, started := false, false, false, 0
	var calls [][]string
	tgPath = func() string {
		if installed {
			return "/fake/augmented-brain-telegram"
		}
		return ""
	}
	tgInstall = func() (string, error) { installed = true; return tgPath(), nil }
	tgStart = func(string) error { started++; return nil }
	tgCall = func(_ context.Context, _ string, args []string, stdin string, v any) error {
		calls = append(calls, args)
		switch args[0] {
		case "status":
			ans := map[string]any{"connected": connected, "app": app}
			if connected {
				ans["account"] = map[string]any{"id": 1, "name": "Lucía Pérez", "username": "lucia"}
			}
			b, _ := json.Marshal(ans)
			return json.Unmarshal(b, v)
		case "app":
			if !strings.Contains(stdin, `"api_id":"123"`) {
				t.Errorf("app stdin %s", stdin)
			}
			app = true
		case "disconnect":
			connected = false
		}
		return nil
	}

	page := func() string { return s.get(t, "/telegram?s=work").Body.String() }
	// A session in the keychain without the program: connected, only the program is
	// missing, and the page says so.
	tgSaved = func() bool { return true }
	if p := page(); !strings.Contains(p, "only augmented brain") || !strings.Contains(p, "Download the Telegram program") {
		t.Fatalf("saved session, no program:\n%.2000s", p)
	}
	tgSaved = func() bool { return false }
	if p := page(); !strings.Contains(p, "Download the Telegram program") || !strings.Contains(p, ">Telegram<") {
		t.Fatalf("not installed:\n%.2000s", p)
	}
	saved(t, s.post(t, "/telegram/install", url.Values{"s": {"work"}}), "install")
	if p := page(); !strings.Contains(p, `name="api_hash"`) || !strings.Contains(p, "my.telegram.org") {
		t.Fatalf("no application:\n%.2000s", p)
	}
	if w := s.post(t, "/telegram/app", url.Values{"s": {"work"}, "api_id": {"123"}}); w.Code != http.StatusBadRequest {
		t.Errorf("app without its hash: %d", w.Code)
	}
	saved(t, s.post(t, "/telegram/app", url.Values{"s": {"work"}, "api_id": {"123"}, "api_hash": {"0123456789abcdef0123456789abcdef"}}), "app")
	if started != 1 {
		t.Errorf("login started %d times", started)
	}

	// The program is running its login: the QR code, then the password.
	login.mu.Lock()
	login.running, login.qr = true, "tg://login?token=abc"
	login.mu.Unlock()
	if p := page(); !strings.Contains(p, `<svg class="qr"`) || !strings.Contains(p, `data-reload="3"`) {
		t.Fatalf("waiting for the scan:\n%.2000s", p)
	}
	login.mu.Lock()
	login.password = true
	login.mu.Unlock()
	if p := page(); !strings.Contains(p, `type="password" name="password"`) || strings.Contains(p, `<svg class="qr"`) {
		t.Fatalf("asking the password:\n%.2000s", p)
	}
	login.mu.Lock()
	login.running, login.password, login.qr = false, false, ""
	login.mu.Unlock()

	connected = true
	if p := page(); !strings.Contains(p, "Lucía Pérez (@lucia)") || !strings.Contains(p, "/telegram/disconnect") {
		t.Fatalf("connected:\n%.2000s", p)
	}

	// Through panel remote: it shows, and refuses every change.
	const rhost, owner = "box.example.ts.net:8443", "lucia@example.com"
	s.remote.Store(&Remote{Host: rhost, Owner: owner})
	body := s.do(t, req{path: s.netBase() + "/telegram?s=work", host: rhost, owner: owner}).Body.String()
	if !strings.Contains(body, "Lucía Pérez") || strings.Contains(body, "/telegram/disconnect\"") {
		t.Errorf("remote page:\n%.2000s", body)
	}
	before := len(calls)
	for _, path := range []string{"/telegram/install", "/telegram/app", "/telegram/login", "/telegram/password", "/telegram/cancel", "/telegram/disconnect"} {
		form := url.Values{"s": {"work"}, "csrf": {s.netCSRF}, "confirm": {"yes"}, "api_id": {"1"}, "api_hash": {"0123456789abcdef0123456789abcdef"}, "password": {"x"}}
		w := s.do(t, req{method: http.MethodPost, path: s.netBase() + path, host: rhost, owner: owner, origin: "https://" + rhost, form: form})
		if w.Code == http.StatusSeeOther {
			t.Errorf("%s went through panel remote", path)
		}
	}
	for _, c := range calls[before:] {
		if c[0] != "status" {
			t.Errorf("remote ran %v", c)
		}
	}

	if w := s.post(t, "/telegram/disconnect", url.Values{"s": {"work"}}); w.Code != http.StatusBadRequest {
		t.Errorf("disconnected without the tick: %d", w.Code)
	}
	saved(t, s.post(t, "/telegram/disconnect", url.Values{"s": {"work"}, "confirm": {"yes"}, "forget": {"yes"}}), "disconnect")
	if !slices.ContainsFunc(calls, func(c []string) bool { return slices.Equal(c, []string{"disconnect", "--forget"}) }) {
		t.Errorf("calls %v", calls)
	}
}

func TestQRSVG(t *testing.T) {
	svg := string(qrSVG("tg://login?token=abc"))
	if !strings.HasPrefix(svg, `<svg class="qr"`) || !strings.Contains(svg, "M4 4h1v1h-1z") {
		t.Errorf("svg: %.200s", svg)
	}
}
