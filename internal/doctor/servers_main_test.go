package doctor

import (
	"os"
	"testing"

	"github.com/icalvete/augmented-brain/internal/state"
)

// The Claude sessions open on the machine that runs the tests are not theirs.
func TestMain(m *testing.M) {
	state.ScanProc = false
	os.Exit(m.Run())
}
