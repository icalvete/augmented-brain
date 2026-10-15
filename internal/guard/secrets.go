package guard

import "regexp"

// High-confidence secret patterns only: a token with its exact shape, never words such
// as "password", so the guard does not block by mistake.
var secretPatterns = []struct {
	re   *regexp.Regexp
	what string
}{
	{regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`), "an AWS access key"},
	{regexp.MustCompile(`\bASIA[0-9A-Z]{16}\b`), "an AWS temporary access key"},
	{regexp.MustCompile(`(?i)aws_secret_access_key\s*[=:]\s*['"]?[A-Za-z0-9/+=]{40}\b`), "an AWS secret key"},
	{regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{30,}\b`), "a GitHub token"},
	{regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{40,}\b`), "a GitHub token"},
	{regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}\b`), "a GitLab token"},
	{regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}`), "an Anthropic API key"},
	{regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_-]{40,}`), "an OpenAI API key"},
	{regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`), "a Google API key"},
	{regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}`), "a Slack token"},
	{regexp.MustCompile(`\b[0-9]{8,10}:AA[A-Za-z0-9_-]{33}\b`), "a Telegram bot token"},
	{regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----`), "a private key"},
}

// FindSecret returns what kind of secret text contains, or "" if none.
func FindSecret(text string) string {
	for _, p := range secretPatterns {
		if p.re.MatchString(text) {
			return p.what
		}
	}
	return ""
}
