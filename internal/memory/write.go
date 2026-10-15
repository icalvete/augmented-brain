package memory

import (
	"bytes"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Render returns the file text: the header, always in the same key order, then the body.
func (n *Note) Render() ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("---\n")
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(&n.Header); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	b.WriteString("---\n")
	b.WriteString(n.Body)
	return b.Bytes(), nil
}

// Save checks the note and writes it under dir, replacing the file in one step so a
// crash never leaves half a note.
func Save(dir string, n *Note) error {
	if err := n.check(); err != nil {
		return err
	}
	data, err := n.Render()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, filepath.FromSlash(n.Path))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
