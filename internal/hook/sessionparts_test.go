package hook

import (
	"bytes"
	"encoding/json"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/session"
)

// startParts runs the nine session-start hooks at once, in a random order, and returns
// each part's text.
func startParts(t *testing.T, cfgDir string, in map[string]any) []string {
	t.Helper()
	data, _ := json.Marshal(in)
	out := make([]string, session.Parts)
	errs := make([]error, session.Parts)
	var wg sync.WaitGroup
	for _, n := range rand.Perm(session.Parts) {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			var buf bytes.Buffer
			if errs[n] = SessionStartPart(bytes.NewReader(data), &buf, cfgDir, n+1); errs[n] != nil {
				return
			}
			var o struct {
				HookSpecificOutput struct {
					AdditionalContext string `json:"additionalContext"`
				} `json:"hookSpecificOutput"`
			}
			errs[n] = json.Unmarshal(buf.Bytes(), &o)
			out[n] = o.HookSpecificOutput.AdditionalContext
		}(n)
	}
	wg.Wait()
	for n, err := range errs {
		if err != nil {
			t.Fatalf("part %d: %v", n+1, err)
		}
	}
	return out
}

// Nine parts at once and in any order give what each part built on its own gives: part
// 1 builds them all and the others take theirs from it.
func TestSessionPartsAtOnce(t *testing.T) {
	cfgDir, work := setupSpace(t)
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := c.SpaceAt(work)
	for round := range 5 {
		in := map[string]any{"cwd": work, "session_id": "sess-" + string(rune('a'+round)), "source": "startup"}
		got := startParts(t, cfgDir, in)
		for n := 2; n <= session.Parts; n++ {
			if want := session.Part(c, s, "", n); got[n-1] != want {
				t.Errorf("round %d, part %d:\n%s\n--- built on its own:\n%s", round, n, got[n-1], want)
			}
		}
		if !strings.Contains(got[0], "# augmented brain · space work") {
			t.Errorf("round %d: part 1 is %q", round, got[0])
		}
		if _, err := os.Stat(partsFile(cfgDir, in["session_id"].(string))); err != nil {
			t.Errorf("round %d: part 1 left no parts: %v", round, err)
		}
	}
}

// A part takes nothing from the parts of another session, another source or an
// earlier start of the same one: it waits, and then builds its own.
func TestSessionPartsOfAnotherStart(t *testing.T) {
	cfgDir, work := setupSpace(t)
	old := PartWait
	PartWait = 50 * time.Millisecond
	t.Cleanup(func() { PartWait = old })
	bogus := make([]string, session.Parts)
	for i := range bogus {
		bogus[i] = "not this start's"
	}
	write := func(p storedParts) {
		t.Helper()
		p.Parts = bogus
		data, _ := json.Marshal(p)
		file := partsFile(cfgDir, "sess-1")
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	part2 := func() string {
		o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 2) },
			map[string]any{"cwd": work, "session_id": "sess-1", "source": "compact"})
		v, _ := o["additionalContext"].(string)
		return v
	}
	for _, p := range []storedParts{
		{Session: "sess-1", Space: "work", Source: "startup", Written: time.Now()},                   // another source
		{Session: "sess-1", Space: "home", Source: "compact", Written: time.Now()},                   // another space
		{Session: "sess-1", Space: "work", Source: "compact", Written: time.Now().Add(-time.Minute)}, // an earlier compaction
	} {
		write(p)
		if got := part2(); strings.Contains(got, "not this start's") {
			t.Errorf("took the parts of %+v", p)
		}
	}
	write(storedParts{Session: "sess-1", Space: "work", Source: "compact", Written: time.Now()})
	if got := part2(); got != "not this start's" {
		t.Errorf("did not take this start's parts: %q", got)
	}
}

// A session that ends takes its parts away, and those left by sessions that never
// ended go a day later.
func TestSessionPartsRemoved(t *testing.T) {
	cfg := t.TempDir()
	now := time.Now()
	for _, id := range []string{"ending", "old", "recent"} {
		if !saveParts(cfg, input{SessionID: id}, "work", []string{"x"}) {
			t.Fatal("not saved")
		}
	}
	past := now.Add(-25 * time.Hour)
	if err := os.Chtimes(partsFile(cfg, "old"), past, past); err != nil {
		t.Fatal(err)
	}
	removeParts(cfg, "ending", now)
	for id, want := range map[string]bool{"ending": false, "old": false, "recent": true} {
		if _, err := os.Stat(partsFile(cfg, id)); (err == nil) != want {
			t.Errorf("%s: kept %v, want %v", id, err == nil, want)
		}
	}
	if partsFile(cfg, "../x") != "" || partsFile(cfg, "") != "" {
		t.Error("a session id that cannot name a file names one")
	}
}
