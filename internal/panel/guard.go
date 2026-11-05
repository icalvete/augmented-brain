package panel

import (
	"errors"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
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

type guardSpace struct {
	Space        string
	Allowed, Ask []string
}

type guardData struct {
	Untouchable []string
	Spaces      []guardSpace
	Roots       []guardRoot
	Unreadable  []string // root_ask entries that cannot be read
	Profiles    []string // the machine-wide list, the one this page edits
	OtherProf   []string // the same from a space, shown only
	Hosts       []string
	OtherHosts  []string
	Protected   []string
	Changed     bool   // the lists are not the ones last sealed
	Problem     string // why guard.toml cannot be read, when it cannot
	Language    string
}

func (s *Server) guardPage(v *view, r *http.Request) (string, page, error) {
	d := guardData{Language: v.lang}
	g, err := config.LoadGuard(v.c.Dir)
	if err != nil {
		d.Problem = err.Error()
	}
	d.Untouchable = g.UntouchableRepos
	for _, name := range v.c.SpaceNames() {
		sp := v.c.Spaces[name]
		d.Spaces = append(d.Spaces, guardSpace{name, sp.GitAllowedRepos, sp.GitAskRepos})
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
	for _, f := range []string{config.FileName, config.GuardFile, config.SecretsFile, "remote.toml", "panel.json"} {
		d.Protected = append(d.Protected, paths.Tilde(filepath.Join(v.c.Dir, f)))
	}
	if seal := state.ReadSeal(v.c.Dir); seal != "" && seal != v.c.GuardFingerprint(g) {
		d.Changed = true
	}
	p := v.frame("guard", v.t("guard"))
	p.D = d
	return "guard", p, nil
}

// sealed runs a change to the lists and, when the lists were the ones the user had
// sealed before it, seals the result. Otherwise the change is written and the lists stay
// unsealed: the page keeps saying something else changed them, until the user accepts.
func (v *view) sealed(change func(g *config.Guard) error) error {
	g, err := config.LoadGuard(v.c.Dir)
	if err != nil {
		return err
	}
	seal := state.ReadSeal(v.c.Dir)
	clean := seal == "" || seal == v.c.GuardFingerprint(g)
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
	if clean {
		return state.WriteSeal(v.c.Dir, c.GuardFingerprint(g2))
	}
	return nil
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
			return config.SaveGuard(v.c.Dir, config.Guard{UntouchableRepos: out})
		case "allowed", "ask":
			sp, ok := v.c.Spaces[field(r, "space")]
			if !ok {
				return errNoSpace
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
			if host == "" || until == "" {
				return errors.New(v.t("err.required", v.t("f.host")))
			}
			if field(r, "confirm") != host {
				return errors.New(v.t("guard.err.typename", host))
			}
			entry := host + " until " + until
			if _, err := config.ParseRootAsk(v.lang, entry); err != nil {
				return err
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
	return "/guard", state.WriteSeal(v.c.Dir, v.c.GuardFingerprint(g))
}
