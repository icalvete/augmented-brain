package panel

import (
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/doctor"
	"github.com/icalvete/augmented-brain/internal/guard"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/update"
	"github.com/icalvete/augmented-brain/internal/version"
)

// The Help page: links to the website's answers, and a problem report or an idea for
// GitHub, written here first. Nothing leaves the computer until the person clicks the
// link to GitHub, and then only the text they see on the page, as they left it.

// maxIssueURL is the longest address the page builds for GitHub's "new issue": GitHub
// refuses much longer ones (about 8 KB). web/static/app.js keeps to the same length.
const maxIssueURL = 8000

// maxCheckText is how much of each check of the health report goes into a report.
const maxCheckText = 300

// owner and site come from update.Repo, the one place that names the repository.
func repoOwner() string { o, _, _ := strings.Cut(update.Repo, "/"); return o }

// siteURL is the website (GitHub Pages of the repository), in the viewer's language.
func siteURL(lang string) string {
	owner, name, _ := strings.Cut(update.Repo, "/")
	u := "https://" + owner + ".github.io/" + name + "/"
	if lang == "es" {
		u += "es/"
	}
	return u
}

// guideURL is one guide of the website, in the viewer's language.
func guideURL(lang, slug string) string {
	if lang == "es" {
		slug += ".es"
	}
	return strings.TrimSuffix(siteURL("en"), "/") + "/guide/" + slug + ".html"
}

// issueField is one field of GitHub's issue form, by its id in .github/ISSUE_TEMPLATE.
type issueField struct{ ID, Value string }

