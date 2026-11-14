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

// checkEvery is how often the latest version is asked for; retryEvery, how long to wait
// after a check that failed (offline), so a session never waits for it twice in a row.
const (
	checkEvery = 24 * time.Hour
	retryEvery = time.Hour
)

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

// Newer reports whether version a is newer than b. The numbers decide first; with the
// same numbers, a release is newer than its pre-releases (1.0.0 after 1.0.0-rc.2), and
// two pre-releases compare part by part as semantic versioning says (rc.2 after rc.1,
// rc after beta after alpha). Build metadata after "+" never counts. Anything that does
// not start with numbers is never newer nor older.
func Newer(a, b string) bool {
	x, preA, okA := parse(a)
	y, preB, okB := parse(b)
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
	return comparePre(preA, preB) > 0
}

// parse splits a version into its numbers and its pre-release ("rc.1" of 1.0.0-rc.1).
func parse(v string) (nums []int, pre string, ok bool) {
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	if i := strings.IndexByte(v, '-'); i >= 0 {
		v, pre = v[:i], v[i+1:]
	}
	for _, part := range strings.Split(v, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, "", false
		}
		nums = append(nums, n)
	}
	return nums, pre, len(nums) > 0
}

// comparePre compares two pre-releases: none is the highest; otherwise part by part,
// numbers as numbers and below words, words in ASCII order, and more parts win a tie.
func comparePre(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == "":
		return 1
	case b == "":
		return -1
	}
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, errA := strconv.Atoi(pa[i])
		nb, errB := strconv.Atoi(pb[i])
		switch {
		case errA == nil && errB == nil:
			if na != nb {
				if na > nb {
					return 1
				}
				return -1
			}
		case errA == nil:
			return -1
		case errB == nil:
			return 1
		default:
			if c := strings.Compare(pa[i], pb[i]); c != 0 {
				return c
			}
		}
	}
	return len(pa) - len(pb)
}

type cache struct {
	Checked time.Time `json:"checked"`
	Latest  string    `json:"latest"`
	Failed  bool      `json:"failed,omitempty"` // the last check got no answer
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
	wait := checkEvery
	if c.Failed {
		wait = retryEvery
	}
	if c.Checked.IsZero() || now.Sub(c.Checked) > wait {
		if latest, err := fetch(); err == nil && latest != "" {
			c = cache{Checked: now, Latest: latest}
		} else {
			// Remember the attempt too: offline, every session would wait for it again.
			c.Checked, c.Failed = now, true
		}
		if data, err := json.Marshal(c); err == nil {
			_ = os.WriteFile(path, data, 0o644)
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
