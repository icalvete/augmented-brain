package guard

import "testing"

// A wrapper is stripped by program before simple looks up the program's rule, so a
// name that is both a wrapper and a rule has a rule that never runs. That is how doas
// went through as if it were the command after it.
func TestWrappersHaveNoRule(t *testing.T) {
	if rules["sudo"] == nil || rules["doas"] == nil {
		t.Fatal("the rules of sudo and doas were not found: has the table been renamed or moved?")
	}
	for name := range wrappers {
		if rules[name] != nil {
			t.Errorf("%q is a wrapper and has a rule of its own: program strips it, so the rule never runs", name)
		}
	}
	for name := range reserved {
		if rules[name] != nil {
			t.Errorf("%q is a reserved word and has a rule of its own: program skips it, so the rule never runs", name)
		}
	}
}

func TestDoasIsRoot(t *testing.T) {
	for cmd, want := range map[string]Verdict{
		"doas id":                 Deny,
		"doas -u root id":         Deny,
		"doas -uroot id":          Deny,
		"doas -n apk add jq":      Deny,
		"doas -s":                 Deny,
		"doas sh -c 'id'":         Deny,
		"/usr/bin/doas id":        Deny,
		"env doas id":             Deny,
		"doas -u deploy id":       Allow,
		"doas -u deploy doas id":  Deny, // chained back to root
		"doas -u deploy git push": Deny, // what runs as another user is still checked
		"doas -L":                 Allow,
	} {
		if got := (Policy{}).Command(cmd).Verdict; got != want {
			t.Errorf("%s: got %v, want %v", cmd, got, want)
		}
	}
}
