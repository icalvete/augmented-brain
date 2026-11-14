package panel

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/detach"
	"github.com/icalvete/augmented-brain/internal/i18n"
)

// The Backup page: for every space, whether its notes have a copy elsewhere, the last
// copies, a copy now, and setting one up (a private repository of the person's GitHub,
// made with gh, or one they already have). Restoring stays a command: it is for a new
// computer, where no panel runs yet. Through the network the page only shows.

// lastCopies is how many of the last copies each space shows.
const lastCopies = 5

type backupCopy struct {
	When, What string
	Rev        string // the commit, to go back to it
	Back       string // it went back to the copy of this date
	Changes    backup.Changes
}

type backupSpace struct {
	Name, URL            string
	Known, OK            bool
	Time, Error, Warning string
	Copies               []backupCopy
	Started              bool   // a copy was just started from here
	WentBack             string // the notes just went back to the copy of this date
	Profile              bool   // this space's backup carries the profile
	NoGit                bool   // set up, but the notes folder is not a repository (yet)
}

type backupData struct {
	Spaces []backupSpace
	Remote bool
	None   int // spaces with no backup at all
}

// copiesOf are the last copies of the notes in memory, newest first, each with what it
// changed. Nothing when the folder is not a repository or git cannot read it.
func copiesOf(memory string) []backupCopy {
	got, err := backup.Copies(memory, lastCopies)
	if err != nil {
		return nil
	}
	var cs []backupCopy
	for _, c := range got {
		bc := backupCopy{When: c.Time.Local().Format(stamp), What: c.Host, Rev: c.Rev, Back: c.Back}
		bc.Changes, _ = backup.ChangesOf(memory, c.Rev)
		cs = append(cs, bc)
	}
	return cs
}

func (s *Server) backupPage(v *view, r *http.Request) (string, page, error) {
	d := backupData{Remote: s.viaRemote(r)}
	started := r.URL.Query().Get("started")
	profile := backup.ProfileSpace(v.c)
	type where struct{ name, memory string }
	var spaces []where
	for _, name := range v.c.SpaceNames() {
		sp := v.c.Spaces[name]
		bs := backupSpace{Name: name, URL: backup.WithoutPassword(sp.Backup), Started: started == name, Profile: name == profile && sp.Backup != ""}
		if started == name {
			bs.WentBack = r.URL.Query().Get("back")
		}
		if st, ok := backup.LastStatus(v.c.Dir, name); ok && sp.Backup != "" {
			bs.Known, bs.OK, bs.Error, bs.Warning = true, st.OK, st.Error, st.Warning
			bs.Time = st.Time.Local().Format(stamp)
		}
		if sp.Backup == "" {
			d.None++
		}
		d.Spaces = append(d.Spaces, bs)
		spaces = append(spaces, where{name, sp.Memory})
	}
	v.unlock() // git reads the history: not with the notes locked
	for i, w := range spaces {
		if d.Spaces[i].URL == "" {
			continue
		}
		d.Spaces[i].Copies = copiesOf(w.memory)
		if _, err := os.Stat(filepath.Join(w.memory, ".git")); err != nil {
			d.Spaces[i].NoGit = true
		}
	}
	p := v.frame("backup", v.t("backup"))
	p.D = d
	return "backup", p, nil
}

// startBackup starts the backup of a space in the background, as the end of a session
// does: the page does not wait for git or the network. Tests replace it.
func (s *Server) startBackup(space string) error {
	if s.backupNow != nil {
		return s.backupNow(space)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "backup", "--auto", "--space", space)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	detach.Cmd(cmd)
	_, err = detach.Start(cmd)
	return err
}

// gitHubRepo makes (or finds) the space's private repository on GitHub. Tests replace it.
func (s *Server) gitHubRepo(space string) (string, error) {
	if s.github != nil {
		return s.github(space)
	}
	return backup.GitHubRepo(space, backup.Exec)
}

// backupStarted is where the browser goes after a copy was started.
func backupStarted(space string) string {
	return "/backup?" + url.Values{"started": {space}}.Encode()
}

// backupRun starts a copy of one space now.
func (s *Server) backupRun(v *view, r *http.Request) (string, error) {
	if s.viaRemote(r) {
		return "", errors.New(v.t("backup.localonly"))
	}
	sp := v.b.Space
	if sp.Backup == "" {
		return "", errors.New(v.t("backup.err.none"))
	}
	if err := s.startBackup(sp.Name); err != nil {
		return "", err
	}
	return backupStarted(sp.Name), nil
}

