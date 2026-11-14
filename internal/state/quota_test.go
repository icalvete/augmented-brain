package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestQuotaStop(t *testing.T) {
	dir := t.TempDir()
	prompt := `{"type":"user","message":{"role":"user","content":"go on"}}` + "\n"
	// As Claude Code 2.1.322 wrote it on 12-nov, without what is not read.
	limit := `{"type":"assistant","isSidechain":false,"message":{"model":"<synthetic>","role":"assistant","content":[{"type":"text","text":"You've hit your 5-hour limit · resets 12:40am (Europe/Madrid)"}]},"apiError":"usage_limit_reached","apiErrorParams":{"rate_limit_info":{"status":"rejected","resetsAt":1794526800,"rateLimitType":"five_hour"}},"quotaLimits":{"status":"rejected","resetsAt":1794526800,"rateLimitType":"five_hour"},"error":"rate_limit","isApiErrorMessage":true,"apiErrorStatus":429}` + "\n"
	noHour := `{"type":"assistant","message":{"model":"<synthetic>","content":[{"type":"text","text":"limit"}]},"apiError":"usage_limit_reached","error":"rate_limit","isApiErrorMessage":true}` + "\n"
	login := `{"type":"assistant","message":{"model":"<synthetic>","content":[{"type":"text","text":"Please run /login"}]},"error":"authentication_failed","isApiErrorMessage":true}` + "\n"
	ok := `{"type":"assistant","message":{"model":"claude-opus-5-5","content":[{"type":"text","text":"On it."}]}}` + "\n"
	tool := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"x"}]}}` + "\n"
	after := `{"type":"system","subtype":"turn_duration"}` + "\n"
	cases := []struct {
		name, body string
		ok         bool
	}{
		{"none", "", false},
		{"limit", prompt + limit + after, true},
		{"limit-tool", prompt + limit + tool, true},
		{"typed", prompt + limit + prompt, false},
		{"recovered", prompt + limit + prompt + ok, false},
		{"no-hour", prompt + noHour, false},
		{"login", prompt + login, false},
		{"ok", prompt + ok, false},
		{"cut", `ut":"x"}` + "\n" + limit, true},
	}
	for _, c := range cases {
		p := filepath.Join(dir, "missing")
		if c.body != "" {
			p = filepath.Join(dir, c.name+".jsonl")
			if err := os.WriteFile(p, []byte(c.body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		resets, got := QuotaStop(p)
		if got != c.ok {
			t.Errorf("%s: ok = %v, want %v", c.name, got, c.ok)
		}
		if want := time.Unix(1794526800, 0); got && !resets.Equal(want) {
			t.Errorf("%s: resets = %v, want %v", c.name, resets, want)
		}
		if !got && !resets.IsZero() {
			t.Errorf("%s: a reset with no stop: %v", c.name, resets)
		}
	}
}
