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
