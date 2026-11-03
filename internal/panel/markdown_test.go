package panel

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func md(src string) string {
	return string(markdown(src, func(id string) (string, bool) {
		if id == "task-known" {
			return "note?id=" + id, true
		}
		return "", false
	}))
}

func TestMarkdownBlocks(t *testing.T) {
	cases := []struct{ src, want string }{
		{"# Title", "<h2>Title</h2>"},
		{"### Sub", "<h4>Sub</h4>"},
		{"one\ntwo", "<p>one two</p>"},
		{"- a\n- b", "<ul>\n<li>a</li>\n<li>b</li></ul>"},
		{"1. a\n2. b", "<ol>\n<li>a</li>\n<li>b</li></ol>"},
		{"- a\n  - b\n- c", "<ul>\n<li>a<ul>\n<li>b</li></ul>\n</li>\n<li>c</li></ul>"},
		{"- a\n  more", "<li>a more</li>"},
		{"- [ ] todo\n- [x] done", `<span class="box-off">☐</span> todo`},
		{"- [x] done", `<span class="box-on">☑</span> done`},
		{"```\n<b>x</b>\n```", "<pre><code>&lt;b&gt;x&lt;/b&gt;</code></pre>"},
		{"| a | b |\n|---|---|\n| 1 | 2 |", "<th>a</th><th>b</th>"},
		{"| a | b |\n|---|---|\n| 1 | 2 |", "<td>1</td><td>2</td>"},
		{"| a | b |\n|---|---|\n| `x|y` | 2 |", "<td><code>x|y</code></td><td>2</td>"},
		{"> quoted", "<blockquote><p>quoted</p>\n</blockquote>"},
		{"---", "<hr>"},
	}
	for _, c := range cases {
		if got := md(c.src); !strings.Contains(got, c.want) {
			t.Errorf("%q:\n got  %q\n want %q", c.src, got, c.want)
		}
	}
}

func TestMarkdownInline(t *testing.T) {
	cases := []struct{ src, want string }{
		{"**bold**", "<strong>bold</strong>"},
		{"an *it* word", "an <em>it</em> word"},
		{"snake_case_name and 2*3*4", "snake_case_name and 2*3*4"},
		{"~~gone~~", "<s>gone</s>"},
		{"`a <b>`", "<code>a &lt;b&gt;</code>"},
		{"`**not bold**`", "<code>**not bold**</code>"},
		{"see [[task-known]]", `<a href="note?id=task-known">task-known</a>`},
		{"see [[task-known|the task]]", `<a href="note?id=task-known">the task</a>`},
		{"see [[task-gone]]", `<span class="nolink" title="task-gone">task-gone</span>`},
		{"[site](https://example.com/a?b=1&c=2)", `<a href="https://example.com/a?b=1&amp;c=2" rel="noreferrer noopener">site</a>`},
		{"go to https://example.com/x.", `<a href="https://example.com/x" rel="noreferrer noopener">https://example.com/x</a>.`},
		{"[**b**](https://e.com)", `<a href="https://e.com" rel="noreferrer noopener"><strong>b</strong></a>`},
	}
	for _, c := range cases {
		if got := md(c.src); !strings.Contains(got, c.want) {
			t.Errorf("%q:\n got  %q\n want %q", c.src, got, c.want)
		}
	}
}

// A note can never put its own HTML, scripts or dangerous links in the page: every tag
// in the output is one the renderer writes, with only its own attributes, and every
// link goes to an http(s) address or within the panel.
func TestMarkdownIsSafe(t *testing.T) {
	bad := []string{
		"<script>alert(1)</script>",
		`<img src=x onerror="alert(1)">`,
		"[x](javascript:alert(1))",
		"[x](JAVASCRIPT:alert(1))",
		"[x](data:text/html,<script>alert(1)</script>)",
		`[x](https://e.com" onmouseover="alert(1))`,
		`[[x" onmouseover="alert(1)]]`,
		"| <b>a</b> |\n|---|\n| <i>1</i> |",
		"# <svg onload=alert(1)>",
		"- <iframe src=x>",
		"`</code><script>`",
		"text \x000\x00 with a fake token",
		"https://e.com/\"><script>alert(1)</script>",
		"https://e.com/a'onmouseover='x",
		"**[x](https://e.com\"x)**",
	}
	tag := regexp.MustCompile(`<(/?)([a-zA-Z0-9]+)([^>]*)>`)
	attr := regexp.MustCompile(`\s([a-z]+)="([^"]*)"`)
	allowed := map[string]bool{"p": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "ul": true, "ol": true,
		"li": true, "pre": true, "code": true, "blockquote": true, "hr": true, "div": true, "table": true, "thead": true,
		"tbody": true, "tr": true, "th": true, "td": true, "a": true, "span": true, "strong": true, "em": true, "s": true}
	for _, src := range bad {
		got := md(src)
		for _, m := range tag.FindAllStringSubmatch(got, -1) {
			if !allowed[m[2]] {
				t.Errorf("%q wrote the tag %q: %q", src, m[2], got)
			}
			rest := attr.ReplaceAllString(m[3], "")
			if strings.TrimSpace(rest) != "" {
				t.Errorf("%q wrote stray attributes %q: %q", src, rest, got)
			}
			for _, a := range attr.FindAllStringSubmatch(m[3], -1) {
				switch a[1] {
				case "class", "title", "rel":
				case "href":
					if !regexp.MustCompile(`^(https?://|/|#|note\?)`).MatchString(a[2]) {
						t.Errorf("%q links to %q", src, a[2])
					}
				default:
					t.Errorf("%q wrote the attribute %q: %q", src, a[1], got)
				}
			}
		}
	}
}

