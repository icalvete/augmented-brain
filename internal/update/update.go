// Package update knows the latest published version of augmented brain and whether this
// one is older. The session hook asks it at most once a day, so a new version reaches
// people who installed it once and never think about it again.
package update

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Repo is where releases are published.
const Repo = "icalvete/augmented-brain"

// cacheFile, in the configuration folder, remembers the last answer.
const cacheFile = ".latest-version.json"

// checkEvery is how often the latest version is asked for.
const checkEvery = 24 * time.Hour

// Latest asks GitHub for the latest release, without its API (no rate limit, no token):
// /releases/latest redirects to /releases/tag/v<version>.
func Latest(client *http.Client) (string, error) {
	c := *client
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := c.Head("https://github.com/" + Repo + "/releases/latest")
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	loc := resp.Header.Get("Location")
	i := strings.LastIndex(loc, "/tag/")
	if i < 0 {
		return "", errors.New("no release found")
	}
	return strings.TrimPrefix(loc[i+len("/tag/"):], "v"), nil
}

// Newer reports whether version a is newer than b. Only the numbers count (0.2.1-alpha
// is 0.2.1); anything that does not start with numbers is never newer nor older.
func Newer(a, b string) bool {
	x, okA := numbers(a)
	y, okB := numbers(b)
	if !okA || !okB {
		return false
	}
	for i := 0; i < len(x) || i < len(y); i++ {
		var p, q int
		if i < len(x) {
			p = x[i]
		}
		if i < len(y) {
			q = y[i]
		}
		if p != q {
			return p > q
		}
	}
	return false
}

func numbers(v string) ([]int, bool) {
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var out []int
	for _, part := range strings.Split(v, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, len(out) > 0
}

type cache struct {
	Checked time.Time `json:"checked"`
	Latest  string    `json:"latest"`
}

// Available returns the latest version when it is newer than current, asking for it
// only when the last answer, kept in cfgDir, is older than a day. fetch is Latest in
// real use. It never fails: no network or no answer just means no news.
func Available(cfgDir, current string, now time.Time, fetch func() (string, error)) string {
	if os.Getenv("AB_NO_UPDATE_CHECK") == "1" {
		return ""
	}
	path := filepath.Join(cfgDir, cacheFile)
	var c cache
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &c)
	}
	if c.Latest == "" || now.Sub(c.Checked) > checkEvery {
		if latest, err := fetch(); err == nil && latest != "" {
			c = cache{Checked: now, Latest: latest}
			if data, err := json.Marshal(c); err == nil {
				_ = os.WriteFile(path, data, 0o644)
			}
		}
	}
	if Newer(c.Latest, current) {
		return c.Latest
	}
	return ""
}

// Fetch is Latest with a short timeout, for the session hook: the session must not wait.
func Fetch() (string, error) {
	return Latest(&http.Client{Timeout: 2 * time.Second})
}
