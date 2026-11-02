package guard

import (
	"strings"
	"testing"
)

func TestFirstOperand(t *testing.T) {
	short := shortOf("p")
	long := oneOf("--port", "-p")
	for _, c := range []struct {
		args       []string
		takesValue func(string) bool
		want       int
	}{
		{nil, short, 0},
		{[]string{"host"}, short, 0},
		{[]string{"-v", "host", "ls"}, short, 1},
		{[]string{"-p", "22", "host"}, short, 2},
		{[]string{"-p22", "host"}, short, 1},     // the value is in the same word
		{[]string{"--port", "22", "h"}, long, 2}, // a long option that takes a value
		{[]string{"--port=22", "h"}, long, 1},
		{[]string{"--", "h"}, short, 1},
		{[]string{"-", "h"}, short, 1}, // a lone - is skipped like an option
		{[]string{"-p"}, short, 1},     // a value that is missing
	} {
		if got := firstOperand(c.args, c.takesValue); got != c.want {
			t.Errorf("firstOperand(%q) = %d, want %d", c.args, got, c.want)
		}
	}
}

func TestCheckHost(t *testing.T) {
	p := Policy{UntouchableHosts: []string{"db01"}}
	for target, want := range map[string]Verdict{
		"":                 Allow,
		"web01":            Allow,
		"db01":             Ask,
		"admin@db01":       Ask,
		"a@b@DB01.example": Ask,
		"db01@web01":       Allow, // the host is what follows the last @
	} {
		if got := p.checkHost("ssh", target).Verdict; got != want {
			t.Errorf("checkHost(%q) = %v, want %v", target, got, want)
		}
	}
	if d := p.checkHost("scp", "u@db01"); !strings.Contains(d.Reason, "scp reaches db01") {
		t.Errorf("reason: %q", d.Reason)
	}
	broken := Policy{ConfigProblem: "bad"}
	if got := broken.checkHost("ssh", "web01").Verdict; got != Ask {
		t.Errorf("unknown configuration: %v, want Ask", got)
	}
	if got := broken.checkHost("ssh", "").Verdict; got != Allow {
		t.Errorf("no host: %v, want Allow", got)
	}
}
