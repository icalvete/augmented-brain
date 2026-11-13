package guard

import (
	"encoding/base64"
	"regexp"
	"strings"
)

// High-confidence secret patterns only: a token with its exact shape, never words such
// as "password", so the guard does not block by mistake.
//
// Each pattern carries may: a cheap test that every match of it passes (a literal it
// must contain, at a word boundary when the pattern starts with one). The expression
// runs only when may says yes. Go finds no literal prefix behind \b, so without it
// each expression walks the whole text, about 2.5 MB/s for all of them; with it, the
// whole search runs at well over 50 MB/s. A new pattern brings its test, and
// TestFilterAgrees in secrets_filter_test.go compares the answers with and without.
var secretPatterns = []struct {
	re   *regexp.Regexp
	what string
	may  func(text string) bool
}{
	{regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`), "an AWS access key", atWord("AKIA")},
	{regexp.MustCompile(`\bASIA[0-9A-Z]{16}\b`), "an AWS temporary access key", atWord("ASIA")},
	{regexp.MustCompile(`(?i)aws_secret_access_key\s*[=:]\s*['"]?[A-Za-z0-9/+=]{40}`), "an AWS secret key", mayAWSSecret}, // no \b: a key may end in / or +
	{regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{30,}\b`), "a GitHub token", atWord("ghp_", "gho_", "ghu_", "ghs_", "ghr_")},
	{regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{40,}\b`), "a GitHub token", atWord("github_pat_")},
	{regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}\b`), "a GitLab token", atWord("glpat-")},
	{regexp.MustCompile(`\bgl(?:rt|dt|ptt|cbt|soat|ft)-[A-Za-z0-9_-]{20,}\b`), "a GitLab token", atWord("glrt-", "gldt-", "glptt-", "glcbt-", "glsoat-", "glft-")},
	{regexp.MustCompile(`\b[rs]k_live_[0-9A-Za-z]{20,}\b`), "a Stripe secret key", atWord("rk_live_", "sk_live_")},
	{regexp.MustCompile(`\bnpm_[A-Za-z0-9]{36}\b`), "an npm token", atWord("npm_")},
	{regexp.MustCompile(`\bhf_[A-Za-z0-9]{30,}\b`), "a Hugging Face token", atWord("hf_")},
	{regexp.MustCompile(`\bdckr_pat_[A-Za-z0-9_-]{20,}\b`), "a Docker Hub token", atWord("dckr_pat_")},
	{regexp.MustCompile(`\bSG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}\b`), "a SendGrid API key", atWord("SG.")},
	{regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}`), "an Anthropic API key", atWord("sk-ant-")},
	{regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_-]{40,}`), "an OpenAI API key", atWord("sk-")},
	{regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`), "a Google API key", atWord("AIza")}, // no \b: a key may end in -
	{regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}`), "a Slack token", atWord("xoxa-", "xoxb-", "xoxp-", "xoxr-", "xoxs-")},
	{regexp.MustCompile(`\bAQ\.[A-Za-z0-9_-]{40,}`), "a Google AI Studio API key", atWord("AQ.")},
	{regexp.MustCompile(`\bphx_[A-Za-z0-9]{30,}\b`), "a PostHog personal API key", atWord("phx_")}, // phc_, the project key, is public
	{regexp.MustCompile(`\bApiKey\s+[A-Za-z0-9+/=]{30,}`), elasticKey, atWord("ApiKey")},           // without ApiKey: encodedElasticKeys
	{regexp.MustCompile(`\b[0-9]{8,10}:[A-Za-z0-9_-]{30,}`), "a Telegram bot token", mayTelegram},
	{regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----`), "a private key", contains("-----BEGIN ")},
}

// examples are the made-up keys AWS's own documentation uses: they open no account, and
// notes and commands quote them. Built in pieces so this file holds no key-shaped text.
var examples = []string{
	"AKIA" + "IOSFODNN7" + "EXAMPLE",
	"wJalrXUtnFEMI/K7MDENG/" + "bPxRfiCYEXAMPLEKEY",
}

