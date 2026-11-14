package panel

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/confighistory"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/state"
)

// The Guard page shows every list the guard answers to and changes them, in the panel
// and nowhere else: the guard denies the assistant the panel's token and pages (see
// guard.ReadFile), so these lists are the user's. Changing one writes the file with the
// user's comments kept, and seals the lists (state.WriteSeal) so the guard knows the
// user saw them. If the lists were already different from the sealed ones, something
// else changed them: the page says so and seals only when the user accepts.

type guardRoot struct {
	Entry, Host, Until string
	Expired            bool
}

// guardRepo is a repository of one of the lists: Space names the space whose list it is
// in (empty for the untouchable list, which is the machine's), and Missing says its
// folder is not on this computer, most often a name typed wrong.
type guardRepo struct {
	Path, Space string
	Missing     bool
}

type guardData struct {
	Untouchable []guardRepo
	Allowed     []guardRepo // every space's, in the order of the spaces
	AskRepos    []guardRepo
	SpaceNames  []string // for the add forms' choice of space
	Current     string   // the space the page was opened in, chosen by default
	Roots       []guardRoot
	Unreadable  []string // root_ask entries that cannot be read
	Profiles    []string // the machine-wide list, the one this page edits
	OtherProf   []string // the same from a space, shown only
	Hosts       []string
	OtherHosts  []string
	Protected   []string
	Changed     bool // the lists are not the ones last sealed
	Changes     []listChange
	Known       bool   // the sealed lists were found among the kept versions
	Problem     string // why guard.toml cannot be read, when it cannot
	Model       string // preferred_model of session.toml
	LastStart   string // what the last session start said about its model
	Ask         bool   // memory_trust of session.toml is "ask" (or a value not known)
	Language    string
	Reach       string   // where augmented brain acts: machine or spaces
	SpaceDirs   []string // the folders of the spaces, with ~
	Remote      bool     // through the network: the page only shows
}

func (s *Server) guardPage(v *view, r *http.Request) (string, page, error) {
	d := guardData{Language: v.lang, Remote: s.viaRemote(r)}
	g, err := config.LoadGuard(v.c.Dir)
	if err != nil {
		d.Problem = err.Error()
	}
	for _, x := range g.UntouchableRepos {
		d.Untouchable = append(d.Untouchable, repoOf(x, ""))
	}
	d.Reach = reachOf(g)
	d.SpaceDirs = tildeEach(v.c.SpaceDirs())
	if ss, err := config.LoadSession(v.c.Dir); err == nil {
		d.Model = ss.PreferredModel
		d.Ask = ss.Trust() == config.TrustAsk
	} else if d.Problem == "" {
		d.Problem = err.Error()
	}
	if d.Model != "" {
		if ls, ok := state.LastStartModel(v.c.Dir); ok {
			said := v.t("guard.model.none")
			if ls.Model != "" {
				said = ls.Model
			}
			d.LastStart = v.t("guard.model.last", said)
		}
	}
	d.SpaceNames, d.Current = v.c.SpaceNames(), v.b.Space.Name
	for _, name := range d.SpaceNames {
		sp := v.c.Spaces[name]
		for _, x := range sp.GitAllowedRepos {
			d.Allowed = append(d.Allowed, repoOf(x, name))
		}
		for _, x := range sp.GitAskRepos {
			d.AskRepos = append(d.AskRepos, repoOf(x, name))
		}
	}
	_, d.Unreadable = v.c.RootExceptions()
	for _, e := range v.c.RootAsk {
		if ra, err := config.ParseRootAsk(v.lang, e); err == nil {
			d.Roots = append(d.Roots, guardRoot{e, ra.Host, ra.Until.Format("2006-01-02 15:04") + " UTC", time.Now().After(ra.Until)})
		}
	}
	d.Profiles, d.Hosts = v.c.ProductionAWSProfiles, v.c.UntouchableHosts
	allProf, allHosts := v.c.GuardLists()
	for _, x := range allProf {
		if !slices.Contains(d.Profiles, x) {
			d.OtherProf = append(d.OtherProf, x)
		}
	}
	for _, x := range allHosts {
		if !slices.Contains(d.Hosts, x) {
			d.OtherHosts = append(d.OtherHosts, x)
		}
	}
	for _, f := range []string{config.FileName, config.GuardFile, config.SecretsFile, config.SessionFile, "remote.toml", "panel.json"} {
		d.Protected = append(d.Protected, paths.Tilde(filepath.Join(v.c.Dir, f)))
	}
	if seal, err := state.ReadSeal(v.c.Dir); err != nil || (seal != "" && seal != v.c.GuardFingerprint(g)) {
		d.Changed = true
		if sc, sg, ok := sealedVersion(v.c.Dir, seal); ok {
			d.Known = true
			d.Changes = listChanges(v, guardListsOf(sc, sg), guardListsOf(v.c, g))
		}
	}
	p := v.frame("guard", v.t("guard"))
	p.D = d
	return "guard", p, nil
}

