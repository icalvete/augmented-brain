package stable

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestMissing(t *testing.T) {
	frozen := Surface{"task_new": {"title: string", "required title"}, "capture": {}}
	now := Surface{"task_new": {"title: integer", "required title", "area: string"}}
	got := Missing(frozen, now)
	want := []string{`"capture"`, `"title: string" of "task_new"`}
	if !slices.Equal(got, want) {
		t.Errorf("gone: %q, want %q", got, want)
	}
	if got := Missing(now, frozen); !slices.Equal(got, []string{`"area: string" of "task_new"`, `"title: integer" of "task_new"`}) {
		t.Errorf("new: %q", got)
	}
}

// Freezing adds what is new and keeps what is frozen, even what the program dropped.
func TestFreezeOnlyAdds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testdata", "x.json")
	t.Setenv(UpdateEnv, "1")
	Check(t, path, Surface{"a": {"1"}})
	Check(t, path, Surface{"a": {"1", "2"}, "b": {}})
	if got := Read(t, path); !slices.Equal(got["a"], []string{"1", "2"}) || got["b"] == nil {
		t.Errorf("frozen: %v", got)
	}
}
