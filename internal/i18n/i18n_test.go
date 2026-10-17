package i18n

import (
	"regexp"
	"testing"
)

var verb = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z]`)

// Every message exists in every language, with the same placeholders in the same order,
// so a translation can never swallow or misplace a value.
func TestCatalog(t *testing.T) {
	for lang, msgs := range messages {
		for key := range messages["en"] {
			if _, ok := msgs[key]; !ok {
				t.Errorf("%s: missing %q", lang, key)
			}
		}
		for key, f := range msgs {
			en, ok := messages["en"][key]
			if !ok {
				t.Errorf("%s: %q does not exist in English", lang, key)
				continue
			}
			if a, b := verb.FindAllString(en, -1), verb.FindAllString(f, -1); fmtString(a) != fmtString(b) {
				t.Errorf("%s %q: placeholders %v, English has %v", lang, key, b, a)
			}
		}
	}
}

func TestT(t *testing.T) {
	if got := T("es", "doctor.inspace", "work"); got != `esta carpeta es del espacio "work"` {
		t.Errorf("T = %q", got)
	}
	if got := T("fr", "notes.1"); got != "1 note" {
		t.Errorf("unknown language must fall back to English: %q", got)
	}
	if got := Count("es", "notes", 1); got != "1 nota" {
		t.Errorf("Count = %q", got)
	}
	t.Setenv("AB_LANGUAGE", "")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "es_ES.UTF-8")
	if FromEnv() != "es" {
		t.Error("LANG es_ES not detected")
	}
	t.Setenv("LC_ALL", "en_US.UTF-8")
	if FromEnv() != "en" {
		t.Error("LC_ALL must win over LANG")
	}
	t.Setenv("AB_LANGUAGE", "es")
	if FromEnv() != "es" {
		t.Error("AB_LANGUAGE must win over the locale")
	}
	for in, want := range map[string]string{"es-ES": "es", "es_419": "es", "ES": "es", "en-US": "en", "": "en", "fr-FR": "en"} {
		if got := pick(in); got != want {
			t.Errorf("pick(%q) = %q, want %q", in, got, want)
		}
	}
}

func fmtString(v []string) string {
	s := ""
	for _, x := range v {
		s += x + ","
	}
	return s
}
