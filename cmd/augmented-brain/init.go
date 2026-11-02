package main

import (
	"bufio"
	"cmp"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// msg is one of init's messages, unformatted.
func msg(lang, key string) string { return i18n.T(lang, "init."+key) }

// asker reads answers from the user, or takes the defaults when interactive is false.
type asker struct {
	in          *bufio.Reader
	out         io.Writer
	interactive bool
}

func (a *asker) ask(question, def string) string {
	if !a.interactive {
		return def
	}
	fmt.Fprintf(a.out, "%s [%s]: ", question, def)
	line, _ := a.in.ReadString('\n')
	if line = strings.TrimSpace(line); line != "" {
		return line
	}
	return def
}

// spaceFlags are the flags init and space add share: a space and its folders, --yes
// and --language.
func spaceFlags(name string, errOut io.Writer) (*flag.FlagSet, *setup.Options, *bool, *string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(errOut)
	o := &setup.Options{}
	fs.StringVar(&o.Space, "space", "", "space name")
	fs.StringVar(&o.WorkFolder, "work", "", "work folder")
	fs.StringVar(&o.Memory, "memory", "", "memory folder")
	yes := fs.Bool("yes", false, "do not ask: take the flags and the defaults")
	lang := fs.String("language", "", "language: en | es")
	return fs, o, yes, lang
}

// root holds every space unless told otherwise: one folder per space, to work in, with
// its notes inside.
const root = "~/augmented-brain"

// defaultSpaces are the spaces `init` creates when it is not asked anything.
func defaultSpaces(lang string) []string {
	return []string{msg(lang, "space.work"), "personal"}
}

// notesFolder is the name of the notes folder inside a space's work folder.
func notesFolder(lang string) string { return msg(lang, "notes") }

// defaults fills in ~/augmented-brain/<space> as the work folder and its notes folder
// as the memory.
func defaults(o *setup.Options, lang string) {
	if o.WorkFolder == "" {
		o.WorkFolder = root + "/" + o.Space
	}
	if o.Memory == "" {
		o.Memory = root + "/" + o.Space + "/" + notesFolder(lang)
	}
}

// cmdInit sets everything up. With no flags it asks nothing: the language comes from
// the system, and it creates two spaces in ~/augmented-brain. --custom asks for the
// language and one space; --space, --work and --memory make that one space without asking.
func cmdInit(args []string, in io.Reader, out, errOut io.Writer) int {
	fs, o, yes, lang := spaceFlags("init", errOut)
	noPlugin := fs.Bool("no-plugin", false, "do not install the Claude Code plugin")
	custom := fs.Bool("custom", false, "ask for the language, the space and its folders")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	a := &asker{in: bufio.NewReader(in), out: out, interactive: *custom && !*yes}
	if *lang == "" {
		*lang = a.ask("Language / Idioma (en, es)", i18n.FromEnv())
	}
	if *lang != "en" && *lang != "es" {
		fmt.Fprintln(errOut, i18n.T(i18n.FromEnv(), "init.language", *lang))
		return 2
	}
	oneSpace := *custom || o.Space != "" || o.WorkFolder != "" || o.Memory != ""
	spaces := initSpaces(a, o, *lang, oneSpace)
	if err := createSpaces(spaces, *lang, *custom, out); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	plugin := true
	if !*noPlugin {
		plugin = installPlugin(a, *lang, out, errOut)
	}
	fmt.Fprintf(out, "\n%s\n", msg(*lang, "ready"))
	for _, sp := range spaces {
		fmt.Fprintf(out, "  %-10s %s\n", sp.Space, sp.WorkFolder)
	}
	if !plugin {
		warn := msg(*lang, "noplugin")
		if _, err := exec.LookPath("git"); err != nil {
			warn += " " + msg(*lang, "nogit")
		}
		fmt.Fprintf(out, "\n%s\n", warn)
	}
	fmt.Fprintf(out, "\n%s\n", nextStep(*lang, spaces[0].WorkFolder))
	return 0
}

// initSpaces are the spaces init creates, in the language lang: the one in o, asking
// for what it lacks, when oneSpace; otherwise the default spaces in ~/augmented-brain.
func initSpaces(a *asker, o *setup.Options, lang string, oneSpace bool) []setup.Options {
	if oneSpace {
		askSpace(a, o, lang, cmp.Or(o.Space, defaultSpaces(lang)[0]))
		o.Language = lang
		return []setup.Options{*o}
	}
	var spaces []setup.Options
	for _, name := range defaultSpaces(lang) {
		sp := setup.Options{Space: name, Language: lang}
		defaults(&sp, lang)
		spaces = append(spaces, sp)
	}
	return spaces
}

// askSpace asks for a space's name (def by default) and then its folders, which
// default to those under ~/augmented-brain.
func askSpace(a *asker, o *setup.Options, lang, def string) {
	o.Space = a.ask(msg(lang, "space"), def)
	defaults(o, lang)
	o.WorkFolder = a.ask(msg(lang, "work"), o.WorkFolder)
	o.Memory = a.ask(msg(lang, "memory"), o.Memory)
}

// createSpaces writes the configuration with the first space and adds the others,
// saying what it created when verbose. With more than one, personal becomes the
// default space.
func createSpaces(spaces []setup.Options, lang string, verbose bool, out io.Writer) error {
	dir, err := config.Dir()
	if err != nil {
		return err
	}
	for i, sp := range spaces {
		create := setup.AddSpace
		if i == 0 {
			create = setup.Init
		}
		res, err := create(dir, sp)
		if err != nil {
			return err
		}
		if verbose {
			report(out, lang, res)
		}
	}
	if len(spaces) > 1 {
		// Sessions outside every folder (the desktop app's scratch folder) use personal.
		return config.SetDefaultSpace(dir, "personal")
	}
	return nil
}

// nextStep is init's last message: how to start working in the folder work. The
// installer says, through the environment, what fits this machine.
func nextStep(lang, work string) string {
	next := "now"
	switch {
	case os.Getenv("AB_DESKTOP") == "1": // install.sh found the Claude desktop app
		next = "desktop"
	case os.Getenv("AB_LAUNCH") == "1": // install.sh offers to open Claude right after
		next = "launch"
	case os.Getenv("AB_NEW_TERMINAL") == "1": // set by install.sh when it changed the PATH
		next = "newterm"
	}
	text := fmt.Sprintf(msg(lang, next), work)
	if runtime.GOOS == "windows" {
		// Windows PowerShell 5.1, the one that comes with Windows, has no &&.
		text = strings.ReplaceAll(text, " && claude", "; claude")
	}
	return text
}

// installPlugin offers to install the Claude Code plugin and reports whether it is
// installed now: Claude Code is asked, since an install can end without an error and
// still leave no plugin (with no git, for one). A failure here does not undo the rest:
// it says how to install the plugin by hand.
func installPlugin(a *asker, lang string, out, errOut io.Writer) bool {
	manual := fmt.Sprintf("  claude plugin marketplace add %s\n  claude plugin install %s --scope user\n",
		claudecode.Marketplace(), claudecode.Plugin)
	if !claudecode.Available() {
		fmt.Fprintf(out, "\n%s\n%s", msg(lang, "noclaude"), manual)
		return false
	}
	answer := strings.ToLower(a.ask(msg(lang, "plugin?"), msg(lang, "yes")))
	if !strings.HasPrefix(answer, "y") && !strings.HasPrefix(answer, "s") {
		fmt.Fprintf(out, "\n%s\n%s", msg(lang, "later"), manual)
		return false
	}
	err := claudecode.Install(claudecode.Exec)
	if err == nil {
		if ok, lerr := claudecode.Installed(claudecode.Exec); lerr != nil || !ok {
			err = fmt.Errorf("%s", msg(lang, "notthere"))
		}
	}
	if err != nil {
		fmt.Fprintf(errOut, "\n%v\n%s\n%s", err, msg(lang, "later"), manual)
		return false
	}
	fmt.Fprintf(out, "\n%s\n", msg(lang, "plugin"))
	return true
}

// cmdSpaceAdd adds one space to the configuration, asking for its name and folders
// unless --yes.
func cmdSpaceAdd(args []string, in io.Reader, out, errOut io.Writer) int {
	fs, o, yes, _ := spaceFlags("space add", errOut)
	// The name may come first (space add personal --work ~/personal): Go's flag package
	// stops at the first argument that is not a flag, so take it out before parsing.
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		o.Space, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 && o.Space == "" {
		o.Space = fs.Arg(0)
	}
	c, ok := loadConfig(errOut)
	if !ok {
		return 1
	}
	lang := c.Language
	a := &asker{in: bufio.NewReader(in), out: out, interactive: !*yes}
	askSpace(a, o, lang, o.Space)
	res, err := setup.AddSpace(c.Dir, *o)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	report(out, lang, res)
	fmt.Fprintf(out, "\n"+msg(lang, "added")+"\n", o.Space)
	if res.DefaultSpace != "" {
		fmt.Fprintf(out, msg(lang, "default")+"\n", res.DefaultSpace)
	}
	return 0
}

// report says where the configuration is and what was created.
func report(out io.Writer, lang string, res *setup.Result) {
	fmt.Fprintf(out, "\n%s %s\n%s\n", msg(lang, "config"), res.ConfigPath, msg(lang, "created"))
	for _, p := range res.Created {
		fmt.Fprintf(out, "  + %s\n", p)
	}
}
