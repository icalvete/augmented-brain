package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// innocentCommands reads testdata/innocent-commands.txt: entries separated by a line
// "----", lines starting with ## left out. Each comes with the line it starts on.
func innocentCommands(t *testing.T) (cmds []string, lines []int) {
	t.Helper()
	data, err := os.ReadFile("testdata/innocent-commands.txt")
	if err != nil {
		t.Fatal(err)
	}
	var cur []string
	start := 0
	flush := func() {
		if c := strings.TrimSpace(strings.Join(cur, "\n")); c != "" {
			cmds, lines = append(cmds, c), append(lines, start)
		}
		cur = nil
	}
	for n, l := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		switch {
		case l == "----":
			flush()
		case strings.HasPrefix(l, "##"):
		default:
			if len(cur) == 0 {
				start = n + 1
			}
			cur = append(cur, l)
		}
	}
	flush()
	return cmds, lines
}

// TestInnocentCommands is the net under false positives: everyday commands, in the
// shapes real sessions use, must pass the guard with no question and no refusal. On
// 9-nov `sh -n` on a script written by the same line was refused, and
// `echo $(( $(date +%s)-1 ))` asked; both are in the corpus now.
func TestInnocentCommands(t *testing.T) {
	cfgDir, work := setupSpace(t)
	cmds, lines := innocentCommands(t)
	if len(cmds) < 100 {
		t.Fatalf("only %d commands read from the corpus: is its format broken?", len(cmds))
	}
	for i, cmd := range cmds {
		in, _ := json.Marshal(map[string]any{"cwd": work, "tool_name": "Bash", "session_id": "innocent",
			"tool_input": map[string]any{"command": cmd}})
		var out bytes.Buffer
		if err := PreToolUse(bytes.NewReader(in), &out, cfgDir); err != nil {
			t.Fatal(err)
		}
		if out.Len() > 0 {
			t.Errorf("testdata/innocent-commands.txt:%d is stopped: %s\n%s", lines[i], strings.TrimSpace(out.String()), cmd)
		}
	}
}
