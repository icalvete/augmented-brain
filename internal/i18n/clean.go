package i18n

// augmented-brain clean, and the doctor line that leads to it.
var cleanMessages = map[string]msg{
	"clean.nothing": {
		en: "Nothing to clean: no worktree of Claude Code without its session, and no scratch folder of a session that ended.",
		es: "No hay nada que limpiar: ningún worktree de Claude Code sin su sesión, ni carpetas temporales de sesiones terminadas.",
	},
	"clean.plan": {
		en: "This is left over from working with Claude Code:",
		es: "Esto ha quedado del trabajo con Claude Code:",
	},
	"clean.worktrees": {
		en: "Worktrees Claude Code made whose session is over, with nothing of their own (git worktree remove):",
		es: "Worktrees que hizo Claude Code, con su sesión terminada y sin nada propio (git worktree remove):",
	},
	"clean.worktree.branch": {
		en: "%s (and its branch %s, whose commits are all on the remote)",
		es: "%s (y su rama %s, con todos sus commits en el remoto)",
	},
	"clean.kept": {
		en: "These stay, with work of their own; look at them yourself:",
		es: "Estos se quedan, con trabajo propio; míralos tú:",
	},
	"clean.kept.changes": {
		en: "%s: changes not committed",
		es: "%s: cambios sin commit",
	},
	"clean.kept.commits": {
		en: "%s: commits not on the remote",
		es: "%s: commits que no están en el remoto",
	},
	"clean.prunable": {
		en: "Repositories that remember worktrees whose folder is gone (git worktree prune; their branches stay):",
		es: "Repositorios que recuerdan worktrees cuya carpeta ya no existe (git worktree prune; sus ramas se quedan):",
	},
	"clean.scratch": {
		en: "Scratch folders of %d Claude sessions that ended over a day ago: %s in %s",
		es: "Carpetas temporales de %d sesiones de Claude terminadas hace más de un día: %s en %s",
	},
	"clean.noscratch": {
		en: "The scratch folders were not looked at: the sessions running could not be read from %s.",
		es: "No he mirado las carpetas temporales: no he podido leer en %s qué sesiones están abiertas.",
	},
	"clean.ask": {
		en: "Remove it? (yes, no)",
		es: "¿Lo quito? (sí, no)",
	},
	"clean.removed.worktrees": {
		en: "Removed %d worktrees.",
		es: "He quitado %d worktrees.",
	},
	"clean.removed.pruned": {
		en: "Forgot the missing worktrees of %d repositories.",
		es: "He olvidado los worktrees perdidos de %d repositorios.",
	},
	"clean.removed.scratch": {
		en: "Removed %d scratch folders: %s freed.",
		es: "He quitado %d carpetas temporales: %s libres.",
	},
	"clean.removed.kept": {
		en: "%s stays: it got work of its own since.",
		es: "%s se queda: desde entonces tiene trabajo propio.",
	},
	"doctor.leftovers": {
		en: "Left over from working with Claude Code: %d worktrees whose session is over, %d records of worktrees whose folder is gone, %s of scratch folders of sessions that ended",
		es: "Ha quedado del trabajo con Claude Code: %d worktrees con su sesión terminada, %d registros de worktrees cuya carpeta ya no existe, %s de carpetas temporales de sesiones terminadas",
	},
	"doctor.leftovers.fix": {
		en: "run augmented-brain clean: it shows the list, and removes it only if you say yes",
		es: "ejecuta augmented-brain clean: enseña la lista, y solo lo quita si dices que sí",
	},
	"doctor.leftovers.kept": {
		en: "%d worktrees Claude Code made have work of their own but no session: %s",
		es: "%d worktrees que hizo Claude Code tienen trabajo propio pero ninguna sesión: %s",
	},
	"doctor.leftovers.kept.fix": {
		en: "commit and push it, or remove it yourself if it is not needed (augmented-brain clean leaves it alone)",
		es: "haz commit y push, o quítalo tú si no hace falta (augmented-brain clean no lo toca)",
	},
	"doctor.lingering.ended": {
		en: "The background session %q (%s) ended or was archived, but its process still runs",
		es: "La sesión de fondo %q (%s) terminó o se archivó, pero su proceso sigue en marcha",
	},
	"doctor.lingering.idle": {
		en: "The background session %q (%s) has been idle for over %d hours and no task opened it: if you archived it in Claude Desktop, its process still runs",
		es: "La sesión de fondo %q (%s) lleva más de %d horas parada y no la abrió ninguna tarea: si la archivaste en Claude Desktop, su proceso sigue en marcha",
	},
	"doctor.lingering.fix": {
		en: "if you no longer need it: claude stop %s",
		es: "si ya no la necesitas: claude stop %s",
	},
}
