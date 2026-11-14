package state

import (
	"bufio"
	"bytes"
	"encoding/json"
	"time"
)

// usageLimit is what Claude Code writes in apiError when the subscription's usage limit
// refused the turn.
const usageLimit = "usage_limit_reached"

// QuotaStop reads the end of a Claude Code transcript and says whether its last answer
// is the one Claude Code writes when the usage limit of the subscription refused the
// turn, and when the limit resets. Measured on 12-nov with 2.1.322, on a background
// session: the answer has apiError "usage_limit_reached", error "rate_limit" and
// quotaLimits.resetsAt, the reset in seconds since 1970 ("You've hit your 5-hour limit ·
// resets 12:40am (Europe/Madrid)" is its text, which is not read). The turn ends there
// and a background session waits for a message, for as long as it takes.
//
// ok is false when a prompt was typed after it (the session is trying again), when
// another answer followed, or when the answer names no reset: without the hour there is
// nothing to wait for.
func QuotaStop(path string) (resets time.Time, ok bool) {
	sc := bufio.NewScanner(bytes.NewReader(transcriptTail(path)))
	sc.Buffer(make([]byte, 0, 64<<10), tailBytes+1)
	for sc.Scan() {
		var e struct {
			Type     string `json:"type"`
			APIError string `json:"apiError"`
			Quota    struct {
				ResetsAt int64 `json:"resetsAt"`
			} `json:"quotaLimits"`
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue // the first line of the tail may be cut: it does not parse
		}
		switch e.Type {
		case "assistant":
			resets, ok = time.Time{}, false
			if e.APIError == usageLimit && e.Quota.ResetsAt > 0 {
				resets, ok = time.Unix(e.Quota.ResetsAt, 0), true
			}
		case "user":
			var typed string
			if json.Unmarshal(e.Message.Content, &typed) == nil {
				resets, ok = time.Time{}, false // a prompt, not a tool's result
			}
		}
	}
	return resets, ok
}
