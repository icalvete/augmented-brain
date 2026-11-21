package i18n

// augmented-brain clean, and the doctor line that leads to it.
var cleanMessages = map[string]msg{
	"clean.nothing": {
		en: "Nothing to clean: no worktree of Claude Code without its session, no scratch folder of a session that ended, and no background session left over.",
		es: "No hay nada que limpiar: ningún worktree de Claude Code sin su sesión, ni carpetas temporales de sesiones terminadas, ni sesiones de fondo que sobren.",
	},
	"clean.sessions": {
		en: "Sessions of tasks parked, put off or closed (claude stop and claude rm: they leave Claude Desktop and claude.ai, and do not come back; their worktrees stay):",
		es: "Sesiones de tareas aparcadas, para algún día o cerradas (claude stop y claude rm: salen de Claude Desktop y claude.ai, y no vuelven; sus worktrees se quedan):",
	},
	"clean.session": {
		en: "\"%s\", of %s (%s)",
		es: "«%s», de %s (%s)",
	},
	"clean.session.gone": {
		en: "\"%s\", of %s (no space has that task any more)",
		es: "«%s», de %s (ningún espacio tiene ya esa tarea)",
	},
	"clean.status.parked":  {en: "parked", es: "aparcada"},
	"clean.status.someday": {en: "someday", es: "para algún día"},
	"clean.status.closed":  {en: "closed", es: "cerrada"},
	"clean.sessions.unread": {
		en: "The background sessions were not looked at: Claude's list could not be read (%v).",
		es: "No he mirado las sesiones de fondo: no he podido leer la lista de Claude (%v).",
	},
	"clean.removed.sessions": {
		en: "Sessions of tasks closed: %d. They left Claude Desktop and claude.ai.",
		es: "Sesiones de tareas cerradas: %d. Han salido de Claude Desktop y claude.ai.",
	},
	"clean.others": {
		en: "Background sessions no task opened, ended or archived. Someone else started them, so I ask for each one (claude stop and claude rm: it leaves Claude Desktop and claude.ai, and its conversation stays for claude --resume):",
		es: "Sesiones de fondo que no abrió ninguna tarea, terminadas o archivadas. Las empezó otro, así que pregunto por cada una (claude stop y claude rm: sale de Claude Desktop y claude.ai, y su conversación se queda para claude --resume):",
	},
	"clean.other": {
		en: "\"%s\" (%s), in %s, %s",
		es: "«%s» (%s), en %s, %s",
	},
	"clean.other.worktree": {
		en: "· its worktree goes too, if nothing would be lost",
		es: "· su worktree también se va, si no se pierde nada",
	},
	"clean.other.ask": {
		en: "Remove this one? (yes, no)",
		es: "¿Quito esta? (sí, no)",
	},
	"clean.other.removed": {
		en: "Removed \"%s\".",
		es: "He quitado «%s».",
	},
	"clean.others.yes": {
		en: "None of these goes with --yes: run augmented-brain clean without it to be asked for each one.",
		es: "Ninguna de estas se va con --yes: ejecuta augmented-brain clean sin él y te pregunto por cada una.",
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
	"clean.kept.ignored": {
		en: "%s: files git ignores that may exist nowhere else: %s",
		es: "%s: ficheros que git ignora que quizá no estén en otro sitio: %s",
	},
	"clean.prunable": {
		en: "Repositories that remember worktrees whose folder is gone (git worktree prune; their branches stay):",
		es: "Repositorios que recuerdan worktrees cuya carpeta ya no existe (git worktree prune; sus ramas se quedan):",
	},
	"clean.scratch": {
		en: "Scratch folders of %d Claude sessions that ended over a day ago: %s in %s",
		es: "Carpetas temporales de %d sesiones de Claude terminadas hace más de un día: %s en %s",
	},
	"clean.oneprofile": {
		en: "Only the sessions of %s count as running: if you use another Claude folder (CLAUDE_CONFIG_DIR) with a session open now, close it first or answer no.",
		es: "Solo cuento como abiertas las sesiones de %s: si usas otra carpeta de Claude (CLAUDE_CONFIG_DIR) con una sesión abierta ahora, ciérrala antes o contesta que no.",
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
		en: "if you no longer need it: claude stop %s, and then augmented-brain clean takes it out of Claude Desktop and claude.ai",
		es: "si ya no la necesitas: claude stop %s, y después augmented-brain clean la quita de Claude Desktop y claude.ai",
	},
	"doctor.spare": {
		en: "%d background sessions are left over (%d of them of tasks parked, put off or closed): they stay in Claude Desktop and claude.ai, and Claude may bring them back when it starts again",
		es: "Sobran %d sesiones de fondo (%d de ellas de tareas aparcadas, para algún día o cerradas): siguen en Claude Desktop y claude.ai, y Claude puede volver a levantarlas cuando arranque",
	},
	"doctor.spare.fix": {
		en: "run augmented-brain clean: it shows them and asks before removing them",
		es: "ejecuta augmented-brain clean: las enseña y pregunta antes de quitarlas",
	},
}
