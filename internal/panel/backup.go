package panel

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

type backupCopy struct{ When, What string }

type backupSpace struct {
	Name, URL            string
	Known, OK            bool
	Time, Error, Warning string
	Copies               []backupCopy
	Started              bool // a copy was just started from here
	Profile              bool // this space's backup carries the profile
	NoGit                bool // set up, but the notes folder is not a repository (yet)
}

type backupData struct {
	Spaces []backupSpace
	Remote bool
	None   int // spaces with no backup at all
}

// copiesOf are the last copies of the notes in memory: the backup's own commits, newest
// first. Nothing when the folder is not a repository or git cannot read it.
func copiesOf(memory string) []backupCopy {
	if _, err := os.Stat(filepath.Join(memory, ".git")); err != nil {
		return nil
	}
	out, err := backup.Git(memory, "log", "-n", "20", "--format=%ct%x09%s", "HEAD")
	if err != nil {
		return nil
	}
	var cs []backupCopy
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		ts, subject, ok := strings.Cut(line, "\t")
		// Only the backup's commits ("backup <host> <date>"), not those a person made.
		host, isBackup := strings.CutPrefix(subject, "backup ")
		if !ok || !isBackup {
			continue
		}
		var sec int64
		for _, c := range ts {
			if c < '0' || c > '9' {
				sec = -1
				break
			}
			sec = sec*10 + int64(c-'0')
		}
		if sec <= 0 {
			continue
		}
		if f := strings.Fields(host); len(f) > 0 {
			host = f[0]
		}
		cs = append(cs, backupCopy{When: time.Unix(sec, 0).Local().Format(stamp), What: host})
		if len(cs) == lastCopies {
			break
		}
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
		bs := backupSpace{Name: name, URL: sp.Backup, Started: started == name, Profile: name == profile && sp.Backup != ""}
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
		s.fail(w, r, nil, http.StatusBadRequest, errors.New(i18n.T(lang, "panel.backup.localonly")))
		return
	}
	if strings.TrimSpace(r.PostForm.Get("url")) == "" {
		c, err := config.Load(s.cfgDir)
		if err != nil {
			s.fail(w, r, nil, http.StatusBadRequest, err)
			return
		}
		name := r.PostForm.Get("s")
		sp, ok := c.Spaces[name]
		if !ok {
			s.fail(w, r, nil, http.StatusBadRequest, errNoSpace)
			return
		}
		if sp.Backup != "" {
			s.fail(w, r, nil, http.StatusBadRequest, errors.New(i18n.T(lang, "panel.backup.err.has", sp.Backup)))
			return
		}
		made, err := s.gitHubRepo(name)
		var steps *backup.ErrNoGitHub
		if errors.As(err, &steps) {
			err = errors.New(i18n.T(lang, "panel.backup.err.gh"))
		}
		if err != nil {
			s.fail(w, r, nil, http.StatusBadRequest, err)
			return
		}
		r.PostForm.Set("url", made)
	}
	s.write(func(v *view, r *http.Request) (string, error) {
		sp := v.b.Space
		if sp.Backup != "" {
			return "", errors.New(v.t("backup.err.has", sp.Backup))
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
