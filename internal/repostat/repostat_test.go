package repostat

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	var st Status
	parse(&st, "# branch.oid abc\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +2 -1\n1 .M N... 100644 100644 100644 a b f.go\n? new.txt\n")
	if st.Branch != "main" || !st.Upstream || st.Ahead != 2 || st.Changed != 2 || !st.Pending() {
		t.Fatalf("%+v", st)
	}
	st = Status{Remote: true}
	parse(&st, "# branch.oid abc\n# branch.head topic\n")
	if st.Upstream || st.Ahead != 1 || !st.Pending() {
		t.Fatalf("a branch never pushed should be pending: %+v", st)
	}
	st = Status{}
	parse(&st, "# branch.oid abc\n# branch.head topic\n")
	if st.Pending() {
		t.Fatalf("a repository with no remote has nowhere to push: %+v", st)
	}
	st = Status{}
	parse(&st, "# branch.oid abc\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +0 -0\n")
	if st.Pending() {
		t.Fatalf("a clean repository is not pending: %+v", st)
	}
}

func TestRead(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	sts := ReadAll([]string{dir, filepath.Join(dir, "missing"), sub, t.TempDir()})
	if sts[0].Err != "" || sts[0].NotRepo || sts[0].Changed != 1 || sts[0].Origin != "" {
		t.Fatalf("%+v", sts[0])
	}
	// The remote is shown without the password its address holds.
	if out, err := exec.Command("git", "-C", dir, "remote", "add", "origin", "https://ana:s3cret@example.com/ana/site.git").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if st := Read(dir); st.Origin != "example.com/ana/site" || !st.Remote {
		t.Fatalf("origin: %+v", st)
	}
	for _, st := range sts[1:] {
		if !st.NotRepo || st.Pending() {
			t.Fatalf("not a repository of its own: %+v", st)
		}
	}
}
