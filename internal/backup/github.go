package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Run runs a command and returns its combined output. Tests replace it.
type Runner func(name string, args ...string) (string, error)

// Exec runs real commands.
func Exec(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// ErrNoGitHub means the GitHub command line (gh) is missing or not logged in, so the
// repository cannot be created for the user. Its text says what to do, step by step.
type ErrNoGitHub struct{ Steps string }

func (e *ErrNoGitHub) Error() string { return e.Steps }

// GitHubRepo creates, or finds, the private repository augmented-brain-<space> in the
// user's GitHub account with gh, lets git use gh's login for it, and returns its URL.
// A public repository with that name is refused: the notes are private.
func GitHubRepo(space string, run Runner) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", &ErrNoGitHub{Steps: GitHubSteps(false)}
	}
	if _, err := run("gh", "auth", "status"); err != nil {
		return "", &ErrNoGitHub{Steps: GitHubSteps(true)}
	}
	name := "augmented-brain-" + space
	var url string
	if out, err := run("gh", "repo", "view", name, "--json", "url,visibility"); err == nil {
		var repo struct{ URL, Visibility string }
		if err := json.Unmarshal([]byte(out), &repo); err != nil {
			return "", fmt.Errorf("reading %s: %v", name, err)
		}
		if !strings.EqualFold(repo.Visibility, "private") {
			return "", fmt.Errorf("the repository %s already exists and is not private: make it private on GitHub, or use another one", repo.URL)
		}
		url = repo.URL
	} else {
		out, err := run("gh", "repo", "create", name, "--private", "--disable-issues", "--disable-wiki",
			"--description", "augmented brain backup of the "+space+" space (private)")
		if err != nil {
			return "", fmt.Errorf("creating %s on GitHub: %v: %s", name, err, strings.TrimSpace(out))
		}
		for _, line := range strings.Fields(out) {
			if strings.HasPrefix(line, "https://") {
				url = line
			}
		}
		if url == "" {
			return "", errors.New("GitHub did not say where the new repository is")
		}
	}
	if out, err := run("gh", "auth", "setup-git"); err != nil {
		return "", fmt.Errorf("letting git use your GitHub login: %v: %s", err, strings.TrimSpace(out))
	}
	return strings.TrimSuffix(url, "/") + ".git", nil
}

// GitHubSteps says how to get gh ready, for a person with no git knowledge. The
// assistant runs what it can; the login needs the user's browser.
func GitHubSteps(installed bool) string {
	install := "Install the GitHub command line, gh: "
	switch runtime.GOOS {
	case "darwin":
		install += "`brew install gh` if Homebrew is there, otherwise the installer from https://cli.github.com"
	case "windows":
		install += "`winget install --id GitHub.cli -e --source winget` (then open a new terminal, so gh is on the PATH)"
	default:
		install += "`sudo apt install gh` on Ubuntu or Debian (other systems: https://cli.github.com)"
	}
	login := "Log in once: `gh auth login --web --git-protocol https`. It shows a short code and opens GitHub in the browser; the user pastes the code there and accepts."
	if installed {
		return "gh is installed but not logged in. " + login + " Then call backup_setup again."
	}
	return "The backup goes to a private repository in the user's GitHub account, created by the GitHub command line. 1) " +
		install + ". 2) " + login + " 3) Call backup_setup again. If the user has no GitHub account, they create one at https://github.com/signup, " +
		"or give backup_setup the URL of any private git repository they can push to."
}
