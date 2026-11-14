package brain

import (
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
)

// sealedNow says whether the guard's lists match the seal, as the guard sees it.
func sealedNow(t *testing.T, cfgDir string) bool {
	t.Helper()
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	g, err := config.LoadGuard(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	seal, err := state.ReadSeal(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	return seal != "" && seal == c.GuardFingerprint(g)
}

func sealCurrent(t *testing.T, cfgDir string) {
	t.Helper()
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	g, err := config.LoadGuard(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.WriteSeal(cfgDir, c.GuardFingerprint(g)); err != nil {
		t.Fatal(err)
	}
}

// Found in the review of 2026-11-13 (P4): after safety_save the lists no longer matched
// the seal, and every commit, push, aws or ssh asked until the next session. Adding only
// makes the guard ask more, so lists the user had accepted stay accepted.
func TestSafetySaveKeepsAcceptedListsSealed(t *testing.T) {
	b := open(t, "en")
	sealCurrent(t, b.ConfigDir)
	if _, err := b.SaveSafety(SafetyInput{ProductionAWSProfiles: []string{" prod "}, UntouchableHosts: []string{"db01"}}); err != nil {
		t.Fatal(err)
	}
	if !sealedNow(t, b.ConfigDir) {
		t.Error("lists accepted before and only added to are left unsealed")
	}
	// Saving what is already there changes nothing and keeps the seal.
	if _, err := b.SaveSafety(SafetyInput{UntouchableHosts: []string{"db01"}}); err != nil {
		t.Fatal(err)
	}
	if !sealedNow(t, b.ConfigDir) {
		t.Error("saving a host already listed left the lists unsealed")
	}
}

// Lists changed some other way before the call stay unsealed: an addition never
// launders a change the user has not accepted.
func TestSafetySaveLeavesUnacceptedChange(t *testing.T) {
	b := open(t, "en")
	sealCurrent(t, b.ConfigDir)
	if err := config.SetSpaceList(b.ConfigDir, "work", "git_allowed_repos", []string{"~/somewhere"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SaveSafety(SafetyInput{UntouchableHosts: []string{"db01"}}); err != nil {
		t.Fatal(err)
	}
	if sealedNow(t, b.ConfigDir) {
		t.Error("safety_save sealed a change of the git lists the user never accepted")
	}
}

// Without a seal (none yet) the session start takes the first one; safety_save does not.
func TestSafetySaveNoSealYet(t *testing.T) {
	b := open(t, "en")
	if _, err := b.SaveSafety(SafetyInput{UntouchableHosts: []string{"db01"}}); err != nil {
		t.Fatal(err)
	}
	if seal, err := state.ReadSeal(b.ConfigDir); err != nil || seal != "" {
		t.Errorf("a seal was written where there was none: %q %v", seal, err)
	}
}

// Something else writing the lists between safety_save's write and its seal leaves them
// unsealed: only the old lists plus what was added are sealed.
func TestSafetySaveConcurrentChange(t *testing.T) {
	b := open(t, "en")
	sealCurrent(t, b.ConfigDir)
	safetyWritten = func() {
		if err := config.SetSpaceList(b.ConfigDir, "work", "git_allowed_repos", []string{"~/somewhere"}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { safetyWritten = func() {} })
	if _, err := b.SaveSafety(SafetyInput{UntouchableHosts: []string{"db01"}}); err != nil {
		t.Fatal(err)
	}
	if sealedNow(t, b.ConfigDir) {
		t.Error("a change made while safety_save wrote was sealed with it")
	}
}
