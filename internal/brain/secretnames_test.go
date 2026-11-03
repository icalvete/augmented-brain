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
	c, err := config.Load(b.ConfigDir)
	if err != nil {
		t.Fatal(err)
	}
	got := SecretNames(c)
	if len(got) != 2 || got[0].Key() != "telegram/alerts-bot" || got[0].What != "the alerts bot" || got[1].Key() != "mail/smtp" {
		t.Fatalf("secrets: %+v", got)
	}
	if err := b.ForgetSecretName("mail", "smtp"); err != nil {
		t.Fatal(err)
	}
	if err := b.ForgetSecretName("mail", "smtp"); err == nil {
		t.Error("forgot one that was not there")
	}
	c, _ = config.Load(b.ConfigDir)
	if strings.Join(c.Secrets, "|") != "telegram/alerts-bot: the alerts bot" {
		t.Errorf("list: %q", c.Secrets)
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