// repoOf is one repository of a list, with whether its folder is on this computer. A
// name with * stands for several folders and is never missing.
func repoOf(path, space string) guardRepo {
	r := guardRepo{Path: path, Space: space}
	if !strings.Contains(path, "*") {
		if a, err := paths.Resolve(path); err == nil {
			_, err = os.Stat(a)
			r.Missing = errors.Is(err, os.ErrNotExist)
		}
	}
	return r
}

// guardList is what the Guard page's template "repos" gets: the page, which list, and
// its repositories.
type guardList struct {
	P     page
	List  string
	Repos []guardRepo
}

func repoList(p page, list string, repos []guardRepo) guardList { return guardList{p, list, repos} }

// listChange is what one of the guard's lists gained and lost since the user last saw
// them.
type listChange struct {
	List           string
	Added, Removed []string
}

// guardListKeys are the lists the seal covers, in the page's order, with their titles.
var guardListKeys = []struct{ key, title string }{
	{"untouchable", "guard.untouchable"}, {"allowed", "guard.allowed"}, {"ask", "guard.ask"},
	{"root", "guard.root"}, {"profiles", "guard.profiles"}, {"hosts", "guard.hosts"},
	{"reach", "guard.reach"},
}

// reachOf is where guard.toml says augmented brain acts, the default written out.
func reachOf(g config.Guard) string {
	if g.OnlySpaces() {
		return config.ReachSpaces
	}
	return config.ReachMachine
}

// guardListsOf are the lists GuardFingerprint covers, by key.
func guardListsOf(c *config.Config, g config.Guard) map[string][]string {
	allowed, ask := c.GitLists()
	profiles, hosts := c.GuardLists()
	return map[string][]string{"untouchable": g.UntouchableRepos, "allowed": allowed, "ask": ask,
		"root": c.RootAsk, "profiles": profiles, "hosts": hosts, "reach": {reachOf(g)}}
}

// listChanges compares the sealed lists with the current ones, list by list.
func listChanges(v *view, before, now map[string][]string) []listChange {
	var out []listChange
	for _, k := range guardListKeys {
		ch := listChange{List: v.t(k.title)}
		for _, x := range now[k.key] {
			if !slices.Contains(before[k.key], x) {
				ch.Added = append(ch.Added, shown(x))
			}
		}
		for _, x := range before[k.key] {
			if !slices.Contains(now[k.key], x) {
				ch.Removed = append(ch.Removed, shown(x))
			}
		}
		if len(ch.Added)+len(ch.Removed) > 0 {
			out = append(out, ch)
		}
	}
	return out
}

// shown is an entry as the page writes it: a folder with ~, anything else as it is.
func shown(x string) string {
	if filepath.IsAbs(x) {
		return paths.Tilde(x)
	}
	return x
}

// sealedVersion finds the lists the user last accepted: the seal is only their
// fingerprint, so it looks for the newest kept version of the configuration whose lists
// have it. False when none has.
func sealedVersion(dir, seal string) (*config.Config, config.Guard, bool) {
	vs, err := confighistory.List(dir)
	if err != nil || len(vs) == 0 {
		return nil, config.Guard{}, false
	}
	for _, ver := range vs {
		c, g, ok := versionLists(ver)
		if ok && c.GuardFingerprint(g) == seal {
			return c, g, true
		}
	}
	return nil, config.Guard{}, false
}

// versionLists reads a kept version of the configuration, from a temporary copy of its
// files, for its guard's lists; false when it does not load.
func versionLists(ver confighistory.Version) (*config.Config, config.Guard, bool) {
	tmp, err := os.MkdirTemp("", "ab-version-")
	if err != nil {
		return nil, config.Guard{}, false
	}
	defer os.RemoveAll(tmp)
	for name, text := range ver.Files {
		if name != filepath.Base(name) {
			continue // only the configuration's own files, by name
		}
		_ = os.WriteFile(filepath.Join(tmp, name), []byte(text), 0o600)
	}
	c, err := config.Load(tmp)
	if err != nil {
		return nil, config.Guard{}, false
	}
	g, err := config.LoadGuard(tmp)
	if err != nil {
		return nil, config.Guard{}, false
	}
	return c, g, true
}

