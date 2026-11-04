package config

import (
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/doccheck"
)

// undocumented are the settings Load accepts that the reference leaves out on purpose.
var undocumented = []string{
	// Where 0.6.0 kept the names of the secrets: only read, to move them to secrets.toml;
	// nobody should write it any more.
	"secrets",
}

// settings are the names Load reads from config.toml (its toml tags), the top level's
// and a space's, each once.
func settings() []string {
	seen := map[string]bool{}
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		for i := range t.NumField() {
			f := t.Field(i)
			name, _, _ := strings.Cut(f.Tag.Get("toml"), ",")
			if name == "-" || name == "" || !f.IsExported() {
				continue
			}
			seen[name] = true
			ft := f.Type
			for ft.Kind() == reflect.Map || ft.Kind() == reflect.Pointer || ft.Kind() == reflect.Slice {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				walk(ft)
			}
		}
	}
	walk(reflect.TypeFor[Config]())
	var out []string
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// The reference page of the configuration, in each language, gives every setting of
// config.toml in backticks. It fails until the page covers them: docs cannot drift.
// AB_DOCS_DIR points it at pages elsewhere.
func TestSettingsDocumented(t *testing.T) {
	all := settings()
	for _, u := range undocumented {
		if !slices.Contains(all, u) {
			t.Errorf("undocumented lists %q, which config.toml no longer has", u)
		}
	}
	dir, err := doccheck.Dir()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range doccheck.Files("configuration") {
		p, err := doccheck.Read(dir, f)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		for _, s := range all {
			if !slices.Contains(undocumented, s) && !doccheck.Mentions(p.Code, s) {
				t.Errorf("%s: lacks the setting `%s`", f, s)
			}
		}
	}
}
