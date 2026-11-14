package panel

import (
	"net/http"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/repostat"
	"github.com/icalvete/augmented-brain/internal/state"
)

// todayDays is how far ahead the first page looks for reminders: the same week the
// session start brings up.
const todayDays = 7

type todayReminder struct {
	brain.Reminder
	TaskTitle string
}

// todaySpace is what one space asks of the user today.
type todaySpace struct {
	Name                     string
	Reminders                []todayReminder
	Mine, MineParked         []taskRow
	Inbox, Dead, Problems    int
	DeadRows                 []sessionRow // the sessions that did not end, to deal with from here
	ReviewDue                bool
	ReviewDays, Live, Parked int
}

// Empty says the space asks nothing of the user today, so its block says so instead of
// showing a bare heading.
func (ts todaySpace) Empty() bool {
	return len(ts.Reminders) == 0 && len(ts.Mine) == 0 && len(ts.MineParked) == 0 && ts.Inbox == 0 &&
		ts.Dead == 0 && !ts.ReviewDue && ts.Problems == 0
}

type taskLink struct{ Space, ID, Title, Status string }

// repoRow is a repository that live tasks name and that holds something not yet
// elsewhere: changes not committed, or commits not pushed.
type repoRow struct {
	repostat.Status
	Name        string
	Tasks       []taskLink
	Forgotten   bool   // every task that names it is parked: nobody is on it
	Description string // what it is, from the space's repositories note; empty: not described yet
	Space       string // the space whose work folders hold it
}

// today is the first page: what every space asks of the user now, and the
// repositories with work that is only on this computer.
func (s *Server) today(v *view, _ *http.Request) (string, page, error) {
	var d struct {
		Spaces     []todaySpace
		Repos      []repoRow
		CleanRepos int
		Nothing    bool
	}
	type repoUse struct {
		name, desc, space string
		tasks             []taskLink
	}
	uses := map[string]*repoUse{}
	var order []string
	now := time.Now()
	for _, name := range v.c.SpaceNames() {
		b := v.b
		if name != v.b.Space.Name {
			var err error
			if b, err = brain.Open(v.c, v.c.Spaces[name]); err != nil {
				continue
			}
		}
		ts := todaySpace{Name: name, Problems: len(b.Problems)}
		for _, r := range b.Reminders(todayDays) {
			title := ""
			if n, ok := b.Mem.Notes[r.TaskID]; ok {
				title = n.Header.Title
			}
			ts.Reminders = append(ts.Reminders, todayReminder{r, title})
		}
		items, _ := b.Pending(0)
		for _, i := range items {
			if i.Status == "parked" {
				ts.Parked++
			} else {
				ts.Live++
			}
			if i.WaitingOn == "me" && i.Status == "parked" {
				ts.MineParked = append(ts.MineParked, taskRow{ID: i.ID, Title: i.Title})
			} else if i.WaitingOn == "me" {
				ts.Mine = append(ts.Mine, taskRow{ID: i.ID, Title: i.Title, Status: i.Status,
					StatusLabel: v.status(i.Status), Next: i.Next, Steps: v.steps(i.StepsDone, i.StepsTotal)})
			}
		}
		entries, _ := b.Inbox()
		ts.Inbox = len(entries)
		ts.ReviewDue, ts.ReviewDays = b.ReviewDue()
		for _, se := range state.Read(v.c.Dir, name).Unclosed("", now) {
			ts.DeadRows = append(ts.DeadRows, sessionRowOf(state.Listed{Session: se}, b.Mem))
		}
		ts.Dead = len(ts.DeadRows)
		d.Spaces = append(d.Spaces, ts)

		for _, n := range b.Mem.OfType("task") {
			h := n.Header
			if h.Status == "closed" {
				continue
			}
			for _, r := range h.Repos {
				// Only inside the space's work folders: git may run programs a
				// repository's own settings name, so a note (an imported one, say)
				// never sends it anywhere else.
				p, ok := b.RepoPath(r)
				if !ok || !slices.ContainsFunc(b.Space.WorkFolders, func(wf string) bool { return paths.Within(p, wf) && p != wf }) {
					continue
				}
				u := uses[p]
				if u == nil {
					u = &repoUse{name: filepath.Base(p), desc: b.RepoDescriptionOf(p), space: name}
					uses[p] = u
					order = append(order, p)
				}
				u.tasks = append(u.tasks, taskLink{name, h.ID, h.Title, h.Status})
			}
		}
	}
	v.unlock() // git can take seconds: not with the notes locked
	for _, st := range repostat.ReadAll(order) {
		u := uses[st.Path]
		if st.NotRepo {
			continue
		}
		if !st.Pending() && st.Err == "" {
			d.CleanRepos++
			continue
		}
		forgotten := !slices.ContainsFunc(u.tasks, func(t taskLink) bool { return t.Status != "parked" && t.Status != "someday" })
		d.Repos = append(d.Repos, repoRow{Status: st, Name: u.name, Tasks: u.tasks, Forgotten: forgotten,
			Description: u.desc, Space: u.space})
	}
	sort.SliceStable(d.Repos, func(i, j int) bool {
		if d.Repos[i].Forgotten != d.Repos[j].Forgotten {
			return d.Repos[i].Forgotten
		}
		return d.Repos[i].Name < d.Repos[j].Name
	})
	d.Nothing = len(d.Repos) == 0
	for _, sp := range d.Spaces {
		if len(sp.Reminders) > 0 || len(sp.Mine) > 0 || sp.Inbox > 0 || sp.Dead > 0 || sp.ReviewDue || sp.Problems > 0 {
			d.Nothing = false
		}
	}
	p := v.frame("today", v.t("today"))
	p.D = d
	return "today", p, nil
}
