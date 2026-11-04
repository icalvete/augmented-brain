// Package i18n holds the messages the program shows to people, in every supported
// language. Messages are fmt formats; a missing key falls back to English, and a key
// missing in English is shown as is, so a gap is visible but never breaks anything.
package i18n

import (
	"fmt"
	"os"
	"strings"
)

// msg is one message in each language.
type msg struct{ en, es string }

// messages is every message, by key. Each subject keeps its own in a file of its own.
var messages = merge(doctorMessages, commandMessages, panelMessages, importMessages, initMessages, noteMessages, helpMessages, guardMessages, hookMessages)

// merge joins the subjects' messages. A key in two of them is a mistake: it panics, so
// the tests catch it.
func merge(groups ...map[string]msg) map[string]msg {
	all := map[string]msg{}
	for _, g := range groups {
		for k, m := range g {
			if _, dup := all[k]; dup {
				panic("i18n: message " + k + " defined twice")
			}
			all[k] = m
		}
	}
	return all
}

// All returns a message in every language, unformatted: for reading back text the
// program wrote in whichever language the user had then.
func All(key string) []string {
	m := messages[key]
	return []string{m.en, m.es}
}

// Known returns lang when there are messages in it, and English otherwise.
func Known(lang string) string {
	if lang == "es" {
		return "es"
	}
	return "en"
}

// T formats the message key in lang.
func T(lang, key string, args ...any) string {
	m, ok := messages[key]
	f := m.en
	switch {
	case !ok:
		f = key
	case Known(lang) == "es":
		f = m.es
	}
	if len(args) == 0 {
		return f
	}
	return fmt.Sprintf(f, args...)
}

// Count returns "1 note" or "3 notes" (key "notes"), in lang.
func Count(lang, key string, n int) string {
	if n == 1 {
		return T(lang, key+".1")
	}
	return T(lang, key+".n", n)
}

// FromEnv guesses the language when there is no configuration yet: AB_LANGUAGE, which
// the installers set; then the locale, in the order the C library reads it (LC_ALL,
// LC_MESSAGES, LANG); then the system's own setting where there is one, as on Windows,
// which has no locale variables.
func FromEnv() string {
	if l := os.Getenv("AB_LANGUAGE"); l == "en" || l == "es" {
		return l
	}
	for _, v := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if l := os.Getenv(v); l != "" {
			return pick(l)
		}
	}
	return pick(systemLanguage())
}

// pick maps a locale or language tag (es_ES.UTF-8, es-ES, en-US) to a language we speak.
func pick(l string) string {
	if strings.HasPrefix(strings.ToLower(l), "es") {
		return "es"
	}
	return "en"
}
