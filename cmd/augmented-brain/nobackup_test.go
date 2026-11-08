package main

import (
	"bytes"
	"strings"
	"testing"
)

// Without a backup, losing the computer loses the notes: doctor says so for each space
// (as information, not a failure), update says it goes ahead with no copy, and
// uninstall --all warns that those notes will be gone for good.
func TestSaysThereIsNoBackup(t *testing.T) {
	home(t) // two spaces, trabajo and personal, in Spanish, none backed up
	var out, errOut bytes.Buffer
	run([]string{"doctor"}, nil, &out, &errOut)
	for _, sp := range []string{"trabajo", "personal"} {
		if !strings.Contains(out.String(), "el espacio "+sp+" no tiene copia de seguridad") {
			t.Errorf("doctor does not say %s has no backup:\n%s", sp, out.String())
		}
	}
	if !strings.Contains(out.String(), "configura la copia de seguridad") {
		t.Errorf("doctor does not say how:\n%s", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := beforeUpdate("es", false, &out, &errOut); code != 0 {
		t.Fatalf("no backup stopped the update: %d %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Ningún espacio tiene copia de seguridad") {
		t.Errorf("update does not say it goes ahead with no copy:\n%s", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run([]string{"uninstall", "--all"}, strings.NewReader("n\n"), &out, &errOut); code == 0 {
		t.Errorf("answering no went ahead")
	}
	if !strings.Contains(out.String(), "se perderán para siempre") {
		t.Errorf("uninstall --all does not warn the notes have no backup:\n%s", out.String())
	}
}
