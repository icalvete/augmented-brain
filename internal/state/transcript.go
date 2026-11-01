package state

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

// Conversation reads a Claude Code transcript (JSONL, one event per line) and returns
// only what the user and the assistant wrote after since: no tool calls, no tool output,
// nothing else. When it is longer than max characters (not bytes), the last part is
// kept, never cutting a character in two.
//
// The format is Claude Code's own and is not documented, so it is read defensively: a
// line that is not what is expected is skipped, never an error. What it relies on: each
// line is a JSON object with "type" ("user" or "assistant"), "timestamp" (RFC 3339) and
// "message" with "content", either a string or a list of blocks of which only those of
// type "text" are taken.
func Conversation(path string, since time.Time, max int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var parts []string
	size := 0
	r := bufio.NewReaderSize(f, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			if who, text, ok := entry(line, since); ok {
				p := who + ": " + text
				parts = append(parts, p)
				size += utf8.RuneCountInString(p) + 2
				for size > max && len(parts) > 1 { // keep the most recent part
					size -= utf8.RuneCountInString(parts[0]) + 2
					parts = parts[1:]
				}
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
	}
	out := strings.Join(parts, "\n\n")
	if rs := []rune(out); len(rs) > max {
		out = "…" + string(rs[len(rs)-max:])
	}
	return out, nil
}

var (
	userMark      = []byte(`"user"`)
	assistantMark = []byte(`"assistant"`)
)

// entry reads one line of the transcript: who wrote it and the text, when it is a
// message of the user or the assistant written after since.
func entry(line []byte, since time.Time) (string, string, bool) {
	// Most lines are tool output, often huge: look before decoding.
	if !bytes.Contains(line, userMark) && !bytes.Contains(line, assistantMark) {
		return "", "", false
	}
	var e struct {
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
		IsMeta    bool   `json:"isMeta"`
		Message   struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &e) != nil || e.IsMeta || (e.Type != "user" && e.Type != "assistant") {
		return "", "", false
	}
	if t, err := time.Parse(time.RFC3339Nano, e.Timestamp); err != nil || !t.After(since) {
		return "", "", false
	}
	var text string
	if json.Unmarshal(e.Message.Content, &text) != nil {
		var blocks []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(e.Message.Content, &blocks) != nil {
			return "", "", false
		}
		var texts []string
		for _, b := range blocks {
			if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
				texts = append(texts, strings.TrimSpace(b.Text))
			}
		}
		text = strings.Join(texts, "\n")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", "", false
	}
	return e.Type, text, true
}
