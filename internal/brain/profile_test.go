package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The panel shows the profile as the parts profile_save takes and saves them back:
// load, save, load again gives the same parts.
func TestLoadProfileRoundTrip(t *testing.T) {
	for _, lang := range []string{"en", "es"} {
		b := open(t, lang)
		in := ProfileInput{Name: "Ana", About: "- Runs a small bakery\n- Learning to code", WorkStyle: "- Short answers"}
		if _, err := b.SaveProfile(in); err != nil {
			t.Fatal(err)
		}
		got, err := LoadProfile(b.ConfigDir)
		if err != nil {
			t.Fatal(err)
		}
		if got != in {
			t.Errorf("%s: LoadProfile = %+v, want %+v", lang, got, in)
		}
	}
}

func TestLoadProfileBlankAndHandEdited(t *testing.T) {
	b := open(t, "es")
	got, err := LoadProfile(b.ConfigDir)
	if err != nil {
		t.Fatal(err)
	}
	// The blank profile of init: no name yet, its other lines kept.
	if got.Name != "" || got.About != "- A qué me dedico:" || got.WorkStyle == "" {
		t.Errorf("blank profile = %+v", got)
	}
	// Edited by hand: a loose line, a heading the program does not know, and a section
	// after the work style. Only the work-style section, found by its heading in either
	// language, is the work style; the rest is About, headings and all.
	hand := "---\nid: profile\ntype: profile\ntitle: Perfil\nsummary: x\ncreated: 2026-10-01\nupdated: 2026-10-01\n---\n" +
		"Una línea suelta.\n## Sobre mí\n- Nombre: Lucía\n\n## Cómo trabajo\n- Corto\n\n## How I like to work\n- Directo\n\n## Más\n- Algo\n"
	if err := os.WriteFile(filepath.Join(b.ConfigDir, ProfileFile), []byte(hand), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = LoadProfile(b.ConfigDir)
	if err != nil {
		t.Fatal(err)
	}
	want := ProfileInput{Name: "Lucía", About: "Una línea suelta.\n\n## Cómo trabajo\n- Corto\n\n## Más\n- Algo", WorkStyle: "- Directo"}
	if got != want {
		t.Errorf("hand-edited profile = %+v", got)
	}
	// Saved and loaded again: nothing lost, the extra headings included.
	if _, err := b.SaveProfile(got); err != nil {
		t.Fatal(err)
	}
	again, err := LoadProfile(b.ConfigDir)
	if err != nil || again != got {
		t.Errorf("after saving: %+v, %v", again, err)
	}
	data, _ := os.ReadFile(filepath.Join(b.ConfigDir, ProfileFile))
	for _, h := range []string{"## Cómo trabajo", "## Más", "## Cómo me gusta trabajar"} {
		if !strings.Contains(string(data), h+"\n") {
			t.Errorf("the saved profile lost %q:\n%s", h, data)
		}
	}
	// The work-style heading inside About (or the work style) would split the profile
	// in the wrong place: refused, in either language.
	for _, bad := range []ProfileInput{{Name: "Lucía", About: "- x\n## Cómo me gusta trabajar\n- y"},
		{Name: "Lucía", About: "## How I like to work"}, {Name: "Lucía", WorkStyle: "- a\n## How I like to work\n- b"}} {
		if _, err := b.SaveProfile(bad); err == nil || !strings.Contains(err.Error(), "Cómo me gusta trabajar") {
			t.Errorf("SaveProfile(%+v): %v", bad, err)
		}
	}
	if got, err := LoadProfile(t.TempDir()); err != nil || got != (ProfileInput{}) {
		t.Errorf("missing profile = %+v, %v", got, err)
	}
}
