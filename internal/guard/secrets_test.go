package guard

import (
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/i18n"
)

// The keys are built at run time, so this file holds no secret-shaped text for a
// scanner (this repository's own, or a user's) to stop on.
func TestMoreSecretShapes(t *testing.T) {
	for what, text := range map[string]string{
		"a Google AI Studio API key": "key=" + "AQ" + "." + strings.Repeat("Ab3_", 12),
		"a PostHog personal API key": "POSTHOG_KEY=" + "phx" + "_" + strings.Repeat("a1B2", 9),
		"an Elasticsearch API key":   "Authorization: " + "Api" + "Key " + strings.Repeat("QmFzZTY0", 6),
		"a Telegram bot token":       "token " + "123456789" + ":" + strings.Repeat("Zx9_", 9),
	} {
		if got := FindSecret(text); got != what {
			t.Errorf("%q: %q, want %q", text, got, what)
		}
		if out := RedactSecrets(text); !strings.Contains(out, "["+what+" removed]") {
			t.Errorf("redact %q: %q", text, out)
		}
	}
	for _, text := range []string{
		"phc" + "_" + strings.Repeat("a1B2", 9), // PostHog's project key is public
		"the ApiKey header",
		"at 12:30 the build took 45 seconds",
		"AQ.short",
	} {
		if got := FindSecret(text); got != "" {
			t.Errorf("%q taken for %s", text, got)
		}
	}
}

// A key that ends in a character that is not a letter or digit (AWS keys use / and +,
// Google's use -) is still a key: the end of a word is not where it ends.
func TestSecretsEndingInSymbols(t *testing.T) {
	aws := strings.Repeat("aB3dE5gH7j", 3) + "kL9nP1rT3"
	google := "AI" + "za" + strings.Repeat("Bc9_x", 6) + "Yz1q"
	for what, texts := range map[string][]string{
		"an AWS secret key": {"AWS_SECRET_ACCESS_KEY=" + aws + "/", "aws_secret_access_key = " + aws + "+\n"},
		"a Google API key":  {"key: " + google + "-", "key: " + google + "-\n"},
	} {
		for _, text := range texts {
			if got := FindSecret(text); got != what {
				t.Errorf("%q: %q, want %q", text, got, what)
			}
		}
	}
	if out := RedactSecrets("AWS_SECRET_ACCESS_KEY=" + aws + "/\nnext"); !strings.HasSuffix(out, " removed]\nnext") {
		t.Errorf("redact keeps what follows the key: %q", out)
	}
}

// Every kind of secret has its words in the catalogue, for the messages people read.
func TestSecretKindsAreTranslated(t *testing.T) {
	for _, p := range secretPatterns {
		for _, lang := range []string{"en", "es"} {
			if got := i18n.T(lang, "secret."+p.what); got == "" || strings.HasPrefix(got, "secret.") || (lang == "es" && got == p.what) {
				t.Errorf("%s in %s: %q", p.what, lang, got)
			}
		}
	}
}
