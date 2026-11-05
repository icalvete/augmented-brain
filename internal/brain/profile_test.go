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

// The assistant's name and personality live in a section of their own, which loads back
// as it was saved, in either language; a profile without them gets no such section.
func TestProfileAssistant(t *testing.T) {
	for _, lang := range []string{"en", "es"} {
		b := open(t, lang)
		in := ProfileInput{Name: "Ana López", WorkStyle: "- Short answers", Assistant: "Argos",
			Personality: "- Calm and direct\n- Dry humour, now and then"}
		if _, err := b.SaveProfile(in); err != nil {
			t.Fatal(err)
		}
		got, err := LoadProfile(b.ConfigDir)
		if err != nil || got != in {
			t.Errorf("%s: LoadProfile = %+v, %v; want %+v", lang, got, err, in)
		}
		if ProfileEmpty(b.ConfigDir) {
			t.Errorf("%s: a profile with an assistant counts as blank", lang)
		}
		data, _ := os.ReadFile(filepath.Join(b.ConfigDir, ProfileFile))
		if body := WithoutAssistant(string(data)); strings.Contains(body, "Argos") || !strings.Contains(body, "Short answers") {
			t.Errorf("%s: WithoutAssistant kept the section or lost the rest:\n%s", lang, body)
		}
		// Without assistant, no section at all.
		in.Assistant, in.Personality = "", ""
		if _, err := b.SaveProfile(in); err != nil {
			t.Fatal(err)
		}
		data, _ = os.ReadFile(filepath.Join(b.ConfigDir, ProfileFile))
		for _, h := range []string{"## Assistant", "## Asistente"} {
			if strings.Contains(string(data), h) {
				t.Errorf("%s: an empty assistant wrote %q:\n%s", lang, h, data)
			}
		}
	}
}

// A heading the program writes cannot come inside a part, or the profile would split
// in the wrong place.
func TestProfileAssistantHeadings(t *testing.T) {
	b := open(t, "es")
	for _, bad := range []ProfileInput{
		{Name: "Ana", About: "## Asistente\n- Nombre: X"},
		{Name: "Ana", Personality: "## Cómo me gusta trabajar\n- x"},
		{Name: "Ana", Personality: "## Assistant"},
	} {
		if _, err := b.SaveProfile(bad); err == nil {
			t.Errorf("SaveProfile(%+v) took a heading", bad)
		}
	}
}

// The signature line names the user by their first name, in the user's language.
func TestSignatureLine(t *testing.T) {
	if got := SignatureLine("es", "Ana López"); got != "asistente de Ana (IA, Claude)" {
		t.Errorf("es: %q", got)
	}
	if got := SignatureLine("en", "Ana"); got != "Ana's assistant (AI, Claude)" {
		t.Errorf("en: %q", got)
	}
}
