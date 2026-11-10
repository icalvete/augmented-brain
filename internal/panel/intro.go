package panel

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"

	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/state"
)

// Every page starts with a short text that says what it is and what it is for, with a
// link to its part of the guide. A person who no longer needs them hides them, on one
// page or on all; the panel keeps that in a file of this machine (hintsFile), not in a
// cookie (those of 127.0.0.1 reach every port) nor in the browser (the panel's address
// changes with its port, and the browser would forget).

// hintsFile is where the panel keeps which introductions the person hid.
const hintsFile = "panel-hints.json"

// hints is which introductions are hidden: all of them, or those of some pages.
type hints struct {
	All    bool     `json:"all,omitempty"`
	Hidden []string `json:"hidden,omitempty"`
}

func (h hints) hidden(key string) bool { return h.All || slices.Contains(h.Hidden, key) }

func hintsPath(cfgDir string) string { return filepath.Join(cfgDir, state.Dir, hintsFile) }

// loadHints reads what was hidden; a missing or broken file hides nothing.
func loadHints(cfgDir string) hints {
	var h hints
	if b, err := os.ReadFile(hintsPath(cfgDir)); err == nil {
		if json.Unmarshal(b, &h) != nil {
			return hints{}
		}
	}
	return h
}

func saveHints(cfgDir string, h hints) error {
	b, err := json.Marshal(h)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(cfgDir, state.Dir), 0o700); err != nil {
		return err
	}
	return paths.WriteAtomic(hintsPath(cfgDir), b, 0o600)
}

// guideAnchors is, for each introduction, the part of the panel's guide that tells
// about its page: the anchor of its heading in docs/guide/panel.md and panel.es.md
// (TestIntroAnchors checks they exist in the built pages).
var guideAnchors = map[string][2]string{ // key → English, Spanish
	"today":     {"today", "hoy"},
	"tasks":     {"tasks", "tareas"},
	"task":      {"tasks", "tareas"},
	"reminders": {"reminders", "recordatorios"},
	"inbox":     {"inbox", "buzón"},
	"review":    {"weekly-review", "revisión-semanal"},
	"areas":     {"areas-knowledge-rules-and-people", "áreas-conocimiento-reglas-y-personas"},
	"knowledge": {"areas-knowledge-rules-and-people", "áreas-conocimiento-reglas-y-personas"},
	"rules":     {"areas-knowledge-rules-and-people", "áreas-conocimiento-reglas-y-personas"},
	"people":    {"areas-knowledge-rules-and-people", "áreas-conocimiento-reglas-y-personas"},
	"note":      {"search-and-reading-notes", "búsqueda-y-lectura-de-notas"},
	"search":    {"search-and-reading-notes", "búsqueda-y-lectura-de-notas"},
	"repos":     {"repositories", "repositorios"},
	"secrets":   {"secrets", "secretos"},
	"sessions":  {"sessions", "sesiones"},
	"profile":   {"profile", "perfil"},
	"guard":     {"guard", "guardia"},
	"versions":  {"versions", "versiones"},
	"space":     {"health", "salud"},
	"help":      {"help", "ayuda"},
}

// introData is what template "intro" (layout.html) writes at the top of a page.
type introData struct {
	Key, Text, Guide string
	Hidden, All      bool
	Base, CSRF       string
	Space, Back      string
}

// intro is the template function behind {{template "intro" (intro $ "key" text)}}.
func intro(p page, key, text string) introData {
	d := introData{Key: key, Text: text, Base: p.Base, CSRF: p.CSRF, Space: p.Space,
		Hidden: p.Hints.hidden(key), All: p.Hints.All, Back: p.Here}
	if a, ok := guideAnchors[key]; ok {
		frag := a[0]
		if p.Lang == "es" {
			frag = a[1]
		}
		d.Guide = guideURL(p.Lang, "panel") + "#" + frag
	}
	return d
}

// hintsChange hides or shows introductions, then sends the browser back to the page
// it came from. It is a preference of this machine, not a note: no notes lock, no
// version of the configuration.
func (s *Server) hintsChange(w http.ResponseWriter, r *http.Request) {
	key := r.PostForm.Get("key")
	if _, ok := guideAnchors[key]; !ok && key != "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	h := loadHints(s.cfgDir)
	switch r.PostForm.Get("do") {
	case "hide":
		if !slices.Contains(h.Hidden, key) {
			h.Hidden = append(h.Hidden, key)
		}
	case "hideall":
		h.All = true
	case "show":
		h.All = false
		h.Hidden = slices.DeleteFunc(h.Hidden, func(k string) bool { return k == key })
	case "showall":
		h = hints{}
	default:
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := saveHints(s.cfgDir, h); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, s.base()+backTo(r.PostForm.Get("back")), http.StatusSeeOther)
}

// backTo is the page to go back to: a path of the panel with its query, never an
// address of another site.
func backTo(back string) string {
	u, err := url.Parse(back)
	if err != nil || u.Scheme != "" || u.Host != "" || len(u.Path) < 2 || u.Path[0] != '/' || u.Path[1] == '/' || u.Path[1] == '\\' {
		return "/today"
	}
	return (&url.URL{Path: u.Path, RawQuery: u.RawQuery}).String()
}
