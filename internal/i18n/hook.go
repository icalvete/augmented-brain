package i18n

// What the hooks tell the assistant in the language of the configuration. These are
// instructions to the assistant, which passes them on in the user's language.
var hookMessages = map[string]msg{
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
