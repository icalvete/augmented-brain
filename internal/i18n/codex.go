package i18n

// The Codex integration: what its session start tells the assistant, on top of the
// space's text, about what is different from Claude Code.
var codexMessages = map[string]msg{
	// What the guard does in Codex. The session start cannot tell whether the user
	// registered it (`hook pre-tool-use codex`), so it says both cases.
	"codex.guard": {
		en: "The command guard runs here only if the user registered it as Codex's PreToolUse hook, and then it never asks: what the rules say it asks about, it refuses. Either way, keep the safety rules yourself; where they say the guard asks or a hook blocks, take it as refused and give the user the command to run instead.",
		es: "El guardia de comandos solo actúa aquí si el usuario lo registró como hook PreToolUse de Codex, y entonces nunca pregunta: lo que según las reglas pregunta, lo rechaza. En cualquier caso, cumple tú las reglas de seguridad; donde digan que el guardia pregunta o que un hook bloquea, tómalo como prohibido y dale al usuario el comando para que lo ejecute él.",
	},
	"codex.note": {
		en: "\n## In Codex\n\nThis session runs in Codex, not in Claude Code. What changes from the rules:\n- %s\n- Your augmented brain tools are only %s, and only in this space. For anything else the rules name (capture, knowledge_new, rule_new, task_close, task_worktree, another space), tell the user to do it from a Claude session or the panel.\n- There is no SendMessage nor PushNotification: tell the user in this chat.\n",
		es: "\n## En Codex\n\nEsta sesión corre en Codex, no en Claude Code. Lo que cambia respecto a las reglas:\n- %s\n- Tus herramientas de augmented brain son solo %s, y solo en este espacio. Para lo demás que nombren las reglas (capture, knowledge_new, rule_new, task_close, task_worktree, otro espacio), dile al usuario que lo haga desde una sesión de Claude o el panel.\n- No hay SendMessage ni PushNotification: díselo al usuario en este chat.\n",
	},
	"codex.nospace": {
		en: "augmented brain: %s belongs to no space (spaces: %s), so no memory is loaded and its tools refuse to work here. Tell the user; to use it, open Codex in a space's work folder or notes folder.",
		es: "augmented brain: %s no es de ningún espacio (espacios: %s), así que no se carga la memoria y sus herramientas no funcionan aquí. Díselo al usuario; para usarla, abre Codex en la carpeta de trabajo o de notas de un espacio.",
	},
}
