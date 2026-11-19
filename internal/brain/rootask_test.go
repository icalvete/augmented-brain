package brain

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
)

func rootAsk(t *testing.T, cfgDir string) []string {
	t.Helper()
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	return c.RootAsk
}

func TestSaveRootException(t *testing.T) {
	b := open(t, "en")
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	sealCurrent(t, b.ConfigDir)
	out, err := b.SaveRootException("db01", "2026-11-25", now)
	if err != nil || !strings.Contains(out, `"db01 until 2026-11-25"`) {
		t.Fatalf("%s %v", out, err)
	}
	if got := rootAsk(t, b.ConfigDir); !slices.Equal(got, []string{"db01 until 2026-11-25"}) {
		t.Errorf("root_ask = %v", got)
	}
	if !sealedNow(t, b.ConfigDir) {
		t.Error("lists accepted before are left unsealed: the exception would not count")
	}
	if n := state.TakeGuardNotice(b.ConfigDir); !strings.Contains(n, "db01") || !strings.Contains(n, "2026-11-25 23:59") {
		t.Errorf("the user is not told: %q", n)
	}
	// The same machine again: its date is replaced, not added.
	if _, err := b.SaveRootException("DB01", "2026-11-21T10:00Z", now); err != nil {
		t.Fatal(err)
	}
	if got := rootAsk(t, b.ConfigDir); !slices.Equal(got, []string{"DB01 until 2026-11-21T10:00Z"}) {
		t.Errorf("root_ask = %v", got)
	}
	// What config.NewRootAsk refuses is not written.
	for _, until := range []string{"2026-11-18", "2027-01-01", "soon"} {
		if _, err := b.SaveRootException("web01", until, now); err == nil {
			t.Errorf("%s was saved", until)
		}
	}
	if _, err := b.SaveRootException("a b", "2026-11-20", now); err == nil {
		t.Error("a host with a space was saved")
	}
	if got := rootAsk(t, b.ConfigDir); len(got) != 1 {
		t.Errorf("root_ask = %v", got)
	}
}

// Lists that were already changed some other way stay unsealed: the user has not seen
// that other change, and confirming the exception is not accepting it.
func TestSaveRootExceptionLeavesChangedListsUnsealed(t *testing.T) {
	b := open(t, "en")
	sealCurrent(t, b.ConfigDir)
	if err := config.SetTopList(b.ConfigDir, "untouchable_hosts", []string{"x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SaveRootException("db01", "2026-11-25", time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if sealedNow(t, b.ConfigDir) {
		t.Error("lists changed outside were sealed")
	}
	// And something that writes the file between the write and the seal is not sealed.
	b2 := open(t, "en")
	sealCurrent(t, b2.ConfigDir)
	rootWritten = func() { _ = config.SetTopList(b2.ConfigDir, "untouchable_hosts", []string{"y"}) }
	defer func() { rootWritten = func() {} }()
	if _, err := b2.SaveRootException("db01", "2026-11-25", time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if sealedNow(t, b2.ConfigDir) {
		t.Error("a change made meanwhile was sealed")
	}
}

func TestRemoveRootException(t *testing.T) {
	b := open(t, "es")
	if err := config.SetTopList(b.ConfigDir, "root_ask", []string{"db01 until 2026-11-25", "web01 until 2026-11-26", "roto"}); err != nil {
		t.Fatal(err)
	}
	sealCurrent(t, b.ConfigDir)
	now := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	if out, err := b.RemoveRootException("DB01", now); err != nil || !strings.Contains(out, "db01 until 2026-11-25") {
		t.Fatalf("%s %v", out, err)
	}
	if _, err := b.RemoveRootException("roto", now); err != nil {
		t.Fatal(err)
	}
	if got := rootAsk(t, b.ConfigDir); !slices.Equal(got, []string{"web01 until 2026-11-26"}) {
		t.Errorf("root_ask = %v", got)
	}
	if !sealedNow(t, b.ConfigDir) {
		t.Error("taking one out unsealed accepted lists")
	}
	if n := state.TakeGuardNotice(b.ConfigDir); !strings.Contains(n, "quitada") {
		t.Errorf("notice: %q", n)
	}
	if out, err := b.RemoveRootException("nadie", now); err != nil || !strings.Contains(out, "nothing to change") {
		t.Errorf("%s %v", out, err)
	}
}
