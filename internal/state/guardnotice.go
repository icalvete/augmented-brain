package state

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/paths"
)

// noticeFile holds what the user must hear about the guard's lists from somewhere the
// assistant does not speak for them: the panel changed or accepted them, or a session
// started with lists nobody had seen. The next hook that can show the user a message
// (Stop, UserPromptSubmit, SessionStart) takes it and shows it once. The assistant may
// have made the change itself, through the panel's pages: a note to the assistant asking
// it to tell the user would leave the user to the assistant's word.
const noticeFile = "guard-notice"

// toolNoticeFile is the same for what an MCP tool saved that every later session loads
// (a rule, the profile, the guard's safety lists): the assistant calls those tools, so
// a hostile text it read could have it save one. The user hears of it from Claude Code,
// not from the assistant, once.
const toolNoticeFile = "tool-notice"

// quotaNoticeFile is the same for what the watch did about the sessions the usage limit
// stopped (remote.WakeQuota): it runs from a timer, with no session to speak through.
const quotaNoticeFile = "quota-notice"

// noticeMax keeps the file small when no session takes it for a long while.
const noticeMax = 20

// AddGuardNotice adds a line to what the next hook shows the user, with the time.
func AddGuardNotice(cfgDir, text string, now time.Time) error {
	return addNotice(cfgDir, noticeFile, text, now)
}

// AddToolNotice adds a line to what the next hook shows the user about what a tool
// saved for every session, with the time.
func AddToolNotice(cfgDir, text string, now time.Time) error {
	return addNotice(cfgDir, toolNoticeFile, text, now)
}

// AddQuotaNotice adds a line to what the next hook shows the user about the sessions
// the usage limit stopped, with the time.
func AddQuotaNotice(cfgDir, text string, now time.Time) error {
	return addNotice(cfgDir, quotaNoticeFile, text, now)
}

// TakeQuotaNotice is TakeGuardNotice for the sessions the usage limit stopped.
func TakeQuotaNotice(cfgDir string) string { return takeNotice(cfgDir, quotaNoticeFile) }

// TakeGuardNotice returns what there is to show the user about the guard's lists and
// removes it, so it is shown once. Nothing ("") when there is none.
func TakeGuardNotice(cfgDir string) string { return takeNotice(cfgDir, noticeFile) }

// TakeToolNotice is TakeGuardNotice for what a tool saved for every session.
func TakeToolNotice(cfgDir string) string { return takeNotice(cfgDir, toolNoticeFile) }

func addNotice(cfgDir, name, text string, now time.Time) error {
	dir := filepath.Join(cfgDir, Dir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f := filepath.Join(dir, name)
	old, err := os.ReadFile(f)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	lines := strings.Split(strings.TrimSpace(string(old)), "\n")
	if lines[0] == "" {
		lines = nil
	}
	lines = append(lines, now.Format("15:04")+" · "+strings.ReplaceAll(text, "\n", " "))
	if len(lines) > noticeMax {
		lines = lines[len(lines)-noticeMax:]
	}
	return paths.WriteAtomic(f, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}

// takeNotice returns what a notice file holds and removes it, so it is shown once: the
// file is renamed first, so two hooks at the same time never both get it.
func takeNotice(cfgDir, name string) string {
	f := filepath.Join(cfgDir, Dir, name)
	if _, err := os.Stat(f); err != nil {
		return ""
	}
	taken := f + ".taken-" + strconv.Itoa(os.Getpid())
	if err := os.Rename(f, taken); err != nil {
		return ""
	}
	data, err := os.ReadFile(taken)
	_ = os.Remove(taken)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
