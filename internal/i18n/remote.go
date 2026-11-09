package i18n

// `augmented-brain remote`: Claude kept reachable from another computer, and its line
// in `doctor`.
var remoteMessages = map[string]msg{
	"usage.remote": {
		en: "usage: augmented-brain remote install | uninstall | status",
		es: "uso: augmented-brain remote install | uninstall | status",
	},
	// Said by `session start | stop | list`, which needs systemd; `remote` itself works on
	// Linux, macOS and Windows.
	"remote.unsupported": {
		en: "this is not available on this system yet (Linux only for now).",
		es: "esto aún no está disponible en este sistema (por ahora solo en Linux).",
	},
	"remote.unsupported.os": {
		en: "remote access is available on Linux, macOS and Windows only.",
		es: "el acceso remoto solo está disponible en Linux, macOS y Windows.",
	},
	// What is wrong with remote.toml, or with the machine.
	"remote.badmode": {
		en: "%s: permission_mode %q is not one of %s",
		es: "%s: permission_mode %q no es ninguno de estos: %s",
	},
	"remote.badprefix": {
		en: "%s: name_prefix %q must be one plain word (letters, digits, dot, dash, underscore)",
		es: "%s: name_prefix %q debe ser una palabra sencilla (letras, cifras, punto, guion, guion bajo)",
	},
	"remote.badspace": {
		en: "%s: the space %q is not one of your spaces (%s)",
		es: "%s: el espacio %q no es ninguno de tus espacios (%s)",
	},
	"remote.badfolder": {
		en: "%s: folders: the folder of the space %s, %q, is not a valid path",
		es: "%s: folders: la carpeta del espacio %s, %q, no es una ruta válida",
	},
	"remote.notwork": {
		en: "%s: folders: %s is not inside any work folder of the space %s, so a session there would not load that space. Choose a folder inside one, or add it to the space's work_folders",
		es: "%s: folders: %s no está dentro de ninguna carpeta de trabajo del espacio %s, así que una sesión ahí no cargaría ese espacio. Elige una carpeta dentro de alguna, o añádela a los work_folders del espacio",
	},
	"remote.nofolder": {
		en: "the folder %s, where the session of the space %s starts, does not exist: create it, or choose another under folders in remote.toml",
		es: "la carpeta %s, donde arranca la sesión del espacio %s, no existe: créala, o elige otra en folders de remote.toml",
	},
	"remote.nospaces": {
		en: "no space to serve: write spaces = [\"name\"] in %s, or choose a default space",
		es: "no hay ningún espacio que servir: escribe spaces = [\"nombre\"] en %s, o elige un espacio por defecto",
	},
	"remote.noclaude": {
		en: "Claude Code (the `claude` program) is not in your PATH: install it first",
		es: "Claude Code (el programa `claude`) no está en tu PATH: instálalo primero",
	},
	"remote.nosystemd": {
		en: "remote access needs systemd's user services, and this computer does not offer them (a container, or WSL without systemd): nothing was changed (%v)",
		es: "el acceso remoto necesita los servicios de usuario de systemd, y este ordenador no los tiene (un contenedor, o WSL sin systemd): no se ha cambiado nada (%v)",
	},
	"remote.nolaunchctl": {
		en: "remote access needs launchctl, the part of macOS that starts programs at login, and it was not found: nothing was changed (%v)",
		es: "el acceso remoto necesita launchctl, la parte de macOS que arranca programas al entrar, y no se ha encontrado: no se ha cambiado nada (%v)",
	},
	"remote.launchctl": {
		en: "launchctl %s failed: %v",
		es: "launchctl %s ha fallado: %v",
	},
	"remote.notasks": {
		en: "remote access needs the Task Scheduler of Windows (%s), and it was not found: nothing was changed (%v)",
		es: "el acceso remoto necesita el Programador de tareas de Windows (%s), y no se ha encontrado: no se ha cambiado nada (%v)",
	},
	"remote.schtasks": {
		en: "schtasks %s failed: %v",
		es: "schtasks %s ha fallado: %v",
	},
	"remote.systemctl": {
		en: "systemctl --user %s failed: %v",
		es: "systemctl --user %s ha fallado: %v",
	},
	// remote install.
	"remote.install.done": {
		en: "Remote access is set up. One service per space keeps Claude reachable:",
		es: "El acceso remoto está listo. Un servicio por espacio mantiene Claude accesible:",
	},
	"remote.install.space": {
		en: "  space %s: session %q, started in %s",
		es: "  espacio %s: sesión %q, arrancada en %s",
	},
	"remote.install.added": {
		en: "  %s is now a work folder of the space %s",
		es: "  %s es ahora una carpeta de trabajo del espacio %s",
	},
	"remote.install.trust": {
		en: "Claude does not trust these folders yet, so its server refuses to start in them. Open Claude once in each one and accept the question:",
		es: "Claude aún no confía en estas carpetas, así que su servidor no arranca en ellas. Abre Claude una vez en cada una y acepta la pregunta:",
	},
	"remote.install.trust.cmd": {
		en: "  cd %s && claude",
		es: "  cd %s && claude",
	},
	// PowerShell, the terminal of Windows, does not take && before version 7.
	"remote.install.trust.cmd.win": {
		en: "  cd \"%s\"; claude",
		es: "  cd \"%s\"; claude",
	},
	"remote.install.trust.after": {
		en: "Then close that Claude. The service tries again by itself within a minute.",
		es: "Luego cierra ese Claude. El servicio lo reintenta solo en menos de un minuto.",
	},
	"remote.install.linger": {
		en: "To start after a reboot, with nobody logged in, an administrator must turn on \"linger\". augmented brain does not do it for you; run:",
		es: "Para que arranque tras reiniciar, sin que nadie haya entrado, un administrador debe activar el «linger». augmented brain no lo hace por ti; lanza:",
	},
	"remote.install.linger.unknown": {
		en: "Could not check \"linger\" (loginctl is not here): without it the services only run while you are logged in.",
		es: "No se pudo comprobar el «linger» (aquí no hay loginctl): sin él los servicios solo funcionan mientras tienes la sesión abierta.",
	},
	"remote.install.atlogin": {
		en: "It starts when you log in to this computer, and again by itself if it stops. After a reboot, log in once: until then nobody can connect.",
		es: "Arranca cuando entras en este ordenador, y otra vez solo si se para. Tras reiniciar, entra una vez: hasta entonces nadie puede conectarse.",
	},
	"remote.install.connect": {
		en: "Connect from claude.ai/code or Claude Desktop and choose the environment named %s.",
		es: "Conéctate desde claude.ai/code o Claude Desktop y elige el entorno llamado %s.",
	},
	// remote uninstall.
	"remote.uninstall.done": {
		en: "Stopped and removed: %s.\nThe folders and their work_folders entries stay; delete them by hand if you want.",
		es: "Parado y quitado: %s.\nLas carpetas y sus entradas de work_folders se quedan; bórralas a mano si quieres.",
	},
	"remote.uninstall.none": {
		en: "No remote access service was installed.",
		es: "No había ningún servicio de acceso remoto instalado.",
	},
	// remote status.
	"remote.status.off": {
		en: "Remote access is not set up. Run `augmented-brain remote install`.",
		es: "El acceso remoto no está puesto. Lanza `augmented-brain remote install`.",
	},
	"remote.status.space": {
		en: "space %s, session %q",
		es: "espacio %s, sesión %q",
	},
	"remote.status.service": {
		en: "  service: %s, %s",
		es: "  servicio: %s, %s",
	},
	"remote.status.folder": {
		en: "  folder: %s, %s",
		es: "  carpeta: %s, %s",
	},
	"remote.status.linger": {
		en: "linger (start at boot without a login): %s",
		es: "linger (arrancar sin que nadie entre): %s",
	},
	"remote.status.atlogin": {
		en: "it starts when you log in to this computer (after a reboot, log in once)",
		es: "arranca cuando entras en este ordenador (tras reiniciar, entra una vez)",
	},
	"remote.status.fix": {
		en: "  → %s",
		es: "  → %s",
	},
	"remote.state.active":     {en: "running", es: "en marcha"},
	"remote.state.inactive":   {en: "stopped", es: "parado"},
	"remote.state.failed":     {en: "failed", es: "ha fallado"},
	"remote.state.activating": {en: "starting", es: "arrancando"},
	"remote.state.missing":    {en: "not installed", es: "no instalado"},
	"remote.state.other":      {en: "state: %s", es: "estado: %s"},
	"remote.enabled.enabled":  {en: "starts at login", es: "arranca al entrar"},
	"remote.enabled.other":    {en: "does not start by itself", es: "no arranca solo"},
	"remote.trust.yes":        {en: "trusted by Claude", es: "Claude confía en ella"},
	"remote.trust.no":         {en: "not trusted by Claude yet", es: "Claude aún no confía en ella"},
	"remote.yes":              {en: "yes", es: "sí"},
	"remote.no":               {en: "no", es: "no"},
	"remote.unknown":          {en: "unknown", es: "no se sabe"},
	// What to do about each problem.
	"remote.fix.install": {
		en: "run `augmented-brain remote install`",
		es: "lanza `augmented-brain remote install`",
	},
	"remote.fix.trust": {
		en: "run `cd %s && claude` once and accept the question",
		es: "lanza `cd %s && claude` una vez y acepta la pregunta",
	},
	"remote.fix.trust.win": {
		en: "run `cd \"%s\"; claude` once and accept the question",
		es: "lanza `cd \"%s\"; claude` una vez y acepta la pregunta",
	},
	"remote.fix.restart": {
		en: "see why with `journalctl --user -u %[1]s -n 20` (or `systemctl --user status %[1]s`); if Claude's login expired, run `claude` and sign in",
		es: "mira por qué con `journalctl --user -u %[1]s -n 20` (o `systemctl --user status %[1]s`); si caducó el acceso de Claude, lanza `claude` e inicia sesión",
	},
	"remote.fix.restart.log": {
		en: "see why in %s; if Claude's login expired, run `claude` and sign in. `augmented-brain remote install` starts it again",
		es: "mira por qué en %s; si caducó el acceso de Claude, lanza `claude` e inicia sesión. `augmented-brain remote install` lo vuelve a arrancar",
	},
	"remote.fix.update": {
		en: "run `augmented-brain remote install` again to bring the service up to date: its sessions then find claude and augmented-brain, and when it fails it says why in the system journal (it restarts remote access)",
		es: "vuelve a lanzar `augmented-brain remote install` para poner el servicio al día: así sus sesiones encuentran claude y augmented-brain, y cuando falla dice por qué en el registro del sistema (reinicia el acceso remoto)",
	},
	"remote.fix.linger": {
		en: "an administrator runs `%s`",
		es: "un administrador lanza `%s`",
	},
	// The line of `doctor`.
	"remote.doctor": {
		en: "remote access, space %s (session %q): service %s, folder %s, linger %s",
		es: "acceso remoto, espacio %s (sesión %q): servicio %s, carpeta %s, linger %s",
	},
	"remote.doctor.atlogin": {
		en: "remote access, space %s (session %q): service %s, folder %s, starts at login",
		es: "acceso remoto, espacio %s (sesión %q): servicio %s, carpeta %s, arranca al entrar",
	},
	// doctor's line about the remote panel (panel remote), once it has been used.
	"doctor.panelremote.ok": {
		en: "remote panel: Tailscale is ready (%s)",
		es: "panel remoto: Tailscale está listo (%s)",
	},
	"doctor.panelremote.missing": {
		en: "remote panel: Tailscale is no longer installed",
		es: "panel remoto: Tailscale ya no está instalado",
	},
	"doctor.panelremote.missing.fix": {
		en: "install it again to open the panel from your other devices: https://tailscale.com/download",
		es: "vuelve a instalarlo para abrir el panel desde tus otros dispositivos: https://tailscale.com/download",
	},
	"doctor.panelremote.stopped": {
		en: "remote panel: Tailscale is not connected",
		es: "panel remoto: Tailscale no está conectado",
	},
	"doctor.panelremote.stopped.fix": {
		en: "open Tailscale and sign in (or run: tailscale up)",
		es: "abre Tailscale e inicia sesión (o lanza: tailscale up)",
	},
	"doctor.panelremote.nohttps": {
		en: "remote panel: your Tailscale network no longer gives HTTPS certificates",
		es: "panel remoto: tu red de Tailscale ya no da certificados HTTPS",
	},
	"doctor.panelremote.nohttps.fix": {
		en: "turn them on in https://login.tailscale.com/admin/dns → HTTPS Certificates",
		es: "actívalos en https://login.tailscale.com/admin/dns → HTTPS Certificates",
	},
	"doctor.panelremote.leftover": {
		en: "remote panel: port %d of your Tailscale network still points at %s, but no remote panel is running",
		es: "panel remoto: el puerto %d de tu red de Tailscale sigue apuntando a %s, pero no hay ningún panel remoto en marcha",
	},
	"doctor.panelremote.leftover.fix": {
		en: "take it back: tailscale serve --https=%d off (starting the panel again does it too)",
		es: "quítalo: tailscale serve --https=%d off (también lo hace volver a arrancar el panel)",
	},
	"remote.doctor.bad": {
		en: "remote access: %v",
		es: "acceso remoto: %v",
	},
	"remote.doctor.bad.fix": {
		en: "fix remote.toml in the configuration folder",
		es: "arregla remote.toml en la carpeta de configuración",
	},
}
