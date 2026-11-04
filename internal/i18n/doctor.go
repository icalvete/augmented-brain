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
	"doctor.badconfig.fix": {
		en: "fix the file, or move it away and run `augmented-brain init`",
		es: "arregla el fichero, o apártalo y lanza `augmented-brain init`",
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
		en: "close and open those sessions again when it suits you",
		es: "cierra y vuelve a abrir esas sesiones cuando te venga bien",
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
