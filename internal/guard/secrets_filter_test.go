package guard

// The search for secrets runs each expression only when its cheap test (may) says the
// text may hold a match, and finds encoded Elasticsearch keys by hand. Both must answer
// exactly as the plain expressions did: those stay here, as the reference, and every
// text below goes through both. A filter that is wrong lets a secret through, so a new
// pattern brings its test and its pieces in randomText.

import (
	"encoding/base64"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// base64Run and encodedElasticKeysRegexp are how encodedElasticKeys worked before it
// was walked by hand.
var base64Run = regexp.MustCompile(`[A-Za-z0-9+/]{40,80}={0,2}`)

func encodedElasticKeysRegexp(text string) [][]int {
	var out [][]int
	for _, loc := range base64Run.FindAllStringIndex(text, -1) {
		if loc[0] > 0 && isB64(text[loc[0]-1]) || loc[1] < len(text) && (isB64(text[loc[1]]) || text[loc[1]] == '=') {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(text[loc[0]:loc[1]])
		if err != nil || len(data) < 36 || data[20] != ':' || !idChars(data[:20]) || !idChars(data[21:]) {
			continue
		}
		out = append(out, loc)
	}
	return out
}

// findSecretRegexp and redactSecretsRegexp are FindSecret and RedactSecrets with no
// filter: every expression over the whole text.
func findSecretRegexp(text string) string {
	for _, p := range secretPatterns {
		for _, m := range p.re.FindAllString(text, -1) {
			if !isExample(m) {
				return p.what
			}
		}
	}
	if len(encodedElasticKeysRegexp(text)) > 0 {
		return elasticKey
	}
	return ""
}

func redactSecretsRegexp(text string) string {
	for _, p := range secretPatterns {
		text = p.re.ReplaceAllStringFunc(text, func(m string) string {
			if isExample(m) {
				return m
			}
			return "[" + p.what + " removed]"
		})
	}
	locs := encodedElasticKeysRegexp(text)
	for i := len(locs) - 1; i >= 0; i-- {
		text = text[:locs[i][0]] + "[" + elasticKey + " removed]" + text[locs[i][1]:]
	}
	return text
}

// pieces of secret-shaped text, built so this file holds none whole.
var pieces = []string{
	"AKIA", "ASIA", "aws_secret_access_key", "AWS_SECRET_ACCESS_KEY", "Aws_Secret_Access_Key",
	"awſ_secret_acceſſ_Key", "aws_ſecret_access_Key",
	"ghp_", "gho_", "ghu_", "ghs_", "ghr_", "github_pat_", "glpat-", "glrt-", "gldt-", "glptt-", "glcbt-", "glsoat-", "glft-",
	"sk_live_", "rk_live_", "npm_", "hf_", "dckr_pat_", "SG.", "sk-ant-", "sk-", "sk-proj-", "AIza", "xoxb-", "xoxp-", "xoxa-", "xox", "AQ.", "phx_",
	"Api" + "Key ", "Api" + "Key\t", "-----BEGIN " + "PRIVATE KEY-----", "-----BEGIN " + "RSA PRIVATE KEY-----", "-----BEGIN ",
	"12345678:", "1234567890:", "123456789012:",
	" ", "\n", "=", "==", "===", " = ", ": ", "'", "\"", ".", "/", "+", "-", "_", "task-", "x", "é", "K",
	"IOSFODNN7" + "EXAMPLE", "wJalrXUtnFEMI/K7MDENG/" + "bPxRfiCYEXAMPLEKEY",
}

const tokenChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// randomText joins pieces, runs of token characters of lengths near the limits of the
// patterns, encoded Elasticsearch keys (and near misses) and plain words.
func randomText(r *rand.Rand) string {
	var b strings.Builder
	for n := r.Intn(12); n >= 0; n-- {
		switch r.Intn(6) {
		case 0, 1:
			b.WriteString(pieces[r.Intn(len(pieces))])
		case 2:
			lens := []int{8, 10, 16, 19, 20, 22, 29, 30, 35, 36, 39, 40, 41, 43, 79, 80, 81, 120}
			chars := tokenChars
			switch r.Intn(4) {
			case 0:
				chars += "_-"
			case 1:
				chars += "+/"
			case 2:
				chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			}
			for k := lens[r.Intn(len(lens))]; k > 0; k-- {
				b.WriteByte(chars[r.Intn(len(chars))])
			}
		case 3:
			id, key := make([]byte, 20), make([]byte, 16+r.Intn(12))
			for i := range id {
				id[i] = tokenChars[r.Intn(len(tokenChars))]
			}
			for i := range key {
				key[i] = tokenChars[r.Intn(len(tokenChars))]
			}
			sep := ":"
			if r.Intn(5) == 0 {
				sep = ";"
			}
			b.WriteString(base64.StdEncoding.EncodeToString([]byte(string(id) + sep + string(key))))
		case 4:
			if r.Intn(2) == 0 { // a key of an exact length, or nearly
				shape := []struct {
					prefix, chars string
					n             int
				}{
					{"AKIA", "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 16},
					{"ASIA", "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 16},
					{"npm_", tokenChars, 36},
					{"AIza", tokenChars + "_-", 35},
					{"aws_secret_access_key = ", tokenChars + "/+", 40},
					{"AWS_\u017fECRET_ACCESS_\u212aEY:'", tokenChars + "/+", 40},
				}[r.Intn(6)]
				b.WriteString(shape.prefix)
				for k := shape.n + r.Intn(3) - 1; k > 0; k-- {
					b.WriteByte(shape.chars[r.Intn(len(shape.chars))])
				}
				break
			}
			if r.Intn(3) == 0 { // a SendGrid key, or nearly
				b.WriteString("SG.")
				for k := 22; k > 0; k-- {
					b.WriteByte(tokenChars[r.Intn(len(tokenChars))])
				}
				b.WriteString(".")
				for k := 42 + r.Intn(3); k > 0; k-- {
					b.WriteByte(tokenChars[r.Intn(len(tokenChars))])
				}
				break
			}
			digit := "0123456789"[r.Intn(10):][:1]
			b.WriteString(strings.Repeat(digit, 7+r.Intn(5)) + ":")
		default:
			b.WriteString("word ")
		}
	}
	return b.String()
}

func TestFilterAgrees(t *testing.T) {
	n := 60000
	if raceEnabled || testing.Short() {
		n = 4000
	}
	r := rand.New(rand.NewSource(1))
	found := map[string]int{}
	for i := 0; i < n; i++ {
		text := randomText(r)
		want, got := findSecretRegexp(text), FindSecret(text)
		if want != got {
			t.Fatalf("without the filter %q, with it %q, for %q", want, got, text)
		}
		if i%8 == 0 {
			if want, got := redactSecretsRegexp(text), RedactSecrets(text); want != got {
				t.Fatalf("RedactSecrets(%q): without the filter %q, with it %q", text, want, got)
			}
		}
		found[want]++
	}
	for _, p := range secretPatterns {
		if found[p.what] == 0 {
			t.Errorf("the random texts never held %s: the comparison does not cover it", p.what)
		}
	}
	t.Logf("kinds found in %d texts: %v", n, found)
}

// Every pattern's test says yes to every match the pattern finds in the random texts:
// stricter than TestFilterAgrees, which only sees the first kind found in each.
func TestEachFilterHoldsItsMatches(t *testing.T) {
	n := 20000
	if raceEnabled || testing.Short() {
		n = 1000
	}
	r := rand.New(rand.NewSource(2))
	for i := 0; i < n; i++ {
		text := randomText(r)
		for _, p := range secretPatterns {
			for _, m := range p.re.FindAllStringIndex(text, -1) {
				// The match itself, with what is just before it (for \b), must pass.
				from := max(m[0]-1, 0)
				if !p.may(text) || !p.may(text[from:m[1]]) {
					t.Fatalf("%s: %q matches in %q, but its test says no", p.what, text[m[0]:m[1]], text)
				}
			}
		}
	}
}

// repositoryTexts are the text files of this repository: code, documentation, scripts
// and test data, some of it secret-shaped on purpose. With -race, one in ten.
func repositoryTexts(tb testing.TB) []string {
	var texts []string
	root := filepath.Join("..", "..")
	seen := 0
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if name := d.Name(); name == ".git" || name == "node_modules" || name == "logs" {
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(p) {
		case ".go", ".md", ".sh", ".py", ".ps1", ".yml", ".json", ".toml", ".txt", ".html", ".js", ".css":
		default:
			return nil
		}
		if seen++; raceEnabled && seen%10 != 0 {
			return nil
		}
		if data, err := os.ReadFile(p); err == nil {
			texts = append(texts, string(data))
		}
		return nil
	})
	if len(texts) < 20 {
		tb.Fatalf("only %d files found under %s", len(texts), root)
	}
	return texts
}

func TestFilterAgreesOnRepository(t *testing.T) {
	for _, text := range repositoryTexts(t) {
		if want, got := findSecretRegexp(text), FindSecret(text); want != got {
			t.Fatalf("without the filter %q, with it %q, on a file of %d bytes", want, got, len(text))
		}
		if want, got := redactSecretsRegexp(text), RedactSecrets(text); want != got {
			t.Fatalf("RedactSecrets differs on a file of %d bytes", len(text))
		}
	}
}

// cleanText is about 1 MB of the guides, prose much like notes, with no secret. (This
// package's own code is a poor sample: it is full of the literals the tests look for.)
func cleanText(tb testing.TB) string {
	files, _ := filepath.Glob(filepath.Join("..", "..", "docs", "guide", "*.md"))
	if len(files) == 0 {
		tb.Fatal("no guides found")
	}
	var b strings.Builder
	for b.Len() < 1<<20 {
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				tb.Fatal(err)
			}
			b.Write(data)
		}
	}
	text := b.String()
	if got := FindSecret(text); got != "" {
		tb.Fatalf("the guides hold %s", got)
	}
	return text
}

// The search reads text with no secret at 20 MB/s at least. Without the filters it read
// about 2.5 MB/s; with them, well over 50 on a laptop. The floor is low so a slow CI
// machine passes, and high enough that losing the filters fails.
func TestFindSecretSpeed(t *testing.T) {
	if raceEnabled || testing.Short() {
		t.Skip("timing means nothing with -race or -short")
	}
	text := cleanText(t)
	res := testing.Benchmark(func(b *testing.B) {
		b.SetBytes(int64(len(text)))
		for b.Loop() {
			_ = FindSecret(text)
		}
	})
	mbs := float64(res.Bytes) * float64(res.N) / 1e6 / res.T.Seconds()
	t.Logf("%.0f MB/s", mbs)
	if mbs < 20 {
		t.Errorf("FindSecret reads %.1f MB/s, want 20 at least", mbs)
	}
}

func BenchmarkFindSecret(b *testing.B) {
	text := cleanText(b)
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		_ = FindSecret(text)
	}
}

func BenchmarkFindSecretWithoutFilter(b *testing.B) {
	text := cleanText(b)
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		_ = findSecretRegexp(text)
	}
}

func BenchmarkRedactSecrets(b *testing.B) {
	text := cleanText(b)
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		_ = RedactSecrets(text)
	}
}
