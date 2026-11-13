package mcpserver

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
)

// setupBackup connects the space to a new private repository in the user's GitHub
// account (made, and checked private, by gh) and makes the first backup. An address
// given is refused: that backup is the user's to set up, in a terminal or the panel.
// When GitHub is not ready, it answers with the steps, as text: the assistant guides the
// user through them.
func setupBackup(b *brain.Brain, url string, run backup.Runner) (string, error) {
	c, err := config.Load(b.ConfigDir)
	if err != nil {
		return "", err
	}
	s := c.Spaces[b.Space.Name]
	if url != "" && url != s.Backup {
		if s.Backup != "" {
			// Moving a backup sends every note somewhere else: only the person does it, by
			// hand, never a tool the assistant may be talked into using.
			return "", fmt.Errorf("space %s already backs up to %s. Changing where it backs up is for the user: they run `augmented-brain backup setup <url>` in a terminal, in a folder of the space", s.Name, backup.WithoutPassword(s.Backup))
		}
		// The same goes for the first one: an address the assistant passes on may come
		// from anything it read, and one call would send every note there, again at the
		// end of each session. Only the repository gh makes in the user's own account is
		// set up from here.
		return "", errors.New("the tool does not take an address: a backup somewhere the user chose is set up by the user, who runs `augmented-brain backup setup <url>` in a terminal, in a folder of the space (or uses the panel's Backup page). " +
			"Without repo_url, this tool creates a private repository in the user's GitHub account")
	}
	if s.Backup != "" {
		msg := fmt.Sprintf("Space %s already backs up to %s, by itself when each session ends.", s.Name, backup.WithoutPassword(s.Backup))
		if st, ok := backup.LastStatus(c.Dir, s.Name); ok {
			if st.OK {
				msg += " Last backup: " + st.Time.Format("2006-01-02 15:04") + ", fine."
			} else {
				msg += " The last one failed: " + st.Error
			}
			if st.Warning != "" {
				msg += " Warning: " + st.Warning + "."
			}
		}
		return msg, nil
	}
	url, err = backup.GitHubRepo(s.Name, run)
	var steps *backup.ErrNoGitHub
	if errors.As(err, &steps) {
		return steps.Steps, nil
	}
	if err != nil {
		return "", err
	}
	if err := backup.Setup(c, s, url); err != nil {
		return "", err
	}
	host, _ := os.Hostname()
	res, err := backup.Run(c, s, host)
	backup.RecordStatusWarning(c.Dir, s.Name, err, res.Warning(c.Language), time.Now())
	if err != nil {
		return "", fmt.Errorf("the backup is set up to %s, but the first copy failed: %w", url, err)
	}
	warn := ""
	if w := res.Warning(c.Language); w != "" {
		warn = " Tell the user: " + w + "."
	}
	return fmt.Sprintf("Backup of space %s set up: %s (private). The first copy is there, and from now on it runs by itself every time a session ends. "+
		"On a new machine: install augmented brain, then `augmented-brain restore %s`.", s.Name, url, url) + warn, nil
}
