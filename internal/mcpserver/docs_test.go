package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/doccheck"
)

// toolParams are the tools the server offers, as a client lists them, each with the
// names of its input parameters.
func toolParams(t *testing.T) map[string][]string {
	t.Helper()
	cs := connect(t, func() (*brain.Brain, error) { return nil, errors.New("listing only") })
	out := map[string][]string{}
	for tl, err := range cs.Tools(context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(tl.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(b, &schema); err != nil {
			t.Fatalf("%s: %v", tl.Name, err)
		}
		params := []string{}
		for p := range schema.Properties {
			params = append(params, p)
		}
		sort.Strings(params)
		out[tl.Name] = params
	}
	return out
}

// The reference page of the MCP tools, in each language, has a section per tool
// (## `tool_name`) that gives each of its parameters in backticks. It fails until the
// page covers the server: docs cannot drift. AB_DOCS_DIR points it at pages elsewhere.
func TestToolsDocumented(t *testing.T) {
	params := toolParams(t)
	var order []string
	want := map[string][]string{}
	for name, ps := range params {
		h := "`" + name + "`"
		order = append(order, h)
		want[h] = ps
	}
	sort.Strings(order)
	dir, err := doccheck.Dir()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range doccheck.Files("mcp-tools") {
		p, err := doccheck.Read(dir, f)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		// A tool's heading is code: ## `task_new`. Other headings are prose.
		ours := func(h string) bool { return strings.HasPrefix(h, "`") }
		for _, g := range doccheck.Check(p, order, want, ours, "tool") {
			t.Error(g)
		}
	}
}
