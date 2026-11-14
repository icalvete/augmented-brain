package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/runlimit"

	"github.com/icalvete/augmented-brain/internal/detach"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/state"
)

// Runner runs a command and returns its combined output. Tests replace it.
type Runner func(name string, args ...string) (string, error)

// ExecLimit is how long Exec waits for each gh call (auth status, repo view, repo
// create, auth setup-git): none asks anything, and a gh stuck on the network would
// leave the MCP tool or the panel's page waiting for good.
const ExecLimit = time.Minute

// Exec runs real commands, within ExecLimit.
func Exec(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ExecLimit)
	defer cancel()
	out, err := runlimit.Command(ctx, name, args...).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("no answer from %s in %s: %w", name, ExecLimit, err)
	}
	return string(out), err
}

// ErrNoGitHub means the GitHub command line (gh) is missing or not logged in, so the
// repository cannot be created for the user. Its text says what to do, step by step.
// Login, when set, is the GitHub login already started for the user: all they do is
// accept it in the browser.
type ErrNoGitHub struct {
	Steps  string
	Login  *Login
	Reason string // why the login could not be started, when gh is there but it failed
}

func (e *ErrNoGitHub) Error() string { return e.Steps }

// LoginStarter starts the GitHub login in the background and says what the person
// types in the browser. GHLogin is the real one.
type LoginStarter func() (*Login, error)