// sealed runs a change to the lists and, when the lists were the ones the user had
// sealed before it, seals the result. Otherwise the change is written and the lists stay
// unsealed: the page keeps saying something else changed them, until the user accepts.
func (v *view) sealed(change func(g *config.Guard) error) error {
	g, err := config.LoadGuard(v.c.Dir)
	if err != nil {
		return err
	}
	seal, err := state.ReadSeal(v.c.Dir)
	clean := err == nil && (seal == "" || seal == v.c.GuardFingerprint(g))
	before := guardListsOf(v.c, g)
	if err := change(&g); err != nil {
		return err
	}
	c, err := config.Load(v.c.Dir)
	if err != nil {
		return err
	}
	g2, err := config.LoadGuard(v.c.Dir)
	if err != nil {
		return err
	}
	v.c = c
	if ch := listChanges(v, before, guardListsOf(c, g2)); len(ch) > 0 {
		_ = state.AddGuardNotice(v.c.Dir, v.t("guard.notice.changed", changesText(ch)), time.Now())
	}
	if clean {
		return state.WriteSeal(v.c.Dir, c.GuardFingerprint(g2))
	}
	return nil
}

// changesText is a list's changes in one line, for the notice the next hook shows the
// user: "Allowed repositories: +~/work/core; Untouchable hosts: -db01".
func changesText(chs []listChange) string {
	var parts []string
	for _, ch := range chs {
		var items []string
		for _, a := range ch.Added {
			items = append(items, "+"+a)
		}
		for _, r := range ch.Removed {
			items = append(items, "-"+r)
		}
		parts = append(parts, ch.List+": "+strings.Join(items, ", "))
	}
	return strings.Join(parts, "; ")
}

func tildeEach(list []string) []string {
	out := make([]string, len(list))
	for i, p := range list {
		out[i] = paths.Tilde(p)
	}
	return out
}

func without(list []string, x string) ([]string, bool) {
	i := slices.Index(list, x)
	if i < 0 {
		return list, false
	}
	return slices.Delete(slices.Clone(list), i, i+1), true
}

// localGuard refuses a change of the Guard page that comes through the network: the
// lists decide what the assistant may do on this computer, so they change only from its
// own panel, as going back to a version does (versionsRestore).
func (s *Server) localGuard(h func(*view, *http.Request) (string, error)) func(*view, *http.Request) (string, error) {
	return func(v *view, r *http.Request) (string, error) {
		if s.viaRemote(r) {
			return "", errors.New(v.t("guard.localonly"))
		}
		return h(v, r)
	}
}

// guardList adds to or removes from one list. Removing an untouchable repository asks
// for its folder's name: the one change that takes a protection away.
func (s *Server) guardList(v *view, r *http.Request) (string, error) {
	list, op, value := field(r, "list"), field(r, "op"), field(r, "value")
	if value == "" || len(value) > 500 || strings.ContainsAny(value, "\r\n") {
		return "", errors.New(v.t("err.required", v.t("f.value")))
	}
	edit := func(cur []string) ([]string, error) {
		switch op {
		case "add":
			if slices.Contains(cur, value) {
				return cur, nil
			}
			return append(slices.Clone(cur), value), nil
		case "remove":
			out, ok := without(cur, value)
			if !ok {
				return nil, errors.New(v.t("guard.err.gone"))
			}
			return out, nil
		}
		return nil, errors.New(v.t("guard.err.op"))
	}
	err := v.sealed(func(g *config.Guard) error {
		switch list {
		case "untouchable":
			if op == "remove" && field(r, "confirm") != filepath.Base(value) {
				return errors.New(v.t("guard.err.typename", filepath.Base(value)))
			}
			if op == "add" {
				a, err := paths.Resolve(value)
				if err != nil {
					return err
				}
				value = a
			}
			out, err := edit(g.UntouchableRepos)
			if err != nil {
				return err
			}
			g.UntouchableRepos = out // the rest of the file as it was
			return config.SaveGuard(v.c.Dir, *g)
		case "allowed", "ask":
			sp, ok := v.c.Spaces[field(r, "space")]
			if !ok {
				return errNoSpace
			}
			// Letting the assistant commit and push without asking is the riskiest list:
			// adding to it needs the box ticked that says so.
			if list == "allowed" && op == "add" && field(r, "confirm") != "yes" {
				return errors.New(v.t("guard.err.allowedconfirm"))
			}
			cur, key := sp.GitAllowedRepos, "git_allowed_repos"
			if list == "ask" {
				cur, key = sp.GitAskRepos, "git_ask_repos"
			}
			if op == "add" {
				a, err := paths.Resolve(value)
				if err != nil {
					return err
				}
				value = a
			}
			out, err := edit(cur)
			if err != nil {
				return err
			}
			return config.SetSpaceList(v.c.Dir, sp.Name, key, tildeEach(out))
		case "profiles":
			out, err := edit(v.c.ProductionAWSProfiles)
			if err != nil {
				return err
			}
			return config.SetTopList(v.c.Dir, "production_aws_profiles", out)
		case "hosts":
			out, err := edit(v.c.UntouchableHosts)
			if err != nil {
				return err
			}
			return config.SetTopList(v.c.Dir, "untouchable_hosts", out)
		}
		return errors.New(v.t("guard.err.op"))
	})
	return "/guard", err
}

