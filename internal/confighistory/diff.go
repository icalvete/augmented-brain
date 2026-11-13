package confighistory

import "strings"

// Line is one line of a comparison: Op is ' ' (the same in both), '-' (only in the
// older) or '+' (only in the newer).
type Line struct {
	Op   byte
	Text string
}

// FileDiff is what changed in one file between two versions. Gone and New say the file
// was missing in one of them.
type FileDiff struct {
	Name  string
	Gone  bool // in the older, not in the newer
	New   bool // in the newer, not in the older
	Lines []Line
}

// Compare says what changed from older to newer, file by file, in the order of Files;
// a file the same in both is left out. Each comparison keeps up to context unchanged
// lines around each change; the rest are dropped.
func Compare(older, newer map[string]string, context int) []FileDiff {
	var out []FileDiff
	for _, name := range Files {
		a, inA := older[name]
		b, inB := newer[name]
		if inA == inB && a == b {
			continue
		}
		d := FileDiff{Name: name, Gone: inA && !inB, New: inB && !inA}
		d.Lines = trim(lines(split(a), split(b)), context)
		out = append(out, d)
	}
	return out
}

func split(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// maxTable is the largest table lines builds, in cells: about 32 MB. Configuration
// files are a few hundred lines; one far longer (a profile pasted into by mistake) is
// shown as replaced whole, rather than taking gigabytes to compare.
const maxTable = 4_000_000

// lines is a line comparison by longest common subsequence, after the lines the same
// at the start and at the end, which need no table.
func lines(a, b []string) []Line {
	var out []Line
	for len(a) > 0 && len(b) > 0 && a[0] == b[0] {
		out = append(out, Line{' ', a[0]})
		a, b = a[1:], b[1:]
	}
	var tail []Line
	for len(a) > 0 && len(b) > 0 && a[len(a)-1] == b[len(b)-1] {
		tail = append(tail, Line{' ', a[len(a)-1]})
		a, b = a[:len(a)-1], b[:len(b)-1]
	}
	if (len(a)+1)*(len(b)+1) > maxTable {
		for _, l := range a {
			out = append(out, Line{'-', l})
		}
		for _, l := range b {
			out = append(out, Line{'+', l})
		}
	} else {
		out = append(out, table(a, b)...)
	}
	for i := len(tail) - 1; i >= 0; i-- {
		out = append(out, tail[i])
	}
	return out
}

// table compares a and b with the square table of the longest common subsequence.
func table(a, b []string) []Line {
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var out []Line
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			out = append(out, Line{' ', a[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, Line{'-', a[i]})
			i++
		default:
			out = append(out, Line{'+', b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, Line{'-', a[i]})
	}
	for ; j < m; j++ {
		out = append(out, Line{'+', b[j]})
	}
	return out
}

// trim keeps the changed lines and up to context unchanged lines around them; a run of
// unchanged lines that is cut becomes one line with Op '~'.
func trim(ls []Line, context int) []Line {
	keep := make([]bool, len(ls))
	for i, l := range ls {
		if l.Op == ' ' {
			continue
		}
		for k := max(0, i-context); k <= min(len(ls)-1, i+context); k++ {
			keep[k] = true
		}
	}
	var out []Line
	for i, l := range ls {
		if keep[i] {
			out = append(out, l)
		} else if len(out) == 0 || out[len(out)-1].Op != '~' {
			out = append(out, Line{Op: '~'})
		}
	}
	return out
}

// Kind names the line's Op for a page: "add", "del", "skip" or "same".
func (l Line) Kind() string {
	switch l.Op {
	case '+':
		return "add"
	case '-':
		return "del"
	case '~':
		return "skip"
	}
	return "same"
}
