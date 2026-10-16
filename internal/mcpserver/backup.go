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

// setupBackup connects the space to a private repository (the one given, or a new one
// in the user's GitHub account) and makes the first backup. When GitHub is not ready,
// it answers with the steps, as text: the assistant guides the user through them.
func setupBackup(b *brain.Brain, url string, run backup.Runner) (string, error) {
	c, err := config.Load(b.ConfigDir)
	if err != nil {
		return "", err
	}
	s := c.Spaces[b.Space.Name]
	if s.Backup != "" && url == "" {
		msg := fmt.Sprintf("Space %s already backs up to %s, by itself when each session ends.", s.Name, s.Backup)
		if st, ok := backup.LastStatus(c.Dir, s.Name); ok {
			if st.OK {
				msg += " Last backup: " + st.Time.Format("2006-01-02 15:04") + ", fine."
			} else {
				msg += " The last one failed: " + st.Error
			}
		}
		return msg, nil
	}
	if url == "" {
		url, err = backup.GitHubRepo(s.Name, run)
		var steps *backup.ErrNoGitHub
		if errors.As(err, &steps) {
			return steps.Steps, nil
		}
		if err != nil {
			return "", err
		}
	}
	if err := backup.Setup(c, s, url); err != nil {
		return "", err
	}
	// No one can type a password into the server: git must fail instead of waiting.
	os.Setenv("GIT_TERMINAL_PROMPT", "0")
	host, _ := os.Hostname()
	_, err = backup.Run(c, s, host)
	backup.RecordStatus(c.Dir, s.Name, err, time.Now())
	if err != nil {
		return "", fmt.Errorf("the backup is set up to %s, but the first copy failed: %w", url, err)
	}
	return fmt.Sprintf("Backup of space %s set up: %s (private). The first copy is there, and from now on it runs by itself every time a session ends. "+
		"On a new machine: install augmented brain, then `augmented-brain restore %s`.", s.Name, url, url), nil
}
