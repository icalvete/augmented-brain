// Package memory reads and checks the Markdown files of a space.
//
// The program only reads the YAML header of each file, never its prose, so the body can
// be written in any language. See docs/spec/memory-format.md.
package memory

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Types of file, and the folder each one lives in.
var Folders = map[string]string{
	"area":      "areas",
	"task":      "tasks",
	"reference": "knowledge",
	"lesson":    "knowledge",
	"rule":      "rules",
	"review":    "reviews",
	"person":    "people",
	"profile":   "",
}

// Task statuses: the six phases, plus the two that pause a task.
var Statuses = []string{
	"definition", "analysis", "plan", "execution", "validation", "closed",
	"parked", "someday",
}

// Watch is a date to look at something again.
type Watch struct {
	Date string `yaml:"date"`
	What string `yaml:"what"`
	Done bool   `yaml:"done"`
}

// Header is the YAML block at the top of a file. Keys are always English.
type Header struct {
	ID      string   `yaml:"id"`
	Type    string   `yaml:"type"`
	Title   string   `yaml:"title"`
	Summary string   `yaml:"summary"`
	Area    string   `yaml:"area,omitempty"`
	Repos   []string `yaml:"repos,omitempty"`
	Related []string `yaml:"related,omitempty"`
	Created string   `yaml:"created"`
	Updated string   `yaml:"updated"`

	// Tasks only.
	Status    string  `yaml:"status,omitempty"`
	WaitingOn string  `yaml:"waiting_on,omitempty"`
	Next      string  `yaml:"next,omitempty"`
	Watch     []Watch `yaml:"watch,omitempty"`
}

// Note is one file of the memory.
type Note struct {
	Path   string // relative to the memory folder, with forward slashes
	Header Header
	Body   string
}

var delim = []byte("---")

// Parse splits a file into its header and body, and checks the header.
// path is relative to the memory folder.
func Parse(path string, data []byte) (*Note, error) {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(data, append(delim, '\n')) {
		return nil, errors.New("missing header: the file must start with a line '---'")
	}
	rest := data[len(delim)+1:]
	end := bytes.Index(rest, append(append([]byte("\n"), delim...), '\n'))
	if end < 0 {
		if bytes.HasSuffix(rest, append([]byte("\n"), delim...)) {
			end = len(rest) - len(delim) - 1
		} else {
			return nil, errors.New("header not closed: add a line '---' after it")
		}
	}
	n := &Note{Path: filepath.ToSlash(path)}
	dec := yaml.NewDecoder(bytes.NewReader(rest[:end]))
	dec.KnownFields(true)
	if err := dec.Decode(&n.Header); err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}
	if body := end + len(delim) + 2; body < len(rest) {
		n.Body = string(rest[body:])
	}
	if err := n.check(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *Note) check() error {
	h := &n.Header
	var problems []string
	add := func(format string, a ...any) { problems = append(problems, fmt.Sprintf(format, a...)) }

	folder, known := Folders[h.Type]
	switch {
	case h.Type == "":
		add("type is missing")
	case !known:
		add("type %q is not one of: area, task, reference, lesson, rule, review, person, profile", h.Type)
	case filepath.ToSlash(filepath.Dir(n.Path)) != orDot(folder):
		add("a %s goes in %s/, not in %s", h.Type, orDot(folder), filepath.ToSlash(filepath.Dir(n.Path)))
	}
	if want := strings.TrimSuffix(filepath.Base(n.Path), ".md"); h.ID != want {
		add("id %q must match the file name %q", h.ID, want)
	}
	if h.Title == "" {
		add("title is missing")
	}
	if h.Type == "task" {
		if !contains(Statuses, h.Status) {
			add("status %q is not one of: %s", h.Status, strings.Join(Statuses, ", "))
		}
		if h.Status != "closed" && h.Next == "" {
			add("next is missing: the single next action, one line")
		}
		if h.Status != "closed" && h.WaitingOn == "" {
			add("waiting_on is missing: me, nobody, a person or a task id")
		}
	} else if h.Status != "" || h.Next != "" || h.WaitingOn != "" {
		add("status, next and waiting_on are only for tasks")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func orDot(folder string) string {
	if folder == "" {
		return "."
	}
	return folder
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
