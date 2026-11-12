package i18n

// augmented-brain watch: following a pull request until it has something to tell, and
// what it writes in the task and tells the assistant when it ends.
var watchMessages = map[string]msg{
	"usage.watch": {
		en: "usage: augmented-brain watch pr NUMBER [--task TASK] [--repo OWNER/NAME] [--max 3h] [--every 30s]",
		es: "uso: augmented-brain watch pr NÚMERO [--task TAREA] [--repo DUEÑO/NOMBRE] [--max 3h] [--every 30s]",
	},
	"watch.nogh": {
		en: "watch needs gh, GitHub's command, and it is not installed: https://cli.github.com",
		es: "watch necesita gh, el programa de GitHub, y no está instalado: https://cli.github.com",
	},
	"watch.start": {
		en: "Watching pull request #%d, for %s at most. It ends on its own and says how it went.",
		es: "Vigilo la pull request #%d, como mucho %s. Termina sola y dice cómo ha ido.",
	},
	"watch.green": {
		en: "PR #%d: every required check passed (%s). It is still open: it waits to be merged.",
		es: "PR #%d: todos los checks obligatorios en verde (%s). Sigue abierta: falta mezclarla.",
	},
	"watch.failed": {
		en: "PR #%d: a required check failed: %s (%s).",
		es: "PR #%d: ha fallado un check obligatorio: %s (%s).",
	},
	"watch.merged": {
		en: "PR #%d was merged (%s).",
		es: "La PR #%d se ha mezclado (%s).",
	},
	"watch.closed": {
		en: "PR #%d was closed without being merged (%s).",
		es: "La PR #%d se ha cerrado sin mezclarse (%s).",
	},
	"watch.timeout": {
		en: "PR #%d: after %s, still pending: %s. I stopped watching it.",
		es: "PR #%d: tras %s, siguen pendientes: %s. He dejado de vigilarla.",
	},
	"watch.broken": {
		en: "PR #%d: I could not follow it, gh says: %s. I stopped watching it (%s).",
		es: "PR #%d: no he podido seguirla, gh dice: %s. He dejado de vigilarla (%s).",
	},
	"watch.others": {
		en: " Not required, still running or failed: %s.",
		es: " No obligatorios, aún en marcha o fallidos: %s.",
	},
	"watch.minutes": {
		en: "%d min",
		es: "%d min",
	},
	"watch.journal": {
		en: "Watch of PR #%d: %s The next step before it was: %s",
		es: "Vigilancia de la PR #%d: %s El siguiente paso de antes era: %s",
	},
	"watch.saved": {
		en: "Written in task %s: in its journal and as its next step.",
		es: "Apuntado en la tarea %s: en su diario y como su siguiente paso.",
	},
	"watch.unsaved": {
		en: "Could not write it in task %s: %s",
		es: "No he podido apuntarlo en la tarea %s: %s",
	},
	"watch.tell": {
		en: "For the assistant: tell the user this now, in their language, even if they are not looking at this session. If another Claude session asked for this task, send it the same line with SendMessage.",
		es: "Para el asistente: díselo ahora al usuario, en su idioma, aunque no esté mirando esta sesión. Si otra sesión de Claude pidió esta tarea, mándale la misma línea con SendMessage.",
	},
}
