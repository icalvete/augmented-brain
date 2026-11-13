package remote

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/stable"
)

// The settings of remote.toml are part of the compatibility promise, like those of
// config.toml (internal/config): frozen in testdata/stable-settings.json.
func TestStableSettings(t *testing.T) {
	ft := reflect.TypeFor[fileSettings]()
	names := []string{}
	for i := range ft.NumField() {
		if name, _, _ := strings.Cut(ft.Field(i).Tag.Get("toml"), ","); name != "" && name != "-" {
			names = append(names, name)
		}
	}
	stable.Check(t, filepath.Join("testdata", "stable-settings.json"), stable.Surface{SettingsFile: names})
}
