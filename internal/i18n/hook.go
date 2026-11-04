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
}
