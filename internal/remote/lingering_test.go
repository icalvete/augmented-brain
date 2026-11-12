package remote

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"
)

func TestLingering(t *testing.T) {
	r := newRestartFake(t)
	now := time.Date(2026, 11, 12, 18, 0, 0, 0, time.UTC)
	since := func(pid int, ago time.Duration) {
		p := filepath.Join(r.sys.ClaudeDir, "sessions", strconv.Itoa(pid)+".json")
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		ms := now.Add(-ago).UnixMilli()
		if err := os.WriteFile(p, []byte(`{"pid":`+strconv.Itoa(pid)+`,"statusUpdatedAt":`+strconv.FormatInt(ms, 10)+`}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	since(901, 20*time.Hour) // archived in Desktop: blocked, no task, long untouched
	since(902, time.Hour)    // blocked, but recently: someone may be on it
	since(903, 20*time.Hour) // a task's, opened by session start
	since(905, 20*time.Hour) // busy
	r.status[901], r.status[902], r.status[903], r.status[905] = "idle", "idle", "idle", "busy"
	dir := launchRecords(r.cfgDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "task-a.json"), []byte(`{"id":"task0903","task":"task-a","folder":"/x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	inner := r.agentList
	r.agentList = func() []map[string]any {
		return append(inner(),
			map[string]any{"pid": 901, "id": "arch0901", "kind": "background", "name": "archivada", "state": "blocked", "status": "idle"},
			map[string]any{"pid": 902, "id": "recn0902", "kind": "background", "name": "reciente", "state": "blocked", "status": "idle"},
			map[string]any{"pid": 903, "id": "task0903", "kind": "background", "name": "de una tarea", "state": "blocked", "status": "idle"},
			map[string]any{"pid": 904, "id": "done0904", "kind": "background", "name": "terminada", "state": "done", "status": "idle"},
			map[string]any{"pid": 905, "id": "busy0905", "kind": "background", "name": "trabajando", "state": "working", "status": "busy"},
			map[string]any{"pid": 906, "id": "noag0906", "kind": "background", "name": "sin fichero", "state": "blocked", "status": "idle"},
		)
	}
	list, err := r.sys.Lingering(r.cfgDir, now)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, se := range list {
		got = append(got, se.ID)
	}
	slices.Sort(got)
	if want := []string{"arch0901", "done0904"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
