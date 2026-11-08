package keyring

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

// TestMain doubles as a program that ends with the exit code in AB_TEST_EXIT, to make
// real exit errors on every system.
func TestMain(m *testing.M) {
	if code := os.Getenv("AB_TEST_EXIT"); code != "" {
		n, _ := strconv.Atoi(code)
		os.Exit(n)
	}
	os.Exit(m.Run())
}

func exitWith(t *testing.T, code int) error {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "AB_TEST_EXIT="+strconv.Itoa(code))
	err := cmd.Run()
	if err == nil {
		t.Fatal("no exit error")
	}
	return err
}

// Only a silent 1 from secret-tool, or 44 from security, is "missing": an error while
// asking (no D-Bus session, a locked keychain) is "unknown", never "store it again".
func TestClassify(t *testing.T) {
	one, two, nf := exitWith(t, 1), exitWith(t, 2), exitWith(t, notFoundMac)
	cases := []struct {
		name   string
		goos   string
		err    error
		stderr string
		want   State
	}{
		{"there", "linux", nil, "", Present},
		{"not there", "linux", one, "", Missing},
		{"no D-Bus", "linux", one, `secret-tool: Failed to execute child process "dbus-launch" (No such file or directory)`, Unknown},
		{"other failure", "linux", two, "", Unknown},
		{"could not start", "linux", errors.New("exec: not found"), "", Unknown},
		{"mac there", "darwin", nil, "", Present},
		{"mac not there", "darwin", nf, "security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain.", Missing},
		{"mac locked", "darwin", exitWith(t, 36), "User interaction is not allowed.", Unknown},
		{"mac 1", "darwin", one, "", Unknown},
	}
	for _, c := range cases {
		if got := classify(c.goos, c.err, c.stderr); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
