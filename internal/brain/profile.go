package brain

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/icalvete/augmented-brain/internal/memory"
)

// ProfileFile is who the user is and how they like to work. It lives in the
// configuration folder, shared by every space.
const ProfileFile = "profile.md"

// emptyName matches the name line of the blank profile that `init` writes.
var emptyName = regexp.MustCompile(`(?m)^- (Name|Nombre):[ \t]*$`)

// ProfileEmpty reports whether the profile in cfgDir is still blank: missing, or with
// nothing after its name. The first session then asks the user and saves the answer.
func ProfileEmpty(cfgDir string) bool {
	data, err := os.ReadFile(filepath.Join(cfgDir, ProfileFile))
	if err != nil {
		return true
	}
	return emptyName.Match(data)
}

var profileWords = map[string][4]string{
	// title, summary, about heading, work heading
	"en": {"Profile", "Who I am and how I like to work", "About me", "How I like to work"},
	"es": {"Perfil", "Quién soy y cómo me gusta trabajar", "Sobre mí", "Cómo me gusta trabajar"},
}

// ProfileInput is what the user says about themselves, in their own words.
type ProfileInput struct {
	Name      string
	About     string // what they do, what they work on
	WorkStyle string // how they like to work with the assistant
}

// SaveProfile writes the user's profile, keeping the date it was first written.
func (b *Brain) SaveProfile(in ProfileInput) (*memory.Note, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("the profile needs the user's name")
	}
	w, ok := profileWords[b.Language]
	if !ok {
		w = profileWords["en"]
	}
	label := "Name"
	if b.Language == "es" {
		label = "Nombre"
	}
	created := b.today()
	if data, err := os.ReadFile(filepath.Join(b.ConfigDir, ProfileFile)); err == nil {
		if old, err := memory.Parse(ProfileFile, data); err == nil && old.Header.Created != "" {
			created = old.Header.Created
		}
	}
	var body strings.Builder
	body.WriteString("## " + w[2] + "\n- " + label + ": " + name + "\n")
	if about := strings.TrimSpace(in.About); about != "" {
		body.WriteString(about + "\n")
	}
	body.WriteString("\n## " + w[3] + "\n")
	if style := strings.TrimSpace(in.WorkStyle); style != "" {
		body.WriteString(style + "\n")
	}
	n := &memory.Note{
		Path: ProfileFile,
		Header: memory.Header{ID: "profile", Type: "profile", Title: w[0], Summary: w[1],
			Created: created, Updated: b.today()},
		Body: body.String(),
	}
	return n, memory.Save(b.ConfigDir, n)
}
