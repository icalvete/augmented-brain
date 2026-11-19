package i18n

// The Codex integration: what its session start tells the assistant, on top of the
// space's text, about what is different from Claude Code.
var codexMessages = map[string]msg{
	// What the guard does in Codex. The session start cannot tell whether the guard is
	// registered (`codex install`, or `hook pre-tool-use codex` by hand), so it says both cases.
	"codex.guard": {
		en: "The command guard runs here when augmented brain is installed in Codex (augmented-brain codex install), and it never asks: what the rules say it asks about, it refuses. Either way, keep the safety rules yourself; where they say the guard asks or a hook blocks, take it as refused and give the user the command to run instead.",
		es: "El guardia de comandos actúa aquí cuando augmented brain está instalado en Codex (augmented-brain codex install), y nunca pregunta: lo que según las reglas pregunta, lo rechaza. En cualquier caso, cumple tú las reglas de seguridad; donde digan que el guardia pregunta o que un hook bloquea, tómalo como prohibido y dale al usuario el comando para que lo ejecute él.",
	},
	"codex.note": {
		en: "\n## In Codex\n\nThis session runs in Codex, not in Claude Code. What changes from the rules:\n- %s\n- Your augmented brain tools are only %s, and only in this space. For anything else the rules name (capture, knowledge_new, rule_new, task_close, task_worktree, another space), tell the user to do it from a Claude session or the panel.\n- There is no SendMessage nor PushNotification: tell the user in this chat.\n",
		es: "\n## En Codex\n\nEsta sesión corre en Codex, no en Claude Code. Lo que cambia respecto a las reglas:\n- %s\n- Tus herramientas de augmented brain son solo %s, y solo en este espacio. Para lo demás que nombren las reglas (capture, knowledge_new, rule_new, task_close, task_worktree, otro espacio), dile al usuario que lo haga desde una sesión de Claude o el panel.\n- No hay SendMessage ni PushNotification: díselo al usuario en este chat.\n",
	},
	"codex.nospace": {
		en: "augmented brain: %s belongs to no space (spaces: %s), so no memory is loaded and its tools refuse to work here. Tell the user; to use it, open Codex in a space's work folder or notes folder.",
		es: "augmented brain: %s no es de ningún espacio (espacios: %s), así que no se carga la memoria y sus herramientas no funcionan aquí. Díselo al usuario; para usarla, abre Codex en la carpeta de trabajo o de notas de un espacio.",
	},

	"usage.codex": {
		en: "usage: augmented-brain codex install | uninstall | status",
		es: "uso: augmented-brain codex install | uninstall | status",
	},
	"codex.missing": {
		en: "Codex is not installed (there is no codex command on the PATH). Install Codex first, then run augmented-brain codex install.",
		es: "Codex no está instalado (no hay orden codex en el PATH). Instala antes Codex y después ejecuta augmented-brain codex install.",
	},
	"codex.installing": {
		en: "Installing augmented brain in Codex: the plugin %s (the space at the start, the guard and the task tools).\nCodex runs a hook only once you trust it: installing trusts augmented brain's hooks for you, and turns off unified_exec_tty (without it, a terminal escapes the guard).",
		es: "Instalando augmented brain en Codex: el plugin %s (el espacio al empezar, el guardia y las herramientas de tareas).\nCodex solo ejecuta un hook en el que confías: instalar da esa confianza a los hooks de augmented brain por ti, y apaga unified_exec_tty (sin eso, un terminal se salta el guardia).",
	},
	"codex.installed": {
		en: "Done. Open a new Codex session in a space's folder: it starts with your space, and the guard checks its commands. In Codex the guard never asks: it refuses.",
		es: "Hecho. Abre una sesión nueva de Codex en la carpeta de un espacio: empieza con tu espacio y el guardia revisa sus comandos. En Codex el guardia nunca pregunta: rechaza.",
	},
	"codex.uninstalled": {
		en: "augmented brain is no longer in Codex: its plugin and the trust in its hooks are gone. unified_exec_tty stays off; it only protects (to turn it back on, remove its line from Codex's config.toml).",
		es: "augmented brain ya no está en Codex: se han quitado su plugin y la confianza en sus hooks. unified_exec_tty sigue apagado; solo protege (para volver a encenderlo, quita su línea del config.toml de Codex).",
	},
	"codex.status.ok": {
		en: "augmented brain is installed in Codex: plugin, %d trusted hooks and unified_exec_tty off.",
		es: "augmented brain está instalado en Codex: plugin, %d hooks de confianza y unified_exec_tty apagado.",
	},
	"codex.status.none": {
		en: "augmented brain is not installed in Codex. To install it: augmented-brain codex install",
		es: "augmented brain no está instalado en Codex. Para instalarlo: augmented-brain codex install",
	},
	"codex.status.hooks": {
		en: "Codex runs %d of augmented brain's %d hooks: without them there is no space at the start, or no guard.",
		es: "Codex ejecuta %d de los %d hooks de augmented brain: sin ellos no hay espacio al empezar, o no hay guardia.",
	},
	"codex.status.modified": {
		en: "augmented brain's hooks changed since Codex trusted them (a new version), so Codex no longer runs them.",
		es: "Los hooks de augmented brain han cambiado desde que Codex confió en ellos (una versión nueva), así que Codex ya no los ejecuta.",
	},
	"codex.status.tty": {
		en: "unified_exec_tty is on in Codex: the assistant can open a terminal and type commands the guard never sees.",
		es: "unified_exec_tty está encendido en Codex: el asistente puede abrir un terminal y escribir comandos que el guardia no ve.",
	},
	"codex.status.error": {
		en: "Could not ask Codex how augmented brain stands there: %v",
		es: "No se ha podido preguntar a Codex cómo está augmented brain allí: %v",
	},
	"codex.status.fix": {
		en: "To fix it: %s",
		es: "Para arreglarlo: %s",
	},
	"doctor.codex.ok": {
		en: "Codex: augmented brain is installed (%s), its %d hooks are trusted and unified_exec_tty is off",
		es: "Codex: augmented brain está instalado (%s), sus %d hooks son de confianza y unified_exec_tty está apagado",
	},
	"doctor.codex.none": {
		en: "Codex is on this computer, without augmented brain. To carry on a task there when Claude's tokens run out: augmented-brain codex install",
		es: "Codex está en este ordenador, sin augmented brain. Para seguir una tarea allí cuando se acaban los tokens de Claude: augmented-brain codex install",
	},
}
