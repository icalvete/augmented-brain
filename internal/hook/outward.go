package hook

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
)

// Outward commands change something outside the machine: a push, a merge, a release, a
// deploy, a change in the cloud. The guard's hook marks the session when it lets one
// through (markOutward), and the Stop hook then asks at once for a journal entry, instead
// of after RemindAfter. On 19-nov a session deployed to production and ended before it
// wrote that down; three days later the note still said it was to do.
//
// It is a plain reading of the command's words, not the guard's: it decides nothing,
// and a command it misses only gets the usual reminder.

// segments splits a command line at its operators, roughly: enough to find each
// command's first words.
var segments = regexp.MustCompile(`&&|\|\||[;|&\n()]|\$\(|` + "`")

// awsChange are the first words of an AWS CLI operation that changes something.
var awsChange = regexp.MustCompile(`^(create|update|delete|put|run|terminate|start|stop|reboot|deploy|invoke|modify|attach|detach|associate|disassociate|register|deregister|tag|untag|send|publish|restore|promote|reset|replace|set|enable|disable|import|execute|cancel)(-|$)`)

// scriptName are scripts whose name says they publish or deploy.
var scriptName = regexp.MustCompile(`(?i)(^|[-_.])(deploy|launch|release|publish|rollout|rollback|promote)([-_.]|$)`)

// outward returns the outward command in command, cut short, or "".
func outward(command string) string {
	for _, seg := range segments.Split(command, -1) {
		w := strings.Fields(seg)
		for len(w) > 0 && (strings.Contains(w[0], "=") && !strings.HasPrefix(w[0], "-") || w[0] == "sudo" || w[0] == "env" || w[0] == "exec" || w[0] == "time" || w[0] == "!") {
			w = w[1:] // VAR=value, sudo, env…: the command comes after
		}
		if len(w) == 0 {
			continue
		}
		if isOutward(w) {
			return short(strings.Join(w, " "))
		}
	}
	return ""
}

func isOutward(w []string) bool {
	name := filepath.Base(w[0])
	args := words(w[1:])
	switch name {
	case "git":
		return first(args, "push")
	case "gh":
		return pair(args, "pr", "merge", "create", "close") || pair(args, "release", "create", "delete", "upload", "edit") ||
			pair(args, "workflow", "run") || pair(args, "run", "rerun", "cancel") || pair(args, "repo", "create", "delete", "edit")
	case "glab":
		return pair(args, "mr", "merge", "create", "close") || pair(args, "release", "create") || pair(args, "ci", "run")
	case "aws":
		return len(args) >= 2 && (awsChange.MatchString(args[1]) || args[0] == "s3" && (args[1] == "cp" || args[1] == "sync" || args[1] == "rm" || args[1] == "mv" || args[1] == "mb" || args[1] == "rb"))
	case "terraform", "tofu":
		return first(args, "apply", "destroy", "import")
	case "kubectl":
		return first(args, "apply", "delete", "create", "replace", "patch", "scale", "rollout", "set", "edit", "drain", "cordon")
	case "helm":
		return first(args, "install", "upgrade", "uninstall", "rollback")
	case "docker", "podman":
		return first(args, "push")
	case "npm", "pnpm", "yarn", "cargo", "gem", "twine":
		return first(args, "publish", "push", "upload")
	case "curl":
		for i, a := range w[1:] {
			if (a == "-X" || a == "--request") && i+2 < len(w) {
				m := strings.ToUpper(w[i+2])
				return m == "POST" || m == "PUT" || m == "PATCH" || m == "DELETE"
			}
			if strings.HasPrefix(a, "-X") && len(a) > 2 {
				m := strings.ToUpper(a[2:])
				return m == "POST" || m == "PUT" || m == "PATCH" || m == "DELETE"
			}
		}
		return false
	}
	return scriptName.MatchString(strings.TrimSuffix(name, filepath.Ext(name)))
}

// words are the arguments that are not options, with the value of a few options that
// take one (git -C dir, aws --profile p…) left out.
func words(args []string) []string {
	var out []string
	skip := false
	for _, a := range args {
		switch {
		case skip:
			skip = false
		case a == "-C" || a == "-c" || a == "--profile" || a == "--region" || a == "-R" || a == "--repo" || a == "--git-dir" || a == "--work-tree":
			skip = true
		case strings.HasPrefix(a, "-"):
		default:
			out = append(out, a)
		}
	}
	return out
}

func first(args []string, any ...string) bool {
	if len(args) == 0 {
		return false
	}
	for _, a := range any {
		if args[0] == a {
			return true
		}
	}
	return false
}

func pair(args []string, group string, any ...string) bool {
	return len(args) >= 2 && args[0] == group && first(args[1:], any...)
}

// short cuts a command to what a reminder needs to name it.
func short(s string) string {
	const max = 80
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

// markOutward records, for the Stop hook, that this session is running an outward
// command. Errors are dropped: a reminder is never worth failing the guard for.
func markOutward(cfgDir string, i input, command string) {
	what := outward(command)
	if what == "" || i.SessionID == "" {
		return
	}
	c, err := config.Load(cfgDir)
	if err != nil {
		return
	}
	s := c.SpaceFor(i.Cwd)
	if s == nil {
		return
	}
	_ = state.Acted(cfgDir, s.Name, state.Who{ID: i.SessionID}, what, Now())
}

// actedSince says whether st ran an outward command after its task was last written
// and after its last reminder.
func actedSince(st state.Session) bool {
	return !st.Acted.IsZero() && st.Acted.After(st.Since) && st.Acted.After(st.Reminded)
}
