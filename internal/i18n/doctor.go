package i18n

// `augmented-brain doctor`, the health check.
var doctorMessages = map[string]msg{
	"doctor.noconfig": {
		en: "no configuration in %s",
		es: "no hay configuración en %s",
	},
	"doctor.noconfig.fix": {
		en: "run `augmented-brain init`",
		es: "lanza `augmented-brain init`",
	},
	"doctor.nobackup": {
		en: "space %s has no backup: if this computer is lost, so are its notes",
		es: "el espacio %s no tiene copia de seguridad: si pierdes este ordenador, pierdes sus notas",
	},
	"doctor.nobackup.fix": {
		en: "ask Claude \"set up the backup\" in a folder of the space: it guides you step by step (you need a private repository, a place for your files on GitHub or similar, which it helps you create). Or, if you have one, run `augmented-brain backup setup <its address>` there",
		es: "pide a Claude «configura la copia de seguridad» en una carpeta del espacio: te guía paso a paso (hace falta un repositorio privado, un sitio para tus ficheros en GitHub o parecido, y te ayuda a crearlo). O, si ya tienes uno, lanza allí `augmented-brain backup setup <su dirección>`",
	},
	"doctor.badconfig.fix": {
		en: "go back to a version that worked: `augmented-brain config history`, then `augmented-brain config restore <number>`; or fix the file by hand",
		es: "vuelve a una versión que funcionaba: `augmented-brain config history` y después `augmented-brain config restore <número>`; o arregla el fichero a mano",
	},
	"doctor.config": {
		en: "configuration: %s",
		es: "configuración: %s",
	},
	"doctor.blankprofile": {
		en: "profile: not filled in yet; Claude will ask you in your first session",
		es: "perfil: aún sin rellenar; Claude te lo preguntará en tu primera sesión",
	},
	"doctor.profile": {
		en: "profile: %s",
		es: "perfil: %s",
	},
	"doctor.inspace": {
		en: "this folder belongs to space %q",
		es: "esta carpeta es del espacio %q",
	},
	"doctor.default": {
		en: "this folder is outside your spaces' folders: Claude uses the default space, %s",
		es: "esta carpeta está fuera de las de tus espacios: Claude usa el espacio por defecto, %s",
	},
	"doctor.nospace": {
		en: "this folder is outside your spaces; open Claude in one of them: %s",
		es: "esta carpeta está fuera de tus espacios; abre Claude en uno de ellos: %s",
	},
	"doctor.nomemory": {
		en: "space %s: memory folder %s: %v",
		es: "espacio %s: carpeta de la memoria %s: %v",
	},
	"doctor.nomemory.fix": {
		en: "create it, or fix the path in %s",
		es: "créala, o corrige la ruta en %s",
	},
	"doctor.invalid": {
		en: "space %s: %d notes are not valid",
		es: "espacio %s: %d notas no son válidas",
	},
	"doctor.invalid.fix": {
		en: "run `augmented-brain check` inside the space to see them",
		es: "lanza `augmented-brain check` dentro del espacio para verlas",
	},
	"doctor.valid": {
		en: "space %s: %s, all valid",
		es: "espacio %s: %s, todas válidas",
	},
	"doctor.stale": {
		en: "space %s: %s is out of date",
		es: "espacio %s: %s está desfasado",
	},
	"doctor.journals": {
		en: "%s: %d tasks are over %d KB, too long to read in one go: %s",
		es: "%s: %d tareas pasan de %d KB, demasiado largas para leerlas de una vez: %s",
	},
	"doctor.journals.fix": {
		en: "run `augmented-brain check` inside the space: it moves their oldest journal entries to archive/journals/, with a line in the journal saying where",
		es: "lanza `augmented-brain check` dentro del espacio: mueve las entradas más antiguas de su diario a archive/journals/, con una línea en el diario que dice dónde",
	},
	"doctor.stale.fix": {
		en: "run `augmented-brain check` inside the space to rebuild it",
		es: "lanza `augmented-brain check` dentro del espacio para rehacerlo",
	},
	"doctor.tool": {
		en: "%s: %s",
		es: "%s: %s",
	},
	"doctor.notool": {
		en: "%s: not found",
		es: "%s: no encontrado",
	},
	"doctor.claude": {
		en: "Claude Code",
		es: "Claude Code",
	},
	"doctor.claude.fix": {
		en: "install it with: curl -fsSL https://claude.ai/install.sh | bash",
		es: "instálalo con: curl -fsSL https://claude.ai/install.sh | bash",
	},
	"doctor.git": {
		en: "git",
		es: "git",
	},
	"doctor.git.fix": {
		en: "install it with: %s",
		es: "instálalo con: %s",
	},
	"doctor.plugin.unknown": {
		en: "Claude Code plugin: cannot tell (%v)",
		es: "plugin de Claude Code: no se puede saber (%v)",
	},
	"doctor.plugin.missing": {
		en: "Claude Code plugin %s: not installed",
		es: "plugin de Claude Code %s: no instalado",
	},
	"doctor.plugin.ok": {
		en: "Claude Code plugin %s: installed",
		es: "plugin de Claude Code %s: instalado",
	},
	"doctor.guardfile.bad": {
		en: "guard.toml cannot be read, so the guard asks about every commit and push (%s)",
		es: "guard.toml no se puede leer, así que el guardia pregunta por cada commit y push (%s)",
	},
	"doctor.guardfile.fix": {
		en: "fix or remove %s, or open the panel's Guard page",
		es: "arregla o quita %s, o abre la página Guardia del panel",
	},
	"doctor.untouchable": {
		en: "guard: %d untouchable repositories (no commit, push or merge there)",
		es: "guardia: %d repositorios intocables (ni commit, ni push, ni merge ahí)",
	},
	"doctor.untouchable.both": {
		en: "%s is untouchable and also in git_allowed_repos or git_ask_repos: the untouchable list wins, so the other entry does nothing",
		es: "%s es intocable y también está en git_allowed_repos o git_ask_repos: gana la lista de intocables, así que la otra entrada no hace nada",
	},
	"doctor.untouchable.both.fix": {
		en: "take it out of the other list, in the panel's Guard page",
		es: "quítalo de la otra lista, en la página Guardia del panel",
	},
	"doctor.guardchanged": {
		en: "guard: its lists (repositories, root exceptions, profiles, hosts) are not the ones saved when you last started a session or used the panel; something changed them. Until you accept them, a push or merge that the lists allow asks first",
		es: "guardia: sus listas (repositorios, excepciones de root, perfiles, hosts) no son las que se guardaron la última vez que empezaste una sesión o usaste el panel; algo las cambió. Hasta que las aceptes, un push o merge que las listas permiten pregunta antes",
	},
	"doctor.guardchanged.fix": {
		en: "look at the Guard page of the panel and press accept, or start a new session",
		es: "mira la página Guardia del panel y pulsa aceptar, o empieza una sesión nueva",
	},
	"doctor.guard.ok": {
		en: "guard: it blocks a risky command, as it should",
		es: "guardia: bloquea un comando arriesgado, como debe",
	},
	"doctor.guard.fail": {
		en: "guard: a test command that must be blocked was not (%s); risky commands go through unchecked",
		es: "guardia: un comando de prueba que debía bloquear no lo bloqueó (%s); los comandos arriesgados pasan sin revisar",
	},
	"doctor.guard.cannotrun": {
		en: "guard: the program could not check a test command (%s). Until this is fixed, Claude Code stops every command and file change the assistant tries (Claude Code 2.1.295 or later; older versions let them all through unchecked)",
		es: "guardia: el programa no pudo revisar un comando de prueba (%s). Hasta que se arregle, Claude Code para todos los comandos y cambios de ficheros que intente el asistente (Claude Code 2.1.295 o posterior; las versiones anteriores los dejan pasar todos sin revisar)",
	},
	"doctor.guard.fix": {
		en: "make sure `augmented-brain` is on your PATH or in ~/.local/bin (the installer puts it there), then run `augmented-brain doctor` again",
		es: "comprueba que `augmented-brain` está en tu PATH o en ~/.local/bin (el instalador lo pone ahí) y lanza otra vez `augmented-brain doctor`",
	},
	"doctor.lists": {
		en: "guard: %d production AWS profiles and %d untouchable hosts, for every space on this machine",
		es: "guardia: %d perfiles de AWS de producción y %d máquinas intocables, para todos los espacios de esta máquina",
	},
	"doctor.rootask": {
		en: "guard: root on %s asks you, until %s UTC (root_ask)",
		es: "guardia: root en %s te pregunta, hasta el %s UTC (root_ask)",
	},
	"doctor.rootask.expired": {
		en: "guard: the root exception for %s ended on %s UTC: it no longer does anything",
		es: "guardia: la excepción de root de %s terminó el %s UTC: ya no hace nada",
	},
	"doctor.rootask.expired.fix": {
		en: "remove it from root_ask in %s, or give it a new date",
		es: "quítala de root_ask en %s, o ponle otra fecha",
	},
	"doctor.rootask.bad": {
		en: "guard: the root exception %q cannot be read, so it grants nothing",
		es: "guardia: la excepción de root %q no se puede leer, así que no permite nada",
	},
	"doctor.rootask.fix": {
		en: "remove it from root_ask in %s, or write it as \"<host> until <date>\" (2026-10-31, or 2026-10-31T23:59Z)",
		es: "quítala de root_ask en %s, o escríbela como \"<máquina> until <fecha>\" (2026-10-31, o 2026-10-31T23:59Z)",
	},
	"doctor.oldservers": {
		en: "%d Claude sessions open on this computer still run an earlier augmented brain (from before the update to %s): their augmented brain tools may fail",
		es: "%d sesiones de Claude abiertas en este ordenador siguen con un augmented brain anterior (de antes de actualizar a la %s): sus herramientas de augmented brain pueden fallar",
	},
	"doctor.oldservers.fix": {
		en: "augmented-brain session restart --old restarts the idle ones, with the same conversation; those of Claude Desktop stay open, only their augmented brain restarts",
		es: "augmented-brain session restart --old reinicia las que están quietas, con la misma conversación; las de Claude Desktop siguen abiertas, solo se reinicia su augmented brain",
	},
	"doctor.memory": {
		en: "a Claude window on this computer is using %s GB of memory (%s, process %d); normal is about 0.5 GB, and if it keeps growing the system may close it, with every Claude session in it",
		es: "una ventana de Claude en este ordenador usa %s GB de memoria (%s, proceso %d); lo normal es unos 0,5 GB, y si sigue creciendo el sistema puede cerrarla, con todas sus sesiones de Claude",
	},
	"doctor.memory.fix": {
		en: "in that window, park the task in progress and go on in a new one",
		es: "en esa ventana, aparca la tarea en curso y sigue en una nueva",
	},
	"doctor.memory.vscode": {
		en: "VS Code's extension host",
		es: "el host de extensiones de VS Code",
	},
	"doctor.memory.claude": {
		en: "Claude in a terminal",
		es: "Claude en un terminal",
	},
	"doctor.automemory": {
		en: "Claude Code keeps a memory of its own for these folders too (%s): a second memory, next to the space's, that loads at the start and drifts from it",
		es: "Claude Code guarda además una memoria propia para estas carpetas (%s): una segunda memoria, junto a la del espacio, que se carga al empezar y se desvía de ella",
	},
	"doctor.automemory.fix": {
		en: "turn it off: add \"autoMemoryEnabled\": false to %s (what is already in it stays, and you can bring it into the space with `augmented-brain import`)",
		es: "apágala: añade \"autoMemoryEnabled\": false a %s (lo que ya tiene se queda, y puedes traerlo al espacio con `augmented-brain import`)",
	},
	"doctor.exposed": {
		en: "space %s: other users of this computer can read its notes (%s)",
		es: "espacio %s: otros usuarios de este ordenador pueden leer sus notas (%s)",
	},
	"doctor.exposed.fix": {
		en: "run `augmented-brain check` in a folder of the space: it makes them yours only",
		es: "lanza `augmented-brain check` en una carpeta del espacio: las deja solo para ti",
	},
	"doctor.nolists": {
		en: "guard: this machine was restored from a backup, and backups do not carry the guard's lists (production AWS profiles, untouchable hosts) nor the repositories Claude may commit in. Until you give them again, nothing asks before touching production, and Claude commits nowhere",
		es: "guardia: esta máquina se recuperó de una copia de seguridad, y las copias no llevan las listas del guardia (perfiles de AWS de producción, máquinas intocables) ni los repositorios donde Claude puede hacer commit. Hasta que los vuelvas a dar, nada pregunta antes de tocar producción, y Claude no hace commit en ninguno",
	},
	"doctor.nolists.fix": {
		en: "open a Claude session: it offers to ask for them. Or add `production_aws_profiles` and `untouchable_hosts` under `language`, and `git_allowed_repos` in each space, in %s",
		es: "abre una sesión de Claude: se ofrece a preguntártelas. O añade `production_aws_profiles` y `untouchable_hosts` bajo `language`, y `git_allowed_repos` en cada espacio, en %s",
	},
	"doctor.nodefault": {
		en: "no default space: no backup keeps your profile, and sessions outside every space's folder load none",
		es: "no hay espacio por defecto: ninguna copia de seguridad guarda tu perfil, y las sesiones fuera de las carpetas de los espacios no cargan ninguno",
	},
	"doctor.nodefault.fix": {
		en: "add `default_space = \"%s\"` (or another of your spaces) under `language` in %s",
		es: "añade `default_space = \"%s\"` (u otro de tus espacios) debajo de `language` en %s",
	},
}
