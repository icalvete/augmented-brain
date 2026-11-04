package i18n

// What the guard says when it blocks a command or asks the user first. The assistant
// reads it, and the user too when asked: so it is in the user's language.
var guardMessages = map[string]msg{
	"guard.unreadable": {
		en: "the tool call could not be read, so it was not checked; try again.",
		es: "no se pudo leer la llamada a la herramienta, así que no se ha revisado; inténtalo otra vez.",
	},
	"guard.powershell": {
		en: "PowerShell commands are not checked by augmented brain yet, so they are blocked. Run the command with the Bash tool instead (Git Bash on Windows), in POSIX syntax. If it can only be done in PowerShell, give the user the exact command to run themselves, and wait for its output.",
		es: "augmented brain todavía no revisa los comandos de PowerShell, así que se bloquean. Lanza el comando con la herramienta Bash (Git Bash en Windows), con sintaxis POSIX. Si solo se puede hacer en PowerShell, dale al usuario el comando exacto para que lo lance él, y espera a su salida.",
	},
	"guard.secret.command": {
		en: "the command has %s written in it. Read secrets from the system keychain, an environment variable or a secrets manager, never from the command line, and never ask the user to paste one in the chat.",
		es: "el comando contiene %s. Lee los secretos del llavero del sistema, de una variable de entorno o de un gestor de secretos, nunca de la línea de comandos, y nunca pidas al usuario que pegue uno en el chat.",
	},
	"guard.secret.inner": {
		en: "the command has %s written in it.",
		es: "el comando contiene %s.",
	},
	"guard.secret.write": {
		en: "the text has %s in it. Secrets never go into files: the code reads them from an environment variable, the system keychain or a secrets manager.",
		es: "el texto contiene %s. Los secretos nunca van en ficheros: el código los lee de una variable de entorno, del llavero del sistema o de un gestor de secretos.",
	},
	"guard.awskey": {
		en: "the command sets %s with a value written in it. Use an AWS profile (--profile or AWS_PROFILE) instead of keys on the command line.",
		es: "el comando da a %s un valor escrito en él. Usa un perfil de AWS (--profile o AWS_PROFILE) en vez de claves en la línea de comandos.",
	},
	"guard.depth": {
		en: "the command runs commands nested too deep for augmented brain to check; the user confirms it.",
		es: "el comando ejecuta comandos anidados demasiado hondo para que augmented brain los revise; el usuario lo confirma.",
	},
	"guard.unknownprogram": {
		en: "the program this runs is only known when it runs (`%s`), so augmented brain cannot check it; the user confirms it.",
		es: "el programa que esto ejecuta solo se sabe al ejecutarlo (`%s`), así que augmented brain no puede revisarlo; el usuario lo confirma.",
	},
	"guard.script": {
		en: "the script %s runs a command that needs it: %s",
		es: "el script %s ejecuta un comando que lo necesita: %s",
	},
	"guard.protected": {
		en: "%s holds the guard's settings (or Claude Code's): changing it can switch a check off. Only the user changes it; ask them, showing the change.",
		es: "%s guarda los ajustes del guardia (o los de Claude Code): cambiarlo puede apagar una comprobación. Solo lo cambia el usuario; pregúntale, enseñándole el cambio.",
	},

	// Root. The reason why comes last, in guard.root.
	"guard.root": {
		en: "this would run as root. Never become root: give the user the exact command to run themselves, and wait for its output. Running as another user (sudo -u <user>) is fine.",
		es: "esto se ejecutaría como root. Nunca te hagas root: dale al usuario el comando exacto para que lo lance él, y espera a su salida. Ejecutar como otro usuario (sudo -u <usuario>) sí vale.",
	},
	"guard.root.admin": {
		en: "`%s` runs as root (or as the administrator). %s",
		es: "`%s` se ejecuta como root (o como administrador). %s",
	},
	"guard.root.runuser": {
		en: "`runuser` runs as root unless given another user. %s",
		es: "`runuser` se ejecuta como root si no se le da otro usuario. %s",
	},
	"guard.root.systemdrun": {
		en: "`systemd-run` starts the command as a system service, as root unless --user. %s",
		es: "`systemd-run` arranca el comando como un servicio del sistema, como root salvo con --user. %s",
	},
	"guard.root.machinectl": {
		en: "`machinectl %s` opens a shell as root unless given another user. %s",
		es: "`machinectl %s` abre una shell como root si no se le da otro usuario. %s",
	},
	"guard.root.su": {
		en: "`%s` switches to another user, root by default. %s",
		es: "`%s` cambia a otro usuario, root si no se le dice otro. %s",
	},
	"guard.root.ssh": {
		en: "this logs in to %s as root. %s",
		es: "esto entra en %s como root. %s",
	},
	// The machine, then the date and time the exception ends.
	"guard.root.exception": {
		en: "this would run as root on %s, which has a temporary root exception until %s UTC (root_ask). The user confirms each command: say first what it does and why.",
		es: "esto se ejecutaría como root en %s, que tiene una excepción temporal para root hasta el %s UTC (root_ask). El usuario confirma cada comando: di antes qué hace y por qué.",
	},
	"guard.root.thismachine": {
		en: "this machine",
		es: "esta máquina",
	},

	// git, gh and glab.
	"guard.git.config": {
		en: "`git -c %s` makes git run a command of its own, which augmented brain cannot check; the user confirms it.",
		es: "`git -c %s` hace que git ejecute un comando propio, que augmented brain no puede revisar; el usuario lo confirma.",
	},
	"guard.git.publish": {
		en: "`git %s` publishes like a git push, so it is for the user. Leave the change ready and checked, and give the user the exact command to run themselves.",
		es: "`git %s` publica como un git push, así que es cosa del usuario. Deja el cambio listo y comprobado, y dale al usuario el comando exacto para que lo lance él.",
	},
	// git commit or push, then the repository.
	"guard.git.ask": {
		en: "`git %s` in %s needs the user's go-ahead each time. Before asking, show them what would be committed or pushed.",
		es: "`git %s` en %s necesita el visto bueno del usuario cada vez. Antes de preguntar, enséñale lo que entraría en el commit o lo que se subiría.",
	},
	"guard.git.rewrite": {
		en: "this push rewrites or deletes what is on the remote (force, delete or mirror). The user confirms it each time, even in a repository the assistant may push to.",
		es: "este push reescribe o borra lo que hay en el remoto (force, delete o mirror). El usuario lo confirma cada vez, también en un repositorio donde el asistente puede hacer push.",
	},
	"guard.git.deny": {
		en: "`git %s` is for the user. Leave the change ready and checked, show what would be committed, and tell the user it is ready. (The user, not the assistant, can let the assistant commit in a repository by adding it to git_allowed_repos, or to git_ask_repos to be asked each time; then use `git -C <repo>`.)",
		es: "`git %s` es cosa del usuario. Deja el cambio listo y comprobado, enseña lo que entraría en el commit y dile al usuario que está listo. (El usuario, no el asistente, puede dejar que el asistente haga commit en un repositorio añadiéndolo a git_allowed_repos, o a git_ask_repos para que se le pregunte cada vez; entonces usa `git -C <repo>`.)",
	},
	"guard.api": {
		en: "`%s api` with this call merges, releases or moves a branch, like a git push, so it is for the user. Give the user the exact command to run themselves.",
		es: "`%s api` con esta llamada fusiona, publica una versión o mueve una rama, como un git push, así que es cosa del usuario. Dale al usuario el comando exacto para que lo lance él.",
	},
	"guard.publish": {
		en: "`%s` publishes or merges, like a git push, so it is for the user. Leave the change ready and checked, and give the user the exact command to run themselves.",
		es: "`%s` publica o fusiona, como un git push, así que es cosa del usuario. Deja el cambio listo y comprobado, y dale al usuario el comando exacto para que lo lance él.",
	},
	// Merging a pull request: the command, then the repository.
	"guard.merge.ask": {
		en: "`%s` merges a pull request in %s, a repository where that needs the user's go-ahead each time (git_ask_repos). Before asking, show them which pull request it merges and what it changes.",
		es: "`%s` fusiona una pull request en %s, un repositorio donde eso necesita el visto bueno del usuario cada vez (git_ask_repos). Antes de preguntar, enséñale qué pull request fusiona y qué cambia.",
	},
	"guard.merge.cd": {
		en: "`%s` merges a pull request, and goes by git_allowed_repos and git_ask_repos only when the command names the repository, as `git -C` does for a push: write it as `cd <repo> && %s …`, with the repository's path. As it is, it is for the user.",
		es: "`%s` fusiona una pull request, y va por git_allowed_repos y git_ask_repos solo cuando el comando nombra el repositorio, como `git -C` en un push: escríbelo como `cd <repo> && %s …`, con la ruta del repositorio. Tal como está, es para el usuario.",
	},
	"guard.merge.admin": {
		en: "`%s --admin` merges a pull request in %s skipping the branch's protection (its required checks and reviews). The user confirms it each time, even in a repository the assistant may merge in.",
		es: "`%s --admin` fusiona una pull request en %s saltándose la protección de la rama (sus comprobaciones y revisiones obligatorias). El usuario lo confirma cada vez, también en un repositorio donde el asistente puede fusionar.",
	},
	"guard.merge.unread": {
		en: "`%s` names its pull request in a way augmented brain cannot read, so it cannot tell which repository it merges in. Use the number of the pull request, run from the repository's folder.",
		es: "`%s` nombra su pull request de una forma que augmented brain no sabe leer, así que no puede saber en qué repositorio fusiona. Usa el número de la pull request, desde la carpeta del repositorio.",
	},
	// The command, the repository it names, then the folder it runs in.
	"guard.merge.remote": {
		en: "`%s` names %s, which is not a remote of %s, the repository the command runs in. Merging there is for the user: give them the exact command to run themselves.",
		es: "`%s` nombra %s, que no es un remoto de %s, el repositorio donde se ejecuta el comando. Fusionar ahí es cosa del usuario: dale el comando exacto para que lo lance él.",
	},

	// rm -r and find -delete.
	"guard.rm.var": {
		en: "`rm -r %s`: if %s is empty or unset, this deletes from the disk root or the current folder. Check the variable first (for example ${VAR:?}), or the user confirms it.",
		es: "`rm -r %s`: si %s está vacía o sin definir, esto borra desde la raíz del disco o desde la carpeta actual. Comprueba antes la variable (por ejemplo ${VAR:?}), o el usuario lo confirma.",
	},
	"guard.rm.all": {
		en: "`rm -r` of the whole disk is never run.",
		es: "`rm -r` de todo el disco nunca se ejecuta.",
	},
	"guard.rm.lost": {
		en: "`rm -r %s` after a cd the guard cannot follow: it cannot tell which folder this deletes. The user confirms it.",
		es: "`rm -r %s` después de un cd que el guardia no puede seguir: no sabe qué carpeta borra. El usuario lo confirma.",
	},
	"guard.rm.disk": {
		en: "`rm -r` of a whole disk (%s) is never run.",
		es: "`rm -r` de un disco entero (%s) nunca se ejecuta.",
	},
	"guard.rm.abovehome": {
		en: "`rm -r` of %s would delete the user's home folder with it; it is never run.",
		es: "`rm -r` de %s borraría con ella la carpeta personal del usuario; nunca se ejecuta.",
	},
	"guard.rm.home": {
		en: "`rm -r` of the user's whole home folder (%s) is never run. Delete the files or folders inside it that need to go, one by one.",
		es: "`rm -r` de toda la carpeta personal del usuario (%s) nunca se ejecuta. Borra uno a uno los ficheros o carpetas de dentro que sobren.",
	},
	"guard.find": {
		en: "`find` deletes everything it finds there: %s",
		es: "`find` borra todo lo que encuentra ahí: %s",
	},

	// Production AWS profiles and untouchable hosts.
	"guard.aws.env": {
		en: "this runs with the production AWS profile %s; the user confirms first.",
		es: "esto se ejecuta con el perfil de AWS de producción %s; el usuario lo confirma antes.",
	},
	"guard.aws.unknown": {
		en: "the AWS profile (%s) is only known when the command runs, and it may be a production one; the user confirms first.",
		es: "el perfil de AWS (%s) solo se sabe al ejecutar el comando, y puede ser uno de producción; el usuario lo confirma antes.",
	},
	"guard.aws.cli": {
		en: "this AWS command may use the production profile %s; the user confirms first.",
		es: "este comando de AWS puede usar el perfil de producción %s; el usuario lo confirma antes.",
	},
	// The tool (cdk, terraform…), then the profile.
	"guard.aws.tool": {
		en: "%s may run with the production AWS profile %s; the user confirms first.",
		es: "%s puede ejecutarse con el perfil de AWS de producción %s; el usuario lo confirma antes.",
	},
	"guard.aws.vault": {
		en: "aws-vault runs this with the production AWS profile %s; the user confirms first.",
		es: "aws-vault ejecuta esto con el perfil de AWS de producción %s; el usuario lo confirma antes.",
	},
	// The program, then the host.
	"guard.host": {
		en: "%s reaches %s, marked as untouchable; the user confirms first.",
		es: "%s llega a %s, marcado como intocable; el usuario lo confirma antes.",
	},
	// What may reach production (the next two), then why the configuration could not be read.
	"guard.config": {
		en: "%s may reach production or an untouchable host, and augmented brain cannot tell: it cannot read its configuration (%s). The user confirms first; `augmented-brain doctor` shows how to fix the configuration.",
		es: "%s puede llegar a producción o a un host intocable, y augmented brain no puede saberlo: no puede leer su configuración (%s). El usuario lo confirma antes; `augmented-brain doctor` enseña cómo arreglar la configuración.",
	},
	"guard.config.aws": {
		en: "this AWS command",
		es: "este comando de AWS",
	},
	// The program, then the host.
	"guard.config.to": {
		en: "%s to %s",
		es: "%s a %s",
	},
}
