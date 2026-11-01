package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
)

// `check` (and `space`, `backup`) in a folder outside every space use the default space,
// as the hooks and the MCP server do, instead of saying the folder belongs to none.
func TestCheckUsesTheDefaultSpace(t *testing.T) {
	_, cfg := home(t)
	if err := config.SetDefaultSpace(cfg, "personal"); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(wd) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"check"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("check outside every space: %d\n%s%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "personal") {
		t.Errorf("check did not use the default space:\n%s", out.String())
	}
}
