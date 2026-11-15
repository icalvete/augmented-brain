package panel

import (
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Every form of the panel, filled to the maxlength of each of its fields, fits in
// maxBody, so what the page lets a person type is never refused for its size. A letter
// of a script such as Japanese takes up to nine bytes once encoded ("%E3%81%82"). A
// field that takes a whole note (a body of hundreds of thousands of characters) only
// has to fit as plain text: past that, the panel says the text is too long.
func TestFormsFitTheBody(t *testing.T) {
	const (
		worstByte = 9       // bytes of one character, percent-encoded
		note      = 100_000 // from this maxlength, a field holds a whole note
		margin    = 4 << 10 // names, hidden fields, the CSRF token
	)
	reForm := regexp.MustCompile(`(?s)<form\b.*?</form>`)
	reMax := regexp.MustCompile(`maxlength="(\d+)"`)
	pages, err := fs.Glob(webFS, "web/*.html")
	if err != nil || len(pages) == 0 {
		t.Fatalf("no pages: %v", err)
	}
	forms := 0
	for _, p := range pages {
		data, err := fs.ReadFile(webFS, p)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range reForm.FindAllString(string(data), -1) {
			forms++
			sum, big := 0, false
			for _, m := range reMax.FindAllStringSubmatch(f, -1) {
				n, _ := strconv.Atoi(m[1])
				sum += n
				big = big || n >= note
			}
			need := sum*worstByte + margin
			if big {
				need = sum + margin
			}
			if need > maxBody {
				start := f[:min(len(f), 120)]
				t.Errorf("%s: a form needs %d bytes, more than maxBody (%d): %s…", p, need, maxBody, strings.ReplaceAll(start, "\n", " "))
			}
		}
	}
	if forms < 10 {
		t.Fatalf("only %d forms found: is the pattern still right?", forms)
	}
}
