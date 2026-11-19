package hook

import (
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/guard"
)

// Codex changes files with apply_patch: its own tool (tool_name "apply_patch", the
// patch in tool_input.command), or the same program named in a shell command
// (apply_patch <<'EOF' … EOF), which Codex runs itself and sends as Bash. A patch
// names each file it touches on a line of its own:
//
//	*** Begin Patch
//	*** Add File: notes/a.txt
//	+the new file's lines
//	*** Update File: b.txt
//	*** Move to: c.txt
//	@@
//	-old line
//	+new line
//	*** Delete File: d.txt
//	*** End Patch

const (
	patchBegin  = "*** Begin Patch"
	patchEnd    = "*** End Patch"
	patchAdd    = "*** Add File: "
	patchUpdate = "*** Update File: "
	patchDelete = "*** Delete File: "
	patchMove   = "*** Move to: "
)

// patchFile is one file a patch writes, and the lines it puts there.
type patchFile struct {
	path, added string
}

// patchFiles reads the files a patch writes, deletes or moves, with the lines it adds
// to each. ok is false when the text is no patch, or names no file: the guard cannot
// tell what it would change.
func patchFiles(patch string) (files []patchFile, ok bool) {
	begin := strings.Index(patch, patchBegin)
	if begin < 0 {
		return nil, false
	}
	var cur *patchFile
	for _, line := range strings.Split(patch[begin+len(patchBegin):], "\n") {
		line = strings.TrimSuffix(line, "\r")
		var path string
		switch {
		case strings.HasPrefix(line, patchEnd):
			return files, len(files) > 0
		case strings.HasPrefix(line, patchAdd):
			path = strings.TrimPrefix(line, patchAdd)
		case strings.HasPrefix(line, patchUpdate):
			path = strings.TrimPrefix(line, patchUpdate)
		case strings.HasPrefix(line, patchDelete):
			path = strings.TrimPrefix(line, patchDelete)
		case strings.HasPrefix(line, patchMove):
			// The lines that follow land in the new path; the old one goes away.
			path = strings.TrimPrefix(line, patchMove)
		case strings.HasPrefix(line, "+") && cur != nil:
			cur.added += line[1:] + "\n"
			continue
		default:
			continue
		}
		path = strings.TrimSpace(path)
		if path == "" {
			return nil, false
		}
		files = append(files, patchFile{path: path})
		cur = &files[len(files)-1]
	}
	// No end line: Codex refuses such a patch, but what it names is still checked.
	return files, len(files) > 0
}

// patchDecision is the guard's answer to a patch applied in dir: the strictest of its
// answers for each file, as a write of the lines the patch adds there.
func patchDecision(p guard.Policy, dir, patch string) (guard.Decision, bool) {
	files, ok := patchFiles(patch)
	if !ok {
		return guard.Decision{}, false
	}
	var worst guard.Decision
	for _, f := range files {
		path := f.path
		if !filepath.IsAbs(path) && dir != "" {
			path = filepath.Join(dir, path)
		}
		if d := p.WriteFile(path, f.added); stricter(d, worst) {
			worst = d
		}
	}
	return worst, true
}

// stricter says whether a is a stricter answer than b: deny over ask over letting through.
func stricter(a, b guard.Decision) bool {
	rank := map[guard.Verdict]int{guard.Ask: 1, guard.Deny: 2}
	return rank[a.Verdict] > rank[b.Verdict]
}

// shellPatchDir is the folder a patch in a shell command is applied in: dir, or the
// one a leading `cd <folder> &&` names. Codex may also put an apply_patch program on
// the shell's path, so any command that holds a patch is checked as one, wherever in
// the command it is.
func shellPatchDir(dir, cmd string) string {
	rest, found := strings.CutPrefix(strings.TrimSpace(cmd), "cd ")
	if !found {
		return dir
	}
	target, _, and := strings.Cut(rest, "&&")
	if !and {
		return dir
	}
	target = strings.Trim(strings.TrimSpace(target), `'"`)
	if filepath.IsAbs(target) || dir == "" {
		return target
	}
	return filepath.Join(dir, target)
}
