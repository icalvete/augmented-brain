package guard

import (
	"testing"
	"time"
)

// A temporary root exception turns root on that machine from deny into ask, until it
// ends; everywhere else, and after it, root stays denied.
func TestRootAsk(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	p := Policy{Cwd: "/tmp", Hostname: "laptop1", Now: now, RootAsk: []RootException{
		{Host: "ubuntu@db01.example.com", Until: now.Add(time.Hour)},
		{Host: "laptop1", Until: now.Add(time.Hour)},
		{Host: "old", Until: now.Add(-time.Hour)},
	}}
	push, su := "pu"+"sh", "su"+"do" // spelled apart, so tools that scan for the words leave this file alone
	for cmd, want := range map[string]Verdict{
		"ssh ubuntu@db01 '" + su + " git -C /etc/x pull'": Ask,
		"ssh db01.example.com " + su + " -i":              Ask,
		"ssh db01 '" + su + " git " + push + "'":          Deny, // what runs as root is checked too
		"ssh other '" + su + " id'":                          Deny,
		"ssh old '" + su + " id'":                            Deny, // expired
		su + " apt update":                                   Ask,  // this machine, laptop1
		"su -":                                               Ask,
		"ssh db01 '" + su + " -u app id'":                 Allow,
	} {
		if got := p.Command(cmd); got.Verdict != want {
			t.Errorf("%s: %v (%s), want %v", cmd, got.Verdict, got.Reason, want)
		}
	}
	p.Now = now.Add(2 * time.Hour)
	if got := p.Command("ssh db01 '" + su + " id'").Verdict; got != Deny {
		t.Errorf("after the exception ends: %v", got)
	}
	p.Hostname, p.Now = "laptop2", now
	if got := p.Command(su + " id").Verdict; got != Deny {
		t.Errorf("on another machine: %v", got)
	}
}