// guardRootAsk adds a root exception or removes one. Adding asks for the machine's name
// typed again, and checks the date reads: "<host> until <date>".
func (s *Server) guardRootAsk(v *view, r *http.Request) (string, error) {
	err := v.sealed(func(*config.Guard) error {
		cur := v.c.RootAsk
		switch field(r, "op") {
		case "add":
			host, until := field(r, "host"), field(r, "until")
			if host == "" {
				return &fieldError{"host", errors.New(v.t("err.required", v.t("f.host")))}
			}
			if until == "" {
				return &fieldError{"until", errors.New(v.t("err.required", v.t("guard.until")))}
			}
			if field(r, "confirm") != host {
				return &fieldError{"confirm", errors.New(v.t("guard.err.typename", host))}
			}
			entry := host + " until " + until
			if _, err := config.ParseRootAsk(v.lang, entry); err != nil {
				return &fieldError{"until", err}
			}
			if !slices.Contains(cur, entry) {
				cur = append(slices.Clone(cur), entry)
			}
		case "remove":
			out, ok := without(cur, field(r, "value"))
			if !ok {
				return errors.New(v.t("guard.err.gone"))
			}
			cur = out
		default:
			return errors.New(v.t("guard.err.op"))
		}
		return config.SetTopList(v.c.Dir, "root_ask", cur)
	})
	return "/guard", err
}

// guardAccept takes the lists as they are as the ones the user has seen.
func (s *Server) guardAccept(v *view, r *http.Request) (string, error) {
	g, err := config.LoadGuard(v.c.Dir)
	if err != nil {
		return "", err
	}
	what := v.t("guard.notice.unknown")
	if seal, err := state.ReadSeal(v.c.Dir); err == nil && seal != "" {
		if sc, sg, ok := sealedVersion(v.c.Dir, seal); ok {
			if ch := listChanges(v, guardListsOf(sc, sg), guardListsOf(v.c, g)); len(ch) > 0 {
				what = changesText(ch)
			}
		}
	}
	if err := state.WriteSeal(v.c.Dir, v.c.GuardFingerprint(g)); err != nil {
		return "", err
	}
	_ = state.AddGuardNotice(v.c.Dir, v.t("guard.notice.accepted", what), time.Now())
	return "/guard", nil
}

// guardReach saves where augmented brain acts (reach of guard.toml), sealed like the
// lists. Keeping it to the spaces takes rules away outside them, so it needs the box
// ticked that says the user understands which. Like every change of this page, only
// from this computer's own panel (localGuard).
func (s *Server) guardReach(v *view, r *http.Request) (string, error) {
	value := field(r, "value")
	if value != config.ReachMachine && value != config.ReachSpaces {
		return "", errors.New(v.t("guard.err.op"))
	}
	if value == config.ReachSpaces && field(r, "confirm") != "yes" {
		return "", errors.New(v.t("guard.err.reachconfirm"))
	}
	err := v.sealed(func(g *config.Guard) error {
		if reachOf(*g) == value {
			return nil
		}
		g.Reach = value
		return config.SaveGuard(v.c.Dir, *g)
	})
	return "/guard", err
}

// guardModel saves the preferred model of session.toml (empty: no check), keeping the
// rest of the file.
func (s *Server) guardModel(v *view, r *http.Request) (string, error) {
	value := field(r, "value")
	if len(value) > 100 || strings.ContainsAny(value, "\r\n\"\\") {
		return "", errors.New(v.t("err.required", v.t("f.model")))
	}
	ss, err := config.LoadSession(v.c.Dir)
	if err != nil {
		return "", err
	}
	ss.PreferredModel = value
	return "/guard", config.SaveSession(v.c.Dir, ss)
}

// guardTrust saves memory_trust of session.toml: whether the assistant asks before it
// saves a rule, the profile or the guard's lists. The next hook tells the user, as
// with the lists: a level changed behind their back would be one they never chose.
func (s *Server) guardTrust(v *view, r *http.Request) (string, error) {
	value := field(r, "value")
	if value != config.TrustTell && value != config.TrustAsk {
		return "", errors.New(v.t("err.required", v.t("guard.trust")))
	}
	ss, err := config.LoadSession(v.c.Dir)
	if err != nil {
		return "", err
	}
	if ss.MemoryTrust == value {
		return "/guard", nil
	}
	ss.MemoryTrust = value
	if err := config.SaveSession(v.c.Dir, ss); err != nil {
		return "", err
	}
	_ = state.AddGuardNotice(v.c.Dir, v.t("guard.trust.notice."+value), time.Now())
	return "/guard", nil
}
