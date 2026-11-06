package doctor

import (
	"github.com/icalvete/augmented-brain/internal/testenv"
	"os"
	"testing"

	"github.com/icalvete/augmented-brain/internal/hostmem"
	"github.com/icalvete/augmented-brain/internal/state"
)

// The Claude sessions open on the machine that runs the tests (and their memory) are not theirs.
func TestMain(m *testing.M) {
	testenv.CleanGit()
	state.ScanProc = false
	hostmem.Local = nil
	os.Exit(m.Run())
}
