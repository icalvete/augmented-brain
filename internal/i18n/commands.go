package i18n

// The other commands: check, space, backup, restore, update, uninstall.
var commandMessages = map[string]msg{
	"notes.1": {
		en: "1 note",
		es: "1 nota",
	},
	"notes.n": {
		en: "%d notes",
		es: "%d notas",
	},
	"problems.1": {
		en: "1 problem",
		es: "1 problema",
	},
	"problems.n": {
		en: "%d problems",
		es: "%d problemas",
	},
	"space.show": {
		en: "space:  %s\nmemory: %s",
		es: "espacio: %s\nmemoria: %s",
	},
	"space.none": {
		en: "%s is not inside any space's work folder.\nSpaces: %s",
		es: "%s no está dentro de la carpeta de trabajo de ningún espacio.\nEspacios: %s",
	},
	"config.missing": {
		en: "%v.\nRun `augmented-brain init` to create one.",
		es: "%v.\nLanza `augmented-brain init` para crearla.",
	},
	"check.summary": {
		en: "space %s: %s, %s",
		es: "espacio %s: %s, %s",
	},
	"check.rebuilt": {
		en: "%s rebuilt",
		es: "%s rehecho",
	},
	"check.format": {
		en: "notes moved from format %d to %d (%s); a copy from before is in %s",
		es: "notas pasadas del formato %d al %d (%s); hay una copia de antes en %s",
	},
	"check.repo": {
		en: "%s: repository %s is in none of the work folders (%s). Was it moved, renamed or deleted? Fix repos: in the note.",
		es: "%s: el repositorio %s no está en ninguna carpeta de trabajo (%s). ¿Se movió, cambió de nombre o se borró? Corrige repos: en la nota.",
	},
	"backup.setup": {
		en: "Space %s will back up to %s. Run `augmented-brain backup` to make the first copy.",
		es: "El espacio %s hará sus copias en %s. Lanza `augmented-brain backup` para hacer la primera.",
	},
	"backup.saved": {
		en: "✓ %s: changes saved and pushed to %s",
		es: "✓ %s: cambios guardados y subidos a %s",
	},
	"backup.same": {
		en: "✓ %s: nothing new; %s is up to date",
		es: "✓ %s: nada nuevo; %s está al día",
	},
	"backup.failed": {
		en: "✗ %s: %v",
		es: "✗ %s: %v",
	},
	"backup.leftout": {
		en: "! %s: left out of the backup, bigger than %[3]d MB: %[2]s. Move them out of the notes folder.",
		es: "! %s: fuera de la copia, de más de %[3]d MB: %[2]s. Sácalos de la carpeta de notas.",
	},
	"restore.done": {
		en: "✓ Space %s restored: its notes are in %s",
		es: "✓ Espacio %s recuperado: sus notas están en %s",
	},
	"restore.next": {
		en: "Now open Claude in it:\n  cd %s && claude",
		es: "Ahora abre Claude en él:\n  cd %s && claude",
	},
	"update.current": {
		en: "✓ augmented brain %s is the latest version.",
		es: "✓ augmented brain %s es la última versión.",
	},
	"update.found": {
		en: "augmented brain %s → %s: updating…",
		es: "augmented brain %s → %s: actualizando…",
	},
	"update.noreach": {
		en: "cannot reach GitHub to see the latest version: %v",
		es: "no puedo llegar a GitHub para ver la última versión: %v",
	},
	"update.failed": {
		en: "the update did not finish: %v",
		es: "la actualización no ha terminado: %v",
	},
	"update.plugin": {
		en: "✓ Claude Code plugin up to date. Restart Claude (close and open the app or the session) to use it.",
		es: "✓ Plugin de Claude Code al día. Reinicia Claude (cierra y abre la app o la sesión) para usarlo.",
	},
	"uninstall.plan": {
		en: "This removes augmented brain:",
		es: "Esto quita augmented brain:",
	},
	"uninstall.plugin": {
		en: "its Claude Code plugin",
		es: "su plugin de Claude Code",
	},
	"uninstall.leftovers": {
		en: "what Claude Code keeps of the plugin: its cache, its data and its server's logs",
		es: "lo que Claude Code guarda del plugin: su caché, sus datos y los registros de su servidor",
	},
	"uninstall.notes": {
		en: "%s (%d notes)",
		es: "%s (%d notas)",
	},
	"uninstall.keeptools": {
		en: "Claude Code, its app and git stay.",
		es: "Claude Code, su app y git se quedan.",
	},
	"uninstall.backups": {
		en: "Your backups are not deleted: %s. To delete one on GitHub: gh repo delete <owner/name>.",
		es: "Tus copias de seguridad no se borran: %s. Para borrar una en GitHub: gh repo delete <usuario/nombre>.",
	},
	"uninstall.doneall": {
		en: "✓ augmented brain removed, notes and configuration included. To start again: the same install command.",
		es: "✓ augmented brain quitado, con tus notas y tu configuración. Para empezar de nuevo: el mismo comando de instalación.",
	},
	"uninstall.keep": {
		en: "Your notes stay where they are: %s. Claude Code, its app and git stay too.",
		es: "Tus notas se quedan donde están: %s. Claude Code, su app y git también se quedan.",
	},
	"uninstall.ask": {
		en: "Go ahead? (yes, no)",
		es: "¿Sigo? (sí, no)",
	},
	"uninstall.nothing": {
		en: "Nothing removed.",
		es: "No he quitado nada.",
	},
	"uninstall.keptself": {
		en: "The program stays at %s: the plugin could not be removed completely, and while it is there Claude Code needs the program to check commands (without it, it stops every command and file change). Fix the error above and run `augmented-brain uninstall` again.",
		es: "El programa se queda en %s: el plugin no se pudo quitar del todo, y mientras esté, Claude Code necesita el programa para revisar los comandos (sin él, para todos los comandos y cambios de ficheros). Arregla el error de arriba y vuelve a lanzar `augmented-brain uninstall`.",
	},
	"uninstall.done": {
		en: "✓ augmented brain removed. To have it back: the same install command; your notes are still there.",
		es: "✓ augmented brain quitado. Para volver a tenerlo: el mismo comando de instalación; tus notas siguen ahí.",
	},
}
