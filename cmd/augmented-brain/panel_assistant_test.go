package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// anyTicket is an address that opens the panel: a ticket or a session, 32 hex digits.
var anyTicket = regexp.MustCompile(`/[st]/[0-9a-f]{32}`)

// Run by the assistant with no browser, `panel` makes no ticket, starts nothing and
// prints no address: it says to run it in a terminal of the user's own (S2).
func TestPanelByAssistantPrintsNoAddress(t *testing.T) {
	_, cfg := home(t)
	t.Setenv("CLAUDECODE", "1")
	for _, args := range [][]string{{"panel", "--no-browser"}, {"panel"}} {
		if len(args) == 1 {
			t.Setenv("AB_NO_BROWSER", "1")
		}
		var out, errOut bytes.Buffer
		if code := run(args, nil, &out, &errOut); code != 0 {
			t.Fatalf("%v: %d %q", args, code, errOut.String())
		}
		if anyTicket.MatchString(out.String()+errOut.String()) || strings.Contains(out.String(), "127.0.0.1") {
			t.Errorf("%v printed an address: %q", args, out.String())
		}
		if !strings.Contains(out.String(), "terminal tuyo") {
			t.Errorf("%v: no way on: %q", args, out.String())
		}
		if _, err := os.Stat(filepath.Join(cfg, "panel.json")); err == nil {
			t.Errorf("%v: a panel was started", args)
		}
	}
}

// Run by the assistant, `panel remote` publishes nothing and prints no address nor QR
// code, once Tailscale is ready: it says to run it in a terminal of the user's own.
func TestPanelRemoteByAssistantPrintsNoAddress(t *testing.T) {
	h, cfg := home(t)
	useFakeSystem(t, h, "linux")
	ready := `{"BackendState":"Running","Self":{"DNSName":"box.example.ts.net.","UserID":1},"User":{"1":{"LoginName":"lucia@example.com"}},"CertDomains":["box.example.ts.net"]}`
	calls := useFakeTailnet(t, ready, false)
	t.Setenv("CLAUDECODE", "1")
	var out, errOut bytes.Buffer
	if code := run([]string{"panel", "remote"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("%d %q", code, errOut.String())
	}
	if anyTicket.MatchString(out.String()) || strings.Contains(out.String(), "ts.net") || strings.Contains(out.String(), "██") {
		t.Errorf("printed an address: %q", out.String())
	}
	if !strings.Contains(out.String(), "augmented-brain panel remote") {
		t.Errorf("no way on: %q", out.String())
	}
	for _, c := range *calls {
		if strings.HasPrefix(c, "serve --") {
			t.Errorf("published: %q", c)
		}
	}
	if _, err := os.Stat(filepath.Join(cfg, "panel.json")); err == nil {
		t.Error("a panel was started")
	}
}
