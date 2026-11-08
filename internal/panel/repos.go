package panel

import (
	"net/http"
	"path/filepath"
	"slices"
	"sort"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/repostat"
)

// repoCard is one repository on the Repositories page.
type repoCard struct {
	repostat.Status
	Name, Shown string // its folder's name, and its path written with ~
	Description string // the user's own line, from the space's repositories note
	Tasks       []taskLink
	Forgotten   bool // it holds pending work and the live tasks that name it are all parked: nobody is on it
}

// reposSpace is one space's repositories.
type reposSpace struct {
	Name      string
	Repos     []repoCard
	More      bool // the search stopped at its limit: there may be more
	NoFolders bool // the space has no work folders to look in
}

// insideWork says whether p is strictly inside one of the work folders: git only ever
// runs there, since a repository's own settings can make it run programs.
func insideWork(p string, folders []string) bool {
	return slices.ContainsFunc(folders, func(wf string) bool { return paths.Within(p, wf) && !paths.Same(p, wf) })
}

// repos is the map of repositories: every git repository found in each space's work
// folders, what is pending in it, the live tasks that name it and what it is.
func (s *Server) repos(v *view, _ *http.Request) (string, page, error) {
	// What the notes say, read under the lock.
	type plan struct {
		name    string
		folders []string
		tasks   map[string][]taskLink // by path
		b       *brain.Brain
		extra   []string // repositories that tasks name, in the order met
	}
	var plans []*plan
	var notes []string // every space's notes folder, and the configuration's: not repositories of the user's
	for _, name := range v.c.SpaceNames() {
		b := v.b
		if name != v.b.Space.Name {
			var err error
			if b, err = brain.Open(v.c, v.c.Spaces[name]); err != nil {
				continue
			}
		}
		notes = append(notes, b.Space.Memory)
		pl := &plan{name: name, folders: b.Space.WorkFolders, tasks: map[string][]taskLink{}, b: b}
		for _, n := range b.Mem.OfType("task") {
			h := n.Header
			if h.Status == "closed" {
				continue
			}
			for _, r := range h.Repos {
				p, ok := b.RepoPath(r)
				if !ok || !insideWork(p, pl.folders) {
					continue
				}
				if _, seen := pl.tasks[p]; !seen {
					pl.extra = append(pl.extra, p)
				}
				pl.tasks[p] = append(pl.tasks[p], taskLink{name, h.ID, h.Title, h.Status})
			}
		}
		plans = append(plans, pl)
	}
	v.unlock() // the search and git can take seconds: not with the notes locked

	// The walk gives resolved paths (the work folders are): compare with these resolved too.
	notes = append(notes, v.c.Dir)
	for i, m := range notes {
		if real, err := paths.Resolve(m); err == nil {
			notes[i] = real
		}
	}
	skip := func(p string) bool {
		return slices.ContainsFunc(notes, func(m string) bool { return paths.Within(p, m) })
	}
	type found struct {
		pl    *plan
		paths []string
		more  bool
	}
	var all []found
	var toRead []string
	read := map[string]bool{}
	for _, pl := range plans {
		f := found{pl: pl}
		seen := map[string]bool{}
		add := func(p string) {
			if !seen[p] {
				seen[p] = true
				f.paths = append(f.paths, p)
			}
			if !read[p] {
				read[p] = true
				toRead = append(toRead, p)
			}
		}
		for _, wf := range pl.folders {
			got, more := repostat.Find(wf, repostat.FindDepth, repostat.FindLimit-len(f.paths), skip)
			f.more = f.more || more
			for _, p := range got {
				if insideWork(p, pl.folders) {
					add(p)
				}
			}
		}
		for _, p := range pl.extra {
			add(p)
		}
		all = append(all, f)
	}
	status := map[string]repostat.Status{}
	for _, st := range repostat.ReadAll(toRead) {
		status[st.Path] = st
	}

	var d struct {
		Spaces []reposSpace
		Total  int
	}
	for _, f := range all {
		rs := reposSpace{Name: f.pl.name, More: f.more, NoFolders: len(f.pl.folders) == 0}
		for _, p := range f.paths {
			st := status[p]
			if st.NotRepo {
				continue
			}
			tasks := f.pl.tasks[p]
			rs.Repos = append(rs.Repos, repoCard{
				Status: st, Name: filepath.Base(p), Shown: paths.Tilde(p),
				Description: f.pl.b.RepoDescriptionOf(p), Tasks: tasks,
				Forgotten: (st.Pending() || st.Err != "") && len(tasks) > 0 && !slices.ContainsFunc(tasks, func(t taskLink) bool { return t.Status != "parked" && t.Status != "someday" }),
			})
		}
		sort.SliceStable(rs.Repos, func(i, j int) bool {
			a, b := rs.Repos[i], rs.Repos[j]
			if pa, pb := a.Pending() || a.Err != "", b.Pending() || b.Err != ""; pa != pb {
				return pa
			}
			if a.Name != b.Name {
				return a.Name < b.Name
			}
			return a.Shown < b.Shown
		})
		d.Total += len(rs.Repos)
		d.Spaces = append(d.Spaces, rs)
	}
	p := v.frame("repos", v.t("repos"))
	p.D = d
	return "repos", p, nil
}

// repoDescribe saves what a repository is, from the Repositories page: an empty text
// takes the description out.
func (s *Server) repoDescribe(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "repo"); err != nil {
		return "", err
	}
	if _, err := v.b.DescribeRepo(field(r, "repo"), field(r, "text")); err != nil {
		return "", err
	}
	return "/repos", nil
}
