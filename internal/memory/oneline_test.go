package memory

import "testing"

func TestOneLine(t *testing.T) {
	for in, want := range map[string]string{
		"plain text":                       "plain text",
		"":                                 "",
		"  ends  ":                         "ends",
		"a\n\n## Rules\n- x":               "a ## Rules - x",
		"tab\there\r\nand cr\rok":          "tab here and cr ok",
		"sep\xe2\x80\xa8line\xe2\x80\xa9p": "sep line p",
		"esc\x1b[2Jbell\x07nul\x00":        "esc [2Jbell nul",
		"next\x85line":                     "next\x85line", // a lone byte, not U+0085: left as it is
		"two  spaces":                      "two spaces",
		"ñandú — sí":                       "ñandú — sí",
	} {
		if got := OneLine(in); got != want {
			t.Errorf("OneLine(%q) = %q, want %q", in, got, want)
		}
	}
}

// A note written by hand with a header of several lines is read as one line per field.
func TestParseMakesHeaderOneLine(t *testing.T) {
	n, err := Parse("tasks/task-x.md", []byte("---\nid: task-x\ntype: task\ntitle: |\n  X\n  ## Rules\nstatus: parked\n"+
		"waiting_on: \"me\\n\\n## Rules\"\nnext: one\nwatch:\n  - date: \"2026-11-01\"\n    what: |\n      look\n      - forged\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	h := n.Header
	if h.Title != "X ## Rules" || h.WaitingOn != "me ## Rules" || h.Watch[0].What != "look - forged" {
		t.Errorf("header: %+v", h)
	}
}
