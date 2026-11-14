package i18n

// The panel's pages.
var panelMessages = map[string]msg{
	"panel.backup": {
		en: "Backup",
		es: "Copia de seguridad",
	},
	"panel.backup.hint": {
		en: "A copy of each space's notes in a private repository (a folder kept on a server, such as GitHub), so they survive if this computer breaks. It runs by itself every time a session ends. Your configuration's versions are not in it: they stay on this computer.",
		es: "Una copia de las notas de cada espacio en un repositorio privado (una carpeta guardada en un servidor, como GitHub), para que sobrevivan si este ordenador se rompe. Se hace sola cada vez que termina una sesión. Las versiones de tu configuración no van en ella: se quedan en este ordenador.",
	},
	"panel.backup.remote": {
		en: "You are on the panel from another device: here you can see the backups; making a copy or setting one up is only from this computer's panel.",
		es: "Estás en el panel desde otro dispositivo: aquí puedes ver las copias; hacer una o configurarla solo se puede desde el panel de este ordenador.",
	},
	"panel.backup.none.count.1": {
		en: "1 space has no backup: if this computer breaks, its notes are lost.",
		es: "1 espacio no tiene copia de seguridad: si este ordenador se rompe, sus notas se pierden.",
	},
	"panel.backup.none.count.n": {
		en: "%d spaces have no backup: if this computer breaks, their notes are lost.",
		es: "%d espacios no tienen copia de seguridad: si este ordenador se rompe, sus notas se pierden.",
	},
	"panel.backup.none": {
		en: "No backup yet.",
		es: "Aún sin copia de seguridad.",
	},
	"panel.backup.started": {
		en: "A copy has started. It takes a few seconds: reload the page to see how it went.",
		es: "Ha empezado una copia. Tarda unos segundos: recarga la página para ver cómo fue.",
	},
	"panel.backup.profile": {
		en: "This backup also carries your profile, the names of your secrets and the notes shared by every space.",
		es: "Esta copia lleva también tu perfil, los nombres de tus secretos y las notas compartidas por todos los espacios.",
	},
	"panel.backup.copies": {
		en: "Last copies",
		es: "Últimas copias",
	},
	"panel.backup.from": {
		en: "from %s",
		es: "desde %s",
	},
	"panel.backup.copies.none": {
		en: "No copy is in the history yet.",
		es: "Aún no hay ninguna copia en el historial.",
	},
	"panel.backup.run": {
		en: "Back up now",
		es: "Hacer una copia ahora",
	},
	"panel.backup.restore": {
		en: "Bring the notes back on another computer",
		es: "Recuperar las notas en otro ordenador",
	},
	"panel.backup.restore.hint": {
		en: "On a new computer, install augmented brain, then run this in a terminal. Restoring is never done from here: it is for a computer that does not have these notes yet.",
		es: "En un ordenador nuevo, instala augmented brain y después ejecuta esto en una terminal. Recuperar nunca se hace desde aquí: es para un ordenador que aún no tiene estas notas.",
	},
	"panel.backup.github.hint": {
		en: "Makes a private repository named augmented-brain-%s in your GitHub account, with the GitHub command line (gh) already logged in, and makes the first copy.",
		es: "Crea un repositorio privado llamado augmented-brain-%s en tu cuenta de GitHub, con la línea de comandos de GitHub (gh) ya conectada, y hace la primera copia.",
	},
	"panel.backup.github": {
		en: "Set it up on my GitHub",
		es: "Configurarla en mi GitHub",
	},
	"panel.backup.own": {
		en: "I already have a private repository",
		es: "Ya tengo un repositorio privado",
	},
	"panel.backup.own.hint": {
		en: "Its address, as git uses it. It must be private, and empty or already a backup of this space.",
		es: "Su dirección, como la usa git. Tiene que ser privado, y estar vacío o ser ya una copia de este espacio.",
	},
	"panel.backup.url": {
		en: "Address of the repository",
		es: "Dirección del repositorio",
	},
	"panel.backup.connect": {
		en: "Connect and make the first copy",
		es: "Conectar y hacer la primera copia",
	},
	"panel.backup.localonly": {
		en: "Making a copy or setting one up is only from the panel on this computer.",
		es: "Hacer una copia o configurarla solo se puede desde el panel de este ordenador.",
	},
	"panel.backup.err.none": {
		en: "This space has no backup yet: set one up first.",
		es: "Este espacio aún no tiene copia de seguridad: configúrala primero.",
	},
	"panel.backup.err.has": {
		en: "This space already backs up to %s. To move it somewhere else, run `augmented-brain backup setup <address>` in a terminal, in a folder of the space.",
		es: "Este espacio ya tiene su copia en %s. Para llevarla a otro sitio, ejecuta `augmented-brain backup setup <dirección>` en una terminal, en una carpeta del espacio.",
	},
	"panel.backup.what": {
		en: "What it changed",
		es: "Qué cambió",
	},
	"panel.backup.nochange": {
		en: "No note changed in this copy.",
		es: "En esta copia no cambió ninguna nota.",
	},
	"panel.backup.more": {
		en: "And %d more notes.",
		es: "Y %d notas más.",
	},
	"panel.backup.binary": {
		en: "(not text: no lines to show)",
		es: "(no es texto: no hay líneas que enseñar)",
	},
	"panel.backup.wentback": {
		en: "went back to the copy of %s",
		es: "volvió a la copia del %s",
	},
	"panel.backup.back": {
		en: "Go back to this copy",
		es: "Volver a esta copia",
	},
	"panel.backup.back.hint": {
		en: "Your notes in space %s become again what they were in the copy of %s. Before that, what you have now is kept as a new copy: to undo, come back here and go back to that one. Your configuration does not change.",
		es: "Tus notas del espacio %s vuelven a ser las de la copia del %s. Antes, lo que tienes ahora se guarda como una copia nueva: para deshacerlo, vuelve aquí y vuelve a esa. Tu configuración no cambia.",
	},
	"panel.backup.back.changes": {
		en: "What would change in your notes",
		es: "Qué cambiaría en tus notas",
	},
	"panel.backup.back.same": {
		en: "Your notes are already as in this copy: nothing would change.",
		es: "Tus notas ya están como en esta copia: no cambiaría nada.",
	},
	"panel.backup.back.confirm": {
		en: "I have looked at what changes, go back to this copy",
		es: "He mirado qué cambia, volver a esta copia",
	},
	"panel.backup.back.type": {
		en: "Type the name of the space to confirm",
		es: "Escribe el nombre del espacio para confirmar",
	},
	"panel.backup.back.name": {
		en: "To confirm, type the name of the space: %s.",
		es: "Para confirmar, escribe el nombre del espacio: %s.",
	},
	"panel.backup.back.remote": {
		en: "You are on the panel from another device: here you can see what would change; going back is only from this computer's panel.",
		es: "Estás en el panel desde otro dispositivo: aquí puedes ver qué cambiaría; volver solo se puede desde el panel de este ordenador.",
	},
	"panel.backup.back.localonly": {
		en: "Going back to a copy is only from the panel on this computer.",
		es: "Volver a una copia solo se puede desde el panel de este ordenador.",
	},
	"panel.backup.back.gone": {
		en: "That copy is not among the copies of this space any more. Open the Backup page again.",
		es: "Esa copia ya no está entre las copias de este espacio. Abre otra vez la página Copia de seguridad.",
	},
	"panel.backup.back.meanwhile": {
		en: "Your notes changed while going back was getting ready (your assistant may be writing). Nothing changed: try again in a moment.",
		es: "Tus notas cambiaron mientras se preparaba la vuelta (puede que tu asistente esté escribiendo). No ha cambiado nada: vuelve a probar dentro de un momento.",
	},
	"panel.backup.back.notvalid": {
		en: "That copy holds notes this program cannot read (%s). Nothing changed.",
		es: "Esa copia tiene notas que este programa no puede leer (%s). No ha cambiado nada.",
	},
	"panel.backup.back.secrets": {
		en: "What you have now could not be kept as a copy first, because these files hold secrets: %s. Remove them and try again. Nothing changed.",
		es: "Lo que tienes ahora no se pudo guardar antes como copia, porque estos ficheros tienen secretos: %s. Quítalos y vuelve a probar. No ha cambiado nada.",
	},
	"panel.backup.back.done": {
		en: "Your notes are back as in the copy of %s. What you had is kept as the copy before; the result is being sent to the backup.",
		es: "Tus notas vuelven a estar como en la copia del %s. Lo que tenías queda guardado en la copia anterior; el resultado se está enviando a la copia de seguridad.",
	},
	"panel.backup.err.gh": {
		en: "The GitHub command line (gh) is not installed, or not logged in. Install it from https://cli.github.com, run `gh auth login` in a terminal, and try again. Or ask your assistant: it guides you step by step.",
		es: "La línea de comandos de GitHub (gh) no está instalada, o no está conectada. Instálala desde https://cli.github.com, ejecuta `gh auth login` en una terminal y vuelve a probar. O pídeselo a tu asistente: te guía paso a paso.",
	},
	"panel.intro.guide": {
		en: "More in the guide ↗",
		es: "Más en la guía ↗",
	},
	"panel.intro.hide": {
		en: "Hide this help",
		es: "Ocultar esta ayuda",
	},
	"panel.intro.hideall": {
		en: "Hide it on every page",
		es: "Ocultarla en todas las páginas",
	},
	"panel.intro.show": {
		en: "What is this page?",
		es: "¿Qué es esta página?",
	},
	"panel.intro.showall": {
		en: "Show the help on every page",
		es: "Mostrar la ayuda en todas las páginas",
	},
	"panel.tasks.intro": {
		en: "Every piece of work is a task. Each goes through phases (definition, analysis, plan, execution, validation, closed), and says who it waits on and what comes next. Your assistant keeps them up to date as it works; here you see them all, filter them and change them.",
		es: "Cada trabajo es una tarea. Pasa por fases (definición, análisis, plan, ejecución, validación, cerrada) y dice a quién espera y qué viene después. Tu asistente las mantiene al día mientras trabaja; aquí las ves todas, las filtras y las cambias.",
	},
	"panel.task.intro": {
		en: "One task: its phase, who it waits on, its next step, its plan and its journal (what was done and decided, newest first). The phases are approved by you; the rest, your assistant writes as it works.",
		es: "Una tarea: su fase, a quién espera, su siguiente paso, su plan y su diario (lo hecho y lo decidido, lo último primero). Las fases las apruebas tú; lo demás lo escribe tu asistente mientras trabaja.",
	},
	"panel.note.intro": {
		en: "One note of your memory: a Markdown file in your notes folder, yours to keep. Your assistant reads it when the work needs it.",
		es: "Una nota de tu memoria: un fichero Markdown en tu carpeta de notas, tuyo. Tu asistente la lee cuando el trabajo la necesita.",
	},
	"panel.search.intro": {
		en: "Looks for words in every note of this space and in the shared ones: tasks, areas, knowledge, rules and people.",
		es: "Busca palabras en todas las notas de este espacio y en las compartidas: tareas, áreas, conocimiento, reglas y personas.",
	},
	"panel.space.intro": {
		en: "Whether this space is well: its notes can be read, its index is up to date, its backup works and the weekly review is not overdue. Red means something to fix, with how.",
		es: "Si este espacio está bien: sus notas se leen, su índice está al día, su copia de seguridad funciona y la revisión semanal no se ha pasado. En rojo, lo que hay que arreglar, con cómo.",
	},
	"panel.guard": {
		en: "Guard",
		es: "Guardia",
	},
	"panel.versions": {
		en: "Versions",
		es: "Versiones",
	},
	"panel.versions.hint": {
		en: "The last %d versions of your configuration (spaces, guard, remote access, preferred model, profile, and the names of your secrets, never their values), kept only on this computer and never in the backup. A new one is kept every time it changes, from here, a command, the assistant or by hand. Your notes are not here: they have their own history in the backup.",
		es: "Las %d últimas versiones de tu configuración (espacios, guardia, acceso remoto, modelo preferido, perfil y los nombres de tus secretos, nunca su valor), guardadas solo en este ordenador y nunca en la copia de seguridad. Se guarda una cada vez que cambia: desde aquí, con un comando, el asistente o a mano. Tus notas no están aquí: tienen su propia historia en la copia de seguridad.",
	},
	"panel.versions.remote": {
		en: "You are on the panel from another device: here you can see the versions; going back to one is only from this computer's panel.",
		es: "Estás en el panel desde otro dispositivo: aquí puedes ver las versiones; volver a una solo se puede desde el panel de este ordenador.",
	},
	"panel.versions.changed": {
		en: "The configuration changed after the newest version (by hand, or by an earlier program). It is kept as a version with the next change, or before going back.",
		es: "La configuración ha cambiado después de la versión más reciente (a mano, o con un programa anterior). Se guarda como versión con el próximo cambio, o antes de volver a otra.",
	},
	"panel.versions.current": {
		en: "current",
		es: "la actual",
	},
	"panel.versions.what": {
		en: "What it changed",
		es: "Qué cambió",
	},
	"panel.versions.oldest": {
		en: "The oldest kept: there is nothing before it to compare it with.",
		es: "La más antigua que se guarda: no hay nada antes con qué compararla.",
	},
	"panel.versions.filegone": {
		en: "(removed)",
		es: "(se quitó)",
	},
	"panel.versions.filenew": {
		en: "(created)",
		es: "(se creó)",
	},
	"panel.versions.restore": {
		en: "Go back to this one",
		es: "Volver a esta",
	},
	"panel.versions.restore.hint": {
		en: "Your whole configuration becomes this version again: spaces, guard, remote access, profile. The one you have now is kept first, so you can come back to it.",
		es: "Toda tu configuración vuelve a ser esta versión: espacios, guardia, acceso remoto, perfil. La que tienes ahora se guarda antes, así que puedes volver a ella.",
	},
	"panel.versions.restore.guard": {
		en: "Careful: this also changes what the guard lets your assistant do. Going back would change these lists:",
		es: "Ojo: esto también cambia lo que el guardia deja hacer a tu asistente. Volver cambiaría estas listas:",
	},
	"panel.versions.restore.confirm": {
		en: "I have looked at what changes, go back to this version",
		es: "He mirado qué cambia, volver a esta versión",
	},
	"panel.versions.none": {
		en: "No versions yet: one is kept the first time the configuration changes.",
		es: "Aún no hay versiones: se guarda una la primera vez que cambia la configuración.",
	},
	"panel.versions.localonly": {
		en: "Going back to a version is only from this computer's panel.",
		es: "Volver a una versión solo se puede desde el panel de este ordenador.",
	},
	"panel.versions.gone": {
		en: "That version is no longer kept: reload the page.",
		es: "Esa versión ya no se guarda: recarga la página.",
	},
	"panel.guard.hint": {
		en: "What the guard allows, asks about and never allows. Only you change it, here: the assistant cannot read this panel's key or open its pages. Every change keeps your comments in the files.",
		es: "Lo que el guardia deja pasar, lo que pregunta y lo que nunca permite. Solo lo cambias tú, aquí: el asistente no puede leer la clave de este panel ni abrir sus páginas. Cada cambio respeta tus comentarios en los ficheros.",
	},
	"panel.guard.problem": {
		en: "The file guard.toml cannot be read, so the guard asks about every commit and push until it is fixed: %s",
		es: "El fichero guard.toml no se puede leer, así que el guardia pregunta por cada commit y push hasta que se arregle: %s",
	},
	"panel.guard.changed": {
		en: "The guard's lists are not the ones you last saw: something changed them outside this page (you by hand, or an assistant). Until you accept them, a commit, push or merge that the lists allow asks first.",
		es: "Las listas del guardia no son las que viste por última vez: algo las cambió fuera de esta página (tú a mano, o un asistente). Hasta que las aceptes, un commit, push o merge que las listas permiten pregunta antes.",
	},
	"panel.task.reopened.panel": {
		en: "from the panel",
		es: "desde el panel",
	},
	"panel.guard.changes": {
		en: "What changed since you last saw them (+ added, - removed):",
		es: "Qué ha cambiado desde la última vez que las viste (+ añadido, - quitado):",
	},
	"panel.guard.changes.unknown": {
		en: "The lists you last saw are not among the kept versions of the configuration, so what changed cannot be shown: read the lists below before accepting.",
		es: "Las listas que viste por última vez no están entre las versiones guardadas de la configuración, así que no se puede enseñar qué cambió: lee las listas de abajo antes de aceptar.",
	},
	"panel.guard.allowed.warn": {
		en: "Careful: in these repositories the assistant commits and pushes without asking you each time. Add only repositories where a mistake can be undone, and where pushing publishes nothing you would regret.",
		es: "Cuidado: en estos repositorios el asistente hace commit y push sin preguntarte cada vez. Añade solo repositorios donde un error se pueda deshacer y donde un push no publique nada de lo que te arrepientas.",
	},
	"panel.guard.allowed.confirm": {
		en: "I understand the assistant will commit and push here without asking",
		es: "Entiendo que el asistente hará commit y push aquí sin preguntar",
	},
	"panel.guard.err.allowedconfirm": {
		en: "to let the assistant commit and push without asking, tick the box that says you understand",
		es: "para dejar que el asistente haga commit y push sin preguntar, marca la casilla que dice que lo entiendes",
	},
	"panel.guard.untouchable.removewarn": {
		en: "Once removed, the other lists decide again for this repository: if it is in one where the assistant may commit and push, it will.",
		es: "Al quitarlo, las demás listas vuelven a decidir sobre este repositorio: si está en una donde el asistente puede hacer commit y push, lo hará.",
	},
	"panel.guard.root.warn": {
		en: "Careful: until that date, on that machine, the assistant may run commands as root, the most powerful user, after you confirm each one. Keep the date close.",
		es: "Cuidado: hasta esa fecha, en esa máquina, el asistente podrá lanzar comandos como root, el usuario con más poder, después de que confirmes cada uno. Pon una fecha cercana.",
	},
	"panel.ticket.used": {
		en: "This link was already used",
		es: "Este enlace ya se usó",
	},
	"panel.ticket.used.text": {
		en: "Each link to the panel opens it once. If you did not open it yourself a moment ago, something else did: tell the assistant to stop, and look at the Guard page. To open the panel again, run /brain:panel.",
		es: "Cada enlace al panel lo abre una sola vez. Si no lo has abierto tú hace un momento, lo ha abierto otra cosa: dile al asistente que pare y mira la página Guardia. Para abrir el panel otra vez, usa /brain:panel.",
	},
	"panel.ticket.expired": {
		en: "This link has run out",
		es: "Este enlace ha caducado",
	},
	"panel.ticket.expired.text": {
		en: "A link to the panel works for a couple of minutes (ten through your network). To open the panel again, run /brain:panel.",
		es: "Un enlace al panel sirve un par de minutos (diez a través de tu red). Para abrir el panel otra vez, usa /brain:panel.",
	},
	"panel.guard.notice.changed": {
		en: "changed in the panel: %s",
		es: "cambiadas en el panel: %s",
	},
	"panel.guard.notice.accepted": {
		en: "accepted in the panel as they are: %s",
		es: "aceptadas en el panel tal como están: %s",
	},
	"panel.guard.notice.unknown": {
		en: "what changed is not known",
		es: "no se sabe qué cambió",
	},
	"panel.guard.accept": {
		en: "I have seen them, accept",
		es: "Las he visto, aceptar",
	},
	"panel.guard.untouchable": {
		en: "Untouchable repositories",
		es: "Repositorios intocables",
	},
	"panel.guard.untouchable.hint": {
		en: "No commit, push or merge there, ever, whatever the lists below say. For the whole machine, in every space. A folder name ending in * covers its siblings (app-*), and a worktree counts as its repository.",
		es: "Ni commit, ni push, ni merge ahí, nunca, digan lo que digan las listas de abajo. Para toda la máquina, en todos los espacios. Un nombre de carpeta que acabe en * cubre a sus hermanas (app-*), y un worktree cuenta como su repositorio.",
	},
	"panel.guard.untouchable.none": {
		en: "No repository is untouchable yet.",
		es: "Todavía no hay ningún repositorio intocable.",
	},
	"panel.guard.remove": {
		en: "Remove",
		es: "Quitar",
	},
	"panel.guard.typename": {
		en: "To confirm, type the folder's name:",
		es: "Para confirmar, escribe el nombre de la carpeta:",
	},
	"panel.guard.repo": {
		en: "Repository folder",
		es: "Carpeta del repositorio",
	},
	"panel.guard.add.repo": {
		en: "Add a repository",
		es: "Añadir un repositorio",
	},
	"panel.guard.add.root": {
		en: "Add an exception",
		es: "Añadir una excepción",
	},
	"panel.guard.add.profile": {
		en: "Add a profile",
		es: "Añadir un perfil",
	},
	"panel.guard.add.host": {
		en: "Add a machine",
		es: "Añadir una máquina",
	},
	"panel.guard.now": {
		en: "Now:",
		es: "Ahora:",
	},
	"panel.guard.change": {
		en: "Change",
		es: "Cambiar",
	},
	"panel.guard.more": {
		en: "What it means",
		es: "Qué significa",
	},
	"panel.guard.protected.see": {
		en: "See which files",
		es: "Ver qué ficheros",
	},
	"panel.guard.missing": {
		en: "not on this computer: check how it is written",
		es: "no está en este ordenador: revisa cómo está escrita",
	},
	"panel.f.space": {
		en: "Space",
		es: "Espacio",
	},
	"panel.guard.allowed": {
		en: "The assistant may commit and push",
		es: "El asistente puede hacer commit y push",
	},
	"panel.guard.allowed.hint": {
		en: "Without asking, when you have asked for that change.",
		es: "Sin preguntar, cuando tú le has pedido ese cambio.",
	},
	"panel.guard.ask": {
		en: "The assistant asks you each time",
		es: "El asistente te pregunta cada vez",
	},
	"panel.guard.ask.hint": {
		en: "Commit, push and merge ask for your go-ahead, with the branch and the remote.",
		es: "Commit, push y merge piden tu visto bueno, con la rama y el remoto.",
	},
	"panel.guard.none": {
		en: "Nothing listed.",
		es: "No hay nada en la lista.",
	},
	"panel.guard.root": {
		en: "Root exceptions",
		es: "Excepciones de root",
	},
	"panel.guard.root.hint": {
		en: "Becoming root is denied. An exception lets the guard ask you instead, on one machine, until a date; it ends by itself.",
		es: "Hacerse root está denegado. Una excepción deja que el guardia te pregunte, en una máquina, hasta una fecha; acaba sola.",
	},
	"panel.guard.root.bad": {
		en: "These entries cannot be read, so they grant nothing: %s",
		es: "Estas entradas no se pueden leer, así que no conceden nada: %s",
	},
	"panel.guard.root.none": {
		en: "No exception now.",
		es: "Ahora no hay ninguna excepción.",
	},
	"panel.guard.expired": {
		en: "ended",
		es: "caducada",
	},
	"panel.guard.until": {
		en: "Until (date, or date and time, UTC)",
		es: "Hasta (fecha, o fecha y hora, UTC)",
	},
	"panel.guard.typehost": {
		en: "To confirm, type the machine's name again:",
		es: "Para confirmar, escribe otra vez el nombre de la máquina:",
	},
	"panel.guard.profiles": {
		en: "Production profiles",
		es: "Perfiles de producción",
	},
	"panel.guard.profiles.hint": {
		en: "AWS profiles that need your confirmation before the assistant uses them.",
		es: "Perfiles de AWS que necesitan tu confirmación antes de que el asistente los use.",
	},
	"panel.guard.hosts": {
		en: "Untouchable machines",
		es: "Máquinas intocables",
	},
	"panel.guard.hosts.hint": {
		en: "Machines that need your confirmation for ssh, scp and rsync.",
		es: "Máquinas que necesitan tu confirmación para ssh, scp y rsync.",
	},
	"panel.guard.reach": {
		en: "Where augmented brain acts",
		es: "Dónde actúa augmented brain",
	},
	"panel.guard.reach.hint": {
		en: "On the whole computer, every Claude Code session gets the memory of a space (the default one outside them) and the whole guard. Only in your spaces, a session outside their folders gets nothing from augmented brain: no memory, no notices, and the guard only protects its own files, your home folder and what your lists name (production profiles, untouchable machines, the repositories of your lists). Inside a space nothing changes.",
		es: "En todo el ordenador, cada sesión de Claude Code recibe la memoria de un espacio (fuera de ellos, la del espacio por defecto) y el guardia entero. Solo en tus espacios, una sesión fuera de sus carpetas no recibe nada de augmented brain: ni memoria ni avisos, y el guardia solo protege sus propios ficheros, tu carpeta personal y lo que nombran tus listas (perfiles de producción, máquinas intocables, los repositorios de tus listas). Dentro de un espacio no cambia nada.",
	},
	"panel.guard.reach.machine": {
		en: "On the whole computer (as it came)",
		es: "En todo el ordenador (como viene)",
	},
	"panel.guard.reach.spaces": {
		en: "Only in my spaces",
		es: "Solo en mis espacios",
	},
	"panel.guard.reach.warn": {
		en: "Only in your spaces, outside them the guard no longer stops becoming root, secrets in commands, or commits and pushes in repositories your lists do not name. Claude Code still asks for permission as it always does.",
		es: "Solo en tus espacios, fuera de ellos el guardia deja de parar que se actúe como root, los secretos en las órdenes y los commits y push en repositorios que tus listas no nombran. Claude Code sigue pidiendo permiso como siempre.",
	},
	"panel.guard.reach.confirm": {
		en: "I understand what the guard stops watching outside my spaces",
		es: "Entiendo lo que el guardia deja de vigilar fuera de mis espacios",
	},
	"panel.guard.reach.dirs": {
		en: "The folders of your spaces:",
		es: "Las carpetas de tus espacios:",
	},
	"panel.guard.err.reachconfirm": {
		en: "To keep augmented brain to your spaces, tick the box that says you understand what the guard stops watching outside them.",
		es: "Para que augmented brain se quede en tus espacios, marca la casilla que dice que entiendes lo que el guardia deja de vigilar fuera de ellos.",
	},
	"panel.guard.trust": {
		en: "What the assistant saves on its own",
		es: "Lo que el asistente guarda solo",
	},
	"panel.guard.trust.hint": {
		en: "A rule, your profile and the guard's lists change how every session works. Choose whether the assistant saves them and you are told, or asks you first. Tasks, the inbox and what it learns are always saved without asking.",
		es: "Una regla, tu perfil y las listas del guardia cambian cómo trabajan todas las sesiones. Elige si el asistente los guarda y se te avisa, o si te pregunta antes. Las tareas, el buzón y lo que aprende se guardan siempre sin preguntar.",
	},
	"panel.guard.trust.tell": {
		en: "Save and tell me (a line with what was saved)",
		es: "Guardar y avisarme (una línea con lo guardado)",
	},
	"panel.guard.trust.ask": {
		en: "Ask me first (Claude Code shows what would be saved, and you say yes or no)",
		es: "Preguntarme antes (Claude Code enseña lo que se guardaría, y dices sí o no)",
	},
	// Shown under "the guard's lists:" (hook.guardnotice).
	"panel.guard.trust.notice.tell": {
		en: "changed in the panel, what the assistant saves on its own: it now saves rules, your profile and the guard's lists without asking, and you are told.",
		es: "cambiado en el panel, lo que el asistente guarda solo: ahora guarda reglas, tu perfil y las listas del guardia sin preguntar, y se te avisa.",
	},
	"panel.guard.trust.notice.ask": {
		en: "changed in the panel, what the assistant saves on its own: it now asks you before it saves a rule, your profile or the guard's lists.",
		es: "cambiado en el panel, lo que el asistente guarda solo: ahora te pregunta antes de guardar una regla, tu perfil o las listas del guardia.",
	},
	"panel.guard.remote": {
		en: "You are on the panel from another device: here you can see the guard's lists; changing or accepting them, the model and what the assistant saves on its own is only from this computer's panel.",
		es: "Estás en el panel desde otro dispositivo: aquí puedes ver las listas del guardia; cambiarlas o aceptarlas, el modelo y lo que el asistente guarda por su cuenta solo se puede desde el panel de este ordenador.",
	},
	"panel.guard.localonly": {
		en: "Changing the guard is only from this computer's panel.",
		es: "Cambiar el guardia solo se puede desde el panel de este ordenador.",
	},
	"panel.guard.model.unset": {
		en: "None: no check.",
		es: "Ninguno: no se comprueba.",
	},
	"panel.guard.model": {
		en: "Model of your sessions",
		es: "Modelo de tus sesiones",
	},
	"panel.guard.model.hint": {
		en: "The model you expect to answer: opus, sonnet, haiku or a full name. When a session answers with another one, the assistant tells you at once. Empty: no check.",
		es: "El modelo que esperas que responda: opus, sonnet, haiku o un nombre completo. Si una sesión responde con otro, el asistente te lo dice en el momento. Vacío: no se comprueba.",
	},
	"panel.guard.model.last": {
		en: "Last session start: Claude Code said %s.",
		es: "Último inicio de sesión: Claude Code dijo %s.",
	},
	"panel.guard.model.none": {
		en: "nothing about its model (the check then comes after an answer, the second at the latest)",
		es: "nada de su modelo (entonces el aviso llega tras una respuesta, como tarde la segunda)",
	},
	"panel.f.model": {
		en: "Model",
		es: "Modelo",
	},
	"panel.guard.fromspaces": {
		en: "Also from a space's own list (change them in config.toml):",
		es: "También de la lista propia de un espacio (cámbialas en config.toml):",
	},
	"panel.guard.protected": {
		en: "Files the assistant cannot change",
		es: "Ficheros que el asistente no puede cambiar",
	},
	"panel.guard.protected.hint": {
		en: "Writing one asks you first. An assistant that writes a script in line could still edit them: the guard then notices the change and asks before a push.",
		es: "Escribir uno te pregunta antes. Un asistente que escriba un script en línea aún podría editarlos: el guardia nota el cambio y pregunta antes de un push.",
	},
	"panel.guard.err.gone": {
		en: "That entry is no longer in the list.",
		es: "Esa entrada ya no está en la lista.",
	},
	"panel.guard.err.op": {
		en: "The panel did not understand that change.",
		es: "El panel no ha entendido ese cambio.",
	},
	"panel.guard.err.typename": {
		en: "Type %s to confirm.",
		es: "Escribe %s para confirmar.",
	},
	"panel.f.host": {
		en: "Machine",
		es: "Máquina",
	},
	"panel.f.value": {
		en: "Value",
		es: "Valor",
	},
	"panel.nav.day": {
		en: "Day to day",
		es: "Día a día",
	},
	"panel.nav.notes": {
		en: "Notes",
		es: "Notas",
	},
	"panel.nav.care": {
		en: "Upkeep",
		es: "Mantenimiento",
	},
	"panel.today": {
		en: "Today",
		es: "Hoy",
	},
	"panel.today.hint": {
		en: "What every space asks of you now, and the work that is only on this computer.",
		es: "Lo que te pide cada espacio ahora, y el trabajo que solo está en este ordenador.",
	},
	"panel.today.nothing": {
		en: "Nothing asks for you today.",
		es: "Hoy no hay nada que te espere.",
	},
	"panel.today.space": {
		en: "Space %s",
		es: "Espacio %s",
	},
	"panel.today.counts": {
		en: "%d under way, %s",
		es: "%d en marcha, %s",
	},
	"panel.today.parked.1": {
		en: "1 parked",
		es: "1 aparcada",
	},
	"panel.today.parked.n": {
		en: "%d parked",
		es: "%d aparcadas",
	},
	"panel.today.inbox": {
		en: "%d in the inbox",
		es: "%d en el buzón",
	},
	"panel.today.review": {
		en: "weekly review due (%d days)",
		es: "toca la revisión semanal (%d días)",
	},
	"panel.today.dead.see": {
		en: "See what was said, on the Sessions page.",
		es: "Mira lo que se dijo en la página Sesiones.",
	},
	"panel.space.sessions": {
		en: "The Claude sessions of this space",
		es: "Las sesiones de Claude de este espacio",
	},
	"panel.today.dead": {
		en: "%d sessions cut off",
		es: "%d sesiones cortadas",
	},
	"panel.today.reminders": {
		en: "Due this week",
		es: "Avisos de esta semana",
	},
	"panel.today.mine": {
		en: "Waiting on you",
		es: "Te esperan a ti",
	},
	"panel.repos": {
		en: "Repositories",
		es: "Repositorios",
	},
	"panel.repos.hint": {
		en: "The repositories your open tasks name (their repos:), when they hold changes not committed or commits not pushed. Read only: commit and push stay with git.",
		es: "Los repositorios que nombran tus tareas abiertas (su repos:), cuando tienen cambios sin commit o commits sin subir. Solo se leen: el commit y el push siguen en git.",
	},
	"panel.repos.none": {
		en: "No repository holds work that is only here.",
		es: "Ningún repositorio tiene trabajo que solo esté aquí.",
	},
	"panel.repos.clean": {
		en: "%d more, with nothing pending.",
		es: "%d más, sin nada pendiente.",
	},
	"panel.repos.forgotten": {
		en: "forgotten?",
		es: "¿olvidado?",
	},
	"panel.repos.changed": {
		en: "%d files changed",
		es: "%d ficheros cambiados",
	},
	"panel.repos.ahead": {
		en: "%d commits not pushed",
		es: "%d commits sin subir",
	},
	"panel.repos.changed1": {
		en: "1 file changed",
		es: "1 fichero cambiado",
	},
	"panel.repos.ahead1": {
		en: "1 commit not pushed",
		es: "1 commit sin subir",
	},
	"panel.repos.noupstream": {
		en: "branch never pushed",
		es: "rama nunca subida",
	},
	"panel.repos.unreadable": {
		en: "git could not read it",
		es: "git no ha podido leerlo",
	},
	"panel.filter.all": {
		en: "All",
		es: "Todas",
	},
	"panel.filter.me": {
		en: "Waiting on you",
		es: "Te esperan",
	},
	"panel.filter.active": {
		en: "Under way",
		es: "En marcha",
	},
	"panel.filter.parked": {
		en: "Parked",
		es: "Aparcadas",
	},
	"panel.filter.stale": {
		en: "Untouched",
		es: "Paradas",
	},
	"panel.filter.waits": {
		en: "Waiting on:",
		es: "Esperan a:",
	},
	"panel.filter.shown": {
		en: "Showing %d of %d.",
		es: "Se ven %d de %d.",
	},
	"panel.stale": {
		en: "untouched %d days",
		es: "parada %d días",
	},
	"panel.person.new": {
		en: "New person",
		es: "Nueva persona",
	},
	"panel.person.hint": {
		en: "So that a first name, in a task or in what you tell Claude, leads to the right person.",
		es: "Para que un nombre de pila, en una tarea o en lo que le dices a Claude, lleve a la persona que es.",
	},
	"panel.person.edit": {
		en: "Their card",
		es: "Su ficha",
	},
	"panel.person.new.hint": {
		en: "They start in the space you are in; on their page you choose where else they show.",
		es: "Empieza en el espacio en el que estás; en su página eliges en qué otros espacios se ve.",
	},
	"panel.person.card.hint": {
		en: "One card for the same person in every space. The name stays: a new name would be a new person. Each space keeps only what you leave visible there.",
		es: "Una sola ficha para la misma persona en todos los espacios. El nombre se queda: un nombre nuevo sería otra persona. Cada espacio guarda solo lo que dejes visible en él.",
	},
	"panel.person.in": {
		en: "In %s",
		es: "En %s",
	},
	"panel.person.nofile": {
		en: "Not here yet. It is created when you save something visible in this space.",
		es: "Aún no está aquí. Se crea cuando guardes algo visible en este espacio.",
	},
	"panel.person.details": {
		en: "Details",
		es: "Detalles",
	},
	"panel.person.details.hint": {
		en: "Each detail shows only in the spaces you tick. Empty the text to remove it.",
		es: "Cada detalle solo se ve en los espacios que marques. Vacía el texto para quitarlo.",
	},
	"panel.person.detail.n": {
		en: "Detail %d",
		es: "Detalle %d",
	},
	"panel.person.visible": {
		en: "Visible in",
		es: "Se ve en",
	},
	"panel.person.nospace": {
		en: "Not saved: tick at least one space, or empty the text to remove this detail.",
		es: "No se guardó: marca al menos un espacio, o vacía el texto para quitar este detalle.",
	},
	"panel.person.more": {
		en: "Another detail",
		es: "Otro detalle",
	},
	"panel.person.more.hint": {
		en: "Reloads the page with one more empty row: save first, or what you typed is lost.",
		es: "Recarga la página con una fila vacía más: guarda antes, o lo que escribiste se pierde.",
	},
	"panel.person.waiting": {
		en: "Waiting on them",
		es: "Le esperan",
	},
	"panel.person.named": {
		en: "Tasks that name them",
		es: "Tareas que la nombran",
	},
	"panel.people.waiting": {
		en: "%d tasks wait on them",
		es: "le esperan %d tareas",
	},
	"panel.f.aliases": {
		en: "Other names",
		es: "Otros nombres",
	},
	"panel.f.aliases.hint": {
		en: "separated by commas",
		es: "separados por comas",
	},
	"panel.f.role": {
		en: "Who they are to you",
		es: "Quién es para ti",
	},
	"panel.f.notes": {
		en: "Notes",
		es: "Notas",
	},
	"panel.nobody": {
		en: "nobody",
		es: "nadie",
	},
	"panel.search": {
		en: "Search",
		es: "Buscar",
	},
	"panel.search.side": {
		en: "Search the notes…",
		es: "Buscar en las notas…",
	},
	"panel.search.hint": {
		en: "Words in a title, a summary or the text",
		es: "Palabras de un título, un resumen o el texto",
	},
	"panel.search.go": {
		en: "Search",
		es: "Buscar",
	},
	"panel.search.found": {
		en: "%d notes hold every word.",
		es: "%d notas tienen todas las palabras.",
	},
	"panel.search.more": {
		en: "And %d more: add a word to narrow it.",
		es: "Y %d más: añade una palabra para afinar.",
	},
	"panel.search.none": {
		en: "No note holds every word.",
		es: "Ninguna nota tiene todas las palabras.",
	},
	"panel.review": {
		en: "Weekly review",
		es: "Revisión semanal",
	},
	"panel.review.last": {
		en: "Last one: %s",
		es: "La última: %s",
	},
	"panel.review.never": {
		en: "There has been none yet.",
		es: "Aún no se ha hecho ninguna.",
	},
	"panel.review.hint": {
		en: "About ten minutes: each capture becomes a task or knowledge, or is dropped; each task untouched for two weeks goes on, waits for some day, or you open it to close it.",
		es: "Unos diez minutos: cada captura pasa a tarea o a conocimiento, o se descarta; cada tarea sin tocar dos semanas sigue, pasa a algún día, o la abres para cerrarla.",
	},
	"panel.review.as.task": {
		en: "New task",
		es: "Tarea nueva",
	},
	"panel.review.as.reference": {
		en: "Reference",
		es: "Referencia",
	},
	"panel.review.as.lesson": {
		en: "Lesson",
		es: "Lección",
	},
	"panel.review.as.drop": {
		en: "Drop it",
		es: "Descartar",
	},
	"panel.review.title.hint": {
		en: "Title (the capture, if empty)",
		es: "Título (la captura, si lo dejas vacío)",
	},
	"panel.review.do": {
		en: "Do it",
		es: "Hecho",
	},
	"panel.review.stale": {
		en: "Untouched for two weeks or more",
		es: "Paradas: dos semanas o más sin tocar",
	},
	"panel.review.stale.none": {
		en: "Every live task was touched in the last two weeks.",
		es: "Todas las tareas vivas se han tocado en las dos últimas semanas.",
	},
	"panel.review.updated": {
		en: "last change %s",
		es: "último cambio %s",
	},
	"panel.review.keep": {
		en: "Goes on",
		es: "Sigue",
	},
	"panel.review.someday": {
		en: "Some day",
		es: "Algún día",
	},
	"panel.review.open": {
		en: "Open it to close it",
		es: "Abrirla para cerrarla",
	},
	"panel.review.kept": {
		en: "Looked at in the weekly review: it goes on as it is.",
		es: "Repasada en la revisión semanal: sigue como está.",
	},
	"panel.review.someday.next": {
		en: "look at it again when there is time",
		es: "volver a mirarla cuando haya tiempo",
	},
	"panel.review.finish": {
		en: "Finish",
		es: "Terminar",
	},
	"panel.review.draft": {
		en: "Decided so far, kept with the review when you finish:",
		es: "Lo decidido hasta ahora, que se guarda con la revisión al terminar:",
	},
	"panel.review.summary": {
		en: "Anything to add (optional)",
		es: "Algo que añadir (opcional)",
	},
	"panel.review.summary.default": {
		en: "Weekly review, done in the panel.",
		es: "Revisión semanal, hecha en el panel.",
	},
	"panel.review.done": {
		en: "Finish the review",
		es: "Terminar la revisión",
	},
	"panel.review.line.task": {
		en: "Capture → task %s (%s)",
		es: "Captura → tarea %s (%s)",
	},
	"panel.review.line.knowledge": {
		en: "Capture → knowledge %s (%s)",
		es: "Captura → conocimiento %s (%s)",
	},
	"panel.review.line.drop": {
		en: "Capture dropped: %s",
		es: "Captura descartada: %s",
	},
	"panel.review.line.keep": {
		en: "%s goes on",
		es: "%s sigue",
	},
	"panel.review.line.someday": {
		en: "%s → some day",
		es: "%s → algún día",
	},
	"panel.f.as": {
		en: "What it becomes",
		es: "En qué se convierte",
	},
	"panel.inbox.totask": {
		en: "Make it a task",
		es: "Convertir en tarea",
	},
	"panel.changed": {
		en: "This note changed since the page opened.",
		es: "Esta nota ha cambiado desde que abriste la página.",
	},
	"panel.changed.reload": {
		en: "See it as it is now",
		es: "Verla como está ahora",
	},
	"panel.area.edit": {
		en: "Change",
		es: "Cambiar",
	},
	"panel.secrets": {
		en: "Secrets",
		es: "Secretos",
	},
	"panel.secrets.hint": {
		en: "The secrets you keep in this computer's keychain, by name: never their values. So you know which there are and what each is for, and a new machine knows which to ask you for. The same list in every space.",
		es: "Los secretos que guardas en el llavero de este ordenador, por su nombre: nunca su valor. Para saber cuáles hay y para qué sirve cada uno, y que una máquina nueva sepa cuáles pedirte. La misma lista en todos los espacios.",
	},
	"panel.secrets.none": {
		en: "No secret is named yet.",
		es: "Aún no hay ningún secreto apuntado.",
	},
	"panel.secrets.bad": {
		en: "%d entries of the list cannot be read: they are written as service/name: what it is for.",
		es: "%d entradas de la lista no se pueden leer: se escriben servicio/nombre: para qué sirve.",
	},
	"panel.secrets.check": {
		en: "Check the keychain",
		es: "Comprobar en el llavero",
	},
	"panel.secrets.check.hint": {
		en: "asks whether each one is there, without reading it; a locked keychain asks for its password",
		es: "pregunta si está cada uno, sin leerlo; si el llavero está bloqueado, el sistema pide su contraseña",
	},
	"panel.secrets.present": {
		en: "in the keychain",
		es: "en el llavero",
	},
	"panel.secrets.missing": {
		en: "not on this computer",
		es: "no está en este ordenador",
	},
	"panel.secrets.unknown": {
		en: "cannot tell here",
		es: "aquí no se puede saber",
	},
	"panel.secrets.how": {
		en: "To store it here (it asks for the value):",
		es: "Para guardarlo aquí (te pide el valor):",
	},
	"panel.secrets.used": {
		en: "Named in:",
		es: "Lo nombran:",
	},
	"panel.secrets.forget": {
		en: "Take it off the list",
		es: "Quitarlo de la lista",
	},
	"panel.secrets.forget.confirm": {
		en: "Only its name leaves the list: the secret stays in the keychain.",
		es: "Solo sale su nombre de la lista: el secreto sigue en el llavero.",
	},
	"panel.secrets.add": {
		en: "Name a secret",
		es: "Apuntar un secreto",
	},
	"panel.secrets.add.hint": {
		en: "Its service and name in the keychain, and what it is for. Never its value: that stays in the keychain.",
		es: "Su servicio y su nombre en el llavero, y para qué sirve. Nunca su valor: ese se queda en el llavero.",
	},
	"panel.f.service": {
		en: "Service",
		es: "Servicio",
	},
	"panel.err.secretname": {
		en: "Service and name are one word each (letters, digits, . _ @ + -, not starting with -), and what it is for is one line of plain text.",
		es: "Servicio y nombre son una palabra cada uno (letras, cifras, . _ @ + -, sin empezar por -), y para qué sirve, una línea de texto sencillo.",
	},
	"panel.err.secretvalue": {
		en: "That looks like the secret itself. Write only its name and what it is for.",
		es: "Eso parece el secreto mismo. Escribe solo su nombre y para qué sirve.",
	},
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
	"panel.ok.task.new": {
		en: "Task created.",
		es: "Tarea creada.",
	},
	"panel.ok.task.update": {
		en: "Task updated.",
		es: "Tarea actualizada.",
	},
	"panel.ok.task.journal": {
		en: "Added to the journal.",
		es: "Añadido al diario.",
	},
	"panel.ok.task.park": {
		en: "Task parked.",
		es: "Tarea aparcada.",
	},
	"panel.ok.task.close": {
		en: "Task closed.",
		es: "Tarea cerrada.",
	},
	"panel.ok.task.step": {
		en: "Plan updated.",
		es: "Plan actualizado.",
	},
	"panel.ok.watch.add": {
		en: "Watch date added.",
		es: "Fecha para mirar añadida.",
	},
	"panel.ok.watch.done": {
		en: "Watch date marked done.",
		es: "Fecha marcada como hecha.",
	},
	"panel.ok.inbox.capture": {
		en: "Written down in the inbox.",
		es: "Apuntado en el buzón.",
	},
	"panel.ok.inbox.done": {
		en: "Taken out of the inbox.",
		es: "Sacado del buzón.",
	},
	"panel.ok.area.new": {
		en: "Area created.",
		es: "Área creada.",
	},
	"panel.ok.area.update": {
		en: "Area saved.",
		es: "Área guardada.",
	},
	"panel.ok.person": {
		en: "Person saved.",
		es: "Persona guardada.",
	},
	"panel.ok.person.card": {
		en: "Person saved.",
		es: "Persona guardada.",
	},
	"panel.ok.secret": {
		en: "Secret's name added.",
		es: "Nombre del secreto añadido.",
	},
	"panel.ok.secret.forget": {
		en: "Secret's name forgotten.",
		es: "Nombre del secreto olvidado.",
	},
	"panel.ok.review.capture": {
		en: "Capture sorted out.",
		es: "Captura resuelta.",
	},
	"panel.ok.review.task": {
		en: "Task reviewed.",
		es: "Tarea revisada.",
	},
	"panel.ok.review.done": {
		en: "Weekly review finished.",
		es: "Revisión semanal terminada.",
	},
	"panel.ok.profile": {
		en: "Profile saved.",
		es: "Perfil guardado.",
	},
	"panel.ok.session.recover": {
		en: "Session recovered.",
		es: "Sesión recuperada.",
	},
	"panel.ok.session.reviewed": {
		en: "Session marked as reviewed.",
		es: "Sesión marcada como revisada.",
	},
	"panel.ok.session.delete": {
		en: "Session deleted.",
		es: "Sesión borrada.",
	},
	"panel.ok.repo.describe": {
		en: "Description saved.",
		es: "Descripción guardada.",
	},
	"panel.ok.guard.list.add": {
		en: "Added to the list.",
		es: "Añadido a la lista.",
	},
	"panel.ok.guard.list.remove": {
		en: "Taken off the list.",
		es: "Quitado de la lista.",
	},
	"panel.ok.guard.root.add": {
		en: "Root exception added.",
		es: "Excepción de root añadida.",
	},
	"panel.ok.guard.root.remove": {
		en: "Root exception removed.",
		es: "Excepción de root quitada.",
	},
	"panel.ok.guard.accept": {
		en: "Lists accepted.",
		es: "Listas aceptadas.",
	},
	"panel.ok.guard.model": {
		en: "Preferred model saved.",
		es: "Modelo preferido guardado.",
	},
	"panel.ok.guard.trust": {
		en: "Setting saved.",
		es: "Ajuste guardado.",
	},
	"panel.ok.versions.restore": {
		en: "Version restored.",
		es: "Versión recuperada.",
	},
	"panel.ok.backup.run": {
		en: "Backup started.",
		es: "Copia en marcha.",
	},
	"panel.ok.backup.setup": {
		en: "Backup set up; the first copy has started.",
		es: "Copia configurada; la primera ya está en marcha.",
	},
	"panel.notsaved": {
		en: "Nothing was saved.",
		es: "No se ha guardado nada.",
	},
	"panel.notsaved.see": {
		en: "See why, next to the form.",
		es: "Mira por qué, junto al formulario.",
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
	"panel.menu": {
		en: "Menu",
		es: "Menú",
	},
	"panel.footer.remote": {
		en: "On this computer and your devices in Tailscale, only with your account. It stops by itself after 30 minutes without use.",
		es: "En este ordenador y en tus dispositivos de Tailscale, solo con tu cuenta. Se para solo tras 30 minutos sin uso.",
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
		es: "creada %s, último cambio %s",
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
	"panel.journal.count": {
		en: "%d entries",
		es: "%d entradas",
	},
	"panel.journal.newest": {
		en: "newest first",
		es: "lo último primero",
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
	"panel.task.reopen": {
		en: "Open it again",
		es: "Reabrirla",
	},
	"panel.f.why": {
		en: "Why (optional)",
		es: "Por qué (opcional)",
	},
	"panel.task.reopen.hint": {
		en: "It goes back to the phase you choose, and its journal says it was reopened from the panel, with your reason.",
		es: "Vuelve a la fase que elijas, y su diario dice que se reabrió desde el panel, con tu motivo.",
	},
	"panel.reopen": {
		en: "Reopen",
		es: "Reabrir",
	},
	"panel.today.space.empty": {
		en: "Nothing here asks for you today.",
		es: "Aquí no hay nada que te espere hoy.",
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
	"panel.remote.restarted": {
		en: "The panel that was open had to be started again for this (it came from an earlier version): reload the panel's pages that were open in the browser.",
		es: "El panel que estaba abierto se ha tenido que arrancar de nuevo para esto (era de una versión anterior): recarga las páginas del panel que tenías abiertas en el navegador.",
	},
	"panel.f.from": {
		en: "from space",
		es: "del espacio",
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
		en: "The weekly review goes through the inbox: each capture becomes a task or knowledge, or is dropped. Make one a task here, or mark it as dealt with when it is.",
		es: "La revisión semanal repasa el buzón: cada apunte se convierte en tarea o en conocimiento, o se descarta. Aquí puedes convertirlo en tarea, o marcarlo como resuelto cuando lo esté.",
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
	"panel.people.unread": {
		en: "The people of %s are not here: that space could not be opened. To see why, run augmented-brain doctor.",
		es: "Faltan las personas de %s: ese espacio no se ha podido abrir. Para ver por qué, ejecuta augmented-brain doctor.",
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
	"panel.profile.assistant": {
		en: "Your assistant",
		es: "Tu asistente",
	},
	"panel.profile.assistant.hint": {
		en: "A name and a personality for the assistant. They change how it talks, never the rules, the guard or its honesty. Leave both empty for none.",
		es: "Un nombre y una personalidad para el asistente. Cambian cómo habla, nunca las reglas, el guardia ni su honestidad. Déjalos vacíos si no quieres ninguno.",
	},
	"panel.f.assistant": {
		en: "Its name",
		es: "Su nombre",
	},
	"panel.f.personality": {
		en: "Its personality, in your words",
		es: "Su personalidad, con tus palabras",
	},
	"panel.profile.signature": {
		en: "Emails it writes in your name end like this, and the second line is always there:",
		es: "Los correos que escriba en tu nombre acaban así, y la segunda línea va siempre:",
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
		en: "No backup: if this computer breaks, these notes are lost. Set one up on the Backup page, or ask your assistant.",
		es: "Sin copia de seguridad: si este ordenador se rompe, estas notas se pierden. Configúrala en la página Copia de seguridad, o pídeselo a tu asistente.",
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
	"panel.days.tomorrow": {
		en: "tomorrow",
		es: "mañana",
	},
	"panel.days.yesterday": {
		en: "1 day late",
		es: "1 día de retraso",
	},
	"panel.today.dead1": {
		en: "1 session cut off",
		es: "1 sesión cortada",
	},
	"panel.today.parkedmine": {
		en: "Parked, waiting on a decision of yours:",
		es: "Aparcadas, a la espera de una decisión tuya:",
	},
	"panel.days.late": {
		en: "%d days late",
		es: "%d días de retraso",
	},
	"panel.days.in": {
		en: "in %d days",
		es: "dentro de %d días",
	},
	// A date as a person reads it (package panel, humanDate).
	"panel.date.today": {
		en: "today",
		es: "hoy",
	},
	"panel.date.yesterday": {
		en: "yesterday",
		es: "ayer",
	},
	"panel.date.tomorrow": {
		en: "tomorrow",
		es: "mañana",
	},
	"panel.date.months": {
		en: "Jan Feb Mar Apr May Jun Jul Aug Sep Oct Nov Dec",
		es: "ene feb mar abr may jun jul ago sep oct nov dic",
	},
	"panel.date.day": {
		en: "%d %s",
		es: "%d %s",
	},
	"panel.date.year": {
		en: "%d %s %d",
		es: "%d %s %d",
	},
	"panel.date.time": {
		en: "%s, %s",
		es: "%s, %s",
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
	"panel.cli.opened": {
		en: "The panel is open in your browser.",
		es: "El panel está abierto en tu navegador.",
	},
	"panel.cli.opened.running": {
		en: "The panel was already running: it is open in your browser.",
		es: "El panel ya estaba en marcha: está abierto en tu navegador.",
	},
	"panel.cli.once": {
		en: "That address opens the panel once, within 2 minutes. To open it again, run this command again.",
		es: "Esa dirección abre el panel una vez, en los próximos 2 minutos. Para abrirlo otra vez, vuelve a lanzar este comando.",
	},
	"panel.cli.nobrowser": {
		en: "Open that address in your browser.",
		es: "Abre esa dirección en tu navegador.",
	},
	"panel.cli.own.terminal": {
		en: "The panel could not open in a browser from here, and its address is not shown here: it is for you, not for the assistant. Run this command in a terminal of your own (not through Claude):\n  augmented-brain panel",
		es: "Desde aquí el panel no se ha podido abrir en un navegador, y su dirección no se enseña aquí: es para ti, no para el asistente. Lanza este comando en un terminal tuyo (no a través de Claude):\n  augmented-brain panel",
	},
	"panel.cli.stopped": {
		en: "Panel stopped.",
		es: "Panel parado.",
	},
	"panel.cli.none": {
		en: "No panel is running.",
		es: "No hay ningún panel en marcha.",
	},
	// panel remote: the panel published to the user's own Tailscale network.
	"panel.remote.open": {
		en: "The panel is open for your other devices at:\n  %s",
		es: "El panel está abierto para tus otros dispositivos en:\n  %s",
	},
	"panel.remote.own.terminal": {
		en: "The address that opens the panel from your other devices is not shown here: it is for you, not for the assistant. Run this command in a terminal of your own (not through Claude):\n  augmented-brain panel remote",
		es: "La dirección que abre el panel desde tus otros dispositivos no se enseña aquí: es para ti, no para el asistente. Lanza este comando en un terminal tuyo (no a través de Claude):\n  augmented-brain panel remote",
	},
	"panel.remote.qr": {
		en: "On your phone, point the camera at this code:",
		es: "En el móvil, apunta la cámara a este código:",
	},
	"panel.remote.idle": {
		en: "Only your devices in your Tailscale network, signed in with your account (%s), can open it. The address opens the panel once, within 10 minutes; the page it opens works until the panel stops: after 30 minutes without use, or with augmented-brain panel stop.",
		es: "Solo pueden abrirla tus dispositivos de tu red de Tailscale, con tu cuenta (%s). La dirección abre el panel una vez, en los próximos 10 minutos; la página que abre funciona hasta que el panel se para: tras 30 minutos sin uso, o con augmented-brain panel stop.",
	},
	"panel.remote.missing": {
		en: "To open the panel from your other devices, augmented brain uses Tailscale, a free app that joins your devices in a private network of their own. It is not installed here.\nInstall it on this computer and on the device you want to use (the same account on both): https://tailscale.com/download\nThen run augmented-brain panel remote again.",
		es: "Para abrir el panel desde tus otros dispositivos, augmented brain usa Tailscale, una aplicación gratuita que une tus dispositivos en una red privada solo suya. Aquí no está instalada.\nInstálala en este ordenador y en el dispositivo que quieras usar (con la misma cuenta en los dos): https://tailscale.com/download\nLuego lanza otra vez augmented-brain panel remote.",
	},
	"panel.remote.stopped": {
		en: "Tailscale is installed but not connected. Open it and sign in (or run: tailscale up), then run augmented-brain panel remote again.",
		es: "Tailscale está instalado pero no está conectado. Ábrelo e inicia sesión (o lanza: tailscale up), y luego otra vez augmented-brain panel remote.",
	},
	"panel.remote.nohttps": {
		en: "Your Tailscale network does not give out HTTPS certificates yet, and the panel only travels encrypted. Turn them on once, in Tailscale's admin page: %s → HTTPS Certificates → Enable HTTPS. (The names of your machines then appear in public certificate logs; nobody outside your devices can reach them.)\nThen run augmented-brain panel remote again.",
		es: "Tu red de Tailscale aún no da certificados HTTPS, y el panel solo viaja cifrado. Actívalos una vez en la página de administración de Tailscale: %s → HTTPS Certificates → Enable HTTPS. (Los nombres de tus máquinas pasan entonces a registros públicos de certificados; nadie fuera de tus dispositivos puede llegar a ellas.)\nLuego lanza otra vez augmented-brain panel remote.",
	},
	"panel.remote.operator": {
		en: "Tailscale lets only an administrator publish on this computer. Let your user do it, once, with this command (it needs your administrator password; augmented brain does not run it for you):\n  %s\nThen run augmented-brain panel remote again.",
		es: "Tailscale solo deja publicar en este ordenador a un administrador. Deja que lo haga tu usuario, una vez, con este comando (pide tu contraseña de administrador; augmented brain no lo lanza por ti):\n  %s\nLuego lanza otra vez augmented-brain panel remote.",
	},
	"panel.remote.taken": {
		en: "You already publish something else on port %d of your Tailscale network, and augmented brain never touches it. Take it off (tailscale serve status shows it) and run augmented-brain panel remote again.",
		es: "Ya publicas otra cosa en el puerto %d de tu red de Tailscale, y augmented brain nunca la toca. Quítala (tailscale serve status la enseña) y lanza otra vez augmented-brain panel remote.",
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
	// The Repositories page (repos.go).
	"panel.repos.page.hint": {
		en: "Every git repository in the work folders of your spaces, up to three folders down: what is pending in it, the open tasks that name it, and what it is, in your own words. Read only: commit and push stay with git.",
		es: "Todos los repositorios git de las carpetas de trabajo de tus espacios, hasta tres carpetas por debajo: qué tienen pendiente, las tareas abiertas que los nombran y qué son, con tus palabras. Solo se leen: el commit y el push siguen en git.",
	},
	"panel.repos.all": {
		en: "See all repositories",
		es: "Ver todos los repositorios",
	},
	"panel.repos.describe": {
		en: "describe it",
		es: "descríbelo",
	},
	"panel.repos.describe.what": {
		en: "What it is, in one line",
		es: "Qué es, en una línea",
	},
	"panel.repos.describe.hint": {
		en: "It is kept in this space's notes (the note Repositories, in knowledge), so the backup keeps it too. Leave it empty and save to take it out.",
		es: "Se guarda en las notas de este espacio (la nota Repositorios, en conocimiento), así que la copia de seguridad también la guarda. Déjalo vacío y guarda para quitarla.",
	},
	"panel.repos.uptodate": {
		en: "nothing pending",
		es: "nada pendiente",
	},
	"panel.repos.noremote": {
		en: "no remote",
		es: "sin remoto",
	},
	"panel.repos.empty": {
		en: "No git repository in the work folders of this space.",
		es: "No hay ningún repositorio git en las carpetas de trabajo de este espacio.",
	},
	"panel.repos.nofolders": {
		en: "This space has no work folders, so there is nowhere to look.",
		es: "Este espacio no tiene carpetas de trabajo, así que no hay dónde buscar.",
	},
	"panel.repos.more": {
		en: "The search stopped before the end (too many folders or repositories): some may be missing.",
		es: "La búsqueda paró antes de acabar (demasiadas carpetas o repositorios): puede faltar alguno.",
	},
	"panel.f.repo": {
		en: "Repository",
		es: "Repositorio",
	},
}
