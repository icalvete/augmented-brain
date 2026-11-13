package tailnet

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// fake is a Tailscale that answers from memory: status and the serve config.
type fake struct {
	status string // what `status --json` prints
	served string // what `serve status --json` prints
	deny   bool   // serve refused as a user without operator
	calls  []string
}

const running = `{"BackendState":"Running","Self":{"DNSName":"box.example.ts.net.","UserID":42},
"User":{"42":{"LoginName":"lucia@example.com"},"7":{"LoginName":"other@example.com"}},
"CertDomains":["box.example.ts.net"]}`

func ours(port string) string {
	return `{"TCP":{"8443":{"HTTPS":true}},"Web":{"box.example.ts.net:8443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:` + port + `"}}}}}`
}

func (f *fake) tn() Tailnet {
	return Tailnet{
		Run: func(name string, args ...string) (string, error) {
			call := strings.Join(args, " ")
			f.calls = append(f.calls, call)
			switch {
			case call == "status --json":
				return f.status, nil
			case call == "serve status --json":
				return f.served, nil
			case strings.HasPrefix(call, "serve --bg"):
				if f.deny {
					return "", errors.New("exit status 1: sending serve config: Access denied: serve config denied")
				}
				f.served = ours(strings.TrimPrefix(args[len(args)-1], "http://127.0.0.1:"))
				return "", nil
			case strings.HasSuffix(call, " off"):
				f.served = "{}"
				return "", nil
			}
			return "", errors.New("unexpected: " + call)
		},
		LookPath: func(string) (string, error) { return "/usr/bin/tailscale", nil },
		GOOS:     "linux",
	}
}

func TestStatus(t *testing.T) {
	f := &fake{status: running}
	st, err := f.tn().Status()
	if err != nil {
		t.Fatal(err)
	}
	if st.Name != "box.example.ts.net" || st.Owner != "lucia@example.com" || !st.HTTPS {
		t.Errorf("status = %+v", st)
	}
	if st.Address() != "https://box.example.ts.net:8443" || st.Host() != "box.example.ts.net:8443" {
		t.Errorf("address %q host %q", st.Address(), st.Host())
	}
	f.status = strings.Replace(running, `"CertDomains":["box.example.ts.net"]`, `"CertDomains":null`, 1)
	if st, _ = f.tn().Status(); st.HTTPS {
		t.Error("no certificate domains: no HTTPS")
	}
	f.status = `{"BackendState":"Stopped"}`
	if _, err := f.tn().Status(); !errors.Is(err, ErrStopped) {
		t.Errorf("stopped: %v", err)
	}
}

func TestMissing(t *testing.T) {
	tn := (&fake{}).tn()
	tn.LookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	if _, err := tn.Status(); !errors.Is(err, ErrMissing) {
		t.Errorf("linux without tailscale: %v", err)
	}
	// On macOS the app keeps its program out of the PATH.
	tn.GOOS = "darwin"
	tn.Stat = func(p string) error {
		if p == macApp {
			return nil
		}
		return errors.New("no")
	}
	if p, err := tn.program(); err != nil || p != macApp {
		t.Errorf("macOS app: %q %v", p, err)
	}
}

func TestServeAndUnserve(t *testing.T) {
	f := &fake{status: running, served: "{}"}
	tn := f.tn()
	if err := tn.Serve(1234); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(f.calls, "serve --bg --https=8443 http://127.0.0.1:1234") {
		t.Errorf("calls = %q", f.calls)
	}
	if cur, _ := tn.Served(); cur != "http://127.0.0.1:1234" {
		t.Errorf("served = %q", cur)
	}
	// The same again: nothing to do.
	f.calls = nil
	if err := tn.Serve(1234); err != nil || len(f.calls) != 1 {
		t.Errorf("again: %v %q", err, f.calls)
	}
	// A leftover of ours on another port of 127.0.0.1 is replaced.
	f.calls = nil
	if err := tn.Serve(999, 1234); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(f.calls, "serve --https=8443 off") || !slices.Contains(f.calls, "serve --bg --https=8443 http://127.0.0.1:999") {
		t.Errorf("replace: %q", f.calls)
	}
	// Not ours: another port is left alone.
	if err := tn.Unserve(1234); err != nil {
		t.Fatal(err)
	}
	if cur, _ := tn.Served(); cur != "http://127.0.0.1:999" {
		t.Errorf("unserve of another port took it: %q", cur)
	}
	if err := tn.Unserve(999); err != nil {
		t.Fatal(err)
	}
	if cur, _ := tn.Served(); cur != "" {
		t.Errorf("after unserve: %q", cur)
	}
	// Nothing of ours: Unserve does nothing.
	f.calls = nil
	if err := tn.Unserve(999); err != nil || slices.ContainsFunc(f.calls, func(c string) bool { return strings.HasSuffix(c, "off") }) {
		t.Errorf("unserve with nothing: %v %q", err, f.calls)
	}
}

