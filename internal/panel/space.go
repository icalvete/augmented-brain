package panel

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/state"
)

func (s *Server) space(v *view, _ *http.Request) (string, page, error) {
	sp := v.b.Space
	var problems []string
	for _, pr := range v.b.Problems {
		problems = append(problems, pr.Text(v.lang))
	}
	current, _ := os.ReadFile(filepath.Join(sp.Memory, memory.IndexFile))
	due, _ := v.b.ReviewDue()
	d := struct {
		Memory, Backup, BackupTime, BackupError, BackupWarning, LastReview string
		WorkFolders, Problems                                              []string
		Mismatches                                                         []brain.Mismatch
		Notes, Dead                                                        int
		IndexStale, BackupKnown, BackupOK, ReviewDue, IsDefault            bool
	}{Memory: paths.Tilde(sp.Memory), Backup: backup.WithoutPassword(sp.Backup), LastReview: v.b.LastReview(), WorkFolders: tildeEach(sp.WorkFolders),
		Problems: problems, Notes: len(v.b.Mem.Notes), IndexStale: string(current) != v.b.Mem.Index(v.lang),
		ReviewDue: due, IsDefault: v.c.DefaultSpace == sp.Name, Mismatches: v.b.Mismatches()}
	d.Dead = len(state.Read(v.c.Dir, sp.Name).Unclosed("", time.Now()))
	if st, ok := backup.LastStatus(v.c.Dir, sp.Name); ok {
		d.BackupKnown, d.BackupOK, d.BackupError, d.BackupWarning = true, st.OK, st.Error, st.Warning
		d.BackupTime = st.Time.Local().Format("2006-01-02 15:04")
	}
	p := v.frame("space", v.t("space.title", sp.Name))
	p.D = d
	return "space", p, nil
}
