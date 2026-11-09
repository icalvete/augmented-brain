package i18n

// `augmented-brain session`: a remote session of its own for one task, and what the
// session start tells the assistant in it.
var launchMessages = map[string]msg{
	"usage.session": {
		en: "usage: augmented-brain session start TASK [--folder DIR] | stop TASK | list",
		es: "uso: augmented-brain session start TAREA [--folder DIR] | stop TAREA | list",
	},
	"session.notask": {
		en: "there is no task %q in space %s.",
		es: "no hay ninguna tarea %q en el espacio %s.",
	},
	"session.nofolder": {
		en: "I cannot tell in which folder to open %s: its repositories (%s) are not inside the work folders of space %s. Say it with --folder DIR.",
		es: "no sé en qué carpeta abrir %s: sus repositorios (%s) no están en las carpetas de trabajo del espacio %s. Dímela con --folder DIR.",
	},
	"session.nofolder.none": {
		en: "%s names no repository, so I cannot tell in which folder to open it. Say it with --folder DIR (a folder of its own: the space's remote service already serves %s).",
		es: "%s no nombra ningún repositorio, así que no sé en qué carpeta abrirla. Dímela con --folder DIR (una carpeta propia: el servicio remoto del espacio ya sirve %s).",
	},
	"session.started": {
		en: "Opened the session \"%s\" for %s, in %s. It shows up in Claude Desktop and claude.ai within a minute. It waits for you: when you write to it, it reads its task first.\nIt closes when that session ends, or with: augmented-brain session stop %s",
		es: "Abierta la sesión «%s» para %s, en %s. Sale en Claude Desktop y claude.ai en menos de un minuto. Te espera: cuando le escribes, lo primero que hace es leer su tarea.\nSe cierra al terminar esa sesión, o con: augmented-brain session stop %s",
	},
	"session.worktree.made": {
		en: "New worktree %s for the task, on branch %s, from %s.",
		es: "Nuevo worktree %s para la tarea, en la rama %s, desde %s.",
	},
	"session.worktree.offline": {
		en: "Could not reach origin: the worktree starts from %s as this computer last saw it.",
		es: "No he podido llegar a origin: el worktree sale de %s tal como lo vio este ordenador la última vez.",
	},
	"session.worktree.taken": {
		en: "%s is there and is not a worktree of %s. Move it, or give a folder with --folder DIR.",
		es: "%s ya existe y no es un worktree de %s. Muévelo, o dame una carpeta con --folder DIR.",
	},
	"session.worktree.removed": {
		en: "Removed its worktree %s and its branch: it held nothing that was not on the remote.",
		es: "Quitado su worktree %s y su rama: no tenía nada que no estuviera en el remoto.",
	},
	"session.worktree.kept.changes": {
		en: "Kept its worktree %s: it has changes not committed.",
		es: "Dejo su worktree %s: tiene cambios sin commitear.",
	},
	"session.worktree.kept.commits": {
		en: "Kept its worktree %s: it has commits not pushed.",
		es: "Dejo su worktree %s: tiene commits sin subir.",
	},
	"session.worktree.swept": {
		en: "Removed %s: the worktree of a session that has ended, with nothing that was not on the remote.",
		es: "Quitado %s: el worktree de una sesión que ya terminó, sin nada que no estuviera en el remoto.",
	},
	"session.worktree.closed": {
		en: "Removed the task's worktree %s and its branch: it held nothing that was not on the remote.",
		es: "Quitado el worktree de la tarea %s y su rama: no tenía nada que no estuviera en el remoto.",
	},
	"session.worktree.closed.kept": {
		en: "The task's worktree %s stays: it has %s. Push or drop that work, then remove it.",
		es: "El worktree de la tarea %s se queda: tiene %s. Sube o descarta ese trabajo, y después quítalo.",
	},
	"session.worktree.why.changes": {
		en: "changes not committed",
		es: "cambios sin commitear",
	},
	"session.worktree.why.commits": {
		en: "commits not pushed",
		es: "commits sin subir",
	},
	"doctor.worktree.kept": {
		en: "The worktree %s of task %s has %s, and its session has ended.",
		es: "El worktree %s de la tarea %s tiene %s, y su sesión ya terminó.",
	},
	"doctor.worktree.kept.fix": {
		en: "Push or drop that work; the next session start or doctor removes it then. Or ask for the task's session again: it uses it.",
		es: "Sube o descarta ese trabajo; después lo quita el próximo session start o doctor. O pide otra vez la sesión de la tarea: lo usa.",
	},
	"doctor.worktree.swept": {
		en: "Removed %d worktree(s) of sessions that had ended, with nothing that was not on the remote.",
		es: "Quitados %d worktree(s) de sesiones que ya terminaron, sin nada que no estuviera en el remoto.",
	},
	"session.untrusted": {
		en: "Claude does not trust %s yet, and its server only starts in a folder that accepted the question itself (trusting a folder above it is not enough). Once, in a terminal on %s (from another computer: ssh %s):",
		es: "Claude aún no confía en %s, y su servidor solo arranca en una carpeta que ha aceptado la pregunta ella misma (que confíe en una carpeta de arriba no basta). Una vez, en un terminal de %s (desde otro ordenador: ssh %s):",
	},
	"session.untrusted.cmd": {
		en: "  cd %s && claude",
		es: "  cd %s && claude",
	},
	"session.untrusted.after": {
		en: "accept the question, close that Claude with /exit, and ask for the session again.",
		es: "acepta la pregunta, cierra ese Claude con /exit y vuelve a pedir la sesión.",
	},
	"session.served": {
		en: "%s is already served by another remote session (the space's service, or another task's), and Claude allows one per folder. Give this task another folder with --folder DIR, for example a git worktree of the repository.",
		es: "%s ya la sirve otra sesión remota (el servicio del espacio u otra tarea), y Claude solo admite una por carpeta. Dale otra carpeta a esta tarea con --folder DIR, por ejemplo un worktree de git del repositorio.",
	},
	"session.running": {
		en: "%s already has its session open. To open it again: augmented-brain session stop %s",
		es: "%s ya tiene su sesión abierta. Para abrirla de nuevo: augmented-brain session stop %s",
	},
	"session.stopped": {
		en: "Closed the session of %s.",
		es: "Cerrada la sesión de %s.",
	},
	"session.list.none": {
		en: "No task sessions open.",
		es: "No hay sesiones de tareas abiertas.",
	},
	"session.list.line": {
		en: "  %s · %s · %s",
		es: "  %s · %s · %s",
	},
	// SessionStart, in a session opened by session start.
	"hook.launched": {
		en: "This session was opened for task %s (`augmented-brain session start`). Before anything else, read it with task_get, check the real state of what it touches, and tell the user in two or three lines where it stands and what its next step is.",
		es: "Esta sesión se abrió para la tarea %s (`augmented-brain session start`). Antes de nada, léela con task_get, comprueba el estado real de lo que toca y dile al usuario en dos o tres líneas cómo está y cuál es su siguiente paso.",
	},
}