func TestNewestFirst(t *testing.T) {
	in := []string{"intro", "- 2026-01-01 — one", "  more of one", "- 2026-01-02 — two"}
	got, n := newestFirst(in)
	want := []string{"intro", "- 2026-01-02 — two", "- 2026-01-01 — one", "  more of one"}
	if n != 2 || strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q (%d)", got, n)
	}
}

// Found in review: a "|" line that is not a table looped for ever, with the notes
// locked. Every input must come back, and soon.
func TestMarkdownAlwaysEnds(t *testing.T) {
	for _, src := range []string{"| a |", "text\n| a |", "| a |\n| b |", "> | a |", "- x\n| a |", "~~~\ncode", "|", "> >"} {
		done := make(chan string, 1)
		go func() { done <- md(src) }()
		select {
		case got := <-done:
			if strings.Contains(src, "a |") && !strings.Contains(got, "| a |") {
				t.Errorf("%q lost its text: %q", src, got)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%q never ended", src)
		}
	}
}

// Found in review: a link's address took in what an earlier rule had made, and its
// HTML landed inside href.
func TestMarkdownNoHTMLInHref(t *testing.T) {
	href := regexp.MustCompile(`href="([^"]*)"`)
	for _, src := range []string{"http://x[t](/a/onclick=foo/)", "http://x[[b c]]", "[x](http://a[[task-known]])", "http://x`\"a`", "[a](https://e.com/`x`)"} {
		got := md(src)
		for _, m := range href.FindAllStringSubmatch(got, -1) {
			if strings.ContainsAny(m[1], "<>") {
				t.Errorf("%q put HTML in an address: %q", src, got)
			}
		}
		if strings.Count(got, "<a ") != strings.Count(got, "</a>") {
			t.Errorf("%q left a link open: %q", src, got)
		}
	}
}

func TestJournalByHeading(t *testing.T) {
	if !isJournal("4. Diario") || !isJournal("Journal") || isJournal("6. Cierre") {
		t.Error("isJournal")
	}
}

// Any text at all: the renderer ends, writes only its own tags and attributes, and
// never puts HTML in an address.
func FuzzMarkdown(f *testing.F) {
	for _, s := range []string{"# a\n- [ ] b\n| c |\n|---|\n| d |", "[x](https://e.com) [[task-known]] `c` **b**", "> | a |\n~~~", "http://x[t](/a/)"} {
		f.Add(s)
	}
	tag := regexp.MustCompile(`<(/?)([a-zA-Z0-9]+)([^>]*)>`)
	attr := regexp.MustCompile(`\s([a-z]+)="([^"]*)"`)
	allowed := map[string]bool{"p": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "ul": true, "ol": true,
		"li": true, "pre": true, "code": true, "blockquote": true, "hr": true, "div": true, "table": true, "thead": true,
		"tbody": true, "tr": true, "th": true, "td": true, "a": true, "span": true, "strong": true, "em": true, "s": true}
	f.Fuzz(func(t *testing.T, src string) {
		got := md(src)
		for _, m := range tag.FindAllStringSubmatch(got, -1) {
			if !allowed[m[2]] {
				t.Fatalf("%q wrote <%s>", src, m[2])
			}
			if rest := strings.TrimSpace(attr.ReplaceAllString(m[3], "")); rest != "" && rest != "/" {
				t.Fatalf("%q wrote stray attributes %q in %q", src, rest, got)
			}
			for _, a := range attr.FindAllStringSubmatch(m[3], -1) {
				switch a[1] {
				case "class", "title", "rel":
				case "href":
					if strings.ContainsAny(a[2], "<>") || !regexp.MustCompile(`^(https?://|/|#|note\?)`).MatchString(a[2]) {
						t.Fatalf("%q links to %q", src, a[2])
					}
				default:
					t.Fatalf("%q wrote the attribute %q", src, a[1])
				}
			}
		}
	})
}
