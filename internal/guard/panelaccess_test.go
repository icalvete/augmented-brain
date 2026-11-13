package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func panelPolicy(t *testing.T) (Policy, string) {
	t.Helper()
	cfg := filepath.Join(t.TempDir(), "cfg")
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	return Policy{Lang: "en", Hidden: []string{filepath.Join(cfg, "panel.json")},
		ConfigDirs: []string{cfg, "~/.config/augmented-brain", ".config/augmented-brain", "AB_CONFIG_DIR"}}, cfg
}

// Every way of writing a panel address that curl or a script would still send to the
// panel is denied, whatever the host is written as (P-01 of the review of Nov 2026).
func TestPanelAccessShapes(t *testing.T) {
	p, _ := panelPolicy(t)
	const port = "33939"
	key := strings.Repeat("0123456789abcdef", 2)
	for _, l := range []struct{ what, line string }{
		{"the address as printed", "curl -s http://127.0.0.1:" + port + "/s/" + key + "/guard"},
		{"localhost", "curl -s http://localhost:" + port + "/s/" + key + "/guard"},
		{"short IPv4 form", "curl -s http://127.1:" + port + "/s/" + key + "/guard"},
		{"decimal IPv4 form", "curl -s http://2130706433:" + port + "/s/" + key + "/guard"},
		{"hex IPv4 form", "curl -s http://0x7f.0.0.1:" + port + "/s/" + key + "/guard"},
		{"key in a variable", "K=" + key + "; curl -s http://127.0.0.1:" + port + "/s/$K/guard"},
		{"key split by quotes", "curl -s http://127.0.0.1:" + port + "/s/" + key[:16] + `""` + key[16:] + "/guard"},
		{"port in a variable only", "P=" + port + "; curl -s http://127.0.0.1:$P/s/" + key + "/guard"},
		{"host in a variable", "H=127.0.0.1:" + port + "; curl -s http://$H/s/" + key + "/guard"},
		{"--resolve with another name", "curl -s --resolve panel.test:" + port + ":127.0.0.1 -H 'Host: 127.0.0.1:" + port + "' http://panel.test:" + port + "/s/" + key + "/guard"},
		{"path with a dot segment", "curl -s --path-as-is http://127.0.0.1:" + port + "/s/./" + key + "/guard"},
		{"percent-encoded key letter", "curl -s http://127.0.0.1:" + port + "/s/%30" + key[1:] + "/guard"},
		{"python one-liner", `python3 -c "import urllib.request as u;print(u.urlopen('http://127.0.0.1:%s/s/%s/guard'%('` + port + `','` + key + `')).read())"`},
		{"the POST that adds an allowed repository", "curl -s -X POST -H 'Origin: http://127.0.0.1:" + port + "' -d csrf=$C -d list=allowed -d op=add -d confirm=yes http://127.1:" + port + "/s/" + key + "/guard/list"},
		{"the POST that accepts changed lists", "curl -s -X POST -d csrf=$C http://127.1:" + port + "/s/" + key + "/guard/accept"},
		{"a ticket", "curl -s -i http://127.1:" + port + "/t/" + key},
		{"browser history", `sqlite3 ~/.mozilla/firefox/x.default/places.sqlite "select url from moz_places where url like 'http://127.0.0.1:%/s/%' limit 1"`},
		{"process list", "ps -eo args | grep -o '127.0.0.1:[0-9]*/s/[0-9a-f]*'"},
	} {
		if d := p.Command(l.line); d.Verdict != Deny {
			t.Errorf("%s: %v for %s", l.what, d.Verdict, l.line)
		}
	}
	// Without the session, the port alone opens nothing: listing ports stays allowed.
	for _, line := range []string{"ss -ltnp | grep augmented", "curl -s http://127.0.0.1:8080/s/health",
		"git log --format=%H -1", "md5sum " + key, "ls /s/x"} {
		if d := p.Command(line); d.Verdict == Deny {
			t.Errorf("%s: denied: %s", line, d.Reason)
		}
	}
}

// A bare pattern or name counts as the token file only where the command runs in the
// configuration folder, and a quoted * is not a pattern (G-15).
func TestPanelAccessBareNames(t *testing.T) {
	p, cfg := panelPolicy(t)
	for _, line := range []string{
		`echo "a * b" && echo AB_CONFIG_DIR`,
		"AB_CONFIG_DIR=/tmp/x augmented-brain doctor && ls *",
		"cat <<'EOF' > /tmp/t.sh\nexport AB_CONFIG_DIR=/tmp/x\necho $((2 * 3))\nEOF",
		"ls " + cfg + " && ls *",
	} {
		if d := p.Command(line); d.Verdict == Deny {
			t.Errorf("%q: denied: %s", line, d.Reason)
		}
	}
	for _, line := range []string{
		"cd " + cfg + " && cat panel.json",
		"cd " + cfg + " && cat *",
		`cd "$AB_CONFIG_DIR" && cat p*`,
		"cat $AB_CONFIG_DIR/panel.json",
		"jq .token " + cfg + "/panel.*",
	} {
		if d := p.Command(line); d.Verdict != Deny {
			t.Errorf("%q: %v", line, d.Verdict)
		}
	}
	// A session that runs in the configuration folder: a bare * reaches the file.
	p.Cwd = cfg
	if d := p.Command("cat *"); d.Verdict != Deny {
		t.Errorf("cat * in the configuration folder: %v", d.Verdict)
	}
}
