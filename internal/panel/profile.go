package panel

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/brain"
)

func (s *Server) profile(v *view, _ *http.Request) (string, page, error) {
	in, err := brain.LoadProfile(v.b.ConfigDir)
	if err != nil {
		return "", page{}, err
	}
	p := v.frame("profile", v.t("profile"))
	p.D = struct {
		P    brain.ProfileInput
		Path string
	}{in, filepath.Join(v.b.ConfigDir, brain.ProfileFile)}
	return "profile", p, nil
}

func (s *Server) profileSave(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "name"); err != nil {
		return "", err
	}
	// Browsers send \r\n in text areas; notes use \n.
	nl := strings.NewReplacer("\r\n", "\n")
	_, err := v.b.SaveProfile(brain.ProfileInput{Name: field(r, "name"),
		About: nl.Replace(field(r, "about")), WorkStyle: nl.Replace(field(r, "work_style"))})
	return "/profile", err
}