// backupSetup connects a space with no backup to a private repository: the address the
// person gives, or, with none, a new one in their GitHub account. Then it starts the
// first copy. Moving a backup that exists is for a terminal, as with the assistant:
// it sends every note somewhere else.
//
// Making the repository talks to GitHub for seconds: it runs before the notes are
// locked, and only the change of the configuration takes the lock (and checks again
// that the space still has no backup).
func (s *Server) backupSetup(w http.ResponseWriter, r *http.Request) {
	lang := s.lang()
	if s.viaRemote(r) {
		s.failForm(w, r, nil, errors.New(i18n.T(lang, "panel.backup.localonly")))
		return
	}
	if strings.TrimSpace(r.PostForm.Get("url")) == "" {
		c, err := config.Load(s.cfgDir)
		if err != nil {
			s.failForm(w, r, nil, err)
			return
		}
		name := r.PostForm.Get("s")
		sp, ok := c.Spaces[name]
		if !ok {
			s.failForm(w, r, nil, errNoSpace)
			return
		}
		if sp.Backup != "" {
			s.failForm(w, r, nil, errors.New(i18n.T(lang, "panel.backup.err.has", backup.WithoutPassword(sp.Backup))))
			return
		}
		made, err := s.gitHubRepo(name)
		var steps *backup.ErrNoGitHub
		if errors.As(err, &steps) {
			err = errors.New(i18n.T(lang, "panel.backup.err.gh"))
		}
		if err != nil {
			s.failForm(w, r, nil, err)
			return
		}
		r.PostForm.Set("url", made)
	}
	s.write(func(v *view, r *http.Request) (string, error) {
		sp := v.b.Space
		if sp.Backup != "" {
			return "", errors.New(v.t("backup.err.has", backup.WithoutPassword(sp.Backup)))
		}
		if err := backup.Setup(v.c, sp, field(r, "url")); err != nil {
			return "", err
		}
		if err := s.startBackup(sp.Name); err != nil {
			return "", err
		}
		return backupStarted(sp.Name), nil
	})(w, r)
}

// The page before going back to a copy: what would change in the notes as they are now,
// and the confirmation (a box, and the space's name typed). Through the network it only
// shows.

type backData struct {
	Space, Rev, When string
	Changes          backup.Changes
	Remote           bool
}

func (s *Server) backupBackPage(v *view, r *http.Request) (string, page, error) {
	sp := v.b.Space
	if sp.Backup == "" {
		return "", page{}, errors.New(v.t("backup.err.none"))
	}
	d := backData{Space: sp.Name, Rev: r.URL.Query().Get("c"), Remote: s.viaRemote(r)}
	v.unlock() // git reads the history: not with the notes locked
	cs, err := backup.Copies(sp.Memory, 200)
	if err != nil {
		return "", page{}, err
	}
	for _, c := range cs {
		if c.Rev == d.Rev {
			d.When = c.Time.Local().Format(stamp)
		}
	}
	if d.When == "" {
		return "", page{}, errors.New(v.t("backup.back.gone"))
	}
	if d.Changes, err = backup.ChangesBack(sp.Memory, d.Rev); err != nil {
		return "", page{}, err
	}
	p := v.frame("backup", v.t("backup.back"))
	p.D = d
	return "backupback", p, nil
}

// goBack is what the panel does to go back to a copy. Tests replace it.
func (s *Server) goBack(c *config.Config, sp *config.Space, rev string) error {
	host, _ := os.Hostname()
	_, err := backup.GoBack(c, sp, rev, host)
	return err
}

// backupBack puts the notes of a space back as they were in one of its copies. It takes
// the locks itself (backup.GoBack), so it runs outside s.write, as backupSetup does;
// then it starts a copy, which sends the result to the backup.
func (s *Server) backupBack(w http.ResponseWriter, r *http.Request) {
	lang := s.lang()
	fail := func(err error) { s.failForm(w, r, nil, err) }
	if s.viaRemote(r) {
		fail(errors.New(i18n.T(lang, "panel.backup.back.localonly")))
		return
	}
	c, err := config.Load(s.cfgDir)
	if err != nil {
		fail(err)
		return
	}
	lang = c.Language
	name := r.PostForm.Get("s")
	sp, ok := c.Spaces[name]
	if !ok {
		fail(errNoSpace)
		return
	}
	if r.PostForm.Get("confirm") != "yes" {
		fail(errors.New(i18n.T(lang, "panel.err.confirm")))
		return
	}
	if strings.TrimSpace(r.PostForm.Get("name")) != name {
		fail(errors.New(i18n.T(lang, "panel.backup.back.name", name)))
		return
	}
	if sp.Backup == "" {
		fail(errors.New(i18n.T(lang, "panel.backup.err.none")))
		return
	}
	rev := r.PostForm.Get("c")
	var when string
	if cs, err := backup.Copies(sp.Memory, 200); err == nil {
		for _, x := range cs {
			if x.Rev == rev {
				when = x.Time.Local().Format(stamp)
			}
		}
	}
	err = s.goBack(c, sp, rev)
	var notValid *backup.NotValidError
	var secrets *backup.SecretsError
	switch {
	case err == nil:
	case errors.Is(err, backup.ErrNotACopy):
		err = errors.New(i18n.T(lang, "panel.backup.back.gone"))
	case errors.Is(err, backup.ErrChangedMeanwhile):
		err = errors.New(i18n.T(lang, "panel.backup.back.meanwhile"))
	case errors.As(err, &notValid):
		err = errors.New(i18n.T(lang, "panel.backup.back.notvalid", strings.Join(notValid.Problems, "; ")))
	case errors.As(err, &secrets):
		var l []string
		for _, f := range secrets.Found {
			l = append(l, f.Path+" ("+i18n.T(lang, "secret."+f.Kind)+")")
		}
		err = errors.New(i18n.T(lang, "panel.backup.back.secrets", strings.Join(l, ", ")))
	}
	if err != nil {
		fail(err)
		return
	}
	if err := s.startBackup(name); err != nil {
		fail(err)
		return
	}
	q := url.Values{"started": {name}, "back": {when}, "s": {name}}
	http.Redirect(w, r, s.base(r)+"/backup?"+q.Encode(), http.StatusSeeOther)
}
