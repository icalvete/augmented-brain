package guard

import (
	"strings"
	"testing"
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
