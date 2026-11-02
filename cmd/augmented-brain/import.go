package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/importer"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// cmdImport brings a memory of Claude Code into the spaces. Without --apply it only
// says what it would do.
func cmdImport(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(errOut)
	mapFile := fs.String("map", "", "file that says which space each area goes to: lines \"area-x = space\", and \"* = space\" for the rest")
	base := fs.String("repos-base", "", "folder the repos: paths of the source are relative to")
	apply := fs.Bool("apply", false, "write the notes (without it, nothing is written)")
	var src string
	// The folder may come before or after the flags.
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		src, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if src == "" && fs.NArg() > 0 {
		src = fs.Arg(0)
	}
	c, ok := loadConfig(errOut)
	if !ok {
		return 1
	}
	dir, lang := c.Dir, c.Language
	var err error
	if src == "" {
		fmt.Fprintln(errOut, i18n.T(lang, "import.usage"))
		return 2
	}
	src, err = paths.Resolve(src)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	o := importer.Options{Language: lang}
	if *mapFile != "" {
		if o.Map, err = readMap(*mapFile); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	}
	if *base != "" {
		if o.ReposBase, err = paths.Resolve(*base); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	}

	unlock, err := filelock.Lock(filepath.Join(dir, backup.NotesLock))
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer unlock()
	p, err := importer.MakePlan(src, c, o)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	importReport(out, lang, p)
	if !*apply {
		again := []string{"import", src}
		if *mapFile != "" {
			again = append(again, "--map", *mapFile)
		}
		if *base != "" {
			again = append(again, "--repos-base", o.ReposBase)
		}
		fmt.Fprintln(out, i18n.T(lang, "import.dry", shellWords(append(again, "--apply"))))
		return 0
	}
	done, err := importer.Apply(p, c, lang)
	for _, sp := range append(c.SpaceNames(), config.SharedFolder) {
		if n := done.Notes[sp] + done.Copies[sp]; n > 0 {
			fmt.Fprintln(out, i18n.T(lang, "import.done", done.Notes[sp], done.Copies[sp], spaceName(lang, sp)))
		}
	}
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return 0
}

// shellWords joins words into a command line that can be pasted, quoting those with a
// space or a quote in them.
func shellWords(words []string) string {
	q := make([]string, len(words))
	for i, w := range words {
		q[i] = w
		if strings.ContainsAny(w, " \t'\"$`\\") {
			q[i] = "'" + strings.ReplaceAll(w, "'", `'\''`) + "'"
		}
	}
	return strings.Join(q, " ")
}

// readMap reads "area = space" or "note = space" lines; # starts a comment.
func readMap(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m := map[string]string{}
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		area, sp, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(area) == "" || strings.TrimSpace(sp) == "" {
			return nil, fmt.Errorf("%s, line %d: write \"area-id = space\" (or \"* = space\")", path, n)
		}
		m[strings.TrimSpace(area)] = strings.TrimSpace(sp)
	}
	return m, sc.Err()
}

// importReport prints what the plan would do: per space, how many of each; then every issue,
// grouped by kind.
func importReport(out io.Writer, lang string, p *importer.Plan) {
	counts := p.Count()
	var spaces []string
	for sp := range counts {
		spaces = append(spaces, sp)
	}
	sort.Strings(spaces)
	for _, sp := range spaces {
		var parts []string
		var kinds []string
		for k := range counts[sp] {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
		for _, k := range kinds {
			parts = append(parts, fmt.Sprintf("%d %s", counts[sp][k], i18n.T(lang, "import.kind."+k)))
		}
		fmt.Fprintln(out, i18n.T(lang, "import.space", spaceName(lang, sp), strings.Join(parts, ", ")))
	}
	if n := p.Left(); n > 0 {
		fmt.Fprintln(out, i18n.T(lang, "import.left", n))
	}
	byKind := map[string][]importer.Issue{}
	var kinds []string
	for _, is := range p.Issues {
		if byKind[is.Kind] == nil {
			kinds = append(kinds, is.Kind)
		}
		byKind[is.Kind] = append(byKind[is.Kind], is)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		fmt.Fprintln(out, "\n"+i18n.T(lang, "import.issue."+k, len(byKind[k])))
		for _, is := range byKind[k] {
			if is.Detail != "" {
				fmt.Fprintf(out, "  %s · %s\n", is.Source, is.Detail)
			} else {
				fmt.Fprintf(out, "  %s\n", is.Source)
			}
		}
	}
	fmt.Fprintln(out)
}

// spaceName is how the report names a space of the plan, the shared notes included.
func spaceName(lang, sp string) string {
	if sp == config.SharedFolder {
		return i18n.T(lang, "import.shared")
	}
	return i18n.T(lang, "import.named", sp)
}
