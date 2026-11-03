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
	"secret.an AWS access key": {
		en: "an AWS access key",
		es: "una clave de acceso de AWS",
	},
	"secret.an AWS temporary access key": {
		en: "an AWS temporary access key",
		es: "una clave temporal de AWS",
	},
	"secret.an AWS secret key": {
		en: "an AWS secret key",
		es: "una clave secreta de AWS",
	},
	"secret.a GitHub token": {
		en: "a GitHub token",
		es: "un token de GitHub",
	},
	"secret.a GitLab token": {
		en: "a GitLab token",
		es: "un token de GitLab",
	},
	"secret.an Anthropic API key": {
		en: "an Anthropic API key",
		es: "una clave de la API de Anthropic",
	},
	"secret.an OpenAI API key": {
		en: "an OpenAI API key",
		es: "una clave de la API de OpenAI",
	},
	"secret.a Google API key": {
		en: "a Google API key",
		es: "una clave de la API de Google",
	},
	"secret.a Slack token": {
		en: "a Slack token",
		es: "un token de Slack",
	},
	"secret.a Google AI Studio API key": {
		en: "a Google AI Studio API key",
		es: "una clave de Google AI Studio",
	},
	"secret.a PostHog personal API key": {
		en: "a PostHog personal API key",
		es: "una clave personal de PostHog",
	},
	"secret.an Elasticsearch API key": {
		en: "an Elasticsearch API key",
		es: "una clave de la API de Elasticsearch",
	},
	"secret.a Telegram bot token": {
		en: "a Telegram bot token",
		es: "un token de bot de Telegram",
	},
	"secret.a Stripe secret key": {
		en: "a Stripe secret key",
		es: "una clave secreta de Stripe",
	},
	"secret.an npm token": {
		en: "an npm token",
		es: "un token de npm",
	},
	"secret.a Hugging Face token": {
		en: "a Hugging Face token",
		es: "un token de Hugging Face",
	},
	"secret.a Docker Hub token": {
		en: "a Docker Hub token",
		es: "un token de Docker Hub",
	},
	"secret.a SendGrid API key": {
		en: "a SendGrid API key",
		es: "una clave de API de SendGrid",
	},
	"secret.a private key": {
		en: "a private key",
		es: "una clave privada",
	},
	"check.secret": {
		en: "%s has a secret written in it (%s). The backup refuses to upload it until it is taken out: keep it in your keychain or a secrets manager.",
		es: "%s tiene un secreto escrito (%s). La copia de seguridad no lo sube hasta que se quite: guárdalo en tu llavero o en un gestor de secretos.",
	},
	"check.shared": {
		en: "shared notes (%s): %s, %s",
		es: "notas compartidas (%s): %s, %s",
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
	"backup.secrets": {
		en: "✗ %s: nothing was backed up: these files hold secrets, remove them first: %s",
		es: "✗ %s: no se ha copiado nada: estos ficheros tienen secretos, quítalos primero: %s",
	},
	"backup.conflict": {
		en: "✗ %s: nothing was backed up: another machine changed the same notes (%s). Open %s, run `git merge origin/main`, keep the right version of each, commit, and back up again",
		es: "✗ %s: no se ha copiado nada: otra máquina cambió las mismas notas (%s). Abre %s, lanza `git merge origin/main`, quédate con la versión buena de cada una, haz commit y vuelve a copiar",
	},
	"backup.leftout": {
		en: "! %s: left out of the backup, bigger than %[3]d MB: %[2]s. Move them out of the notes folder.",
		es: "! %s: fuera de la copia, de más de %[3]d MB: %[2]s. Sácalos de la carpeta de notas.",
	},
	"restore.done": {
		en: "✓ Space %s restored: its notes are in %s",
		es: "✓ Espacio %s recuperado: sus notas están en %s",
	},
	"restore.nolists": {
		en: "The guard's lists (production AWS profiles, untouchable hosts) are not in any backup: your first Claude session offers to ask for them again. Until then, nothing asks before touching production.",
		es: "Las listas del guardia (perfiles de AWS de producción, máquinas intocables) no van en ninguna copia: tu primera sesión de Claude se ofrece a preguntártelas otra vez. Hasta entonces, nada pregunta antes de tocar producción.",
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
	"args.extra": {
		en: "This command does not take: %s. Run `augmented-brain help` to see what it takes.",
		es: "Este comando no lleva: %s. Lanza `augmented-brain help` para ver qué lleva.",
	},
	"usage": {
		en: `augmented brain: memory, method and a guard for AI coding assistants.

Usage:
  augmented-brain init         set up augmented brain: the work and personal spaces and the plugin (--custom: choose them yourself)
  augmented-brain space        show which space this folder belongs to
  augmented-brain space add    add another space (work, personal…)
  augmented-brain check        check the notes of this folder's space and rebuild its index
  augmented-brain doctor       say what is missing, and how to fix it
  augmented-brain import DIR   bring in a memory of Claude Code (--map FILE, --repos-base DIR; --apply to write it)
  augmented-brain backup       save this folder's space to its private repository (--all: every space)
  augmented-brain backup setup URL   connect this folder's space to a private git repository
  augmented-brain restore URL  bring a space back from its backup, on a new machine
  augmented-brain panel        open the panel in your browser: see and edit your tasks, inbox, profile… (panel stop: stop it)
  augmented-brain update       update augmented brain and its Claude Code plugin
  augmented-brain uninstall    remove the program and the plugin (your notes stay; --all: start from scratch)
  augmented-brain mcp          run the MCP server (your assistant starts it, not you)
  augmented-brain hook EVENT   answer an assistant hook: session-start, pre-tool-use, stop, post-compact, session-end
  augmented-brain version      show the version
`,
		es: `augmented brain: memoria, método y un guardia para asistentes de programación con IA.

Uso:
  augmented-brain init         prepara augmented brain: los espacios trabajo y personal y el plugin (--custom: elígelos tú)
  augmented-brain space        dice de qué espacio es esta carpeta
  augmented-brain space add    añade otro espacio (trabajo, personal…)
  augmented-brain check        revisa las notas del espacio de esta carpeta y rehace su índice
  augmented-brain doctor       dice qué falta y cómo arreglarlo
  augmented-brain import DIR   trae una memoria de Claude Code (--map FICHERO, --repos-base DIR; --apply para escribirla)
  augmented-brain backup       guarda el espacio de esta carpeta en su repositorio privado (--all: todos los espacios)
  augmented-brain backup setup URL   conecta el espacio de esta carpeta a un repositorio git privado
  augmented-brain restore URL  recupera un espacio de su copia, en una máquina nueva
  augmented-brain panel        abre el panel en el navegador: ve y cambia tus tareas, buzón, perfil… (panel stop: lo para)
  augmented-brain update       actualiza augmented brain y su plugin de Claude Code
  augmented-brain uninstall    quita el programa y el plugin (tus notas se quedan; --all: empezar de cero)
  augmented-brain mcp          arranca el servidor MCP (lo arranca tu asistente, no tú)
  augmented-brain hook EVENTO  responde a un hook del asistente: session-start, pre-tool-use, stop, post-compact, session-end
  augmented-brain version      muestra la versión
`,
	},
	"usage.unknown": {
		en: "unknown command %q",
		es: "no conozco el comando %q",
	},
	"usage.backup.setup": {
		en: "usage: augmented-brain backup setup <url of a private git repository>",
		es: "uso: augmented-brain backup setup <url de un repositorio git privado>",
	},
	"usage.restore": {
		en: "usage: augmented-brain restore <url of the space's backup> [--memory dir] [--work dir]",
		es: "uso: augmented-brain restore <url de la copia del espacio> [--memory carpeta] [--work carpeta]",
	},
	"usage.panel": {
		en: "usage: augmented-brain panel [--no-browser] | panel stop",
		es: "uso: augmented-brain panel [--no-browser] | panel stop",
	},
	"space.unknown": {
		en: "there is no space called %q",
		es: "no hay ningún espacio llamado %q",
	},
	"init.language": {
		en: "language %q: use en or es",
		es: "idioma %q: usa en o es",
	},
	"backup.warning": {
		en: "these files are bigger than %d MB and were left out of the backup: %s. Move them out of the notes folder (or make them smaller); the rest is backed up",
		es: "estos ficheros pasan de %d MB y se han quedado fuera de la copia: %s. Sácalos de la carpeta de notas (o hazlos más pequeños); el resto está copiado",
	},
}
