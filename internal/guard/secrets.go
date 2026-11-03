package guard

import (
	"encoding/base64"
	"regexp"
	"strings"
)

// High-confidence secret patterns only: a token with its exact shape, never words such
// as "password", so the guard does not block by mistake.
var secretPatterns = []struct {
	re   *regexp.Regexp
	what string
}{
	{regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`), "an AWS access key"},
	{regexp.MustCompile(`\bASIA[0-9A-Z]{16}\b`), "an AWS temporary access key"},
	{regexp.MustCompile(`(?i)aws_secret_access_key\s*[=:]\s*['"]?[A-Za-z0-9/+=]{40}`), "an AWS secret key"}, // no \b: a key may end in / or +
	{regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{30,}\b`), "a GitHub token"},
	{regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{40,}\b`), "a GitHub token"},
	{regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}\b`), "a GitLab token"},
	{regexp.MustCompile(`\bgl(?:rt|dt|ptt|cbt|soat|ft)-[A-Za-z0-9_-]{20,}\b`), "a GitLab token"},
	{regexp.MustCompile(`\b[rs]k_live_[0-9A-Za-z]{20,}\b`), "a Stripe secret key"},
	{regexp.MustCompile(`\bnpm_[A-Za-z0-9]{36}\b`), "an npm token"},
	{regexp.MustCompile(`\bhf_[A-Za-z0-9]{30,}\b`), "a Hugging Face token"},
	{regexp.MustCompile(`\bdckr_pat_[A-Za-z0-9_-]{20,}\b`), "a Docker Hub token"},
	{regexp.MustCompile(`\bSG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}\b`), "a SendGrid API key"},
	{regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}`), "an Anthropic API key"},
	{regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_-]{40,}`), "an OpenAI API key"},
	{regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`), "a Google API key"}, // no \b: a key may end in -
	{regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}`), "a Slack token"},
	{regexp.MustCompile(`\bAQ\.[A-Za-z0-9_-]{40,}`), "a Google AI Studio API key"},
	{regexp.MustCompile(`\bphx_[A-Za-z0-9]{30,}\b`), "a PostHog personal API key"}, // phc_, the project key, is public
	{regexp.MustCompile(`\bApiKey\s+[A-Za-z0-9+/=]{30,}`), elasticKey},             // without ApiKey: encodedElasticKeys
	{regexp.MustCompile(`\b[0-9]{8,10}:[A-Za-z0-9_-]{30,}`), "a Telegram bot token"},
	{regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----`), "a private key"},
}

// examples are the made-up keys AWS's own documentation uses: they open no account, and
// notes and commands quote them. Built in pieces so this file holds no key-shaped text.
var examples = []string{
	"AKIA" + "IOSFODNN7" + "EXAMPLE",
	"wJalrXUtnFEMI/K7MDENG/" + "bPxRfiCYEXAMPLEKEY",
}

const elasticKey = "an Elasticsearch API key"

// base64Run is a candidate for an Elasticsearch API key in its "encoded" form, alone:
// base64 of "<id>:<key>". The edges are checked by hand (Go has no lookbehind).
var base64Run = regexp.MustCompile(`[A-Za-z0-9+/]{40,80}={0,2}`)

// encodedElasticKeys returns where text holds an Elasticsearch API key written without
// "ApiKey" before it. Only a run that decodes to a 20-character id, a colon and a key of
// the same alphabet counts, so other base64 (a hash, an image) is not taken for one.
func encodedElasticKeys(text string) [][]int {
	var out [][]int
	for _, loc := range base64Run.FindAllStringIndex(text, -1) {
		if loc[0] > 0 && isB64(text[loc[0]-1]) || loc[1] < len(text) && (isB64(text[loc[1]]) || text[loc[1]] == '=') {
			continue // part of a longer run
		}
		data, err := base64.StdEncoding.DecodeString(text[loc[0]:loc[1]])
		if err != nil || len(data) < 36 || data[20] != ':' || !idChars(data[:20]) || !idChars(data[21:]) {
			continue
		}
		out = append(out, loc)
	}
	return out
}

func isB64(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '+' || c == '/'
}

func idChars(b []byte) bool {
	for _, c := range b {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

// FindSecret returns what kind of secret text contains, or "" if none.
func FindSecret(text string) string {
	for _, p := range secretPatterns {
		for _, m := range p.re.FindAllString(text, -1) {
			if !isExample(m) {
				return p.what
			}
		}
	}
	if len(encodedElasticKeys(text)) > 0 {
		return elasticKey
	}
	return ""
}

// CountSecrets returns how many secrets text holds (AWS's examples apart).
func CountSecrets(text string) int {
	return strings.Count(RedactSecrets(text), " removed]") - strings.Count(text, " removed]")
}

// RedactSecrets replaces each secret in text (AWS's examples apart) with a marker, for
// text written by a program, such as a conversation summary, that must never hold one.
func RedactSecrets(text string) string {
	for _, p := range secretPatterns {
		text = p.re.ReplaceAllStringFunc(text, func(m string) string {
			if isExample(m) {
				return m
			}
			return "[" + p.what + " removed]"
		})
	}
	locs := encodedElasticKeys(text)
	for i := len(locs) - 1; i >= 0; i-- {
		text = text[:locs[i][0]] + "[" + elasticKey + " removed]" + text[locs[i][1]:]
	}
	return text
}

func isExample(match string) bool {
	for _, e := range examples {
		if strings.Contains(match, e) {
			return true
		}
	}
	return false
}
