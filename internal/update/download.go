package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

// Where release files are published. Variables, so tests can point them at a server
// of their own.
var (
	// PublicBase is the public address of a release's files: <PublicBase>/v<version>/<file>.
	PublicBase = "https://github.com/" + Repo + "/releases/download"
	// APIBase is GitHub's API for the repository. On 8-nov-2026 the public address
	// answered 504 for over an hour while the API served the very same files.
	APIBase = "https://api.github.com/repos/" + Repo
)

// Tries is how many times a download is tried; RetryDelay, the wait between two tries.
var (
	Tries      = 3
	RetryDelay = 5 * time.Second
)

// maxFile caps what a download keeps in memory: release files are a few MB.
const maxFile = 64 << 20

// Download fetches one file of a release into dst. With base (AB_RELEASE_URL, for
// tests and the rehearsal) it is <base>/<name>, http(s) or file://; otherwise the file
// of release version from the public address, and when that fails from GitHub's API.
// It tries Tries times. dst is written only when the whole file arrived; the error
// says what each address answered.
func Download(client *http.Client, base, version, name, dst string) error {
	var last error
	for try := 1; try <= Tries; try++ {
		if try > 1 {
			time.Sleep(RetryDelay)
		}
		var data []byte
		var err error
		if base != "" {
			data, err = get(client, strings.TrimRight(base, "/")+"/"+name, "")
		} else {
			data, err = get(client, PublicBase+"/v"+strings.TrimPrefix(version, "v")+"/"+name, "")
			if err != nil {
				var errAPI error
				if data, errAPI = viaAPI(client, version, name); errAPI == nil {
					err = nil
				} else {
					err = fmt.Errorf("%v; API: %v", err, errAPI)
				}
			}
		}
		if err == nil {
			return os.WriteFile(dst, data, 0o600)
		}
		last = err
	}
	return last
}

// viaAPI fetches a release file through GitHub's API: the release by its tag, the
// asset by its name, and the asset itself as plain bytes. No token: an update makes two
// calls, well inside the anonymous limit.
func viaAPI(client *http.Client, version, name string) ([]byte, error) {
	body, err := get(client, APIBase+"/releases/tags/v"+strings.TrimPrefix(version, "v"), "application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	var rel struct {
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, err
	}
	for _, a := range rel.Assets {
		if a.Name == name {
			return get(client, a.URL, "application/octet-stream")
		}
	}
	return nil, fmt.Errorf("the release v%s has no %s", strings.TrimPrefix(version, "v"), name)
}

// get reads an address whole: http(s), or file:// (the tests' releases).
func get(client *http.Client, addr, accept string) ([]byte, error) {
	if strings.HasPrefix(addr, "file://") {
		u, err := url.Parse(addr)
		if err != nil {
			return nil, err
		}
		p := u.Path
		if runtime.GOOS == "windows" && len(p) > 2 && p[0] == '/' && p[2] == ':' {
			p = p[1:] // file:///C:/… is C:/…
		}
		return os.ReadFile(p)
	}
	req, err := http.NewRequest(http.MethodGet, addr, nil)
	if err != nil {
		return nil, err
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %s", addr, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFile+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxFile {
		return nil, errors.New(addr + ": too big")
	}
	return data, nil
}
