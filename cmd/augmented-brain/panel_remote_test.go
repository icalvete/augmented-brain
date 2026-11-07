package main

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/tailnet"
	"rsc.io/qr"
)

// useFakeTailnet points `panel remote` at a Tailscale that answers status with status
// (missing: no program at all) and has nothing published.
func useFakeTailnet(t *testing.T, status string, missing bool) *[]string {
	t.Helper()
	var calls []string
	old := panelTailnet
	t.Cleanup(func() { panelTailnet = old })
	panelTailnet = func() tailnet.Tailnet {
		return tailnet.Tailnet{
			Run: func(name string, args ...string) (string, error) {
				call := strings.Join(args, " ")
				calls = append(calls, call)
				switch call {
				case "status --json":
					return status, nil
				case "serve status --json":
					return "{}", nil
				}
				return "", errors.New("unexpected " + call)
			},
			LookPath: func(string) (string, error) {
				if missing {
					return "", exec.ErrNotFound
				}
				return "/usr/bin/tailscale", nil
			},
			GOOS: "linux",
		}
	}
	return &calls
}

// What is missing is explained, in the configuration's language, and nothing starts.
func TestPanelRemoteExplains(t *testing.T) {
	noHTTPS := `{"BackendState":"Running","Self":{"DNSName":"box.example.ts.net.","UserID":1},"User":{"1":{"LoginName":"lucia@example.com"}}}`
	for _, c := range []struct {
		name, status string
		missing      bool
		want         string
	}{
		{"missing", "", true, "tailscale.com/download"},
		{"stopped", `{"BackendState":"NeedsLogin"}`, false, "tailscale up"},
		{"no https", noHTTPS, false, "Enable HTTPS"},
	} {
		h, _ := home(t)
		useFakeSystem(t, h, "linux")
		calls := useFakeTailnet(t, c.status, c.missing)
		var out, errOut bytes.Buffer
		if code := run([]string{"panel", "remote"}, nil, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), c.want) {
			t.Errorf("%s: %d %q", c.name, code, errOut.String())
		}
		if !strings.Contains(errOut.String(), "augmented-brain panel remote") || strings.Contains(errOut.String(), "Install it") {
			t.Errorf("%s: not in Spanish, or no way back: %q", c.name, errOut.String())
		}
		for _, call := range *calls {
			if strings.HasPrefix(call, "serve --") {
				t.Errorf("%s: published anyway: %q", c.name, call)
			}
		}
	}
}

func TestExplainTailnet(t *testing.T) {
	if got := explainTailnet("en", tailnet.ErrOperator); !strings.Contains(got, "sudo tailscale set --operator=") {
		t.Errorf("operator: %q", got)
	}
	if got := explainTailnet("en", tailnet.ErrPortTaken); !strings.Contains(got, "8443") {
		t.Errorf("port taken: %q", got)
	}
	if got := explainTailnet("en", errors.New("boom")); got != "boom" {
		t.Errorf("other: %q", got)
	}
}

// The QR code is square, with its quiet zone, and holds the address.
func TestQRText(t *testing.T) {
	address := "https://box.example.ts.net:8443/s/0123456789abcdef0123456789abcdef/today?s=work"
	text := qrText(address)
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	code, err := qr.Encode(address, qr.L)
	if err != nil {
		t.Fatal(err)
	}
	width := code.Size + 4
	if len(lines) != (width+1)/2 {
		t.Errorf("%d lines for %d modules", len(lines), width)
	}
	for i, l := range lines {
		if n := len([]rune(strings.TrimPrefix(l, "  "))); n != width {
			t.Errorf("line %d is %d wide, want %d", i, n, width)
		}
	}
	if !strings.HasPrefix(strings.TrimPrefix(lines[0], "  "), "██") {
		t.Error("no light quiet zone around the code")
	}
}
