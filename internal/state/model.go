package state

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// ModelFile keeps, for the model check (config.Session), the warnings already given (a
// session is warned once per model, not after every answer) and what the last session
// start said about its model: whether Claude Code sent one at all, which only a real
// start can tell.
const ModelFile = "model.json"

// modelKeep is how long a warning is remembered.
const modelKeep = 7 * 24 * time.Hour

type modelWarning struct {
	Model string    `json:"model"`
	At    time.Time `json:"at"`
}

// StartModel is what the last session start said about its model.
type StartModel struct {
	At     time.Time `json:"at"`
	Source string    `json:"source,omitempty"`
	Model  string    `json:"model,omitempty"` // empty: Claude Code sent none
}

type modelMarks struct {
	Warned    map[string]modelWarning `json:"warned,omitempty"`
	LastStart *StartModel             `json:"last_start,omitempty"`
}

func modelFile(cfgDir string) string { return filepath.Join(cfgDir, Dir, ModelFile) }

func readModel(cfgDir string) modelMarks {
	var m modelMarks
	if data, err := os.ReadFile(modelFile(cfgDir)); err == nil {
		_ = json.Unmarshal(data, &m)
	}
	return m
}

// ModelWarned is the model a session was already warned about ("" for none). No lock:
// one small file read, after every answer.
func ModelWarned(cfgDir, session string) string {
	return readModel(cfgDir).Warned[session].Model
}

// LastStartModel is what the last session start said about its model, if recorded.
func LastStartModel(cfgDir string) (StartModel, bool) {
	m := readModel(cfgDir)
	if m.LastStart == nil {
		return StartModel{}, false
	}
	return *m.LastStart, true
}

func changeModel(cfgDir string, now time.Time, change func(*modelMarks)) error {
	unlock, err := filelock.Lock(filepath.Join(cfgDir, Dir, ".lock"))
	if err != nil {
		return err
	}
	defer unlock()
	m := readModel(cfgDir)
	change(&m)
	for k, w := range m.Warned {
		if now.Sub(w.At) > modelKeep {
			delete(m.Warned, k)
		}
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return paths.WriteAtomicQuick(modelFile(cfgDir), data, 0o600)
}

// MarkModel records that a session was warned about model.
func MarkModel(cfgDir, session, model string, now time.Time) error {
	return changeModel(cfgDir, now, func(m *modelMarks) {
		if m.Warned == nil {
			m.Warned = map[string]modelWarning{}
		}
		m.Warned[session] = modelWarning{model, now}
	})
}

// RecordStartModel records what a session start said about its model.
func RecordStartModel(cfgDir, source, model string, now time.Time) error {
	return changeModel(cfgDir, now, func(m *modelMarks) {
		m.LastStart = &StartModel{At: now, Source: source, Model: model}
	})
}

// tailBytes is how much of the end of a transcript LastModel reads: the last answers,
// never the whole file, which can be hundreds of megabytes.
const tailBytes = 512 << 10

// LastModel is the model of the last answer in a Claude Code transcript (JSONL), or ""
// when it has none yet or cannot be read. Answers Claude Code makes up itself (an error
// shown as an answer) carry a model in angle brackets and are skipped.
func LastModel(path string) string {
	data := transcriptTail(path)
	model := ""
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64<<10), tailBytes+1)
	for sc.Scan() {
		line := sc.Bytes()
		if !bytes.Contains(line, []byte(`"assistant"`)) {
			continue
		}
		var e struct {
			Type    string `json:"type"`
			Message struct {
				Model string `json:"model"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &e) != nil || e.Type != "assistant" {
			continue // the first line of the tail may be cut: it does not parse
		}
		if m := e.Message.Model; m != "" && m[0] != '<' {
			model = m
		}
	}
	return model
}

// transcriptTail is the end of a transcript (tailBytes at most), or nil when it cannot
// be read.
func transcriptTail(path string) []byte {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > tailBytes {
		if _, err := f.Seek(-tailBytes, io.SeekEnd); err != nil {
			return nil
		}
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil
	}
	return data
}

// LastAnswer reads the end of a Claude Code transcript: whether it holds an answer at
// all (answered), and, when the last one is an error Claude Code wrote itself because
// the API refused the turn, its text (apiErr), such as "Login reached its time limit ·
// Please run /login" (measured on 2.1.322: model "<synthetic>", isApiErrorMessage true,
// error "authentication_failed"). A message the user typed after the error clears it:
// the session is trying again.
func LastAnswer(path string) (answered bool, apiErr string) {
	data := transcriptTail(path)
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64<<10), tailBytes+1)
	for sc.Scan() {
		var e struct {
			Type     string `json:"type"`
			APIError bool   `json:"isApiErrorMessage"`
			Error    string `json:"error"`
			Message  struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue // the first line of the tail may be cut: it does not parse
		}
		switch e.Type {
		case "assistant":
			answered = true
			apiErr = ""
			if e.APIError {
				apiErr = answerText(e.Message.Content)
				if apiErr == "" {
					apiErr = e.Error
				}
				if apiErr == "" {
					apiErr = "API error"
				}
			}
		case "user":
			var typed string
			if json.Unmarshal(e.Message.Content, &typed) == nil {
				apiErr = "" // a prompt, not a tool's result
			}
		}
	}
	return answered, apiErr
}

// answerText is the text of an answer's content: a string, or its text blocks.
func answerText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			parts = append(parts, strings.TrimSpace(b.Text))
		}
	}
	return strings.Join(parts, " ")
}
