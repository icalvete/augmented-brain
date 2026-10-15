// Package i18n holds the messages the program shows to people, in every supported
// language. Messages are fmt formats; a missing key falls back to English, and a key
// missing in English is shown as is, so a gap is visible but never breaks anything.
package i18n

import (
	"fmt"
	"os"
	"strings"
)

var messages = map[string]map[string]string{
	"en": {
		// doctor
		"doctor.noconfig":       "no configuration in %s",
		"doctor.noconfig.fix":   "run `augmented-brain init`",
		"doctor.badconfig.fix":  "fix the file, or move it away and run `augmented-brain init`",
		"doctor.config":         "configuration: %s",
		"doctor.noprofile":      "no profile at %s",
		"doctor.noprofile.fix":  "write who you are and how you like to work there",
		"doctor.profile":        "profile: %s",
		"doctor.inspace":        "this folder belongs to space %q",
		"doctor.nospace":        "this folder belongs to no space (spaces: %s)",
		"doctor.nospace.fix":    "open your assistant inside a space's work folder, or run `augmented-brain space add`",
		"doctor.nomemory":       "space %s: memory folder %s: %v",
		"doctor.nomemory.fix":   "create it, or fix the path in %s",
		"doctor.invalid":        "space %s: %d notes are not valid",
		"doctor.invalid.fix":    "run `augmented-brain check` inside the space to see them",
		"doctor.valid":          "space %s: %s, all valid",
		"doctor.stale":          "space %s: %s is out of date",
		"doctor.stale.fix":      "run `augmented-brain check` inside the space to rebuild it",
		"doctor.tool":           "%s: %s",
		"doctor.notool":         "%s: not found",
		"doctor.claude":         "Claude Code",
		"doctor.claude.fix":     "install it: https://docs.claude.com/en/docs/claude-code",
		"doctor.git":            "git, for backups",
		"doctor.git.fix":        "install git with your system's package manager",
		"doctor.plugin.unknown": "Claude Code plugin: cannot tell (%v)",
		"doctor.plugin.missing": "Claude Code plugin %s: not installed",
		"doctor.plugin.ok":      "Claude Code plugin %s: installed",
		"notes.1":               "1 note",
		"notes.n":               "%d notes",
		"problems.1":            "1 problem",
		"problems.n":            "%d problems",
		// commands
		"space.show":     "space:  %s\nmemory: %s",
		"space.none":     "%s is not inside any space's work folder.\nSpaces: %s",
		"config.missing": "%v.\nRun `augmented-brain init` to create one.",
		"check.summary":  "space %s: %s, %s",
		"check.rebuilt":  "%s rebuilt",
		"backup.setup":   "Space %s will back up to %s. Run `augmented-brain backup` to make the first copy.",
		"backup.saved":   "✓ %s: changes saved and pushed to %s",
		"backup.same":    "✓ %s: nothing new; %s is up to date",
		"backup.failed":  "✗ %s: %v",
		"restore.done":   "Space %s restored in %s.\nWork folder: %s\nNext: run `augmented-brain doctor`.",
	},
	"es": {
		"doctor.noconfig":       "no hay configuración en %s",
		"doctor.noconfig.fix":   "lanza `augmented-brain init`",
		"doctor.badconfig.fix":  "arregla el fichero, o apártalo y lanza `augmented-brain init`",
		"doctor.config":         "configuración: %s",
		"doctor.noprofile":      "no hay perfil en %s",
		"doctor.noprofile.fix":  "escribe ahí quién eres y cómo te gusta trabajar",
		"doctor.profile":        "perfil: %s",
		"doctor.inspace":        "esta carpeta es del espacio %q",
		"doctor.nospace":        "esta carpeta no es de ningún espacio (espacios: %s)",
		"doctor.nospace.fix":    "abre tu asistente dentro de la carpeta de trabajo de un espacio, o lanza `augmented-brain space add`",
		"doctor.nomemory":       "espacio %s: carpeta de la memoria %s: %v",
		"doctor.nomemory.fix":   "créala, o corrige la ruta en %s",
		"doctor.invalid":        "espacio %s: %d notas no son válidas",
		"doctor.invalid.fix":    "lanza `augmented-brain check` dentro del espacio para verlas",
		"doctor.valid":          "espacio %s: %s, todas válidas",
		"doctor.stale":          "espacio %s: %s está desfasado",
		"doctor.stale.fix":      "lanza `augmented-brain check` dentro del espacio para rehacerlo",
		"doctor.tool":           "%s: %s",
		"doctor.notool":         "%s: no encontrado",
		"doctor.claude":         "Claude Code",
		"doctor.claude.fix":     "instálalo: https://docs.claude.com/en/docs/claude-code",
		"doctor.git":            "git, para las copias de seguridad",
		"doctor.git.fix":        "instala git con el gestor de paquetes de tu sistema",
		"doctor.plugin.unknown": "plugin de Claude Code: no se puede saber (%v)",
		"doctor.plugin.missing": "plugin de Claude Code %s: no instalado",
		"doctor.plugin.ok":      "plugin de Claude Code %s: instalado",
		"notes.1":               "1 nota",
		"notes.n":               "%d notas",
		"problems.1":            "1 problema",
		"problems.n":            "%d problemas",
		"space.show":            "espacio: %s\nmemoria: %s",
		"space.none":            "%s no está dentro de la carpeta de trabajo de ningún espacio.\nEspacios: %s",
		"config.missing":        "%v.\nLanza `augmented-brain init` para crearla.",
		"check.summary":         "espacio %s: %s, %s",
		"check.rebuilt":         "%s rehecho",
		"backup.setup":          "El espacio %s hará sus copias en %s. Lanza `augmented-brain backup` para hacer la primera.",
		"backup.saved":          "✓ %s: cambios guardados y subidos a %s",
		"backup.same":           "✓ %s: nada nuevo; %s está al día",
		"backup.failed":         "✗ %s: %v",
		"restore.done":          "Espacio %s recuperado en %s.\nCarpeta de trabajo: %s\nSiguiente: lanza `augmented-brain doctor`.",
	},
}

// T formats the message key in lang.
func T(lang, key string, args ...any) string {
	f, ok := messages[lang][key]
	if !ok {
		if f, ok = messages["en"][key]; !ok {
			f = key
		}
	}
	if len(args) == 0 {
		return f
	}
	return fmt.Sprintf(f, args...)
}

// Count returns "1 note" or "3 notes" (key "notes"), in lang.
func Count(lang, key string, n int) string {
	if n == 1 {
		return T(lang, key+".1")
	}
	return T(lang, key+".n", n)
}

// FromEnv guesses the language from $LANG when there is no configuration yet.
func FromEnv() string {
	if strings.HasPrefix(os.Getenv("LANG"), "es") {
		return "es"
	}
	return "en"
}
