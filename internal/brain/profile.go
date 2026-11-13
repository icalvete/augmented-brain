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
	// Assistant and Personality are the name and personality the user gives their
	// assistant, in the profile's "Assistant" section. Both may be empty.
	Assistant   string
	Personality string
}

// SaveProfile writes the user's profile, keeping the date it was first written.
func (b *Brain) SaveProfile(in ProfileInput) (*memory.Note, error) {
	// The names are one line each: a name with a line break in it would write a
	// heading of its own into the profile, which every session reads.
	name, assistant := memory.OneLine(in.Name), memory.OneLine(in.Assistant)
	if name == "" {
		return nil, errors.New("the profile needs the user's name")
	}
	for _, f := range []struct{ what, text string }{{"user's name", name}, {"assistant's name", assistant}} {
		if err := tooLong(f.what, f.text, maxName, "a name goes here; the rest goes in about or in the personality"); err != nil {
			return nil, err
		}
	}
	word := func(key string) string { return i18n.T(b.Language, "profile."+key) }
	if workStyleIn(in.About) || workStyleIn(in.WorkStyle) {
		return nil, errors.New(i18n.T(b.Language, "profile.heading", word("work")))
	}
	for _, text := range []string{in.About, in.WorkStyle, in.Personality} {
		if headingIn(text, "profile.assistant") {
			return nil, errors.New(i18n.T(b.Language, "profile.heading.field", word("assistant")))
		}
	}
	for _, key := range []string{"profile.work", "profile.about"} {
		if headingIn(in.Personality, key) {
			return nil, errors.New(i18n.T(b.Language, "profile.heading.field", word(strings.TrimPrefix(key, "profile."))))
		}
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
	personality := strings.TrimSpace(in.Personality)
	if assistant != "" || personality != "" {
		// Only lines with something in them: an empty "- Name:" line is what marks a
		// blank profile (ProfileEmpty).
		body.WriteString("\n## " + word("assistant") + "\n")
		if assistant != "" {
			body.WriteString("- " + label + ": " + assistant + "\n")
		}
		if personality != "" {
			body.WriteString(personality + "\n")
		}
	}
	n := &memory.Note{
		Path: ProfileFile,
		Header: memory.Header{ID: "profile", Type: "profile", Title: word("title"), Summary: word("summary"),
			Created: created, Updated: b.today()},
		Body: body.String(),
	}
	if err := tooLong("profile", n.Body, maxProfile, "keep in the profile who the user is and how they like to work, and put the rest in knowledge notes (knowledge_new)"); err != nil {
		return nil, err
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
func workStyleIn(text string) bool { return headingIn(text, "profile.work") }

// headingIn reports whether text has a line that is the profile heading key.
func headingIn(text, key string) bool {
	for _, l := range strings.Split(text, "\n") {
		if heading(l, key) {
			return true
		}
	}
	return false
}

// takeSection splits lines into the section under the heading key (without the
// heading, up to the next "## " heading) and the lines left. found is false when the
// profile has no such heading.
func takeSection(lines []string, key string) (section, rest []string, found bool) {
	for i, l := range lines {
		if !heading(l, key) {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(lines[j], "## ") {
				end = j
				break
			}
		}
		return lines[i+1 : end], append(slices.Clip(lines[:i]), lines[end:]...), true
	}
	return nil, lines, false
}

// takeAll takes every section under the heading key out of lines (see takeSection):
// their lines, one after the other, and the lines left.
func takeAll(lines []string, key string) (sections, rest []string) {
	rest = lines
	for {
		section, left, found := takeSection(rest, key)
		if !found {
			return sections, rest
		}
		if len(sections) > 0 {
			sections = append(sections, "")
		}
		sections, rest = append(sections, section...), left
	}
}

// assistantFrom reads the Assistant section: its first name line is the assistant's
// name, everything else is the personality.
func assistantFrom(section []string) (name, personality string) {
	var rest []string
	for _, l := range section {
		if m := profileName.FindStringSubmatch(l); m != nil && name == "" {
			name = strings.TrimSpace(m[1])
			continue
		}
		rest = append(rest, l)
	}
	return name, strings.TrimSpace(strings.Join(rest, "\n"))
}

// WithoutAssistant is a profile's body without its Assistant section, which the
// session start says in words of its own (see session.assistantText).
func WithoutAssistant(body string) string {
	_, rest, _ := takeSection(strings.Split(body, "\n"), "profile.assistant")
	return strings.Join(rest, "\n")
}

// SignatureLine is the line that goes under the assistant's name in every email or
// message it writes in the user's name, so whoever reads it knows no person wrote it.
// It names the user by their first name.
func SignatureLine(lang, userName string) string {
	first := userName
	if f := strings.Fields(userName); len(f) > 0 {
		first = f[0]
	}
	return i18n.T(lang, "assistant.signature", first)
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
	var in ProfileInput
	// A heading written twice (by hand, or by an earlier program that took a name of
	// several lines) has its sections joined, so what is loaded can always be saved.
	assistant, rest := takeAll(strings.Split(n.Body, "\n"), "profile.assistant")
	in.Assistant, in.Personality = assistantFrom(assistant)
	work, rest := takeAll(rest, "profile.work")
	in.WorkStyle = strings.TrimSpace(strings.Join(work, "\n"))
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
