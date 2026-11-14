package remote

import "strings"

// Claude's daemon starts with the environment of whoever starts it, and every session
// it starts later, of any task and any space, gets that environment (launch.go). So it
// gets a list of what passes, not of what is taken out: whatever a terminal or a
// session happens to export (a token, an AB_* switch of a detached copy, a test's
// AB_CONFIG_DIR) stays with whoever exported it (R-07).

// daemonNames are the variables that pass, by name. Compared in upper case: Windows
// does not tell case apart in names, and on Unix nothing here has a lower-case twin
// worth keeping out.
var daemonNames = map[string]bool{
	// Who and where.
	"HOME": true, "USER": true, "LOGNAME": true, "SHELL": true, "PATH": true, "TMPDIR": true,
	// Language and time.
	"LANG": true, "LANGUAGE": true, "TZ": true,
	// The terminal and the desktop session.
	"TERM": true, "COLORTERM": true, "DISPLAY": true, "WAYLAND_DISPLAY": true, "XAUTHORITY": true,
	"DBUS_SESSION_BUS_ADDRESS": true,
	// Keys kept by an agent: git over ssh, signed commits. The agent holds the key, not
	// the variable.
	"SSH_AUTH_SOCK": true, "GNUPGHOME": true,
	// Proxies and certificates of a network that needs them.
	"HTTP_PROXY": true, "HTTPS_PROXY": true, "NO_PROXY": true, "ALL_PROXY": true,
	"SSL_CERT_FILE": true, "SSL_CERT_DIR": true, "NODE_EXTRA_CA_CERTS": true, "REQUESTS_CA_BUNDLE": true,
	// Windows: what a program needs to find the system and the user's folders.
	"SYSTEMROOT": true, "WINDIR": true, "SYSTEMDRIVE": true, "COMSPEC": true, "PATHEXT": true,
	"USERPROFILE": true, "USERNAME": true, "USERDOMAIN": true, "HOMEDRIVE": true, "HOMEPATH": true,
	"APPDATA": true, "LOCALAPPDATA": true, "PROGRAMDATA": true, "PUBLIC": true,
	"PROGRAMFILES": true, "PROGRAMFILES(X86)": true, "PROGRAMW6432": true,
	"COMMONPROGRAMFILES": true, "COMMONPROGRAMFILES(X86)": true, "COMMONPROGRAMW6432": true,
	"TEMP": true, "TMP": true, "OS": true, "PROCESSOR_ARCHITECTURE": true, "NUMBER_OF_PROCESSORS": true,
}

// daemonPrefixes pass by the start of their name: the locale, the desktop's folders,
// and Claude's own settings, which say how the daemon itself logs in and where it keeps
// its files (CLAUDE_CONFIG_DIR, ANTHROPIC_API_KEY, a Bedrock or Vertex setup): without
// them the daemon of someone who logs in that way could not start a session.
var daemonPrefixes = []string{"LC_", "XDG_", "CLAUDE_", "ANTHROPIC_"}

// daemonEnv is the part of env (as os.Environ gives it) that Claude's daemon may get.
func daemonEnv(env []string) []string {
	out := []string{}
	for _, kv := range env {
		name, _, ok := strings.Cut(kv, "=")
		if !ok || name == "" {
			continue
		}
		if daemonPasses(strings.ToUpper(name)) {
			out = append(out, kv)
		}
	}
	return out
}

func daemonPasses(name string) bool {
	if daemonNames[name] {
		return true
	}
	for _, p := range daemonPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
