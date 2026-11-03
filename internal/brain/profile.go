package brain

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/guard"
	"github.com/icalvete/augmented-brain/internal/i18n"
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
	word := func(key string) string { return i18n.T(b.Language, "profile."+key) }
	if workStyleIn(in.About) || workStyleIn(in.WorkStyle) {
		return nil, errors.New(i18n.T(b.Language, "profile.heading", word("work")))
	}
	label := i18n.T(b.Language, "profile.name")
	created := b.today()
	if data, err := os.ReadFile(filepath.Join(b.ConfigDir, ProfileFile)); err == nil {
		if old, err := memory.Parse(ProfileFile, data); err == nil && old.Header.Created != "" {
			created = old.Header.Created
		}
	}
	var body strings.Builder
	body.WriteString("## " + word("about") + "\n- " + label + ": " + name + "\n")
	if about := strings.TrimSpace(in.About); about != "" {
		body.WriteString(about + "\n")
	}
	body.WriteString("\n## " + word("work") + "\n")
	if style := strings.TrimSpace(in.WorkStyle); style != "" {
		body.WriteString(style + "\n")
	}
	n := &memory.Note{
		Path: ProfileFile,
		Header: memory.Header{ID: "profile", Type: "profile", Title: word("title"), Summary: word("summary"),
			Created: created, Updated: b.today()},
		Body: body.String(),
	}
	if what := guard.FindSecret(n.Body); what != "" {
		return nil, secretError(what)
	}
	return n, memory.Save(b.ConfigDir, n)
}

// profileName matches the name line of a profile, in either language.
var profileName = regexp.MustCompile(`^- (?:Name|Nombre):[ \t]*(.*)$`)

// heading reports whether line is the profile's heading key ("profile.about",
// "profile.work"), in either language.
func heading(line string, key string) bool {
	l := strings.TrimSpace(line)
	for _, w := range i18n.All(key) {
		if l == "## "+w {
			return true
		}
	}
	return false
}

// workStyleIn reports whether text has a line that is the work-style heading: saved,
// it would split the profile in the wrong place.
func workStyleIn(text string) bool {
	for _, l := range strings.Split(text, "\n") {
		if heading(l, "profile.work") {
			return true
		}
	}
	return false
}

// LoadProfile reads the profile in cfgDir back into the parts SaveProfile takes: the
// name, the work style (the section under the heading "How I like to work" or "Cómo me
// gusta trabajar", up to the next "## " heading), and About, which is everything else
// as it is, other headings included, but the About heading and the name line. Saving
// those parts and loading them again gives the same parts, so a profile edited by hand
// loses nothing. A missing profile is an empty one.
func LoadProfile(cfgDir string) (ProfileInput, error) {
	data, err := os.ReadFile(filepath.Join(cfgDir, ProfileFile))
	if errors.Is(err, os.ErrNotExist) {
		return ProfileInput{}, nil
	}
	if err != nil {
		return ProfileInput{}, err
	}
	n, err := memory.Parse(ProfileFile, data)
	if err != nil {
		return ProfileInput{}, err
	}
	lines := strings.Split(n.Body, "\n")
	rest := lines
	var in ProfileInput
	for i, l := range lines {
		if !heading(l, "profile.work") {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(lines[j], "## ") {
				end = j
				break
			}
		}
		in.WorkStyle = strings.TrimSpace(strings.Join(lines[i+1:end], "\n"))
		rest = append(slices.Clip(lines[:i]), lines[end:]...)
		break
	}
	var about []string
	aboutHeading, otherHeading := false, false
	for _, l := range rest {
		if strings.HasPrefix(l, "## ") {
			if !aboutHeading && !otherHeading && heading(l, "profile.about") {
				aboutHeading = true
				continue
			}
			otherHeading = true
		}
		if m := profileName.FindStringSubmatch(l); m != nil && in.Name == "" && !otherHeading {
			in.Name = strings.TrimSpace(m[1])
			continue
		}
		about = append(about, l)
	}
	in.About = strings.TrimSpace(strings.Join(about, "\n"))
	return in, nil
}
