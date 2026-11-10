package panel

import (
	"errors"
	"maps"
	"net/http"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/confighistory"
)

// The Versions page: the last versions of the configuration (confighistory), what each
// one changed from the one before, and going back to one. Going back is only for the
// panel on this computer: through the network the page only shows.

type versionRow struct {
	N       int
	ID      string
	When    string
	Who     string
	Current bool
	Diff    []confighistory.FileDiff // what it changed from the one before; nil for the oldest
	Oldest  bool
	// Guard is how going back to it would change the guard's lists, from what they are now.
	Guard []listChange
}

type versionsData struct {
	Versions []versionRow
	Changed  bool // the configuration changed after the newest version
	Remote   bool // through the network: no going back from here
	Keep     int
}

func (s *Server) versionsPage(v *view, r *http.Request) (string, page, error) {
	vs, err := confighistory.List(v.c.Dir)
	if err != nil {
		return "", page{}, err
	}
	d := versionsData{Remote: s.viaRemote(r), Keep: confighistory.Keep}
	d.Changed, _ = confighistory.Changed(v.c.Dir)
	cur, _ := confighistory.Snapshot(v.c.Dir)
	var now map[string][]string
	if g, err := config.LoadGuard(v.c.Dir); err == nil {
		now = guardListsOf(v.c, g)
	}
	for i, x := range vs {
		row := versionRow{N: i + 1, ID: x.ID, When: x.Time.Local().Format("2006-01-02 15:04"),
			Who: confighistory.Who(v.lang, x), Current: maps.Equal(cur, x.Files), Oldest: i == len(vs)-1}
		if !row.Oldest {
			row.Diff = confighistory.Compare(vs[i+1].Files, x.Files, 2)
		}
		if !row.Current && now != nil {
			if c, g, ok := versionLists(x); ok {
				row.Guard = listChanges(v, now, guardListsOf(c, g))
			}
		}
		d.Versions = append(d.Versions, row)
	}
	p := v.frame("versions", v.t("versions"))
	p.D = d
	return "versions", p, nil
}

// versionsRestore goes back to one version. The guard's lists are sealed afterwards when
// they were the sealed ones before, as any change made on the Guard page: the user is at
// this computer, doing it.
func (s *Server) versionsRestore(v *view, r *http.Request) (string, error) {
	if s.viaRemote(r) {
		return "", errors.New(v.t("versions.localonly"))
	}
	if field(r, "confirm") != "yes" {
		return "", errors.New(v.t("err.confirm"))
	}
	id := field(r, "id")
	err := v.sealed(func(*config.Guard) error {
		return confighistory.Restore(v.c.Dir, id, func() error { return confighistory.Check(v.c.Dir) })
	})
	if errors.Is(err, confighistory.ErrNotFound) {
		err = errors.New(v.t("versions.gone"))
	}
	return "/versions", err
}
