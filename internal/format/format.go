// Package format keeps the version of the files augmented brain writes, and moves them
// forward when a new version of the program changes how they are written.
//
// There are two versioned things. Each space's notes say their version in
// .augmented-brain/format inside the memory folder, so the version travels with the
// backup and a restore on another machine knows what it holds. The configuration
// folder (the configuration and the profile) says its version in config.toml, as
// format = N. A missing version is 1: every release before versions were written
// wrote format 1.
//
// Moving forward runs the migrations one step at a time, after a full copy of what
// they change, and writes the new version after each step: a migration that stops
// half way runs again from that step next time, so every step must be safe to repeat.
// Notes of a newer version than the program knows are never touched.
package format

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// MetaDir is the program's folder inside the memory folder (the backup keeps the
// space's configuration there too); File, the name of the version file in it.
const (
	MetaDir = ".augmented-brain"
	File    = "format"
)

// CopiesDir, in the configuration folder, keeps the copy made before each migration.
const CopiesDir = "migrations"

// Migration moves a folder from version From to From+1. Run must be safe to repeat.
type Migration struct {
	From int
	What string // one line, for the user
	Run  func(dir string) error
}

// NotesMigrations and ConfigMigrations are the steps, in order, the first from
// version 1; tests replace them.
var (
	NotesMigrations  []Migration
	ConfigMigrations []Migration
)

// Notes is the version of the notes this program writes, and Config that of the
// configuration folder: one more than the steps that lead to it.
func Notes() int  { return 1 + len(NotesMigrations) }
func Config() int { return 1 + len(ConfigMigrations) }

// NewerError says the files were written by a newer program than this one. It is
// said in Lang, the user's language; when that is not known (it can come before the
// configuration, and its language, could be read), in the system's.
type NewerError struct {
	What    string // the memory folder or the configuration folder
	Version int
	Known   int
	Lang    string
}

func (e *NewerError) Error() string {
	return i18n.T(cmp.Or(e.Lang, i18n.FromEnv()), "format.newer", e.What, e.Version, e.Known)
}

// stepError is an error of a migration, said in the user's language, that keeps the
// error under it for errors.Is and errors.As.
type stepError struct {
	msg string
	err error
}

func (e *stepError) Error() string { return e.msg }
func (e *stepError) Unwrap() error { return e.err }

// Of reads the version of a memory folder: 1 when it says none. What it says when it
// cannot is in the system's language.
func Of(memDir string) (int, error) { return of(memDir, i18n.FromEnv()) }

// of is Of, saying what it cannot read in lang.
func of(memDir, lang string) (int, error) {
	data, err := os.ReadFile(filepath.Join(memDir, MetaDir, File))
	if errors.Is(err, fs.ErrNotExist) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || v < 1 {
		return 0, errors.New(i18n.T(lang, "format.notversion", filepath.Join(memDir, MetaDir, File), strings.TrimSpace(string(data))))
	}
	return v, nil
}

// Write records the version of a memory folder.
func Write(memDir string, v int) error {
	dir := filepath.Join(memDir, MetaDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return paths.WriteAtomic(filepath.Join(dir, File), []byte(strconv.Itoa(v)+"\n"), 0o600)
}

// CheckNotes refuses notes written by a newer program. It reads one small file, so
// every open of a space can afford it. lang is the user's language ("": the system's).
func CheckNotes(memDir, lang string) error {
	lang = cmp.Or(lang, i18n.FromEnv())
	v, err := of(memDir, lang)
	if err != nil {
		return err
	}
	if v > Notes() {
		return &NewerError{What: memDir, Version: v, Known: Notes(), Lang: lang}
	}
	return nil
}

// Result says what Ensure did.
type Result struct {
	From, To int
	Copy     string   // where the copy made before migrating is; empty when nothing ran
	Done     []string // what each step did
}

// Migrated reports whether any step ran.
func (r Result) Migrated() bool { return r.From != r.To }

// EnsureNotes brings a memory folder to the version this program writes: it records
// the version when the folder says none, and runs the missing steps, after copying the
// whole folder into <cfgDir>/migrations/. name names the copy (the space); lang is the
// language of what it says when it cannot ("": the system's).
func EnsureNotes(memDir, cfgDir, name, lang string, now time.Time) (Result, error) {
	lang = cmp.Or(lang, i18n.FromEnv())
	v, err := of(memDir, lang)
	if err != nil {
		return Result{}, err
	}
	return ensure(memDir, cfgDir, "notes-"+name, lang, v, Notes(), NotesMigrations, now,
		func(n int) error { return Write(memDir, n) })
}

// EnsureConfig does the same for the configuration folder, given its version as read
// from config.toml and how to write a new one there. The copy leaves out the copies
// of earlier migrations and the machine's state.
func EnsureConfig(cfgDir string, v int, write func(int) error, lang string, now time.Time) (Result, error) {
	return ensure(cfgDir, cfgDir, "config", cmp.Or(lang, i18n.FromEnv()), v, Config(), ConfigMigrations, now, write)
}

func ensure(dir, cfgDir, name, lang string, v, target int, steps []Migration, now time.Time, write func(int) error) (Result, error) {
	r := Result{From: v, To: v}
	switch {
	case v > target:
		return r, &NewerError{What: dir, Version: v, Known: target, Lang: lang}
	case v == target:
		return r, write(v) // records it when it was only implied; cheap
	}
	r.Copy = filepath.Join(cfgDir, CopiesDir, fmt.Sprintf("%s-%d-%s", name, v, now.Format("20060102-150405")))
	if err := paths.CopyTree(dir, r.Copy, CopiesDir, "state", ".git"); err != nil { // the machine's state and earlier copies stay out
		return r, &stepError{i18n.T(lang, "format.copying", dir, target, err), err}
	}
	for r.To < target {
		var step *Migration
		for i := range steps {
			if steps[i].From == r.To {
				step = &steps[i]
			}
		}
		if step == nil {
			return r, errors.New(i18n.T(lang, "format.noway", r.To, r.To+1))
		}
		if err := step.Run(dir); err != nil {
			return r, &stepError{i18n.T(lang, "format.moving", dir, r.To, r.To+1, step.What, err, r.Copy), err}
		}
		if err := write(r.To + 1); err != nil {
			return r, err
		}
		r.To++
		r.Done = append(r.Done, step.What)
	}
	return r, nil
}