// What the user publishes on our port is never touched.
func TestOthersAreLeftAlone(t *testing.T) {
	for _, served := range []string{
		`{"TCP":{"8443":{"HTTPS":true}},"Web":{"box.example.ts.net:8443":{"Handlers":{"/":{"Path":"/srv/site"}}}}}`,
		`{"TCP":{"8443":{"HTTPS":true}},"Web":{"box.example.ts.net:8443":{"Handlers":{"/":{"Proxy":"http://10.0.0.5:80"}}}}}`,
		`{"TCP":{"8443":{"HTTPS":true}},"Web":{"box.example.ts.net:8443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:1"},"/x":{"Proxy":"http://127.0.0.1:2"}}}}}`,
	} {
		f := &fake{status: running, served: served}
		tn := f.tn()
		if err := tn.Serve(1234); !errors.Is(err, ErrPortTaken) {
			t.Errorf("%s: serve = %v", served, err)
		}
		if err := tn.Unserve(1, 1234); err != nil {
			t.Errorf("unserve: %v", err)
		}
		for _, c := range f.calls {
			if strings.HasSuffix(c, "off") || strings.HasPrefix(c, "serve --bg") {
				t.Errorf("touched someone else's entry: %q", c)
			}
		}
	}
	// Another port of the user's stays too.
	f := &fake{status: running, served: `{"TCP":{"443":{"HTTPS":true}},"Web":{"box.example.ts.net:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:3000"}}}}}`}
	if cur, err := f.tn().Served(); err != nil || cur != "" {
		t.Errorf("port 443 is not ours: %q %v", cur, err)
	}
}

// The user's own `tailscale serve` to another local port (a development server) is not
// ours: never replaced, never taken back (P-03). With no port of ours, Unserve does not
// even run tailscale.
func TestALocalProxyOfTheUser(t *testing.T) {
	f := &fake{status: running, served: ours("3000")}
	tn := f.tn()
	if err := tn.Unserve(); err != nil || len(f.calls) != 0 {
		t.Errorf("no port of ours: %v %q", err, f.calls)
	}
	if err := tn.Unserve(1234); err != nil {
		t.Fatal(err)
	}
	if err := tn.Serve(1234); !errors.Is(err, ErrPortTaken) {
		t.Errorf("serve over the user's proxy: %v", err)
	}
	for _, c := range f.calls {
		if strings.HasSuffix(c, "off") || strings.HasPrefix(c, "serve --bg") {
			t.Errorf("touched the user's proxy: %q", c)
		}
	}
	if cur, _ := tn.Served(); cur != "http://127.0.0.1:3000" {
		t.Errorf("served = %q", cur)
	}
}

func TestRecord(t *testing.T) {
	dir := t.TempDir()
	if Published(dir) != 0 {
		t.Fatal("nothing recorded yet")
	}
	if err := Record(dir, 4321); err != nil {
		t.Fatal(err)
	}
	if Published(dir) != 4321 {
		t.Fatalf("published = %d", Published(dir))
	}
	Forget(dir, 1111) // another port: kept
	if Published(dir) != 4321 {
		t.Fatal("forgot a port it was not asked to")
	}
	Forget(dir, 4321)
	if Published(dir) != 0 {
		t.Fatal("not forgotten")
	}
}

func TestOperator(t *testing.T) {
	f := &fake{status: running, served: "{}", deny: true}
	if err := f.tn().Serve(1234); !errors.Is(err, ErrOperator) {
		t.Errorf("no operator: %v", err)
	}
}
