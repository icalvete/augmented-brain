package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/stable"
)

// typeOf is a parameter's JSON schema type in a few words: "string", "array of string".
func typeOf(raw json.RawMessage) string {
	var p struct {
		Type  any             `json:"type"`
		Items json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return "?"
	}
	var ts []string
	switch v := p.Type.(type) {
	case string:
		ts = []string{v}
	case []any:
		for _, x := range v {
			if s, ok := x.(string); ok && s != "null" {
				ts = append(ts, s)
			}
		}
	}
	slices.Sort(ts)
	t := strings.Join(ts, "|")
	if t == "" {
		t = "any"
	}
	if len(p.Items) > 0 {
		t += " of " + typeOf(p.Items)
	}
	return t
}

// The MCP tools are part of the compatibility promise: every tool keeps its name, every
// parameter its name and type, and no tool gets a new required parameter. They are
// frozen in testdata/stable-tools.json: "param: type" and "required param".
func TestStableTools(t *testing.T) {
	cs := connect(t, func() (*brain.Brain, error) { return nil, errors.New("listing only") })
	now := stable.Surface{}
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
			Required   []string                   `json:"required"`
		}
		if err := json.Unmarshal(b, &schema); err != nil {
			t.Fatalf("%s: %v", tl.Name, err)
		}
		attrs := []string{}
		for p, raw := range schema.Properties {
			attrs = append(attrs, p+": "+typeOf(raw))
		}
		for _, r := range schema.Required {
			attrs = append(attrs, "required "+r)
		}
		now[tl.Name] = attrs
	}
	path := filepath.Join("testdata", "stable-tools.json")
	// A required parameter the frozen tool did not require breaks every call written
	// without it. A new tool may require what it likes.
	frozen := stable.Read(t, path)
	var breaks []string
	for name, attrs := range now {
		old, ok := frozen[name]
		if !ok {
			continue
		}
		for _, a := range attrs {
			if strings.HasPrefix(a, "required ") && !slices.Contains(old, a) {
				breaks = append(breaks, fmt.Sprintf("%q now has %q", name, a))
			}
		}
	}
	stable.Check(t, path, now, breaks...)
}