const elasticKey = "an Elasticsearch API key"

// encodedElasticKeys returns where text holds an Elasticsearch API key written without
// "ApiKey" before it: base64 of "<id>:<key>", alone. A candidate is a run of 40 to 80
// base64 characters that is not part of a longer one, with up to two "=" after it; it
// counts only when it decodes to a 20-character id, a colon and a key of the same
// alphabet, so other base64 (a hash, an image) is not taken for one. Walked by hand: the
// expression it replaces, [A-Za-z0-9+/]{40,80}={0,2} with its edges checked, ran at about
// 20 MB/s; the tests keep it to compare (encodedElasticKeysRegexp).
func encodedElasticKeys(text string) [][]int {
	var out [][]int
	for i := 0; i < len(text); {
		if !isB64(text[i]) {
			i++
			continue
		}
		start := i
		for i < len(text) && isB64(text[i]) {
			i++
		}
		if n := i - start; n < 40 || n > 80 {
			continue
		}
		end := i
		for end < len(text) && end-i < 2 && text[end] == '=' {
			end++
		}
		if end < len(text) && (text[end] == '=' || isB64(text[end])) {
			continue // a third "=", or more base64 after the padding: part of something longer
		}
		data, err := base64.StdEncoding.DecodeString(text[start:end])
		if err != nil || len(data) < 36 || data[20] != ':' || !idChars(data[:20]) || !idChars(data[21:]) {
			continue
		}
		out = append(out, []int{start, end})
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
		if !p.may(text) {
			continue
		}
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
		if !p.may(text) {
			continue
		}
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

// The tests of secretPatterns' may. Each one must say yes to every text the pattern
// matches; saying yes to more only costs time.

func wordByte(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_'
}

// contains is the test of a pattern that holds lit wherever it matches.
func contains(lit string) func(string) bool {
	return func(text string) bool { return strings.Contains(text, lit) }
}

// atWord is the test of a pattern that starts with \b and one of lits: one of them is in
// the text with no word character just before it. Go's \b knows only ASCII word
// characters, as wordByte does.
func atWord(lits ...string) func(string) bool {
	return func(text string) bool {
		for _, lit := range lits {
			for from := 0; ; {
				i := strings.Index(text[from:], lit)
				if i < 0 {
					break
				}
				i += from
				if i == 0 || !wordByte(text[i-1]) {
					return true
				}
				from = i + 1
			}
		}
		return false
	}
}

// mayAWSSecret is the test of (?i)aws_secret_access_key. Under (?i), k and s also match
// the Kelvin sign and the long s, so it looks only for the part with neither,
// "ecret_acce", whose letters fold within ASCII.
func mayAWSSecret(text string) bool {
	const before, after = "ecret", "acce"
	for from := 0; ; {
		i := strings.IndexByte(text[from:], '_')
		if i < 0 {
			return false
		}
		i += from
		if i >= len(before) && i+1+len(after) <= len(text) &&
			strings.EqualFold(text[i-len(before):i], before) && strings.EqualFold(text[i+1:i+1+len(after)], after) {
			return true
		}
		from = i + 1
	}
}

func tokenByte(c byte) bool { return wordByte(c) || c == '-' }

// mayTelegram is the test of \b[0-9]{8,10}:[A-Za-z0-9_-]{30,}: a colon with eight digits
// just before it and thirty token characters after it.
func mayTelegram(text string) bool {
	for from := 0; ; {
		i := strings.IndexByte(text[from:], ':')
		if i < 0 {
			return false
		}
		i += from
		from = i + 1
		if i < 8 || i+31 > len(text) {
			continue
		}
		if allBytes(text[i-8:i], func(c byte) bool { return c >= '0' && c <= '9' }) && allBytes(text[i+1:i+31], tokenByte) {
			return true
		}
	}
}

func allBytes(s string, ok func(byte) bool) bool {
	for i := 0; i < len(s); i++ {
		if !ok(s[i]) {
			return false
		}
	}
	return true
}
