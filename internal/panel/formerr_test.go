package panel

import (
	"html"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// postFrom is a form sent from one of the panel's pages, which the browser names in
// Referer.
func (s *Server) postFrom(t *testing.T, path, from string, form url.Values) (int, string, http.Header) {
	t.Helper()
	form.Set("csrf", s.csrf)
	w := s.do(t, req{method: http.MethodPost, path: s.base() + path, origin: "http://" + host,
		referer: "http://" + host + s.base() + from, form: form})
	return w.Code, html.UnescapeString(w.Body.String()), w.Header()
}

// ferr is the error a template writes next to a field or at the end of its form.
var ferr = regexp.MustCompile(`(?s)<p class="flash err ferr" role="alert" id="form-error" tabindex="-1" autofocus>(.*?)</p>`)

// A form that cannot be saved comes back on its own page, with what was typed, and the
// error next to the field, in the panel's language (the "done when" of the task).
func TestFormErrorNextToField(t *testing.T) {
	s, _ := fixture(t, "es")
	w := s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Hornear pan"}})
	loc, _ := url.Parse(w.Header().Get("Location"))
	id := loc.Query().Get("id")
	if id == "" {
		t.Fatalf("no task: %d %s", w.Code, w.Header().Get("Location"))
	}
	from := "/task?s=work&id=" + id

	code, body, _ := s.postFrom(t, "/watch/add", from, url.Values{"s": {"work"}, "id": {id}, "date": {"mañana"}, "what": {"si ha subido la masa"}})
	if code != http.StatusBadRequest {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(body, "<h1>Hornear pan</h1>") {
		t.Errorf("not the task's page:\n%s", body)
	}
	if !strings.Contains(body, `name="date" required value="mañana"`) || !strings.Contains(body, `value="si ha subido la masa"`) {
		t.Errorf("what was typed is not back in the form:\n%s", body)
	}
	m := ferr.FindAllStringSubmatch(body, -1)
	if len(m) != 1 || m[0][1] != `fecha "mañana": escríbela como AAAA-MM-DD` {
		t.Fatalf("error next to the field: %q", m)
	}
	// Next to the date, not at the end of the form; and the form is unfolded.
	if i, j := strings.Index(body, `name="date"`), strings.Index(body, m[0][0]); j < i || j > strings.Index(body, `name="what"`) {
		t.Error("the error is not right after the date")
	}
	if !strings.Contains(body, `<details class="box" open>
  <summary>Añadir fecha para mirar`) && !regexp.MustCompile(`<details class="box" open>\s*<summary>[^<]*</summary>\s*<form method="post" action="[^"]*/watch/add"`).MatchString(body) {
		t.Error("the form that failed is folded away")
	}
	if !strings.Contains(body, "No se ha guardado nada.") {
		t.Error("the top of the page does not say nothing was saved")
	}
	// The other forms of the page are as they were: nothing typed in them.
	if strings.Count(body, `value="si ha subido la masa"`) != 1 {
		t.Error("what was typed went into other forms")
	}
}

// A field the guard reads as free text: the date of a root exception. The error comes
// from config, in Spanish, next to "until".
func TestFormErrorRootUntil(t *testing.T) {
	s, _ := fixture(t, "es")
	code, body, _ := s.postFrom(t, "/guard/root", "/guard?s=work", url.Values{"s": {"work"}, "op": {"add"}, "form": {"add"},
		"host": {"lab-pc"}, "until": {"el viernes"}, "confirm": {"lab-pc"}})
	if code != http.StatusBadRequest {
		t.Fatalf("code %d", code)
	}
	m := ferr.FindAllStringSubmatch(body, -1)
	if len(m) != 1 || !strings.Contains(m[0][1], `la fecha "el viernes" no es ni un día`) {
		t.Fatalf("error: %q", m)
	}
	until := strings.Index(body, `name="until"`)
	if j := strings.Index(body, m[0][0]); j < until || j > strings.Index(body[until:], `name="confirm"`)+until {
		t.Error("the error is not next to the date")
	}
	if !strings.Contains(body, `value="el viernes"`) || !strings.Contains(body, `name="host" required maxlength="200" autocomplete="off" value="lab-pc"`) {
		t.Error("what was typed is not back")
	}
}

// A form repeated on a page (one per item) shows its error only in the one sent.
func TestFormErrorRepeatedForm(t *testing.T) {
	s, _ := fixture(t, "en")
	for _, d := range []string{"a", "b"} {
		if err := os.MkdirAll(filepath.Join(os.Getenv("HOME"), "work", d, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	code, body, _ := s.postFrom(t, "/repo/describe", "/repos?s=work", url.Values{"s": {"work"}, "repo": {"~/work/a"}, "form": {"~/work/a"},
		"text": {strings.Repeat("x", 400)}})
	if code != http.StatusBadRequest {
		t.Fatalf("code %d", code)
	}
	if m := ferr.FindAllStringSubmatch(body, -1); len(m) != 1 || !strings.Contains(m[0][1], "400 characters long") {
		t.Fatalf("errors: %q", m)
	}
}

// Without a page of the panel to go back to, the error has a page of its own, still in
// the panel's language.
func TestFormErrorNoReferer(t *testing.T) {
	s, _ := fixture(t, "es")
	w := s.post(t, "/inbox/capture", url.Values{"s": {"work"}, "text": {"llamar"}, "remind_on": {"mañana"}})
	body := html.UnescapeString(w.Body.String())
	if w.Code != http.StatusBadRequest || !strings.Contains(body, `fecha "mañana": escríbela como AAAA-MM-DD`) {
		t.Errorf("%d %s", w.Code, body)
	}
}

// An error no template places (a form with nothing typed) goes at the top of the page.
func TestFormErrorAtTop(t *testing.T) {
	s, _ := fixture(t, "es")
	code, body, _ := s.postFrom(t, "/inbox/done", "/inbox?s=work", url.Values{"s": {"work"}, "date": {"2026-10-15"}, "text": {"no está"}})
	if code != http.StatusBadRequest {
		t.Fatalf("code %d", code)
	}
	m := ferr.FindAllStringSubmatch(body, -1)
	if len(m) != 1 || strings.Index(body, m[0][0]) > strings.Index(body, "<h1>") {
		t.Errorf("not at the top: %q", m)
	}
}

// "Saved" says what was saved, and only in the address the change sent the browser to:
// the page itself, read again without it, says nothing (app.js takes ok= out of the
// address, so reloading does not say it again).
func TestSavedSaysWhat(t *testing.T) {
	s, _ := fixture(t, "es")
	w := s.post(t, "/inbox/capture", url.Values{"s": {"work"}, "text": {"llamar a Lucía"}})
	loc := w.Header().Get("Location")
	if w.Code != http.StatusSeeOther || !strings.Contains(loc, "ok=inbox.capture") {
		t.Fatalf("%d %q", w.Code, loc)
	}
	g := s.do(t, req{path: loc})
	if !strings.Contains(g.Body.String(), `<p class="flash ok" role="status">Apuntado en el buzón.</p>`) {
		t.Errorf("after saving:\n%s", g.Body)
	}
	if strings.Contains(s.get(t, "/inbox?s=work").Body.String(), `flash ok`) {
		t.Error("the page says saved without ok=")
	}
	if !strings.Contains(s.get(t, "/inbox?s=work&ok=1").Body.String(), `role="status">Guardado.</p>`) {
		t.Error("ok=1, from an earlier panel, no longer says saved")
	}
	js, err := os.ReadFile("web/static/app.js")
	if err != nil || !strings.Contains(string(js), `q.delete("ok")`) {
		t.Error("app.js does not take ok= out of the address")
	}
}

// Every change the panel makes says what it did, in both languages.
func TestSavedMessages(t *testing.T) {
	s, _ := fixture(t, "en")
	for _, p := range []string{"task/new", "task/update", "task/journal", "task/park", "task/close", "task/step", "watch/add", "watch/done",
		"inbox/capture", "inbox/done", "area/new", "area/update", "person", "person/card", "secret", "secret/forget",
		"review/capture", "review/task", "review/done", "profile", "session/recover", "session/reviewed", "session/delete",
		"repo/describe", "guard/list/add", "guard/list/remove", "guard/root/add", "guard/root/remove", "guard/accept",
		"guard/model", "guard/trust", "versions/restore", "backup/run", "backup/setup"} {
		k := strings.ReplaceAll(p, "/", ".")
		if got := savedLine("es", k); got == "Guardado." {
			t.Errorf("%s says only Saved", p)
		}
	}
	_ = s
}

// Every form with something to type has a place for its error, so what went wrong is
// next to it and not only at the top of the page.
func TestEveryFormHasItsError(t *testing.T) {
	files, _ := filepath.Glob("web/*.html")
	form := regexp.MustCompile(`(?s)<form method="post".*?</form>(\s*\{\{template "ferr"[^}]*\}\})?`)
	typed := regexp.MustCompile(`<(input|textarea|select)\b[^>]*name=`)
	for _, f := range files {
		b, _ := os.ReadFile(f)
		for _, m := range form.FindAllString(string(b), -1) {
			var own []string
			for _, in := range typed.FindAllString(m, -1) {
				if !strings.Contains(in, `type="hidden"`) {
					own = append(own, in)
				}
			}
			if len(own) > 0 && !strings.Contains(m, `{{template "ferr"`) {
				t.Errorf("%s: a form with no place for its error: %.120s", f, m)
			}
		}
	}
}
