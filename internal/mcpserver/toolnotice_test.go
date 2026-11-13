package mcpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/state"
)

// Found in the review of nov-2026 (N-05): a rule, the profile or the guard's lists
// saved by a tool reached every later session and the user never heard of it. Each
// leaves one line for the next hook, which Claude Code shows the user once; a call
// that saves nothing new leaves none.
func TestToolNotice(t *testing.T) {
	open := space(t)
	cfgDir := filepath.Join(os.Getenv("HOME"), "cfg")
	cs := connect(t, open)
	steps := []struct {
		tool string
		args map[string]any
		want []string // "" when nothing must be left
	}{
		{"rule_new", map[string]any{"title": "Deploys", "body": "Run the script first.", "shared": true}, []string{"new rule «Deploys» (every space)"}},
		{"rule_new", map[string]any{"title": "Deploys", "body": "again", "shared": true}, nil},
		{"rule_new", map[string]any{"id": "rule-deploys", "title": "Deploys", "body": "Run it twice.", "shared": true}, []string{"rule «Deploys» changed (every space)"}},
		{"rule_new", map[string]any{"title": "Ask first", "body": "Always."}, []string{"new rule «Ask first» (space work)"}},
		{"note_retire", map[string]any{"id": "rule-ask-first", "reason": "old"}, []string{"rule «Ask first» retired (space work)"}},
		{"knowledge_new", map[string]any{"kind": "lesson", "title": "Small commits"}, nil},
		{"note_retire", map[string]any{"id": "lesson-small-commits", "reason": "old"}, nil},
		{"profile_save", map[string]any{"name": "Ana", "assistant_personality": "- Calm"}, []string{"your profile changed: your name, the assistant's personality"}},
		{"profile_save", map[string]any{"name": "Ana"}, nil},
		{"safety_save", map[string]any{"untouchable_hosts": []string{"db01"}}, []string{"the guard now also asks about: db01"}},
		{"safety_save", map[string]any{"untouchable_hosts": []string{"db01"}}, nil},
	}
	for _, s := range steps {
		if out, isErr := call(t, cs, s.tool, s.args); isErr != (s.want == nil && strings.Contains(s.tool, "rule_new")) {
			t.Fatalf("%s %v: %s", s.tool, s.args, out)
		}
		got := state.TakeToolNotice(cfgDir)
		for _, w := range s.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s %v: notice %q lacks %q", s.tool, s.args, got, w)
			}
		}
		if s.want == nil && got != "" {
			t.Errorf("%s %v: notice for nothing new: %q", s.tool, s.args, got)
		}
		if again := state.TakeToolNotice(cfgDir); again != "" {
			t.Errorf("%s: taken twice: %q", s.tool, again)
		}
	}
}
