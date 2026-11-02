package i18n

// The panel's pages.
var panelMessages = map[string]msg{
	"panel.tasks": {
		en: "Tasks",
		es: "Tareas",
	},
	"panel.reminders": {
		en: "Reminders",
		es: "Recordatorios",
	},
	"panel.inbox": {
		en: "Inbox",
		es: "Buzón",
	},
	"panel.areas": {
		en: "Areas",
		es: "Áreas",
	},
	"panel.knowledge": {
		en: "Knowledge",
		es: "Conocimiento",
	},
	"panel.rules": {
		en: "Rules",
		es: "Reglas",
	},
	"panel.people": {
		en: "People",
		es: "Personas",
	},
	"panel.profile": {
		en: "Profile",
		es: "Perfil",
	},
	"panel.space": {
		en: "Health",
		es: "Salud",
	},
	"panel.space.select": {
		en: "Space",
		es: "Espacio",
	},
	"panel.profile.all": {
		en: "Shared by every space",
		es: "Común a todos los espacios",
	},
	"panel.go": {
		en: "Go",
		es: "Ir",
	},
	"panel.saved": {
		en: "Saved.",
		es: "Guardado.",
	},
	"panel.problems": {
		en: "%d files of this space are not valid notes; nothing writes over them until they are fixed.",
		es: "%d ficheros de este espacio no son notas válidas; nada los sobrescribe hasta que se arreglen.",
	},
	"panel.see": {
		en: "See which",
		es: "Ver cuáles",
	},
	"panel.footer": {
		en: "Only on this computer. It stops by itself after 30 minutes without use.",
		es: "Solo en este ordenador. Se para solo tras 30 minutos sin uso.",
	},
	"panel.stop": {
		en: "Stop the panel",
		es: "Parar el panel",
	},
	"panel.waits.you": {
		en: "waiting on you",
		es: "te espera a ti",
	},
	"panel.waits": {
		en: "Waits on",
		es: "Espera a",
	},
	"panel.next": {
		en: "Next step",
		es: "Siguiente paso",
	},
	"panel.tasks.none": {
		en: "No live tasks. Open one below, or ask your assistant.",
		es: "No hay tareas vivas. Abre una abajo, o pídesela a tu asistente.",
	},
	"panel.task.new": {
		en: "New task",
		es: "Tarea nueva",
	},
	"panel.f.title": {
		en: "Title",
		es: "Título",
	},
	"panel.f.summary": {
		en: "Summary (one line)",
		es: "Resumen (una línea)",
	},
	"panel.f.area": {
		en: "Area",
		es: "Área",
	},
	"panel.task.open": {
		en: "Open the task",
		es: "Abrir la tarea",
	},
	"panel.tasks.someday": {
		en: "Someday",
		es: "Algún día",
	},
	"panel.tasks.closed": {
		en: "Closed",
		es: "Cerradas",
	},
	"panel.f.status": {
		en: "Status",
		es: "Estado",
	},
	"panel.f.dates": {
		en: "Dates",
		es: "Fechas",
	},
	"panel.dates": {
		en: "created %s, last changed %s",
		es: "creada el %s, último cambio el %s",
	},
	"panel.f.id": {
		en: "Id",
		es: "Id",
	},
	"panel.watch": {
		en: "Watch dates",
		es: "Fechas para revisar",
	},
	"panel.done": {
		en: "Done",
		es: "Hecho",
	},
	"panel.steps": {
		en: "Steps",
		es: "Pasos",
	},
	"panel.steps.count": {
		en: "%d/%d steps",
		es: "%d/%d pasos",
	},
	"panel.steps.hint": {
		en: "The small parts of the task, ticked as you go. They are the checkboxes of its Plan.",
		es: "Las partes pequeñas de la tarea, que se marcan según se hacen. Son las casillas de su Plan.",
	},
	"panel.step.add": {
		en: "Add a step",
		es: "Añadir un paso",
	},
	"panel.step.tick": {
		en: "Mark as done",
		es: "Marcar como hecho",
	},
	"panel.step.untick": {
		en: "Mark as not done",
		es: "Marcar como no hecho",
	},
	"panel.f.step": {
		en: "Step",
		es: "Paso",
	},
	"panel.subtasks": {
		en: "Subtasks",
		es: "Subtareas",
	},
	"panel.partof": {
		en: "Part of",
		es: "Parte de",
	},
	"panel.subtask.new": {
		en: "New subtask",
		es: "Subtarea nueva",
	},
	"panel.subtask.hint": {
		en: "A subtask is a part with its own status and next step: for a part that waits on someone else or goes at its own pace. Anything smaller is a step. This task cannot close while a subtask is open.",
		es: "Una subtarea es una parte con su propio estado y siguiente paso: para una parte que espera a otra persona o lleva su propio ritmo. Lo más pequeño es un paso. Esta tarea no se puede cerrar mientras haya una subtarea abierta.",
	},
	"panel.subtask.open": {
		en: "Open the subtask",
		es: "Abrir la subtarea",
	},
	"panel.task.body": {
		en: "The whole task",
		es: "La tarea entera",
	},
	"panel.task.journal": {
		en: "Write in the journal",
		es: "Escribir en el diario",
	},
	"panel.f.entry": {
		en: "What was done, what came out, what was decided",
		es: "Qué se hizo, qué salió, qué se decidió",
	},
	"panel.add": {
		en: "Add",
		es: "Añadir",
	},
	"panel.task.edit": {
		en: "Change status, next step…",
		es: "Cambiar estado, siguiente paso…",
	},
	"panel.task.edit.hint": {
		en: "Only what you change here is saved; what changed elsewhere in the meantime stays. A change of status goes into the journal. Empty the summary or choose no area to remove them. \"Waits on\" and \"Next step\" cannot be empty: me means you, and nobody means no one.",
		es: "Solo se guarda lo que cambies aquí; lo que haya cambiado en otro sitio mientras tanto se queda. Un cambio de estado queda en el diario. Vacía el resumen o elige ninguna área para quitarlos. \"Espera a\" y \"Siguiente paso\" no pueden quedar vacíos: me eres tú, y nobody, nadie.",
	},
	"panel.you": {
		en: "you",
		es: "ti",
	},
	"panel.save": {
		en: "Save",
		es: "Guardar",
	},
	"panel.watch.add": {
		en: "Add a watch date",
		es: "Añadir una fecha para revisar",
	},
	"panel.f.date": {
		en: "Date",
		es: "Fecha",
	},
	"panel.f.what": {
		en: "What to look at",
		es: "Qué mirar",
	},
	"panel.task.park": {
		en: "Park it (pause without losing anything)",
		es: "Aparcarla (pausar sin perder nada)",
	},
	"panel.f.resume": {
		en: "Exactly how to resume it",
		es: "Cómo retomarla, exactamente",
	},
	"panel.f.next": {
		en: "Next step",
		es: "Siguiente paso",
	},
	"panel.f.essentials": {
		en: "Essentials: a few lines enough to resume it (optional; they replace the first section)",
		es: "Lo esencial: unas líneas que basten para retomarla (opcional; sustituyen la primera sección)",
	},
	"panel.f.someday": {
		en: "Someday: it will not be touched for weeks",
		es: "Algún día: no se tocará en semanas",
	},
	"panel.park": {
		en: "Park",
		es: "Aparcar",
	},
	"panel.task.close": {
		en: "Close it",
		es: "Cerrarla",
	},
	"panel.f.result": {
		en: "What was achieved",
		es: "Qué se consiguió",
	},
	"panel.task.close.hint": {
		en: "A closed task is kept, never deleted.",
		es: "Una tarea cerrada se guarda, nunca se borra.",
	},
	"panel.close": {
		en: "Close",
		es: "Cerrar",
	},
	"panel.reminders.hint": {
		en: "Watch dates of tasks and inbox reminders, overdue first, up to %d days ahead.",
		es: "Fechas para revisar de las tareas y recordatorios del buzón, primero lo atrasado, hasta %d días por delante.",
	},
	"panel.f.task": {
		en: "Task",
		es: "Tarea",
	},
	"panel.reminders.none": {
		en: "Nothing due in the next days.",
		es: "Nada pendiente en los próximos días.",
	},
	"panel.reminders.inbox": {
		en: "To be reminded of something that is not a task, capture it with a date in the",
		es: "Para que te recuerde algo que no es una tarea, apúntalo con fecha en el",
	},
	"panel.f.capture": {
		en: "An idea, a link, a loose end",
		es: "Una idea, un enlace, un cabo suelto",
	},
	"panel.f.text": {
		en: "Text",
		es: "Texto",
	},
	"panel.f.remind": {
		en: "Remind me on (optional)",
		es: "Recuérdamelo el (opcional)",
	},
	"panel.capture": {
		en: "Capture",
		es: "Apuntar",
	},
	"panel.inbox.hint": {
		en: "The weekly review goes through the inbox: each capture becomes a task, goes to an area or knowledge, or is dropped. Mark one as dealt with when it is.",
		es: "La revisión semanal repasa el buzón: cada apunte se convierte en tarea, va a un área o al conocimiento, o se descarta. Márcalo como resuelto cuando lo esté.",
	},
	"panel.inbox.done": {
		en: "Dealt with",
		es: "Resuelto",
	},
	"panel.inbox.none": {
		en: "The inbox is empty.",
		es: "El buzón está vacío.",
	},
	"panel.areas.hint": {
		en: "An area is a responsibility with no end date. It groups tasks and knowledge.",
		es: "Un área es una responsabilidad sin fecha de fin. Agrupa tareas y conocimiento.",
	},
	"panel.areas.tasks": {
		en: "%d live tasks",
		es: "%d tareas vivas",
	},
	"panel.areas.none": {
		en: "No areas yet.",
		es: "Aún no hay áreas.",
	},
	"panel.area.new": {
		en: "New area",
		es: "Área nueva",
	},
	"panel.create": {
		en: "Create",
		es: "Crear",
	},
	"panel.knowledge.hint": {
		en: "References (how something works) and lessons (what was learned), of this space and shared by every space. Your assistant writes them; here you can read them.",
		es: "Referencias (cómo funciona algo) y lecciones (qué se aprendió), de este espacio y compartidas por todos. Las escribe tu asistente; aquí puedes leerlas.",
	},
	"panel.knowledge.none": {
		en: "No references or lessons yet.",
		es: "Aún no hay referencias ni lecciones.",
	},
	"panel.rules.hint": {
		en: "How your assistant must work: the ones of this space, and the shared ones, which hold in every space. Your assistant loads them at the start of each session and writes them when you give it one; here you can read them. The rules that come with the program are not listed.",
		es: "Cómo debe trabajar tu asistente: las de este espacio y las compartidas, que valen en todos. Tu asistente las carga al empezar cada sesión y las escribe cuando le das una; aquí puedes leerlas. No salen las que trae el programa.",
	},
	"panel.rules.none": {
		en: "No rules of your own yet.",
		es: "Aún no hay reglas tuyas.",
	},
	"panel.shared": {
		en: "shared",
		es: "compartida",
	},
	"panel.shared.long": {
		en: "shared by every space",
		es: "compartida por todos los espacios",
	},
	"panel.people.hint": {
		en: "People you work or deal with, so a first name leads to the right person.",
		es: "Gente con la que trabajas o tratas, para que un nombre de pila lleve a la persona correcta.",
	},
	"panel.people.none": {
		en: "No people yet.",
		es: "Aún no hay personas.",
	},
	"panel.f.kind": {
		en: "Kind",
		es: "Tipo",
	},
	"panel.profile.hint": {
		en: "Who you are and how you like to work. Every space shares it, and every session starts by reading it. It is kept in %s.",
		es: "Quién eres y cómo te gusta trabajar. Lo comparten todos los espacios, y cada sesión empieza leyéndolo. Se guarda en %s.",
	},
	"panel.f.name": {
		en: "Name",
		es: "Nombre",
	},
	"panel.f.about": {
		en: "About you: what you do and work on",
		es: "Sobre ti: a qué te dedicas y en qué trabajas",
	},
	"panel.f.style": {
		en: "How you like to work with the assistant",
		es: "Cómo te gusta trabajar con el asistente",
	},
	"panel.profile.lines": {
		en: "Write short lines that start with \"- \".",
		es: "Escribe líneas cortas que empiecen por \"- \".",
	},
	"panel.space.title": {
		en: "Space %s",
		es: "Espacio %s",
	},
	"panel.space.notes": {
		en: "%d notes",
		es: "%d notas",
	},
	"panel.space.invalid": {
		en: "%d files are not valid notes",
		es: "%d ficheros no son notas válidas",
	},
	"panel.space.invalid.fix": {
		en: "Fix the header of each one (between the --- lines at the top), or ask your assistant to. `augmented-brain check` lists them too.",
		es: "Arregla la cabecera de cada uno (entre las líneas --- de arriba), o pídeselo a tu asistente. `augmented-brain check` también los lista.",
	},
	"panel.space.index.stale": {
		en: "The index MEMORY.md is out of date; it is rebuilt with the next change, or with `augmented-brain check`.",
		es: "El índice MEMORY.md está desfasado; se rehace con el próximo cambio, o con `augmented-brain check`.",
	},
	"panel.space.index.ok": {
		en: "The index MEMORY.md is up to date.",
		es: "El índice MEMORY.md está al día.",
	},
	"panel.space.backup.ok": {
		en: "Last backup: %s, fine.",
		es: "Última copia de seguridad: %s, bien.",
	},
	"panel.space.backup.failed": {
		en: "The last backup failed (%s): %s",
		es: "La última copia de seguridad falló (%s): %s",
	},
	"panel.space.backup.never": {
		en: "No backup has run yet on this computer.",
		es: "Aún no se ha hecho ninguna copia en este ordenador.",
	},
	"panel.space.backup.to": {
		en: "Backs up to %s",
		es: "Copia en %s",
	},
	"panel.space.backup.none": {
		en: "No backup: if this computer breaks, these notes are lost. Ask your assistant to set one up.",
		es: "Sin copia de seguridad: si este ordenador se rompe, estas notas se pierden. Pide a tu asistente que la prepare.",
	},
	"panel.space.review": {
		en: "Last weekly review: %s",
		es: "Última revisión semanal: %s",
	},
	"panel.space.review.never": {
		en: "No weekly review yet.",
		es: "Aún no hay ninguna revisión semanal.",
	},
	"panel.space.review.due": {
		en: "one is due",
		es: "toca hacer una",
	},
	"panel.space.memory": {
		en: "Notes folder",
		es: "Carpeta de notas",
	},
	"panel.space.folders": {
		en: "Work folders",
		es: "Carpetas de trabajo",
	},
	"panel.space.default": {
		en: "Default space",
		es: "Espacio por defecto",
	},
	"panel.yes": {
		en: "yes",
		es: "sí",
	},
	"panel.back": {
		en: "Back",
		es: "Volver",
	},
	"panel.stopped": {
		en: "Panel stopped",
		es: "Panel parado",
	},
	"panel.stopped.text": {
		en: "You can close this tab. To open it again: `augmented-brain panel`, or /brain:panel in Claude.",
		es: "Puedes cerrar esta pestaña. Para abrirlo otra vez: `augmented-brain panel`, o /brain:panel en Claude.",
	},
	"panel.err": {
		en: "That did not work",
		es: "No ha funcionado",
	},
	"panel.err.space": {
		en: "There is no space called %q.",
		es: "No hay ningún espacio llamado %q.",
	},
	"panel.err.required": {
		en: "Fill in: %s.",
		es: "Rellena: %s.",
	},
	"panel.err.status": {
		en: "%q is not a status the panel can set; park or close the task with their own forms.",
		es: "%q no es un estado que el panel pueda poner; aparca o cierra la tarea con sus propios formularios.",
	},
	"panel.err.inbox": {
		en: "That capture is no longer in the inbox.",
		es: "Ese apunte ya no está en el buzón.",
	},
	"panel.err.empty": {
		en: "%s cannot be empty while the task is open.",
		es: "%s no puede quedar vacío mientras la tarea esté abierta.",
	},
	"panel.err.note": {
		en: "There is no note %q in this space.",
		es: "No hay ninguna nota %q en este espacio.",
	},
	"panel.status.definition": {
		en: "definition",
		es: "definición",
	},
	"panel.status.analysis": {
		en: "analysis",
		es: "análisis",
	},
	"panel.status.plan": {
		en: "plan",
		es: "plan",
	},
	"panel.status.execution": {
		en: "execution",
		es: "ejecución",
	},
	"panel.status.validation": {
		en: "validation",
		es: "validación",
	},
	"panel.status.closed": {
		en: "closed",
		es: "cerrada",
	},
	"panel.status.parked": {
		en: "parked",
		es: "aparcada",
	},
	"panel.status.someday": {
		en: "someday",
		es: "algún día",
	},
	"panel.area.none": {
		en: "(none)",
		es: "(ninguna)",
	},
	"panel.kind.area": {
		en: "area",
		es: "área",
	},
	"panel.kind.reference": {
		en: "reference",
		es: "referencia",
	},
	"panel.kind.lesson": {
		en: "lesson",
		es: "lección",
	},
	"panel.kind.person": {
		en: "person",
		es: "persona",
	},
	"panel.kind.review": {
		en: "review",
		es: "revisión",
	},
	"panel.kind.rule": {
		en: "rule",
		es: "regla",
	},
	"panel.kind.profile": {
		en: "profile",
		es: "perfil",
	},
	"panel.kind.task": {
		en: "task",
		es: "tarea",
	},
	"panel.days.today": {
		en: "today",
		es: "hoy",
	},
	"panel.days.late": {
		en: "%d days late",
		es: "%d días de retraso",
	},
	"panel.days.in": {
		en: "in %d days",
		es: "dentro de %d días",
	},
	"panel.cli.started": {
		en: "The panel is open at:\n  %s",
		es: "El panel está abierto en:\n  %s",
	},
	"panel.cli.running": {
		en: "The panel is already running at:\n  %s",
		es: "El panel ya está en marcha en:\n  %s",
	},
	"panel.cli.idle": {
		en: "It works only on this computer, and stops by itself after 30 minutes without use. To stop it now: augmented-brain panel stop",
		es: "Solo funciona en este ordenador, y se para solo tras 30 minutos sin uso. Para pararlo ya: augmented-brain panel stop",
	},
	"panel.cli.nobrowser": {
		en: "Open that address in your browser.",
		es: "Abre esa dirección en tu navegador.",
	},
	"panel.cli.stopped": {
		en: "Panel stopped.",
		es: "Panel parado.",
	},
	"panel.cli.none": {
		en: "No panel is running.",
		es: "No hay ningún panel en marcha.",
	},
	"panel.sessions": {
		en: "Sessions",
		es: "Sesiones",
	},
	"panel.sessions.hint": {
		en: "The Claude sessions of this space on this computer, from the last 30 days. Their conversation stays in Claude Code's transcripts, never in your notes, until you keep part of it.",
		es: "Las sesiones de Claude de este espacio en este ordenador, de los últimos 30 días. Su conversación se queda en las transcripciones de Claude Code, nunca en tus notas, hasta que guardes una parte.",
	},
	"panel.sessions.dead": {
		en: "Did not end",
		es: "No terminaron",
	},
	"panel.sessions.dead.hint": {
		en: "Sessions closed by force, or cut off when the computer went down. What was said after the last journal entry may be missing from your notes.",
		es: "Sesiones cerradas a la fuerza, o cortadas al apagarse el ordenador. Lo dicho después de la última anotación del diario puede faltar en tus notas.",
	},
	"panel.sessions.dead.none": {
		en: "Every session ended normally.",
		es: "Todas las sesiones terminaron bien.",
	},
	"panel.sessions.live": {
		en: "Running now",
		es: "En marcha",
	},
	"panel.sessions.live.none": {
		en: "No session of this space is running.",
		es: "No hay ninguna sesión de este espacio en marcha.",
	},
	"panel.sessions.closed": {
		en: "Ended",
		es: "Terminadas",
	},
	"panel.sessions.closed.hint": {
		en: "Their transcripts take %s on disk. Deleting one frees that space, but that session can no longer be resumed.",
		es: "Sus transcripciones ocupan %s en disco. Borrar una libera ese espacio, pero esa sesión ya no se podrá retomar.",
	},
	"panel.sessions.closed.none": {
		en: "No transcript of an ended session is left.",
		es: "No queda ninguna transcripción de sesiones terminadas.",
	},
	"panel.sessions.notask": {
		en: "No task",
		es: "Sin tarea",
	},
	"panel.sessions.started": {
		en: "started %s",
		es: "empezó %s",
	},
	"panel.sessions.written": {
		en: "last written %s",
		es: "escribió por última vez %s",
	},
	"panel.sessions.said": {
		en: "What was said since %s",
		es: "Lo dicho desde %s",
	},
	"panel.sessions.said.none": {
		en: "Nothing was said after %s.",
		es: "No se dijo nada después de %s.",
	},
	"panel.sessions.recover": {
		en: "Keep it in the notes",
		es: "Guardarlo en las notas",
	},
	"panel.sessions.reviewed": {
		en: "Mark as reviewed",
		es: "Dar por revisada",
	},
	"panel.sessions.delete": {
		en: "Delete the transcript",
		es: "Borrar la transcripción",
	},
	"panel.sessions.delete.confirm": {
		en: "I understand this session can no longer be resumed (claude --resume).",
		es: "Entiendo que esta sesión ya no se podrá retomar (claude --resume).",
	},
	"panel.sessions.archive": {
		en: "# Session that did not end, started %s\n\nWhat the user and the assistant said from %s on, kept from the panel. Tool calls and their output are left out.",
		es: "# Sesión que no terminó, empezada el %s\n\nLo que dijeron el usuario y el asistente desde %s, guardado desde el panel. Sin las llamadas a herramientas ni su salida.",
	},
	"panel.err.session": {
		en: "That session is no longer waiting to be recovered.",
		es: "Esa sesión ya no está esperando a que la recuperes.",
	},
	"panel.err.confirm": {
		en: "Tick the box to confirm.",
		es: "Marca la casilla para confirmar.",
	},
	"panel.err.delete": {
		en: "That transcript cannot be deleted: only those of ended sessions, not written in the last minutes.",
		es: "Esa transcripción no se puede borrar: solo las de sesiones terminadas que no se han escrito en los últimos minutos.",
	},
	"panel.repos.missing": {
		en: "not in any work folder of this space on this computer. Was it moved, renamed or deleted? Fix it in the task's repos:, or ask your assistant to.",
		es: "no está en ninguna carpeta de trabajo de este espacio en este ordenador. ¿Se movió, cambió de nombre o se borró? Corrígelo en repos: de la tarea, o pídeselo a tu asistente.",
	},
	"panel.space.repos": {
		en: "Repositories that notes point at and that are not on this computer: %d",
		es: "Repositorios a los que apuntan las notas y que no están en este ordenador: %d",
	},
	"panel.space.repos.fix": {
		en: "They are in none of the space's work folders: moved, renamed or deleted. Nothing is changed by itself; fix repos: in each note, or ask your assistant to.",
		es: "No están en ninguna carpeta de trabajo del espacio: se movieron, cambiaron de nombre o se borraron. Nada se cambia solo; corrige repos: en cada nota, o pídeselo a tu asistente.",
	},
}
