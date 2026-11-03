package guard

import (
	"strings"
	"testing"
)

// Found in the review of 2026-11-03: common token shapes the guard did not know. The
// keys are built here, so this file holds no key-shaped text.
func TestMoreSecrets(t *testing.T) {
	r := func(n int) string { return strings.Repeat("a1B2", n)[:n] }
	for key, want := range map[string]string{
		"glrt-" + r(24):                  "a GitLab token",
		"sk" + "_live_" + r(24):          "a Stripe secret key",
		"npm" + "_" + r(36):              "an npm token",
		"hf" + "_" + r(34):               "a Hugging Face token",
		"dckr" + "_pat_" + r(27):         "a Docker Hub token",
		"SG" + "." + r(22) + "." + r(43): "a SendGrid API key",
	} {
		if got := FindSecret("TOKEN=" + key); got != want {
			t.Errorf("%s…: %q, want %q", key[:8], got, want)
		}
	}
	if got := FindSecret("npm_install and hf_hub are not tokens"); got != "" {
		t.Errorf("plain words taken for a token: %q", got)
	}
}
