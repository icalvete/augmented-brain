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
		"doctor.blankprofile":   "profile: not filled in yet; Claude will ask you in your first session",
		"doctor.profile":        "profile: %s",
		"doctor.inspace":        "this folder belongs to space %q",
		"doctor.default":        "this folder is outside your spaces' folders: Claude uses the default space, %s",
		"doctor.nospace":        "this folder is outside your spaces; open Claude in one of them: %s",
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
		"doctor.claude.fix":     "install it with: curl -fsSL https://claude.ai/install.sh | bash",
		"doctor.git":            "git",
		"doctor.git.fix":        "install it with: %s",
		"doctor.plugin.unknown": "Claude Code plugin: cannot tell (%v)",
		"doctor.plugin.missing": "Claude Code plugin %s: not installed",
		"doctor.plugin.ok":      "Claude Code plugin %s: installed",
		"notes.1":               "1 note",
		"notes.n":               "%d notes",
		"problems.1":            "1 problem",
		"problems.n":            "%d problems",
		// commands
		"space.show":          "space:  %s\nmemory: %s",
		"space.none":          "%s is not inside any space's work folder.\nSpaces: %s",
		"config.missing":      "%v.\nRun `augmented-brain init` to create one.",
		"check.summary":       "space %s: %s, %s",
		"check.rebuilt":       "%s rebuilt",
		"backup.setup":        "Space %s will back up to %s. Run `augmented-brain backup` to make the first copy.",
		"backup.saved":        "✓ %s: changes saved and pushed to %s",
		"backup.same":         "✓ %s: nothing new; %s is up to date",
		"backup.failed":       "✗ %s: %v",
		"restore.done":        "✓ Space %s restored: its notes are in %s",
		"restore.next":        "Now open Claude in it:\n  cd %s && claude",
		"update.current":      "✓ augmented brain %s is the latest version.",
		"update.found":        "augmented brain %s → %s: updating…",
		"update.noreach":      "cannot reach GitHub to see the latest version: %v",
		"update.failed":       "the update did not finish: %v",
		"update.plugin":       "✓ Claude Code plugin up to date. Restart Claude (close and open the app or the session) to use it.",
		"uninstall.plan":      "This removes augmented brain:",
		"uninstall.plugin":    "its Claude Code plugin",
		"uninstall.leftovers": "what Claude Code keeps of the plugin: its cache, its data and its server's logs",
		"uninstall.notes":     "%s (%d notes)",
		"uninstall.keeptools": "Claude Code, its app and git stay.",
		"uninstall.backups":   "Your backups are not deleted: %s. To delete one on GitHub: gh repo delete <owner/name>.",
		"uninstall.doneall":   "✓ augmented brain removed, notes and configuration included. To start again: the same install command.",
		"uninstall.keep":      "Your notes stay where they are: %s. Claude Code, its app and git stay too.",
		"uninstall.ask":       "Go ahead? (yes, no)",
		"uninstall.nothing":   "Nothing removed.",
		"uninstall.done":      "✓ augmented brain removed. To have it back: the same install command; your notes are still there.",
	},
	"es": {
		"doctor.noconfig":       "no hay configuración en %s",
		"doctor.noconfig.fix":   "lanza `augmented-brain init`",
		"doctor.badconfig.fix":  "arregla el fichero, o apártalo y lanza `augmented-brain init`",
		"doctor.config":         "configuración: %s",
		"doctor.blankprofile":   "perfil: aún sin rellenar; Claude te lo preguntará en tu primera sesión",
		"doctor.profile":        "perfil: %s",
		"doctor.inspace":        "esta carpeta es del espacio %q",
		"doctor.default":        "esta carpeta está fuera de las de tus espacios: Claude usa el espacio por defecto, %s",
		"doctor.nospace":        "esta carpeta está fuera de tus espacios; abre Claude en uno de ellos: %s",
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
		"doctor.claude.fix":     "instálalo con: curl -fsSL https://claude.ai/install.sh | bash",
		"doctor.git":            "git",
		"doctor.git.fix":        "instálalo con: %s",
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
		"restore.done":          "✓ Espacio %s recuperado: sus notas están en %s",
		"restore.next":          "Ahora abre Claude en él:\n  cd %s && claude",
		"update.current":        "✓ augmented brain %s es la última versión.",
		"update.found":          "augmented brain %s → %s: actualizando…",
		"update.noreach":        "no puedo llegar a GitHub para ver la última versión: %v",
		"update.failed":         "la actualización no ha terminado: %v",
		"update.plugin":         "✓ Plugin de Claude Code al día. Reinicia Claude (cierra y abre la app o la sesión) para usarlo.",
		"uninstall.plan":        "Esto quita augmented brain:",
		"uninstall.plugin":      "su plugin de Claude Code",
		"uninstall.leftovers":   "lo que Claude Code guarda del plugin: su caché, sus datos y los registros de su servidor",
		"uninstall.notes":       "%s (%d notas)",
		"uninstall.keeptools":   "Claude Code, su app y git se quedan.",
		"uninstall.backups":     "Tus copias de seguridad no se borran: %s. Para borrar una en GitHub: gh repo delete <usuario/nombre>.",
		"uninstall.doneall":     "✓ augmented brain quitado, con tus notas y tu configuración. Para empezar de nuevo: el mismo comando de instalación.",
		"uninstall.keep":        "Tus notas se quedan donde están: %s. Claude Code, su app y git también se quedan.",
		"uninstall.ask":         "¿Sigo? (sí, no)",
		"uninstall.nothing":     "No he quitado nada.",
		"uninstall.done":        "✓ augmented brain quitado. Para volver a tenerlo: el mismo comando de instalación; tus notas siguen ahí.",
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

// FromEnv guesses the language when there is no configuration yet: AB_LANGUAGE, which
// the installers set; then the locale, in the order the C library reads it (LC_ALL,
// LC_MESSAGES, LANG); then the system's own setting where there is one, as on Windows,
// which has no locale variables.
func FromEnv() string {
	if l := os.Getenv("AB_LANGUAGE"); l == "en" || l == "es" {
		return l
	}
	for _, v := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if l := os.Getenv(v); l != "" {
			return pick(l)
		}
	}
	return pick(systemLanguage())
}

// pick maps a locale or language tag (es_ES.UTF-8, es-ES, en-US) to a language we speak.
func pick(l string) string {
	if strings.HasPrefix(strings.ToLower(l), "es") {
		return "es"
	}
	return "en"
}
