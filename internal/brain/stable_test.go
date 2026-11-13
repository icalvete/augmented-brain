package brain

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
)

// secrets.toml is part of the compatibility promise: its one setting, secrets, as 1.0
// writes it, is read by every 1.x.
func TestStableSecretsFile(t *testing.T) {
	dir := t.TempDir()
	written := "secrets = [\n  \"github token: gh in keyring\",\n]\n"
	if err := os.WriteFile(filepath.Join(dir, config.SecretsFile), []byte(written), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readSecretList(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"github token: gh in keyring"}) {
		t.Errorf("secrets.toml of 1.0 read as %q", got)
	}
}
