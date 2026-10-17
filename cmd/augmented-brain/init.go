package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/setup"
)

var messages = map[string]map[string]string{
	"en": {
		"space":    "Space name (one part of your life: work, personal…)",
		"work":     "Work folder: opening your assistant here selects this space",
		"memory":   "Memory folder: where this space's notes will live",
		"created":  "Created:",
		"config":   "Configuration:",
		"added":    "Space %q added.",
		"plugin?":  "Install the Claude Code plugin now? (yes, no)",
		"yes":      "yes",
		"plugin":   "Claude Code plugin installed. It loads your space when a session starts, and guards commands and file writes.",
		"noclaude": "Claude Code was not found. Install it, then add the plugin with:",
		"later":    "To install the Claude Code plugin later:",
		"ready":    "augmented brain is ready. Your spaces:",
		"now":      "Now open Claude in your work space:\n  cd %s && claude\nThe first time, it will ask who you are and how you like to work.",
		"newterm":  "Now open a new terminal (so it finds what was just installed) and type:\n  cd %s && claude\nThe first time, Claude will ask who you are and how you like to work.",
		"launch":   "From now on, to work: open a new terminal and type\n  cd %s && claude",
		"desktop":  "From now on, to work: open the Claude app, go to the Code tab and choose the folder %s.\n(Or, in a new terminal: cd %[1]s && claude)",
	},
	"es": {
		"space":    "Nombre del espacio (una parte de tu vida: work, personal…)",
		"work":     "Carpeta de trabajo: abrir tu asistente aquí elige este espacio",
		"memory":   "Carpeta de la memoria: donde vivirán las notas de este espacio",
		"created":  "Creado:",
		"config":   "Configuración:",
		"added":    "Espacio %q añadido.",
		"plugin?":  "¿Instalo ahora el plugin de Claude Code? (sí, no)",
		"yes":      "sí",
		"plugin":   "Plugin de Claude Code instalado. Carga tu espacio al empezar cada sesión y vigila los comandos y las escrituras de ficheros.",
		"noclaude": "No encuentro Claude Code. Instálalo y después añade el plugin con:",
		"later":    "Para instalar el plugin de Claude Code más adelante:",
		"ready":    "augmented brain está listo. Tus espacios:",
		"now":      "Ahora abre Claude en tu espacio de trabajo:\n  cd %s && claude\nLa primera vez te preguntará quién eres y cómo te gusta trabajar.",
		"newterm":  "Ahora abre una terminal nueva (para que encuentre lo recién instalado) y escribe:\n  cd %s && claude\nLa primera vez, Claude te preguntará quién eres y cómo te gusta trabajar.",
		"launch":   "A partir de ahora, para trabajar: abre una terminal nueva y escribe\n  cd %s && claude",
		"desktop":  "A partir de ahora, para trabajar: abre la app de Claude, ve a la pestaña Code y elige la carpeta %s.\n(O, en una terminal nueva: cd %[1]s && claude)",
	},
}

func msg(lang, key string) string {
	if m, ok := messages[lang]; ok {
		return m[key]
	}
	return messages["en"][key]
}

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
	if lang == "es" {
		return []string{"trabajo", "personal"}
	}
	return []string{"work", "personal"}
}

func notesFolder(lang string) string {
	if lang == "es" {
		return "notas"
	}
	return "notes"
}

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
		fmt.Fprintf(errOut, "language %q: use en or es\n", *lang)
		return 2
	}
	var spaces []setup.Options
	if *custom || o.Space != "" || o.WorkFolder != "" || o.Memory != "" {
		o.Space = a.ask(msg(*lang, "space"), firstNonEmpty(o.Space, defaultSpaces(*lang)[0]))
		defaults(o, *lang)
		o.WorkFolder = a.ask(msg(*lang, "work"), o.WorkFolder)
		o.Memory = a.ask(msg(*lang, "memory"), o.Memory)
		spaces = append(spaces, *o)
	} else {
		for _, name := range defaultSpaces(*lang) {
			sp := setup.Options{Space: name}
			defaults(&sp, *lang)
			spaces = append(spaces, sp)
		}
	}

	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	for i, sp := range spaces {
		sp.Language = *lang
		var res *setup.Result
		if i == 0 {
			res, err = setup.Init(dir, sp)
		} else {
			res, err = setup.AddSpace(dir, sp)
		}
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		if *custom {
			report(out, *lang, res)
		}
	}
	if len(spaces) > 1 {
		// Sessions outside every folder (the desktop app's scratch folder) use personal.
		if err := config.SetDefaultSpace(dir, "personal"); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	}
	if !*noPlugin {
		installPlugin(a, *lang, out, errOut)
	}
	fmt.Fprintf(out, "\n%s\n", msg(*lang, "ready"))
	for _, sp := range spaces {
		fmt.Fprintf(out, "  %-10s %s\n", sp.Space, sp.WorkFolder)
	}
	next := "now"
	switch {
	case os.Getenv("AB_DESKTOP") == "1": // install.sh found the Claude desktop app
		next = "desktop"
	case os.Getenv("AB_LAUNCH") == "1": // install.sh offers to open Claude right after
		next = "launch"
	case os.Getenv("AB_NEW_TERMINAL") == "1": // set by install.sh when it changed the PATH
		next = "newterm"
	}
	text := fmt.Sprintf(msg(*lang, next), spaces[0].WorkFolder)
	if runtime.GOOS == "windows" {
		// Windows PowerShell 5.1, the one that comes with Windows, has no &&.
		text = strings.ReplaceAll(text, " && claude", "; claude")
	}
	fmt.Fprintf(out, "\n%s\n", text)
	return 0
}

// installPlugin offers to install the Claude Code plugin. A failure here does not undo
// the rest: it says how to install the plugin by hand.
func installPlugin(a *asker, lang string, out, errOut io.Writer) {
	manual := fmt.Sprintf("  claude plugin marketplace add %s\n  claude plugin install %s --scope user\n",
		claudecode.Marketplace(), claudecode.Plugin)
	if !claudecode.Available() {
		fmt.Fprintf(out, "\n%s\n%s", msg(lang, "noclaude"), manual)
		return
	}
	answer := strings.ToLower(a.ask(msg(lang, "plugin?"), msg(lang, "yes")))
	if !strings.HasPrefix(answer, "y") && !strings.HasPrefix(answer, "s") {
		fmt.Fprintf(out, "\n%s\n%s", msg(lang, "later"), manual)
		return
	}
	if err := claudecode.Install(claudecode.Exec); err != nil {
		fmt.Fprintf(errOut, "\n%v\n%s\n%s", err, msg(lang, "later"), manual)
		return
	}
	fmt.Fprintf(out, "\n%s\n", msg(lang, "plugin"))
}

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
	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	c, err := config.Load(dir)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	lang := c.Language
	a := &asker{in: bufio.NewReader(in), out: out, interactive: !*yes}
	o.Space = a.ask(msg(lang, "space"), o.Space)
	defaults(o, lang)
	o.WorkFolder = a.ask(msg(lang, "work"), o.WorkFolder)
	o.Memory = a.ask(msg(lang, "memory"), o.Memory)
	res, err := setup.AddSpace(dir, *o)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	report(out, lang, res)
	fmt.Fprintf(out, "\n"+msg(lang, "added")+"\n", o.Space)
	return 0
}

func report(out io.Writer, lang string, res *setup.Result) {
	fmt.Fprintf(out, "\n%s %s\n%s\n", msg(lang, "config"), res.ConfigPath, msg(lang, "created"))
	for _, p := range res.Created {
		fmt.Fprintf(out, "  + %s\n", p)
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
