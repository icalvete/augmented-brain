package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

var messages = map[string]map[string]string{
	"en": {
		"space":    "Space name (one part of your life: work, personal…)",
		"work":     "Work folder: opening your assistant here selects this space",
		"memory":   "Memory folder: where this space's notes will live",
		"created":  "Created:",
		"config":   "Configuration:",
		"done":     "Done. Space %q is ready.",
		"next":     "Next: run `augmented-brain doctor`, and fill in your profile:\n  %s",
		"added":    "Space %q added.",
		"plugin?":  "Install the Claude Code plugin now? (yes, no)",
		"yes":      "yes",
		"plugin":   "Claude Code plugin installed. It loads your space when a session starts, and guards commands and file writes.",
		"noclaude": "Claude Code was not found. Install it, then add the plugin with:",
		"later":    "To install the Claude Code plugin later:",
	},
	"es": {
		"space":    "Nombre del espacio (una parte de tu vida: work, personal…)",
		"work":     "Carpeta de trabajo: abrir tu asistente aquí elige este espacio",
		"memory":   "Carpeta de la memoria: donde vivirán las notas de este espacio",
		"created":  "Creado:",
		"config":   "Configuración:",
		"done":     "Hecho. El espacio %q está listo.",
		"next":     "Siguiente: lanza `augmented-brain doctor` y rellena tu perfil:\n  %s",
		"added":    "Espacio %q añadido.",
		"plugin?":  "¿Instalo ahora el plugin de Claude Code? (sí, no)",
		"yes":      "sí",
		"plugin":   "Plugin de Claude Code instalado. Carga tu espacio al empezar cada sesión y vigila los comandos y las escrituras de ficheros.",
		"noclaude": "No encuentro Claude Code. Instálalo y después añade el plugin con:",
		"later":    "Para instalar el plugin de Claude Code más adelante:",
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

func defaults(o *setup.Options, space string) {
	if o.Space == "" {
		o.Space = space
	}
	if o.WorkFolder == "" {
		if wd, err := os.Getwd(); err == nil {
			o.WorkFolder = wd
		}
	}
	if o.Memory == "" {
		o.Memory = "~/augmented-brain/" + o.Space
	}
}

func cmdInit(args []string, in io.Reader, out, errOut io.Writer) int {
	fs, o, yes, lang := spaceFlags("init", errOut)
	noPlugin := fs.Bool("no-plugin", false, "do not install the Claude Code plugin")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	a := &asker{in: bufio.NewReader(in), out: out, interactive: !*yes}
	if *lang == "" {
		*lang = a.ask("Language / Idioma (en, es)", "en")
	}
	if *lang != "en" && *lang != "es" {
		fmt.Fprintf(errOut, "language %q: use en or es\n", *lang)
		return 2
	}
	o.Language = *lang
	o.Space = a.ask(msg(*lang, "space"), firstNonEmpty(o.Space, "work"))
	defaults(o, o.Space)
	o.WorkFolder = a.ask(msg(*lang, "work"), o.WorkFolder)
	o.Memory = a.ask(msg(*lang, "memory"), o.Memory)

	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	res, err := setup.Init(dir, *o)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	report(out, *lang, res)
	if !*noPlugin {
		installPlugin(a, *lang, out, errOut)
	}
	fmt.Fprintf(out, "\n"+msg(*lang, "done")+"\n", o.Space)
	fmt.Fprintf(out, msg(*lang, "next")+"\n", filepath.Join(dir, "profile.md"))
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
	o.Space = a.ask(msg(lang, "space"), firstNonEmpty(o.Space, "personal"))
	defaults(o, o.Space)
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
