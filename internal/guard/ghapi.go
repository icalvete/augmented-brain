package guard

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// What gh and glab can do through their API, or through an alias, that their own
// commands for it would show: merging, committing, publishing, starting a run again.

// apiDenied are the paths where a write merges, publishes, moves a branch or commits a
// file, in GitHub's API and in GitLab's.
var apiDenied = []string{"/merge", "/releases", "/git/refs", "/contents/", "/repository/commits", "/repository/files"}

// graphqlDenied are the GraphQL mutations (lower case) that do what apiDenied does.
var graphqlDenied = []string{"mergepullrequest", "enablepullrequestautomerge", "mergebranch",
	"updateref", "createref", "deleteref", "createcommitonbranch", "mergerequestaccept",
	"mergerequestsetautomerge", "commitcreate", "releasecreate", "releaseupdate"}

// api is the answer to gh api or glab api with path (lower case).
func (p Policy) api(name string, args []string, path string) Decision {
	if path == "graphql" {
		return p.graphql(name, args)
	}
	method := strings.ToUpper(optValue(args, "-X", "--method"))
	writes := method != "" && method != "GET" || slices.ContainsFunc(args, func(a string) bool {
		return a == "-f" || a == "-F" || a == "--field" || a == "--raw-field" || a == "--input"
	})
	if !writes {
		return allow
	}
	if slices.ContainsFunc(apiDenied, func(s string) bool { return strings.Contains(path, s) }) {
		return Decision{Deny, i18n.T(p.Lang, "guard.api", name)}
	}
	if _, rest, ok := strings.Cut(path, "/actions/workflows/"); name == "gh" && ok {
		if wf, ok := strings.CutSuffix(strings.TrimRight(rest, "/"), "/dispatches"); ok {
			return p.dispatch(wf)
		}
	}
	if strings.Contains(path, "/rerun") {
		return Decision{Ask, i18n.T(p.Lang, "guard.gh.rerun", name+" api "+path)}
	}
	if strings.HasSuffix(strings.TrimRight(path, "/"), "/dispatches") {
		return Decision{Ask, i18n.T(p.Lang, "guard.api.dispatch", name+" api "+path)}
	}
	return allow
}

// graphql is the answer to gh api graphql or glab api graphql: a query only reads; a
// mutation that merges, commits or publishes is the user's; any other mutation, or a
// query the guard cannot read (from a file, a variable), the user confirms.
func (p Policy) graphql(name string, args []string) Decision {
	query, unknown := "", slices.Contains(args, "--input")
	for i := 0; i < len(args); i++ {
		a, v := args[i], ""
		switch {
		case slices.Contains([]string{"-f", "-F", "--field", "--raw-field"}, a) && i+1 < len(args):
			i++
			v = args[i]
		case strings.HasPrefix(a, "--field="), strings.HasPrefix(a, "--raw-field="):
			_, v, _ = strings.Cut(a, "=")
		case (strings.HasPrefix(a, "-f") || strings.HasPrefix(a, "-F")) && len(a) > 2 && !strings.HasPrefix(a, "--"):
			v = a[2:]
		default:
			continue
		}
		if k, q, ok := strings.Cut(v, "="); ok && k == "query" {
			query = q
		}
	}
	if query == "" && !unknown {
		return allow // nothing to run: gh says so
	}
	if unknown || strings.HasPrefix(query, "@") || strings.HasPrefix(query, "$") || strings.ContainsRune(query, subst) {
		return Decision{Ask, i18n.T(p.Lang, "guard.api.graphql", name)}
	}
	q := strings.ToLower(query)
	if !strings.Contains(q, "mutation") {
		return allow
	}
	if slices.ContainsFunc(graphqlDenied, func(m string) bool { return strings.Contains(q, m) }) {
		return Decision{Deny, i18n.T(p.Lang, "guard.api", name)}
	}
	return Decision{Ask, i18n.T(p.Lang, "guard.api.graphql", name)}
}

// ghCommands are the commands of gh and glab themselves: an alias with one of these
// names is never used, so it is not looked up.
var ghCommands = map[string]bool{}

func init() {
	for _, c := range strings.Fields(`agent-task alias api attestation auth browse cache changelog
		check-update ci cluster codespace completion config copilot deploy-key duo extension gist
		gpg-key help incident issue iteration job label milestone mr opentofu org pr preview
		project release repo ruleset run schedule search secret securefile snippet ssh-key stack
		status token user variable version work-items workflow`) {
		ghCommands[c] = true
	}
}

// ghAlias returns what name stands for among the aliases gh or glab keeps (gh in the
// aliases section of its config.yml, glab in aliases.yml), read without running them.
func (p Policy) ghAlias(prog string, env map[string]string, name string) (string, bool) {
	get := func(k string) string {
		if v, ok := env[k]; ok {
			return v
		}
		return p.Env[k]
	}
	home, _ := paths.Home()
	var file string
	section := prog == "gh"
	switch {
	case prog == "gh" && get("GH_CONFIG_DIR") != "":
		file = filepath.Join(get("GH_CONFIG_DIR"), "config.yml")
	case prog == "gh" && get("XDG_CONFIG_HOME") != "":
		file = filepath.Join(get("XDG_CONFIG_HOME"), "gh", "config.yml")
	case prog == "gh" && runtime.GOOS == "windows" && get("AppData") != "":
		file = filepath.Join(get("AppData"), "GitHub CLI", "config.yml")
	case prog == "gh":
		file = filepath.Join(home, ".config", "gh", "config.yml")
	case get("GLAB_CONFIG_DIR") != "":
		file = filepath.Join(get("GLAB_CONFIG_DIR"), "aliases.yml")
	case get("XDG_CONFIG_HOME") != "":
		file = filepath.Join(get("XDG_CONFIG_HOME"), "glab-cli", "aliases.yml")
	default:
		file = filepath.Join(home, ".config", "glab-cli", "aliases.yml")
	}
	f, err := os.Open(file)
	if err != nil {
		return "", false
	}
	defer f.Close()
	return yamlAlias(io.LimitReader(f, maxGitConfig), section, name)
}

// yamlAlias reads name from a YAML map of aliases: the whole file, or its aliases:
// section. A small reader of the plain form gh and glab write, not of all YAML.
func yamlAlias(r io.Reader, section bool, name string) (string, bool) {
	in := !section
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), maxGitConfig)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t\r")
		if line == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		indented := line[0] == ' ' || line[0] == '\t'
		if section && !indented {
			in = line == "aliases:"
			continue
		}
		if !in {
			continue
		}
		k, v, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && yamlScalar(k) == name {
			return yamlScalar(v), true
		}
	}
	return "", false
}

// yamlScalar is a YAML scalar without its quotes.
func yamlScalar(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
	}
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return strings.ReplaceAll(s[1:len(s)-1], "''", "'")
	}
	return s
}
