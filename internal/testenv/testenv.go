// Package testenv is for tests only: it makes the machine running them look like a
// plain terminal.
package testenv

import (
	"os"
	"os/exec"
	"strings"
)

// CleanGit takes out of the environment the variables that tie git to one repository
// (GIT_DIR, GIT_INDEX_FILE, GIT_WORK_TREE…, as `git rev-parse --local-env-vars` lists
// them). Git sets them for its hooks, so tests run from a pre-push hook would otherwise
// make their throwaway repositories inside the real one: on 6-nov they committed to the
// branch being pushed. Call it first in TestMain of every package whose tests run git.
func CleanGit() {
	vars := []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY",
		"GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_COMMON_DIR", "GIT_PREFIX", "GIT_CONFIG",
		"GIT_CONFIG_PARAMETERS", "GIT_CONFIG_COUNT", "GIT_IMPLICIT_WORK_TREE", "GIT_GRAFT_FILE",
		"GIT_NO_REPLACE_OBJECTS", "GIT_REPLACE_REF_BASE", "GIT_SHALLOW_FILE"}
	if out, err := exec.Command("git", "rev-parse", "--local-env-vars").Output(); err == nil {
		vars = append(vars, strings.Fields(string(out))...)
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}
}
