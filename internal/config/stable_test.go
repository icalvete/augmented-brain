package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/stable"
)

// tags are the toml names of t's fields.
func tags(t reflect.Type) []string {
	out := []string{}
	for i := range t.NumField() {
		if name, _, _ := strings.Cut(t.Field(i).Tag.Get("toml"), ","); name != "" && name != "-" {
			out = append(out, name)
		}
	}
	return out
}

// The settings of the configuration files are part of the compatibility promise: a
// setting Load accepts today, every 1.x accepts. Frozen in testdata/stable-settings.json,
// one entry per file (remote.toml is frozen in internal/remote).
func TestStableSettings(t *testing.T) {
	now := stable.Surface{
		FileName:    settings(),
		GuardFile:   tags(reflect.TypeFor[Guard]()),
		SessionFile: tags(reflect.TypeFor[Session]()),
	}
	stable.Check(t, filepath.Join("testdata", "stable-settings.json"), now)
}
