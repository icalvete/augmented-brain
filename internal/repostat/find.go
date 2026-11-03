package repostat

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Limits of Find: how deep under a folder it looks, how many repositories it returns at
// most, and how many folders it opens at most, so a huge folder cannot hold a page.
const (
	FindDepth = 3
	FindLimit = 120
	findDirs  = 5000
)

// heavy are folders that hold dependencies or builds, never a repository of one's own.
var heavy = map[string]bool{
	"node_modules": true, "vendor": true, "venv": true, "__pycache__": true,
	"target": true, "bower_components": true,
}

// Find returns the git repositories inside root (not root itself), shallowest first, at
// most depth levels down and at most limit of them; more reports that it stopped
// before looking everywhere. It never follows a symbolic link, never enters hidden
// folders (.git among them) or the heavy ones (node_modules, vendor…), stops
// going down at the first repository on each path, and leaves out the folders skip
// says (the notes themselves). It only lists folders: git does not run here.
func Find(root string, depth, limit int, skip func(string) bool) (found []string, more bool) {
	type dir struct {
		path  string
		depth int
	}
	queue := []dir{{root, 0}}
	opened := 0
	for len(queue) > 0 {
		d := queue[0]
		queue = queue[1:]
		if d.depth > 0 && isRepo(d.path) {
			if len(found) == limit {
				return found, true
			}
			found = append(found, d.path)
			continue
		}
		if d.depth == depth {
			continue
		}
		if opened == findDirs {
			return found, true
		}
		opened++
		entries, err := os.ReadDir(d.path)
		if err != nil {
			continue
		}
		for _, e := range entries {
			// A DirEntry says what the entry itself is: a link to a folder is not a folder.
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || heavy[e.Name()] {
				continue
			}
			p := filepath.Join(d.path, e.Name())
			if skip != nil && skip(p) {
				continue
			}
			queue = append(queue, dir{p, d.depth + 1})
		}
	}
	return found, false
}

// isRepo says whether a folder is the top of a git repository: it has a .git folder, or
// a .git file (a worktree, a submodule). A .git that is a link does not count.
func isRepo(p string) bool {
	info, err := os.Lstat(filepath.Join(p, ".git"))
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

// CleanRemote turns a remote's address into host/owner/name, with no user, password,
// port, query or ".git": https://user:token@example.com/me/site.git and
// git@example.com:me/site.git are both example.com/me/site. A folder on this
// computer is its last part.
func CleanRemote(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "" // what cannot be read may hold a password: show nothing
		}
		p := strings.Trim(u.Path, "/")
		if u.Hostname() == "" {
			return trimGit(filepath.Base(filepath.FromSlash(p)))
		}
		return strings.TrimSuffix(u.Hostname()+"/"+trimGit(p), "/")
	}
	// The scp form, [user@]host:path, but not a Windows drive (C:\…).
	if c := strings.Index(raw, ":"); c > 1 && !strings.ContainsAny(raw[:c], `/\`) {
		host := raw[:c]
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
		return strings.TrimSuffix(host+"/"+trimGit(strings.Trim(raw[c+1:], "/")), "/")
	}
	return trimGit(filepath.Base(raw))
}

func trimGit(p string) string { return strings.TrimSuffix(p, ".git") }
