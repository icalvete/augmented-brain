package confighistory

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func put(t *testing.T, dir, name, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRecordOnlyWhenChanged(t *testing.T) {
	dir := t.TempDir()
	if kept, err := Record(dir, ByCommand, "init"); err != nil || kept {
		t.Fatalf("no configuration yet: kept=%v err=%v", kept, err)
	}
	put(t, dir, "config.toml", "language = \"en\"\n")
	if kept, err := Record(dir, ByCommand, "init"); err != nil || !kept {
		t.Fatalf("first version: kept=%v err=%v", kept, err)
	}
	if kept, _ := Record(dir, ByPanel, ""); kept {
		t.Error("nothing changed, yet a version was kept")
	}
	if ch, _ := Changed(dir); ch {
		t.Error("Changed with nothing changed")
	}
	put(t, dir, "guard.toml", "untouchable_repos = []\n")
	if ch, _ := Changed(dir); !ch {
		t.Error("a new file is not a change")
	}
	if kept, _ := Record(dir, ByOutside, ""); !kept {
		t.Error("a new file kept no version")
	}
	if first, _ := List(dir); first[1].By != ByCommand {
		t.Errorf("a first version with who: %+v", first[1])
	}
	vs, err := List(dir)
	if err != nil || len(vs) != 2 {
		t.Fatalf("versions: %d, %v", len(vs), err)
	}
	if vs[0].By != ByOutside || vs[1].By != ByCommand || vs[1].Detail != "init" {
		t.Errorf("newest first, with who: %+v %+v", vs[0], vs[1])
	}
	if _, ok := vs[1].Files["guard.toml"]; ok {
		t.Error("the first version holds a file that did not exist then")
	}
}

func TestFirstIsWhereItStarts(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, "config.toml", "x\n")
	Record(dir, ByOutside, "")
	if vs, _ := List(dir); len(vs) != 1 || vs[0].By != ByFirst {
		t.Errorf("first version: %+v", vs)
	}
}

func TestKeepsTen(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	for i := range Keep + 3 {
		put(t, dir, "config.toml", strings.Repeat("x", i+1))
		if _, err := record(dir, ByCommand, "", start.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	vs, _ := List(dir)
	if len(vs) != Keep {
		t.Fatalf("%d versions kept, want %d", len(vs), Keep)
	}
	if vs[0].Files["config.toml"] != strings.Repeat("x", Keep+3) || vs[Keep-1].Files["config.toml"] != strings.Repeat("x", 4) {
		t.Error("the newest were not the ones kept")
	}
}

func TestRestore(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, "config.toml", "a = 1\n")
	Record(dir, ByCommand, "")
	first, _ := List(dir)
	put(t, dir, "config.toml", "a = 2\n")
	put(t, dir, "guard.toml", "g\n")
	// Changed by hand: no Record. Restore keeps it as a version before going back.
	if err := Restore(dir, first[0].ID, nil); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "config.toml")); string(data) != "a = 1\n" {
		t.Errorf("config.toml: %q", data)
	}
	if _, err := os.Stat(filepath.Join(dir, "guard.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Error("a file the version did not have is still there")
	}
	vs, _ := List(dir)
	if len(vs) != 3 || vs[0].By != ByRestore || vs[0].Detail != first[0].ID || vs[1].By != ByOutside {
		t.Fatalf("versions after restore: %+v", vs)
	}
	if vs[1].Files["config.toml"] != "a = 2\n" {
		t.Error("the configuration before going back was not kept")
	}

	// A check that fails puts everything back.
	err := Restore(dir, vs[1].ID, func() error { return errors.New("bad") })
	if err == nil || err.Error() != "bad" {
		t.Fatalf("err = %v", err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "config.toml")); string(data) != "a = 1\n" {
		t.Errorf("not put back: %q", data)
	}
	if after, _ := List(dir); len(after) != 3 {
		t.Errorf("a failed restore kept a version: %d", len(after))
	}
}

func TestGetRefusesOtherPaths(t *testing.T) {
	dir := t.TempDir()
	for _, id := range []string{"", "../config", `..\x`, "a/b", "x y"} {
		if _, err := Get(dir, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%q) = %v", id, err)
		}
	}
}

func TestCompare(t *testing.T) {
	older := map[string]string{"config.toml": "a\nb\nc\nd\ne\nf\ng\n", "remote.toml": "r\n"}
	newer := map[string]string{"config.toml": "a\nb\nc\nD\ne\nf\ng\n", "guard.toml": "x\n"}
	ds := Compare(older, newer, 1)
	if len(ds) != 3 {
		t.Fatalf("%d files, want 3: %+v", len(ds), ds)
	}
	if ds[0].Name != "config.toml" || ds[1].Name != "guard.toml" || !ds[1].New || ds[2].Name != "remote.toml" || !ds[2].Gone {
		t.Errorf("files: %+v", ds)
	}
	var got []string
	for _, l := range ds[0].Lines {
		got = append(got, string(l.Op)+l.Text)
	}
	if want := "~| c|-d|+D| e|~"; strings.Join(got, "|") != want {
		t.Errorf("lines %q, want %q", strings.Join(got, "|"), want)
	}
}

// Two long versions of a file that differ in the middle are compared without building
// a table of every line against every other: past maxTable, the part that changed is
// shown as replaced whole, and the lines the same at both ends stay as they are.
func TestCompareLongFiles(t *testing.T) {
	var a, b strings.Builder
	a.WriteString("top\n")
	b.WriteString("top\n")
	for i := range 3000 {
		fmt.Fprintf(&a, "old %d\n", i)
		fmt.Fprintf(&b, "new %d\n", i)
	}
	a.WriteString("bottom\n")
	b.WriteString("bottom\n")
	ds := Compare(map[string]string{"profile.md": a.String()}, map[string]string{"profile.md": b.String()}, 1)
	if len(ds) != 1 {
		t.Fatalf("%+v", ds)
	}
	ls := ds[0].Lines
	if ls[0].Op != ' ' || ls[0].Text != "top" || ls[1].Op != '-' || ls[3001].Op != '+' || ls[len(ls)-1].Text != "bottom" {
		t.Errorf("lines: %+v ... %+v", ls[:3], ls[len(ls)-2:])
	}
}

func TestWhoNamesThePanelPage(t *testing.T) {
	for detail, want := range map[string]string{"/guard/list": "panel, página Guardia", "/nope/x": "panel", "": "panel"} {
		if got := Who("es", Version{By: ByPanel, Detail: detail}); got != want {
			t.Errorf("%q: %q, want %q", detail, got, want)
		}
	}
}
