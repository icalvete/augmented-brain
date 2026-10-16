package claudecode

import (
	"errors"
	"strings"
	"testing"
)

type fake struct {
	calls []string
	out   map[string]string
	fail  map[string]bool
}

func (f *fake) run(name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, cmd)
	for prefix, out := range f.out {
		if strings.HasPrefix(cmd, prefix) {
			if f.fail[prefix] {
				return out, errors.New("exit status 1")
			}
			return out, nil
		}
	}
	return "", nil
}

func TestInstall(t *testing.T) {
	t.Setenv("AB_MARKETPLACE", "/src/augmented-brain")
	f := &fake{}
	if err := Install(f.run); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"claude plugin marketplace add /src/augmented-brain",
		"claude plugin install brain@augmented-brain --scope user",
	}
	if strings.Join(f.calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("calls:\n%s", strings.Join(f.calls, "\n"))
	}

	// Repeating is fine.
	f = &fake{out: map[string]string{"claude plugin marketplace add": "already installed"},
		fail: map[string]bool{"claude plugin marketplace add": true}}
	if err := Install(f.run); err != nil {
		t.Errorf("a marketplace already added must not fail: %v", err)
	}

	// A real failure is reported with Claude Code's own message.
	f = &fake{out: map[string]string{"claude plugin install": "network is unreachable"},
		fail: map[string]bool{"claude plugin install": true}}
	if err := Install(f.run); err == nil || !strings.Contains(err.Error(), "network is unreachable") {
		t.Errorf("err = %v", err)
	}
}

func TestInstalled(t *testing.T) {
	f := &fake{out: map[string]string{"claude plugin list": "Installed plugins:\n  ❯ brain@augmented-brain\n"}}
	if ok, err := Installed(f.run); !ok || err != nil {
		t.Errorf("installed = %v, %v", ok, err)
	}
	f = &fake{out: map[string]string{"claude plugin list": "No plugins installed"}}
	if ok, _ := Installed(f.run); ok {
		t.Error("reported installed when it is not")
	}
	if Marketplace() != "icalvete/augmented-brain" {
		t.Errorf("default marketplace = %s", Marketplace())
	}
}

func TestUpdate(t *testing.T) {
	f := &fake{}
	if err := Update(f.run); err != nil {
		t.Fatal(err)
	}
	want := "claude plugin marketplace update augmented-brain\nclaude plugin update brain@augmented-brain"
	if got := strings.Join(f.calls, "\n"); got != want {
		t.Errorf("calls:\n%s", got)
	}
	f = &fake{out: map[string]string{"claude plugin update": "plugin not found"},
		fail: map[string]bool{"claude plugin update": true}}
	if err := Update(f.run); err == nil || !strings.Contains(err.Error(), "plugin not found") {
		t.Errorf("err = %v", err)
	}
}

func TestUninstall(t *testing.T) {
	f := &fake{}
	if err := Uninstall(f.run); err != nil {
		t.Fatal(err)
	}
	want := "claude plugin uninstall brain@augmented-brain\nclaude plugin marketplace remove augmented-brain"
	if got := strings.Join(f.calls, "\n"); got != want {
		t.Errorf("calls:\n%s", got)
	}
	// Already gone: fine.
	f = &fake{out: map[string]string{"claude plugin uninstall": "plugin not installed"},
		fail: map[string]bool{"claude plugin uninstall": true}}
	if err := Uninstall(f.run); err != nil {
		t.Errorf("a plugin already gone must not fail: %v", err)
	}
}