// GitHubRepo creates, or finds, the private repository augmented-brain-<space> in the
// user's GitHub account with gh, lets git use gh's login for it, and returns its URL.
// A public repository with that name is refused: the notes are private. When gh is
// not logged in and login is given, it starts the login, so the person only has to
// accept it in the browser, and answers with the code (ErrNoGitHub with Login).
func GitHubRepo(space string, run Runner, login LoginStarter) (string, error) {
	gh := ghCommand()
	if gh == "" {
		return "", &ErrNoGitHub{Steps: GitHubSteps(false)}
	}
	if _, err := run(gh, "auth", "status"); err != nil {
		if login != nil {
			l, lerr := login()
			if lerr == nil {
				return "", &ErrNoGitHub{Steps: LoginSteps(l), Login: l}
			}
			return "", &ErrNoGitHub{Steps: "The GitHub login could not be started (" + lerr.Error() + "). " + GitHubSteps(true), Reason: lerr.Error()}
		}
		return "", &ErrNoGitHub{Steps: GitHubSteps(true)}
	}
	name := "augmented-brain-" + space
	var url string
	if out, err := run(gh, "repo", "view", name, "--json", "url,visibility"); err == nil {
		var repo struct{ URL, Visibility string }
		if err := json.Unmarshal([]byte(out), &repo); err != nil {
			return "", fmt.Errorf("reading %s: %v", name, err)
		}
		if !strings.EqualFold(repo.Visibility, "private") {
			return "", fmt.Errorf("the repository %s already exists and is not private: make it private on GitHub, or use another one", repo.URL)
		}
		url = repo.URL
	} else {
		out, err := run(gh, "repo", "create", name, "--private", "--disable-issues", "--disable-wiki",
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
	if out, err := run(gh, "auth", "setup-git"); err != nil {
		return "", fmt.Errorf("letting git use your GitHub login: %v: %s", err, strings.TrimSpace(out))
	}
	return strings.TrimSuffix(url, "/") + ".git", nil
}

// ghCommand is how gh is run: "gh" when it is on the PATH, or its full path when it is
// next to this program (where the installer puts it) but that folder is not on the PATH
// of whoever started us. Empty when there is none. Tests replace it.
var ghCommand = func() string {
	if _, err := exec.LookPath("gh"); err == nil {
		return "gh"
	}
	if self, err := os.Executable(); err == nil {
		name := "gh"
		if runtime.GOOS == "windows" {
			name = "gh.exe"
		}
		if p := filepath.Join(filepath.Dir(self), name); fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

// Login is a GitHub login gh is waiting on: the person opens URL, types Code and
// accepts. gh finishes it by itself, in the background, as soon as they do.
type Login struct {
	Code    string    `json:"code"`
	URL     string    `json:"url"`
	PID     int       `json:"pid"`
	Started time.Time `json:"started"`
}

const (
	loginFile = "gh-login.json"
	loginLog  = "gh-login.log"
	// loginLasts is how long a code is offered again: GitHub keeps one for 15 minutes,
	// and a code with a minute left is not worth showing.
	loginLasts = 14 * time.Minute
	// loginWait is how long gh has to say its code; it only asks GitHub for one.
	loginWait = 20 * time.Second
	deviceURL = "https://github.com/login/device"
)

var (
	codeRe = regexp.MustCompile(`\b([A-Z0-9]{4}-[A-Z0-9]{4})\b`)
	urlRe  = regexp.MustCompile(`https://\S+/login/device\b`)
)

// GHLogin starts gh's login (GitHub's device flow: no terminal, no password typed
// anywhere but in GitHub's own page) and keeps what it said in the configuration
// folder, so the panel and every session show the same code while it lasts.
func GHLogin(cfgDir string) LoginStarter {
	return func() (*Login, error) { return startLogin(cfgDir, ghCommand(), loginWait) }
}

// CurrentLogin is the login started earlier, while gh still waits on it; nil otherwise.
func CurrentLogin(cfgDir string) *Login {
	data, err := os.ReadFile(filepath.Join(cfgDir, "state", loginFile))
	if err != nil {
		return nil
	}
	var l Login
	if json.Unmarshal(data, &l) != nil || l.Code == "" || time.Since(l.Started) > loginLasts || !state.Alive(l.PID) {
		return nil
	}
	return &l
}

func startLogin(cfgDir, gh string, wait time.Duration) (*Login, error) {
	if gh == "" {
		return nil, errors.New("gh is not installed")
	}
	if l := CurrentLogin(cfgDir); l != nil {
		return l, nil
	}
	dir := filepath.Join(cfgDir, "state")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	logPath := filepath.Join(dir, loginLog)
	out, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	// With no terminal, gh asks nothing: it writes the code and the address, then waits
	// for the person to accept in the browser, and saves the login.
	cmd := exec.Command(gh, "auth", "login", "--web", "--git-protocol", "https", "--hostname", "github.com")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, out, out
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1")
	detach.Cmd(cmd)
	proc, err := detach.Start(cmd)
	if err != nil {
		out.Close()
		return nil, err
	}
	done := make(chan struct{})
	go func() { <-proc.Done(); out.Close(); close(done) }()
	deadline := time.Now().Add(wait)
	for {
		said, _ := os.ReadFile(logPath)
		if m := codeRe.FindSubmatch(said); m != nil {
			l := &Login{Code: string(m[1]), URL: deviceURL, PID: proc.Pid, Started: time.Now()}
			if u := urlRe.Find(said); u != nil {
				l.URL = string(u)
			}
			data, _ := json.Marshal(l)
			if err := paths.WriteAtomic(filepath.Join(dir, loginFile), data, 0o600); err != nil {
				return nil, err
			}
			return l, nil
		}
		select {
		case <-done:
			return nil, fmt.Errorf("gh stopped without a code: %s", lastLine(said))
		default:
		}
		if time.Now().After(deadline) {
			_ = proc.Kill()
			return nil, errors.New("GitHub did not answer in time; check the internet connection")
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func lastLine(b []byte) string {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// LoginSteps is what the assistant tells the person once the login is started.
func LoginSteps(l *Login) string {
	return fmt.Sprintf("gh is installed but not logged in to GitHub, so the login has been started and waits for the user. Tell them, in their language: "+
		"1) open %s in the browser and sign in to GitHub (with no account, they create one for free at https://github.com/signup first); "+
		"2) type the code %s and accept. The code works for about 15 minutes. "+
		"When they say it is done, call backup_setup again: it creates the private repository and makes the first copy. "+
		"Do not run gh auth login yourself: it is already waiting.", l.URL, l.Code)
}

// GitHubSteps says how to get gh ready, for a person with no git knowledge, when the
// login cannot be started for them. The login needs the user's browser.
func GitHubSteps(installed bool) string {
	install := "Install the GitHub command line, gh: "
	switch runtime.GOOS {
	case "darwin":
		install += "`brew install gh` if Homebrew is there; otherwise the user runs the augmented brain installer again, which offers gh and needs no password"
	case "windows":
		install += "the user runs the augmented brain installer again, which offers gh and needs no administrator (or `winget install --id GitHub.cli -e --source winget`, then a new terminal)"
	default:
		install += "the user runs the augmented brain installer again (`curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh`), which offers gh and needs no password. " +
			"`sudo apt install gh` works too, but only the user can run it: it asks for their password"
	}
	login := "Log in once: `gh auth login --web --git-protocol https`. It shows a short code; the user opens https://github.com/login/device, types it and accepts."
	if installed {
		return "gh is installed but not logged in. " + login + " Then call backup_setup again."
	}
	return "The backup goes to a private repository in the user's GitHub account, created by the GitHub command line. 1) " +
		install + ". 2) Call backup_setup again: it starts the GitHub login and gives a code for the user to type in the browser. " +
		"If the user has no GitHub account, they create one for free at https://github.com/signup. Or the backup goes somewhere else (GitLab, their own server): " +
		"the user runs `augmented-brain backup setup <url>` in a terminal, in a folder of the space, or uses the panel's Backup page."
}
