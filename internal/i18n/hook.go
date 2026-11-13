package i18n

// What the hooks tell the assistant in the language of the configuration. These are
// instructions to the assistant, which passes them on in the user's language.
var hookMessages = map[string]msg{
	"fresh.head": {
		en: "augmented brain: since this session last looked, these tasks changed outside it (another session, the panel or a hand edit). What you read about them earlier (the index at the start, an earlier answer) is out of date: before you state their status, say they are pending or done, or act on them, read them with task_get. Mention it to the user only when it bears on what you are doing.\n",
		es: "augmented brain: desde la última vez que esta sesión miró, estas tareas han cambiado fuera de ella (otra sesión, el panel o una edición a mano). Lo que leíste de ellas antes (el índice del inicio, una respuesta anterior) está desfasado: antes de decir su estado, darlas por pendientes o hechas, o actuar sobre ellas, léelas con task_get. Coméntaselo al usuario solo si afecta a lo que estás haciendo.\n",
	},
	"hook.panic": {
		en: "the guard failed while checking this action. It is a bug in augmented brain, not in the command: do not try variants of it. Tell the user, who can report it from the Help page of the panel.",
		es: "el guardia ha fallado al revisar esta acción. Es un error de augmented brain, no del comando: no pruebes variantes. Díselo al usuario, que puede informar de él desde la página Ayuda del panel.",
	},
	"hook.panic.log": {
		en: "The details are in %s.",
		es: "Los detalles están en %s.",
	},
	"hook.guardnotice": {
		en: "augmented brain · the guard's lists:\n%s\nIf you did not do it, look at the Guard page of the panel.",
		es: "augmented brain · las listas del guardia:\n%s\nSi no has sido tú, mira la página Guardia del panel.",
	},
	"hook.toolnotice": {
		en: "augmented brain · the assistant saved, for every session from now on:\n%s\nIf you did not ask for it, tell the assistant to undo it, or look at the panel (Rules, Profile, Guard).",
		es: "augmented brain · el asistente ha guardado, para todas las sesiones a partir de ahora:\n%s\nSi no se lo has pedido, dile que lo deshaga, o míralo en el panel (Reglas, Perfil, Guardia).",
	},
	"hook.toolnotice.rule.new": {
		en: "new rule «%s» (%s)",
		es: "regla nueva «%s» (%s)",
	},
	"hook.toolnotice.rule.changed": {
		en: "rule «%s» changed (%s)",
		es: "regla «%s» cambiada (%s)",
	},
	"hook.toolnotice.rule.retired": {
		en: "rule «%s» retired (%s): it is no longer loaded",
		es: "regla «%s» retirada (%s): ya no se carga",
	},
	"hook.toolnotice.where.shared": {
		en: "every space",
		es: "todos los espacios",
	},
	"hook.toolnotice.where.space": {
		en: "space %s",
		es: "espacio %s",
	},
	"hook.toolnotice.profile": {
		en: "your profile changed: %s",
		es: "tu perfil ha cambiado: %s",
	},
	"hook.toolnotice.part.name": {
		en: "your name",
		es: "tu nombre",
	},
	"hook.toolnotice.part.about": {
		en: "about you",
		es: "sobre ti",
	},
	"hook.toolnotice.part.work": {
		en: "how you like to work",
		es: "cómo te gusta trabajar",
	},
	"hook.toolnotice.part.assistant": {
		en: "the assistant's name",
		es: "el nombre del asistente",
	},
	"hook.toolnotice.part.personality": {
		en: "the assistant's personality",
		es: "la personalidad del asistente",
	},
	"hook.toolnotice.safety": {
		en: "the guard now also asks about: %s",
		es: "el guardia ahora también pregunta por: %s",
	},
	"hook.seal.startnotice": {
		en: "a session started with lists that changed outside the panel since the last one; they are taken as the ones you have seen",
		es: "una sesión ha empezado con listas que cambiaron fuera del panel desde la anterior; se toman como las que has visto",
	},
	"hook.seal.taken": {
		en: "The guard's lists (repositories it allows, asks about or never allows, root exceptions, production profiles, untouchable hosts) are not the ones saved when a session last started, and the panel did not change them. If the user did not do it, say so.",
		es: "Las listas del guardia (repositorios que permite, por los que pregunta o que nunca permite, excepciones de root, perfiles de producción, hosts intocables) no son las que se guardaron cuando empezó la última sesión, y no las cambió el panel. Si no fue el usuario, dilo.",
	},
	"hook.seal.kept": {
		en: "The guard's lists (repositories, root exceptions, production profiles, untouchable hosts) are not the ones the user last saw, and the panel did not change them. Until the user accepts them on the panel's Guard page, a commit, push or merge they would let through asks first. If the user did not do it, say so.",
		es: "Las listas del guardia (repositorios, excepciones de root, perfiles de producción, hosts intocables) no son las que el usuario vio por última vez, y no las cambió el panel. Hasta que las acepte en la página Guardia del panel, un commit, push o merge que dejarían pasar pregunta antes. Si no fue el usuario, dilo.",
	},
	"hook.seal.unread": {
		en: "The record of the guard's lists the user last saw cannot be read (%v): until it can, a commit, push or merge the lists would let through asks first. `augmented-brain doctor` shows the problem.",
		es: "No se puede leer el registro de las listas del guardia que el usuario vio por última vez (%v): hasta que se pueda, un commit, push o merge que las listas dejarían pasar pregunta antes. `augmented-brain doctor` enseña el problema.",
	},
	"hook.seal.unwritten": {
		en: "The guard's lists changed outside the panel since the last session, and they could not be saved as the ones the user has seen (%v): a commit, push or merge they would let through asks first.",
		es: "Las listas del guardia cambiaron fuera del panel desde la última sesión, y no se han podido guardar como las que el usuario ha visto (%v): un commit, push o merge que dejarían pasar pregunta antes.",
	},
	"fresh.changed": {
		en: "- %s (%s): now %s, waits on %s, next: %s\n",
		es: "- %s (%s): ahora %s, espera a %s, siguiente: %s\n",
	},
	"fresh.new": {
		en: "- %s (%s): new task, %s\n",
		es: "- %s (%s): tarea nueva, %s\n",
	},
	"fresh.gone": {
		en: "- %s: no longer exists in this space\n",
		es: "- %s: ya no existe en este espacio\n",
	},
	"fresh.more": {
		en: "- and %d more: see pending\n",
		es: "- y %d más: mira pending\n",
	},
	"fresh.format": {
		en: "augmented brain: the notes of this space moved to a new format version since this session started, so every note may have been rewritten. What you read earlier may be out of date: read again (task_get, pending) whatever you are going to state or act on.\n",
		es: "augmented brain: las notas de este espacio han pasado a otra versión de formato desde que empezó esta sesión, así que cualquier nota puede haberse reescrito. Lo que leíste antes puede estar desfasado: vuelve a leer (task_get, pending) lo que vayas a afirmar o sobre lo que vayas a actuar.\n",
	},
	"hook.memory.vscode": {
		en: "augmented brain: this VS Code window's extension host (the process that runs Claude and the other extensions of the window) is using %s GB of memory; normal is about 0.5 GB. " +
			"If it keeps growing, the system may kill it, and every Claude session of this window with it. " +
			"This is not a message from the user, and your answer is already given: tell them now, in two or three short lines in the language they write in (English if you cannot tell), " +
			"and suggest parking the task in progress (task_park), so nothing is lost, and going on in a new VS Code window. Then end your turn.",
		es: "augmented brain: el host de extensiones de esta ventana de VS Code (el proceso que ejecuta Claude y las demás extensiones de la ventana) usa %s GB de memoria; lo normal es unos 0,5 GB. " +
			"Si sigue creciendo, el sistema puede matarlo, y con él todas las sesiones de Claude de esta ventana. " +
			"Esto no es un mensaje del usuario, y tu respuesta ya está dada: díselo ahora, en dos o tres líneas cortas en el idioma en que escribe (español si no lo sabes), " +
			"y propón aparcar la tarea en curso (task_park), para no perder nada, y seguir en una ventana nueva de VS Code. Luego termina tu turno.",
	},
	"hook.memory.claude": {
		en: "augmented brain: the Claude process of this session is using %s GB of memory; normal is about 0.5 GB. " +
			"If it keeps growing, the system may kill it, and this session with it. " +
			"This is not a message from the user, and your answer is already given: tell them now, in two or three short lines in the language they write in (English if you cannot tell), " +
			"and suggest parking the task in progress (task_park), so nothing is lost, and going on in a new session. Then end your turn.",
		es: "augmented brain: el proceso de Claude de esta sesión usa %s GB de memoria; lo normal es unos 0,5 GB. " +
			"Si sigue creciendo, el sistema puede matarlo, y con él esta sesión. " +
			"Esto no es un mensaje del usuario, y tu respuesta ya está dada: díselo ahora, en dos o tres líneas cortas en el idioma en que escribe (español si no lo sabes), " +
			"y propón aparcar la tarea en curso (task_park), para no perder nada, y seguir en una sesión nueva. Luego termina tu turno.",
	},
	"hook.model.start": {
		en: "\naugmented brain: this session started with model %s, but the user wants %s (preferred_model in session.toml). " +
			"In one short line at the very start of your first answer, in the language the user writes in (English if you cannot tell), tell them which model is answering and that they can change it with /model or the model picker. Do not change it yourself.\n",
		es: "\naugmented brain: esta sesión ha empezado con el modelo %s, pero el usuario quiere %s (preferred_model en session.toml). " +
			"En una línea corta al principio de tu primera respuesta, en el idioma en que escribe el usuario (español si no lo sabes), dile qué modelo responde y que puede cambiarlo con /model o con el selector de modelo. No lo cambies tú.\n",
	},
	"hook.model.stop": {
		en: "augmented brain: your last answer was written by model %s, but the user wants %s (preferred_model in session.toml). " +
			"This is not a message from the user, and your answer is already given: tell them now, in one or two short lines in the language they write in (English if you cannot tell), which model is answering and that they can change it with /model or the model picker. Do not change it yourself. Then end your turn.",
		es: "augmented brain: tu última respuesta la ha escrito el modelo %s, pero el usuario quiere %s (preferred_model en session.toml). " +
			"Esto no es un mensaje del usuario, y tu respuesta ya está dada: díselo ahora, en una o dos líneas cortas en el idioma en que escribe (español si no lo sabes): qué modelo responde y que puede cambiarlo con /model o con el selector de modelo. No lo cambies tú. Luego termina tu turno.",
	},
	"hook.compact": {
		en: "\nThe conversation was just compacted: its details are gone and only a summary is left, which augmented brain kept in archive/sessions/. " +
			"The summary can be wrong: do not take what it says as fact. Before stating anything from before the compaction (what was done, decided or agreed, what the user allowed), check it against the real state or the notes, or say it is from the summary. " +
			"In one short line at the start of your first answer, in the language the user writes in (English if you cannot tell), tell them the session was compacted and suggest parking the task in progress (task_park) and going on in a new session, which starts from the notes instead of a summary.\n",
		es: "\nLa conversación se acaba de compactar: sus detalles se han perdido y solo queda un resumen, que augmented brain ha guardado en archive/sessions/. " +
			"El resumen puede estar mal: no tomes lo que dice como un hecho. Antes de afirmar algo de antes de la compactación (lo hecho, lo decidido o acordado, lo que el usuario permitió), compruébalo con el estado real o con las notas, o di que viene del resumen. " +
			"En una línea corta al principio de tu primera respuesta, en el idioma en que escribe el usuario (español si no lo sabes), dile que la sesión se ha compactado y propón aparcar la tarea en curso (task_park) y seguir en una sesión nueva, que empieza desde las notas y no desde un resumen.\n",
	},
}
