package i18n

// `augmented-brain init` and `space add`: the questions and what comes after.
var initMessages = map[string]msg{
	"init.space": {
		en: "Space name (one part of your life: work, personal…)",
		es: "Nombre del espacio (una parte de tu vida: work, personal…)",
	},
	"init.work": {
		en: "Work folder: opening your assistant here selects this space",
		es: "Carpeta de trabajo: abrir tu asistente aquí elige este espacio",
	},
	"init.memory": {
		en: "Memory folder: where this space's notes will live",
		es: "Carpeta de la memoria: donde vivirán las notas de este espacio",
	},
	"init.created": {
		en: "Created:",
		es: "Creado:",
	},
	"init.config": {
		en: "Configuration:",
		es: "Configuración:",
	},
	"init.added": {
		en: "Space %q added.",
		es: "Espacio %q añadido.",
	},
	"init.default": {
		en: "%q is now your default space: it is used outside every space's folder, and its backup keeps your profile.",
		es: "%q es ahora tu espacio por defecto: se usa fuera de las carpetas de los espacios, y su copia de seguridad guarda tu perfil.",
	},
	"init.plugin?": {
		en: "Install the Claude Code plugin now? (yes, no)",
		es: "¿Instalo ahora el plugin de Claude Code? (sí, no)",
	},
	"init.yes": {
		en: "yes",
		es: "sí",
	},
	"init.plugin": {
		en: "Claude Code plugin installed. It loads your space when a session starts, and guards commands and file writes.",
		es: "Plugin de Claude Code instalado. Carga tu espacio al empezar cada sesión y vigila los comandos y las escrituras de ficheros.",
	},
	"init.noclaude": {
		en: "Claude Code was not found. Install it, then add the plugin with:",
		es: "No encuentro Claude Code. Instálalo y después añade el plugin con:",
	},
	"init.notthere": {
		en: "Claude Code finished, but the plugin is not in its list.",
		es: "Claude Code terminó, pero el plugin no está en su lista.",
	},
	"init.noplugin": {
		en: "Warning: the Claude Code plugin is not installed. Until it is, Claude does not load your notes and nothing guards its commands. Install it with the command above; `augmented-brain doctor` checks it.",
		es: "Atención: el plugin de Claude Code no está instalado. Hasta que lo esté, Claude no carga tus notas y nada vigila sus comandos. Instálalo con el comando de arriba; `augmented-brain doctor` lo comprueba.",
	},
	"init.later": {
		en: "To install the Claude Code plugin later:",
		es: "Para instalar el plugin de Claude Code más adelante:",
	},
	"init.ready": {
		en: "augmented brain is ready. Your spaces:",
		es: "augmented brain está listo. Tus espacios:",
	},
	"init.now": {
		en: "Now open Claude in your work space:\n  cd %s && claude\nThe first time, it will ask who you are and how you like to work.",
		es: "Ahora abre Claude en tu espacio de trabajo:\n  cd %s && claude\nLa primera vez te preguntará quién eres y cómo te gusta trabajar.",
	},
	"init.newterm": {
		en: "Now open a new terminal (so it finds what was just installed) and type:\n  cd %s && claude\nThe first time, Claude will ask who you are and how you like to work.",
		es: "Ahora abre una terminal nueva (para que encuentre lo recién instalado) y escribe:\n  cd %s && claude\nLa primera vez, Claude te preguntará quién eres y cómo te gusta trabajar.",
	},
	"init.newterm.note": {
		en: "This terminal does not find augmented-brain yet: for its commands, open a new one.",
		es: "Esta terminal todavía no encuentra augmented-brain: para sus comandos, abre una nueva.",
	},
	"init.launch": {
		en: "From now on, to work: open a new terminal and type\n  cd %s && claude\nThe first time, Claude will ask who you are and how you like to work.",
		es: "A partir de ahora, para trabajar: abre una terminal nueva y escribe\n  cd %s && claude\nLa primera vez, Claude te preguntará quién eres y cómo te gusta trabajar.",
	},
	"init.desktop": {
		en: "From now on, to work: open the Claude app, go to the Code tab and click + New. Without choosing a folder you work in your personal space; for the work one, pick the folder %s with Choose….\n(Or, in a new terminal: cd %[1]s && claude)\nThe first time, sign in with your account; Claude will ask who you are and how you like to work.",
		es: "A partir de ahora, para trabajar: abre la app de Claude, ve a la pestaña Code y pulsa + New. Sin elegir carpeta trabajas en tu espacio personal; para el de trabajo, elige con Choose… la carpeta %s.\n(O, en una terminal nueva: cd %[1]s && claude)\nLa primera vez, entra con tu cuenta; Claude te preguntará quién eres y cómo te gusta trabajar.",
	},
	"init.space.work": {
		en: "work",
		es: "trabajo",
	},
	"init.notes": {
		en: "notes",
		es: "notas",
	},
}