// issueURL builds GitHub's "new issue" address for template, with the fields filled
// in, in order. When it would be longer than max, the field cut (by id) is shortened
// and ends with note, so the address still opens.
func issueURL(repo, template string, fields []issueField, cut, note string, max int) string {
	build := func(fs []issueField) string {
		var b strings.Builder
		b.WriteString("https://github.com/" + repo + "/issues/new?template=" + url.QueryEscape(template))
		for _, f := range fs {
			b.WriteString("&" + url.QueryEscape(f.ID) + "=" + url.QueryEscape(f.Value))
		}
		return b.String()
	}
	full := build(fields)
	if len(full) <= max {
		return full
	}
	at := -1
	for i, f := range fields {
		if f.ID == cut {
			at = i
		}
	}
	if at < 0 {
		return full
	}
	fs := append([]issueField(nil), fields...)
	runes := []rune(fields[at].Value)
	with := func(n int) string { // the first n characters of the field, then the note
		fs[at].Value = strings.TrimRight(string(runes[:n]), "\n") + "\n\n" + note
		return build(fs)
	}
	// The longest start of the field that fits.
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if len(with(mid)) <= max {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return with(lo)
}

// redactor takes out of a report what says who the person is or what they work on.
type redactor struct {
	lang  string
	homes []string          // the home folder, as written and with links resolved
	exact []replacement     // folders of the notes and the work, longest first
	words *regexp.Regexp    // hosts, AWS profiles, space names, longest first
	to    map[string]string // what each word (in lower case) is replaced with
}

type replacement struct{ from, to string }

var emailRe = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9\-]+(\.[A-Za-z0-9\-]+)+`)

// newRedactor knows what to take out from the configuration c (nil: only the home
// folder, e-mails and secrets).
func newRedactor(lang string, c *config.Config) *redactor {
	t := func(k string) string { return i18n.T(lang, "panel.help.redact."+k) }
	r := &redactor{lang: lang, to: map[string]string{}}
	if h, err := paths.Home(); err == nil && h != "" {
		r.homes = append(r.homes, h)
		if real, err := filepath.EvalSymlinks(h); err == nil && real != h {
			r.homes = append(r.homes, real)
		}
	}
	if c == nil {
		return r
	}
	addPath := func(p, to string) {
		if strings.TrimSpace(p) == "" {
			return
		}
		ways := []string{p}
		if res, err := paths.Resolve(p); err == nil {
			ways = append(ways, res)
		}
		for _, w := range append([]string(nil), ways...) {
			ways = append(ways, paths.Tilde(w), filepath.ToSlash(w))
		}
		for _, w := range ways {
			if w != "" && w != "~" && w != "/" {
				r.exact = append(r.exact, replacement{w, to})
			}
		}
	}
	addWord := func(w, to string) {
		if w = strings.ToLower(strings.TrimSpace(w)); w != "" {
			if _, seen := r.to[w]; !seen {
				r.to[w] = to
			}
		}
	}
	for _, name := range c.SpaceNames() {
		sp := c.Spaces[name]
		addPath(sp.Memory, t("notes"))
		addPath(sp.Backup, t("backup"))
		for _, w := range sp.WorkFolders {
			addPath(w, t("work"))
		}
	}
	profiles, hosts := c.GuardLists()
	roots, bad := c.RootExceptions()
	for _, x := range roots {
		hosts = append(hosts, x.Host)
	}
	for _, b := range bad { // an entry the program cannot read still names a host
		if f := strings.Fields(b); len(f) > 0 {
			hosts = append(hosts, f[0])
		}
	}
	for _, h := range hosts {
		addWord(h, t("host"))
	}
	for _, p := range profiles {
		addWord(p, t("profile"))
	}
	for _, name := range c.SpaceNames() {
		addWord(name, t("space"))
	}
	// Longest first, so a folder inside another, or a name inside a longer one, is
	// taken out whole.
	sort.SliceStable(r.exact, func(i, j int) bool { return len(r.exact[i].from) > len(r.exact[j].from) })
	if len(r.to) > 0 {
		var alts []string
		for w := range r.to {
			alts = append(alts, w)
		}
		sort.Slice(alts, func(i, j int) bool {
			if len(alts[i]) != len(alts[j]) {
				return len(alts[i]) > len(alts[j])
			}
			return alts[i] < alts[j]
		})
		for i, a := range alts {
			alts[i] = regexp.QuoteMeta(a)
		}
		r.words = regexp.MustCompile(`(?i)` + strings.Join(alts, "|"))
	}
	return r
}

// wordChar is a character that continues a name, a host or a path segment.
func wordChar(b byte) bool {
	return b == '_' || b == '-' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= 0x80
}

// replaceWords replaces each known word that stands whole (not inside a longer word),
// in one pass, so a replacement is never replaced again.
func (r *redactor) replaceWords(s string) string {
	if r.words == nil {
		return s
	}
	var b strings.Builder
	last := 0
	for _, m := range r.words.FindAllStringIndex(s, -1) {
		before := m[0] == 0 || !wordChar(s[m[0]-1]) && s[m[0]-1] != '.'
		after := m[1] == len(s) || !wordChar(s[m[1]])
		if !before || !after {
			continue
		}
		b.WriteString(s[last:m[0]])
		b.WriteString(r.replacement(s[m[0]:m[1]]))
		last = m[1]
	}
	b.WriteString(s[last:])
	return b.String()
}

// replacement is what the word w, as found in the text, is replaced with.
func (r *redactor) replacement(w string) string {
	if to, ok := r.to[strings.ToLower(w)]; ok {
		return to
	}
	for k, to := range r.to { // a case the simple lower case does not match
		if strings.EqualFold(k, w) {
			return to
		}
	}
	return "[…]"
}

// line redacts one line, or says to leave it out (false) when it holds what looks like
// a secret, the same check the guard makes on commands.
func (r *redactor) line(s string) (string, bool) {
	if guard.FindSecret(s) != "" {
		return "", false
	}
	for _, e := range r.exact {
		s = strings.ReplaceAll(s, e.from, e.to)
	}
	for _, h := range r.homes {
		s = strings.ReplaceAll(s, h, "~")
		s = strings.ReplaceAll(s, filepath.ToSlash(h), "~")
	}
	s = emailRe.ReplaceAllString(s, i18n.T(r.lang, "panel.help.redact.email"))
	return r.replaceWords(s), true
}

// text redacts every line of s; a line left out leaves a note in its place.
func (r *redactor) text(s string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if red, ok := r.line(l); ok {
			out = append(out, red)
		} else {
			out = append(out, i18n.T(r.lang, "panel.help.redact.secret"))
		}
	}
	return strings.Join(out, "\n")
}

// diagnostics is what a problem report says about this computer: the program's
// version, the system, and the health report (each check's status and its short
// message, never how to fix it), with what is personal taken out.
func diagnostics(lang string, checks []doctor.Check, r *redactor) string {
	marks := map[doctor.Status]string{doctor.OK: "[ok]  ", doctor.Info: "[info]", doctor.Warning: "[warn]", doctor.Failed: "[fail]"}
	var b strings.Builder
	b.WriteString("augmented brain " + version.Version + "\n")
	b.WriteString(i18n.T(lang, "panel.help.system", runtime.GOOS+"/"+runtime.GOARCH) + "\n\n")
	b.WriteString(i18n.T(lang, "panel.help.checks") + "\n")
	for _, c := range checks {
		what := strings.Join(strings.Fields(c.What), " ")
		if utf8.RuneCountInString(what) > maxCheckText {
			what = string([]rune(what)[:maxCheckText]) + "…"
		}
		b.WriteString(marks[c.Status] + " " + what + "\n")
	}
	return r.text(strings.TrimRight(b.String(), "\n"))
}

type helpLink struct{ Label, Hint, URL string }

// runDoctor is the health check behind a report; tests replace it.
func (s *Server) runDoctor(lang string) []doctor.Check {
	if s.diagnose != nil {
		return s.diagnose(lang)
	}
	return doctor.Run(doctor.Env{ConfigDir: s.cfgDir, Lang: lang})
}

func (s *Server) help(v *view, _ *http.Request) (string, page, error) {
	l := v.lang
	t := func(k string) string { return v.t("help." + k) }
	d := struct {
		Answers               []helpLink
		Idea, Sponsor, Issues string
	}{
		Answers: []helpLink{
			{t("faq"), t("faq.hint"), siteURL(l) + "#faq"},
			{t("docs"), t("docs.hint"), guideURL(l, "index")},
			{t("trouble"), t("trouble.hint"), guideURL(l, "troubleshooting")},
			{t("start"), t("start.hint"), guideURL(l, "getting-started")},
			{t("how"), t("how.hint"), guideURL(l, "how-it-works")},
			{t("install"), t("install.hint"), guideURL(l, "install")},
			{t("connect"), t("connect.hint"), guideURL(l, "connect-services")},
		},
		Idea:    issueURL(update.Repo, "idea.yml", nil, "", "", maxIssueURL),
		Sponsor: "https://github.com/sponsors/" + repoOwner(),
		Issues:  "https://github.com/" + update.Repo + "/issues",
	}
	p := v.frame("help", t("title"))
	p.D = d
	return "help", p, nil
}

// helpReport runs the health check now, when the person clicks "Report a problem on
// GitHub", and sends the browser to GitHub's problem form with the version, the
// system and the check filled in, redacted. It changes nothing here, so it is a GET
// that needs no CSRF token (the Host check and the session in the address still
// apply). A link, not a form: browsers apply the CSP form-action to a form's
// redirects, and a form sent here could not go on to github.com.
func (s *Server) helpReport(w http.ResponseWriter, r *http.Request) {
	// No notes lock: the health check only reads, and runs git, Claude Code and the
	// guard, which take seconds Claude should not wait.
	lang := s.lang()
	c, err := config.Load(s.cfgDir)
	if err != nil {
		c = nil // the home folder, e-mails and secrets are still taken out
	}
	body := diagnostics(lang, s.runDoctor(lang), newRedactor(lang, c))
	to := issueURL(update.Repo, "problem.yml", []issueField{{"what", ""}, {"diagnostics", body}},
		"diagnostics", i18n.T(lang, "panel.help.cut"), maxIssueURL)
	http.Redirect(w, r, to, http.StatusSeeOther)
}
