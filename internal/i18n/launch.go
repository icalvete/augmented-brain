package i18n

// `augmented-brain session`: a remote session of its own for one task, and what the
// session start tells the assistant in it.
var launchMessages = map[string]msg{
	"usage.session": {
		en: "usage: augmented-brain session start TASK [--folder DIR] | stop TASK [--when-idle] | restart SESSION [--when-idle] | restart --old | list",
		es: "uso: augmented-brain session start TAREA [--folder DIR] | stop TAREA [--when-idle] | restart SESIÓN [--when-idle] | restart --old | list",
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
		en: "%s names no repository and its space has no work folder, so I cannot tell in which folder to open it. Say it with --folder DIR.",
		es: "%s no nombra ningún repositorio y su espacio no tiene carpeta de trabajo, así que no sé en qué carpeta abrirla. Dímela con --folder DIR.",
	},
	"session.woken": {
		en: "Brought back the session \"%s\" of %s, in %s, with its whole conversation: Claude had stopped it. It is connected again in Claude Desktop and claude.ai.\nTo close it (and take it out of Claude Desktop and claude.ai): augmented-brain session stop %s",
		es: "Recuperada la sesión «%s» de %s, en %s, con toda su conversación: Claude la había parado. Vuelve a estar conectada en Claude Desktop y claude.ai.\nPara cerrarla (y quitarla de Claude Desktop y claude.ai): augmented-brain session stop %s",
	},
	"session.started": {
		en: "Opened the session \"%s\" for %s, in %s. It shows up in Claude Desktop and claude.ai within a minute, and it starts on its own: it reads its task and gets going.\nTo close it (and take it out of Claude Desktop and claude.ai): augmented-brain session stop %s",
		es: "Abierta la sesión «%s» para %s, en %s. Sale en Claude Desktop y claude.ai en menos de un minuto, y empieza sola: lee su tarea y se pone a ello.\nPara cerrarla (y quitarla de Claude Desktop y claude.ai): augmented-brain session stop %s",
	},
	// The first message of a launched session: what has it start on its own.
	"session.first": {
		en: "This session was opened for task %[1]s (`augmented-brain session start`). Before anything else, read it with task_get, check the real state of what it touches, and tell the user in two or three lines where it stands and what its next step is. Then get on with that step, within the phase the task is in, unless it waits on the user. To wait for a pull request's CI, run `augmented-brain watch pr N --task %[1]s` in the background, not a loop of your own.",
		es: "Esta sesión se abrió para la tarea %[1]s (`augmented-brain session start`). Antes de nada, léela con task_get, comprueba el estado real de lo que toca y dile al usuario en dos o tres líneas cómo está y cuál es su siguiente paso. Después ponte con ese paso, dentro de la fase en la que está la tarea, salvo que espere al usuario. Para esperar la CI de una pull request, lanza en segundo plano `augmented-brain watch pr N --task %[1]s`, no un bucle propio.",
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
	"session.closing": {
		en: "The task's own session closes when it finishes this answer, and leaves Claude Desktop and claude.ai; its worktree goes with it when nothing would be lost.",
		es: "La sesión propia de la tarea se cierra cuando termine esta respuesta, y sale de Claude Desktop y claude.ai; su worktree se va con ella si no se pierde nada.",
	},
	"session.closed.retired": {
		en: "Closed the task's own session, which Claude had stopped: it leaves Claude Desktop and claude.ai.",
		es: "Cerrada la sesión propia de la tarea, que Claude había parado: sale de Claude Desktop y claude.ai.",
	},
	"session.worktree.closed.kept": {
		en: "The task's worktree %s stays: it has %s. Push or drop that work, then remove it.",
		es: "El worktree de la tarea %s se queda: tiene %s. Sube o descarta ese trabajo, y después quítalo.",
	},
	"session.worktree.why.changes": {
		en: "changes not committed",
		es: "cambios sin commitear",
	},
	"session.worktree.why.in-use": {
		en: "this session works in it; once you leave it, git worktree remove takes it",
		es: "esta sesión trabaja en él; cuando salgas de él, lo quita git worktree remove",
	},
	"task.worktree.made": {
		en: "Made the task's worktree %s, on branch %s from %s. Work there; closing the task removes it when it holds nothing not committed or not pushed.",
		es: "Creado el worktree de la tarea %s, en la rama %s desde %s. Trabaja ahí; cerrar la tarea lo quita si no tiene nada sin commitear ni sin subir.",
	},
	"task.worktree.again": {
		en: "The task's worktree %s is there already, on branch %s.",
		es: "El worktree de la tarea %s ya existe, en la rama %s.",
	},
	"task.worktree.noted": {
		en: "Noted %s for %s: closing the task removes it when it holds nothing not committed or not pushed (its branch stays).",
		es: "Apuntado %s para %s: cerrar la tarea lo quita si no tiene nada sin commitear ni sin subir (su rama se queda).",
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
		en: "Claude does not trust %s yet. A task's session opens only in a folder that accepted Claude's question itself (trusting a folder above it is not enough), so a repository's own settings and hooks never run unasked. Once, in a terminal on %s (from another computer: ssh %s):",
		es: "Claude aún no confía en %s. La sesión de una tarea solo se abre en una carpeta que ha aceptado ella misma la pregunta de Claude (que confíe en una carpeta de arriba no basta), para que los ajustes y hooks de un repositorio nunca corran sin preguntar. Una vez, en un terminal de %s (desde otro ordenador: ssh %s):",
	},
	"session.untrusted.cmd": {
		en: "  cd %s && claude",
		es: "  cd %s && claude",
	},
	"session.untrusted.after": {
		en: "accept the question, close that Claude with /exit, and ask for the session again.",
		es: "acepta la pregunta, cierra ese Claude con /exit y vuelve a pedir la sesión.",
	},
	"session.first.report": {
		en: "Another Claude session asked for you. When you stop (the task closed or parked, or waiting on the user), tell it in one line where the task stands, with SendMessage to %s. If it no longer answers, carry on.",
		es: "Te ha pedido otra sesión de Claude. Cuando pares (la tarea cerrada o aparcada, o esperando al usuario), dile en una línea cómo queda la tarea, con SendMessage a %s. Si ya no contesta, sigue.",
	},
	"session.served": {
		en: "Another task's session already works in %s. Two tasks in one folder would step on each other's changes: give this one another folder with --folder DIR, for example a git worktree of the repository.",
		es: "En %s ya trabaja la sesión de otra tarea. Dos tareas en una carpeta se pisarían los cambios: dale a esta otra carpeta con --folder DIR, por ejemplo un worktree de git del repositorio.",
	},
	"session.running": {
		en: "%s already has its session open. To open it again: augmented-brain session stop %s",
		es: "%s ya tiene su sesión abierta. Para abrirla de nuevo: augmented-brain session stop %s",
	},
	"session.stopped": {
		en: "Closed the session of %s.",
		es: "Cerrada la sesión de %s.",
	},
	"session.left": {
		en: "Claude could not take the session of %s out of Claude Desktop and claude.ai: it no longer runs and is no longer the task's. To take it out later: claude rm %s",
		es: "Claude no pudo quitar la sesión de %s de Claude Desktop y claude.ai: ya no corre y ya no es de la tarea. Para quitarla más tarde: claude rm %s",
	},
	"session.none": {
		en: "%s has no session open: nothing to close.",
		es: "%s no tiene ninguna sesión abierta: no hay nada que cerrar.",
	},
	"session.busy": {
		en: "The session of %s kept answering, or waiting for a permission, for 30 minutes: it stays open. Close it with augmented-brain session stop once it is done.",
		es: "La sesión de %s siguió respondiendo, o esperando un permiso, durante 30 minutos: se queda abierta. Ciérrala con augmented-brain session stop cuando acabe.",
	},
	"session.list.retired": {
		en: "retired (Claude stopped it, after hours unused or with its background service; claude.ai shows it disconnected). To bring it back with its conversation: augmented-brain session start %s",
		es: "retirada (Claude la paró, tras horas sin uso o con su servicio de fondo; claude.ai la enseña desconectada). Para traerla de vuelta con su conversación: augmented-brain session start %s",
	},
	"session.list.failed": {
		en: "failed: its last answer was an error (%s). Write to it in Claude Desktop or claude.ai to try again, or open it again: augmented-brain session start %s",
		es: "fallida: su última respuesta fue un error (%s). Escríbele en Claude Desktop o claude.ai para que lo intente otra vez, o ábrela de nuevo: augmented-brain session start %s",
	},
	"session.list.failed.login": {
		en: "failed: Claude Code's login expired (%s). Log in again with claude auth login, then open it again: augmented-brain session start %s",
		es: "fallida: caducó el login de Claude Code (%s). Vuelve a entrar con claude auth login y ábrela de nuevo: augmented-brain session start %s",
	},
	"session.loggedout": {
		en: "Claude Code is not logged in, or its login has expired: the session could not work, so I did not open it. Log in again with:\n  claude auth login\nThen open it again:\n  augmented-brain session start %s",
		es: "Claude Code no tiene la sesión iniciada, o le ha caducado: la sesión no podría trabajar, así que no la abro. Vuelve a entrar con:\n  claude auth login\nY ábrela de nuevo:\n  augmented-brain session start %s",
	},
	"session.firstturn.login": {
		en: "The session for %s started, but its first answer was an error: %s\nClaude Code's login has expired. I closed the session. Log in again with:\n  claude auth login\nThen open it again:\n  augmented-brain session start %s",
		es: "La sesión de %s arrancó, pero su primera respuesta fue un error: %s\nEl login de Claude Code ha caducado. He cerrado la sesión. Vuelve a entrar con:\n  claude auth login\nY ábrela de nuevo:\n  augmented-brain session start %s",
	},
	"session.firstturn": {
		en: "The session for %s started, but its first answer was an error: %s\nI closed it. Try again in a while:\n  augmented-brain session start %s",
		es: "La sesión de %s arrancó, pero su primera respuesta fue un error: %s\nLa he cerrado. Prueba otra vez dentro de un rato:\n  augmented-brain session start %s",
	},
	"session.list.none": {
		en: "No task sessions open.",
		es: "No hay sesiones de tareas abiertas.",
	},
	"session.list.line": {
		en: "  %s · %s · %s",
		es: "  %s · %s · %s",
	},
	// SessionStart, in every session on Linux: the launcher exists and how to use it.
	"hook.launcher": {
		en: "To give a task a session of its own, use the launcher, here or from a remote session: `augmented-brain session start <task>` (Bash). It opens the session in the task's own folder (a worktree of its own for a bare repository), and it shows up in Claude Desktop and claude.ai within a minute. `--folder <dir>` says the folder when the task names none; `augmented-brain session list` shows the ones open; `augmented-brain session stop <task>` closes one, and closing the task (task_close) closes its session too, once it finishes its answer: do not ask the user to run stop. Use it rather than Claude Desktop's start_session, which does not work from a remote session.",
		es: "Para dar a una tarea una sesión propia, usa el lanzador, aquí o desde una sesión remota: `augmented-brain session start <tarea>` (Bash). Abre la sesión en la carpeta de la tarea (en un worktree propio si es un repositorio bare) y sale en Claude Desktop y claude.ai en menos de un minuto. `--folder <dir>` dice la carpeta cuando la tarea no nombra ninguna; `augmented-brain session list` enseña las abiertas; `augmented-brain session stop <tarea>` cierra una, y cerrar la tarea (task_close) también cierra su sesión, cuando termina su respuesta: no le pidas al usuario que lance stop. Úsalo en vez de start_session de Claude Desktop, que no funciona desde una sesión remota.",
	},
	// session restart: a session goes on with the same conversation and the program of
	// now (internal/remote/restart.go).
	"restart.done": {
		en: "✓ Restarted \"%s\": it goes on with the same conversation and the program of now.",
		es: "✓ Reiniciada «%s»: sigue con la misma conversación y el programa de ahora.",
	},
	"restart.busy": {
		en: "\"%s\" is answering (or waiting for a permission): left as it is. To restart it once it finishes: augmented-brain session restart %s --when-idle",
		es: "«%s» está respondiendo (o esperando un permiso): se queda como está. Para reiniciarla cuando acabe: augmented-brain session restart %s --when-idle",
	},
	"restart.self": {
		en: "\"%s\" is the session this command runs in: it cannot restart in the middle of its own answer. To restart it once the answer ends: augmented-brain session restart %s --when-idle",
		es: "«%s» es la sesión desde la que se lanza esta orden: no puede reiniciarse a mitad de su propia respuesta. Para reiniciarla cuando acabe la respuesta: augmented-brain session restart %s --when-idle",
	},
	"restart.shared": {
		en: "\"%s\" shares its remote service (%s) with a session that is answering: left as it is. Try again when it finishes.",
		es: "«%s» comparte su servicio remoto (%s) con una sesión que está respondiendo: se queda como está. Prueba otra vez cuando acabe.",
	},
	"restart.server": {
		en: "✓ Updated \"%s\" (Claude Desktop or a terminal) without closing it: its augmented brain starts again, with the program of now, the next time it uses it.",
		es: "✓ Puesta al día «%s» (Claude Desktop o un terminal) sin cerrarla: su augmented brain arranca otra vez, con el programa de ahora, la próxima vez que lo use.",
	},
	"restart.unsupported": {
		en: "\"%s\" is a session of Claude Desktop or a terminal, and on this system augmented brain cannot tell which of its servers is that session's. Close it and open it again when it suits you.",
		es: "«%s» es una sesión de Claude Desktop o de un terminal, y en este sistema augmented brain no sabe cuál de sus servidores es el de esa sesión. Ciérrala y ábrela otra vez cuando te venga bien.",
	},
	"restart.gone": {
		en: "\"%s\" no longer runs: when it is opened again, it starts with the program of now.",
		es: "«%s» ya no está en marcha: cuando se vuelva a abrir, arranca con el programa de ahora.",
	},
	"restart.failed": {
		en: "Could not restart \"%s\": %v",
		es: "No he podido reiniciar «%s»: %v",
	},
	"restart.stillbusy": {
		en: "\"%s\" was still answering after %d minutes: left as it is.",
		es: "«%s» seguía respondiendo al cabo de %d minutos: se queda como está.",
	},
	"restart.none": {
		en: "✓ No Claude session is still running an earlier augmented brain.",
		es: "✓ Ninguna sesión de Claude sigue con un augmented brain anterior.",
	},
	"restart.old": {
		en: "%d Claude sessions still run an earlier augmented brain. Restarting the idle ones, one at a time:",
		es: "%d sesiones de Claude siguen con un augmented brain anterior. Reinicio las que están quietas, una a una:",
	},
	"restart.unknown": {
		en: "%d more servers of an earlier augmented brain are running, and I cannot tell which session they belong to: close and open your Claude sessions when it suits you.",
		es: "Hay %d servidores más de un augmented brain anterior y no sé de qué sesión son: cierra y abre tus sesiones de Claude cuando te venga bien.",
	},
	"restart.notfound": {
		en: "no running Claude session answers to %q: give a task with a session, a session's id or its name (`claude agents` lists them).",
		es: "ninguna sesión de Claude en marcha responde a %q: da una tarea con sesión, el id de una sesión o su nombre (`claude agents` las lista).",
	},
	"restart.ambiguous": {
		en: "more than one session answers to %q: give its id (`claude agents` lists them).",
		es: "más de una sesión responde a %q: da su id (`claude agents` las lista).",
	},
	"restart.waiting": {
		en: "\"%s\" will restart once it finishes its answer (it waits at most %d minutes). How it went: %s",
		es: "«%s» se reiniciará cuando acabe su respuesta (espera como mucho %d minutos). Cómo fue: %s",
	},
}
