package brain

import (
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
)

// The secrets list keeps names and what they are for, never a value.
func TestSecretNames(t *testing.T) {
	b := open(t, "en")
	if _, err := b.SaveSecretName(SecretName{Service: "telegram", Name: "alerts-bot", What: "alerts by Telegram"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SaveSecretName(SecretName{Service: "mail", Name: "smtp"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SaveSecretName(SecretName{Service: "telegram", Name: "alerts-bot", What: "the alerts bot"}); err != nil {
		t.Fatal(err)
	}
	key := "AK" + "IA" + "QWERTYUIOPASDFGH" // built here, so this file holds no key-shaped text
	if _, err := b.SaveSecretName(SecretName{Service: "aws", Name: "deploy", What: "key " + key}); err != ErrSecretValue {
		t.Errorf("a value was kept: %v", err)
	}
	for _, bad := range []SecretName{{Service: "two words", Name: "x"}, {Service: "a", Name: ""}, {Service: "a/b", Name: "c"}, {Service: "-x", Name: "c"}, {Service: "a", Name: "--help"}} {
		if _, err := b.SaveSecretName(bad); err == nil {
			t.Errorf("%+v was kept", bad)
		}
	}
	got, _ := SecretNames(b.ConfigDir)
	if len(got) != 2 || got[0].Key() != "telegram/alerts-bot" || got[0].What != "the alerts bot" || got[1].Key() != "mail/smtp" {
		t.Fatalf("secrets: %+v", got)
	}
	if err := b.ForgetSecretName("mail", "smtp"); err != nil {
		t.Fatal(err)
	}
	if err := b.ForgetSecretName("mail", "smtp"); err == nil {
		t.Error("forgot one that was not there")
	}
	if l := SecretList(b.ConfigDir); strings.Join(l, "|") != "telegram/alerts-bot: the alerts bot" {
		t.Errorf("list: %q", l)
	}
	// Never in config.toml: programs before 0.6.0 refuse a setting they do not know.
	if c, err := config.Load(b.ConfigDir); err != nil || len(c.Secrets) != 0 {
		t.Errorf("config.toml: %v %v", c.Secrets, err)
	}
}

// Found in review: a control character could not be written to the configuration, and
// a backup with one made the whole restore fail.
func TestSecretNamesPlain(t *testing.T) {
	b := open(t, "en")
	for _, what := range []string{"bell\a", "two\nlines", "a \"quote\"", "back\\slash", "bad \xff"} {
		if _, err := b.SaveSecretName(SecretName{Service: "s", Name: "n", What: what}); err != ErrSecretName {
			t.Errorf("%q: %v", what, err)
		}
	}
	got := CleanSecretNames([]string{"a/b: fine", "c/d: bell\a", "not an entry", "e/f"})
	if strings.Join(got, "|") != "a/b: fine|e/f" {
		t.Errorf("clean: %q", got)
	}
}

// 0.6.0 kept the list in config.toml: it moves to its own file, once, and the
// configuration is left as earlier programs can read it.
func TestMoveSecrets(t *testing.T) {
	b := open(t, "en")
	if err := config.SetTopList(b.ConfigDir, "secrets", []string{"mail/smtp: sending mail", "bad entry"}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := MoveSecrets(b.ConfigDir); err != nil {
			t.Fatal(err)
		}
	}
	if c, err := config.Load(b.ConfigDir); err != nil || len(c.Secrets) != 0 {
		t.Errorf("config.toml still has the list: %v %v", c, err)
	}
	if l := SecretList(b.ConfigDir); strings.Join(l, "|") != "mail/smtp: sending mail" {
		t.Errorf("file: %q", l)
	}
}
