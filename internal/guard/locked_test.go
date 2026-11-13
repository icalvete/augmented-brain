package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The guard cannot be switched off or softened (Rev 29a, closing S-02 and S-03 of the
// search for ways around it, Rev 29): sudo-rs and su-rs are root like sudo and su;
// commands that write a folder holding the guard's files ask as a write of the file
// does; and while the lists are not the ones the user last saw, what they would let
// through no longer does. S-01 and the other half of S-03 show only in the hook
// (internal/hook/locked_test.go).

// lockedPolicy is pinnedPolicy with the configuration folder in its home: guard.toml
// and config.toml protected, and the plugin's cache folder protected as a whole.
func lockedPolicy(t *testing.T) (Policy, string) {
	t.Helper()
	p := pinnedPolicy(t)
	cfg := filepath.Join(p.Cwd, ".config", "augmented-brain")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"guard.toml", "config.toml"} {
		if err := os.WriteFile(filepath.Join(cfg, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p.Protected = []string{filepath.Join(cfg, "guard.toml"), filepath.Join(cfg, "config.toml")}
	cache := filepath.Join(p.Cwd, ".claude", "plugins", "cache", "augmented-brain")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	p.ProtectedDirs = []string{cache}
	return p, cfg
}

func TestRootByAnyName(t *testing.T) {
	p := pinnedPolicy(t)
	checkPinned(t, p, []pinned{
		{"su_do-rs id", Deny},
		{"su_do-rs -u www-data id", Allow},
		{"/usr/bin/su_do-rs id", Deny},
		{"su-rs", Deny},
		{"su-rs -c id", Deny},
		{"timeout 5 su_do-rs id", Deny},
	})
}

func TestGuardFolderWrites(t *testing.T) {
	p, cfg := lockedPolicy(t)
	cache := p.ProtectedDirs[0]
	r := strings.NewReplacer("CFG", filepath.ToSlash(cfg), "CACHE", filepath.ToSlash(cache))
	rows := []pinned{
		// The folder that holds the guard's files, as a destination.
		{"cp /tmp/x CFG/", Ask},
		{"cp /tmp/x CFG", Ask},
		{"cp -t CFG /tmp/guard.toml", Ask},
		{"cp --target-directory=CFG /tmp/guard.toml", Ask},
		{"rsync /tmp/x CFG/guard.toml", Ask},
		{"rsync -a /tmp/x/ CFG/", Ask},
		{"tar -xf x.tar -C CFG", Ask},
		{"tar -xf x.tar --directory=CFG", Ask},
		{"unzip x.zip -d CFG", Ask},
		{"patch CFG/guard.toml x.diff", Ask},
		{"awk -i inplace 1 CFG/guard.toml", Ask},
		{"gawk -i inplace 1 CFG/config.toml", Ask},
		// Any file of the plugin's cache, any version, and the cache itself.
		{"cp /tmp/h.json CACHE/brain/9.9.9/hooks/hooks.json", Ask},
		{"echo x > CACHE/brain/9.9.9/hooks/hooks.json", Ask},
		{"rm -rf CACHE", Ask},
		{"rm -rf CACHE/brain", Ask},
		// What does not write them goes through as before.
		{"awk '{print}' CFG/guard.toml", Allow},
		{"cp /tmp/x /tmp/y", Allow},
		{"tar -czf /tmp/b.tgz src", Allow},
		{"rsync -a src/ /tmp/dst/", Allow},
		{"cat CACHE/brain/9.9.9/hooks/hooks.json", Allow},
		// The limits a guard that reads text has: written down, not fixed.
		{"python3 -c 'open(\"CFG/config.toml\",\"a\").write(\"x\")'", Allow},
		{"cd \"$D\" && cp x guard.toml", Allow},
	}
	for i := range rows {
		rows[i].cmd = r.Replace(rows[i].cmd)
	}
	checkPinned(t, p, rows)
	if d := p.WriteFile(filepath.Join(cache, "brain", "9.9.9", "hooks", "hooks.json"), "{}"); d.Verdict != Ask {
		t.Errorf("Write in the plugin's cache = %v", d.Verdict)
	}
}

func TestListsChangedOutsideThePanel(t *testing.T) {
	p := pinnedPolicy(t)
	p.RootAsk = append(p.RootAsk, RootException{Host: "laptop", Until: p.RootAsk[0].Until})
	p.ProductionAWSProfiles, p.UntouchableHosts = nil, nil // what the change took out
	checkPinned(t, p, []pinned{
		{"su_do id", Ask}, // the lists as the user knows them: the exception counts
		{"aws --profile prod s3 ls", Allow},
		{"ssh db01", Allow},
	})
	p.ConfigChanged = true
	checkPinned(t, p, []pinned{
		{"su_do id", Deny},
		{"su_do -u www-data id", Allow},
		{"aws --profile prod s3 ls", Ask},
		{"AWS_PROFILE=prod make deploy", Allow}, // no AWS program in sight: as before
		{"terraform apply", Ask},
		{"ssh db01", Ask},
		{"scp x db01:/tmp", Ask},
		{"git status", Allow},
		{"ls", Allow},
	})
	if d := p.Command(unspell("su_do id")); !strings.Contains(d.Reason, "Guard page") {
		t.Errorf("the reason does not say how to accept the lists: %s", d.Reason)
	}
}
