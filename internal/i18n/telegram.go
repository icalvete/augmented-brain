package i18n

// augmented-brain telegram: the assistant on the person's own Telegram account, from
// the terminal and from the panel.
var telegramMessages = map[string]msg{
	"usage.telegram": {
		en: "usage: augmented-brain telegram connect | status | disconnect [--forget]",
		es: "uso: augmented-brain telegram connect | status | disconnect [--forget]",
	},
	"telegram.unsupported": {
		en: "Telegram needs the system keychain to keep its session, and this computer has none augmented brain can use (Linux with secret-tool, or macOS). Windows is not supported yet.",
		es: "Telegram necesita el llavero del sistema para guardar su sesión, y este ordenador no tiene uno que augmented brain sepa usar (Linux con secret-tool, o macOS). Windows aún no está soportado.",
	},
	"telegram.app.intro": {
		en: "First, your own Telegram application (a minute, once):\n  1. Open https://my.telegram.org and log in with your phone number (the code arrives in Telegram).\n  2. Go to \"API development tools\" and create an application: any title, Desktop platform.\n  3. Copy its api_id and api_hash here. They go to the keychain, never to a file.",
		es: "Primero, tu propia aplicación de Telegram (un minuto, una sola vez):\n  1. Abre https://my.telegram.org y entra con tu número de teléfono (el código te llega a Telegram).\n  2. Entra en «API development tools» y crea una aplicación: cualquier título, plataforma Desktop.\n  3. Copia aquí su api_id y su api_hash. Van al llavero, nunca a un fichero.",
	},
	"telegram.ask.id": {
		en: "api_id: ",
		es: "api_id: ",
	},
	"telegram.ask.hash": {
		en: "api_hash (not shown as you type): ",
		es: "api_hash (no se ve al escribir): ",
	},
	"telegram.ask.password": {
		en: "Your account has two-step verification. Its password (not shown, not saved): ",
		es: "Tu cuenta tiene verificación en dos pasos. Su contraseña (no se ve ni se guarda): ",
	},
	"telegram.noterminal": {
		en: "telegram connect needs a terminal to ask for the application and to show the QR code. Or connect from the panel: augmented-brain panel → Telegram.",
		es: "telegram connect necesita un terminal para pedir la aplicación y enseñar el código QR. O conéctalo desde el panel: augmented-brain panel → Telegram.",
	},
	"telegram.qr": {
		en: "Scan this code with your phone: Telegram → Settings → Devices → Link Desktop Device. It changes every 30 seconds or so; scan the newest one.",
		es: "Escanea este código con el móvil: Telegram → Ajustes → Dispositivos → Vincular dispositivo de escritorio. Cambia cada 30 segundos o así; escanea el último.",
	},
	"telegram.connected": {
		en: "Connected as %s. Start a new Claude session to use it. To end it: augmented-brain telegram disconnect, or close \"augmented brain\" in Telegram → Settings → Devices.",
		es: "Conectado como %s. Abre una sesión nueva de Claude para usarlo. Para cerrarlo: augmented-brain telegram disconnect, o cierra «augmented brain» en Telegram → Ajustes → Dispositivos.",
	},
	"telegram.status.on": {
		en: "Telegram: connected as %s.",
		es: "Telegram: conectado como %s.",
	},
	"telegram.status.off": {
		en: "Telegram: not connected. To connect: augmented-brain telegram connect, or the panel's Telegram page.",
		es: "Telegram: no conectado. Para conectarlo: augmented-brain telegram connect, o la página Telegram del panel.",
	},
	"telegram.disconnected": {
		en: "Disconnected: the session is closed in Telegram and gone from the keychain.",
		es: "Desconectado: la sesión está cerrada en Telegram y fuera del llavero.",
	},
	"telegram.forgot": {
		en: " The application (api_id, api_hash) is gone from the keychain too.",
		es: " La aplicación (api_id, api_hash) también está fuera del llavero.",
	},
	"telegram.installing": {
		en: "Downloading augmented brain's Telegram program (about 6 MB, once)…",
		es: "Descargando el programa de Telegram de augmented brain (unos 6 MB, una sola vez)…",
	},
	"telegram.installfailed": {
		en: "Could not install the Telegram program: %v",
		es: "No se ha podido instalar el programa de Telegram: %v",
	},
	"telegram.timeout": {
		en: "Nobody scanned the code in %s. Run it again when you have the phone at hand.",
		es: "Nadie ha escaneado el código en %s. Vuelve a lanzarlo cuando tengas el móvil a mano.",
	},

	// The panel's Telegram page.
	"panel.telegram": {en: "Telegram", es: "Telegram"},
	"panel.telegram.hint": {
		en: "Connect your own Telegram account, as one more device of it, so your assistant can read a chat when a task needs it and write from your account. To anyone but you, it shows you the text first. The session is your whole account: it stays in this computer's keychain.",
		es: "Conecta tu propia cuenta de Telegram, como un dispositivo más, para que tu asistente pueda leer un chat cuando una tarea lo necesite y escribir desde tu cuenta. A cualquiera que no seas tú, te enseña antes el texto. La sesión es tu cuenta entera: se queda en el llavero de este ordenador.",
	},
	"panel.telegram.unsupported": {
		en: "Telegram needs a keychain augmented brain can use: Linux (with secret-tool) or macOS. Windows is not supported yet.",
		es: "Telegram necesita un llavero que augmented brain sepa usar: Linux (con secret-tool) o macOS. Windows aún no está soportado.",
	},
	"panel.telegram.localonly": {
		en: "Connecting Telegram is done only on the computer itself, not through panel remote: it hands over your whole account.",
		es: "Telegram solo se conecta en el propio ordenador, no a través del panel remoto: entrega tu cuenta entera.",
	},
	"panel.telegram.install.hint": {
		en: "First, augmented brain's Telegram program: about 6 MB, downloaded once from this version's release and checked against its checksums.",
		es: "Primero, el programa de Telegram de augmented brain: unos 6 MB, descargado una sola vez de esta versión y comprobado con sus sumas de verificación.",
	},
	"panel.telegram.install": {en: "Download the Telegram program", es: "Descargar el programa de Telegram"},
	"panel.telegram.installfailed": {
		en: "Could not install the Telegram program: %v",
		es: "No se ha podido instalar el programa de Telegram: %v",
	},
	"panel.telegram.notinstalled": {
		en: "The Telegram program is not installed: download it first.",
		es: "El programa de Telegram no está instalado: descárgalo primero.",
	},
	"panel.telegram.app.1": {
		en: "Open my.telegram.org and log in with your phone number, in international format (+34…). The code arrives in Telegram itself.",
		es: "Abre my.telegram.org y entra con tu número de teléfono, en formato internacional (+34…). El código te llega al propio Telegram.",
	},
	"panel.telegram.app.2": {
		en: "Go to \"API development tools\" and create an application: App title \"augmented brain\", Short name \"abrain\", Platform Desktop.",
		es: "Entra en «API development tools» y crea una aplicación: App title «augmented brain», Short name «abrain», Platform Desktop.",
	},
	"panel.telegram.app.3": {
		en: "Copy its api_id and api_hash below.",
		es: "Copia aquí abajo su api_id y su api_hash.",
	},
	"panel.telegram.app.hint": {
		en: "They go to this computer's keychain, never to a file. Then comes the QR code.",
		es: "Van al llavero de este ordenador, nunca a un fichero. Después viene el código QR.",
	},
	"panel.telegram.app.save": {en: "Save and show the QR code", es: "Guardar y enseñar el código QR"},
	"panel.telegram.login.hint": {
		en: "Your application is saved. Log in by scanning a QR code with your phone.",
		es: "Tu aplicación está guardada. Entra escaneando un código QR con el móvil.",
	},
	"panel.telegram.login": {en: "Show the QR code", es: "Enseñar el código QR"},
	"panel.telegram.scan.1": {
		en: "On your phone: Telegram → Settings → Devices → Link Desktop Device.",
		es: "En el móvil: Telegram → Ajustes → Dispositivos → Vincular dispositivo de escritorio.",
	},
	"panel.telegram.scan.2": {
		en: "Scan this code. It changes every 30 seconds or so; this page follows it.",
		es: "Escanea este código. Cambia cada 30 segundos o así; esta página lo sigue.",
	},
	"panel.telegram.wait":     {en: "Asking Telegram for the code…", es: "Pidiendo el código a Telegram…"},
	"panel.telegram.cancel":   {en: "Cancel", es: "Cancelar"},
	"panel.telegram.password": {en: "Two-step verification password", es: "Contraseña de la verificación en dos pasos"},
	"panel.telegram.password.hint": {
		en: "Your account has two-step verification. The password goes to Telegram once and is not kept anywhere.",
		es: "Tu cuenta tiene verificación en dos pasos. La contraseña va a Telegram una vez y no se guarda en ningún sitio.",
	},
	"panel.telegram.password.send": {en: "Log in", es: "Entrar"},
	"panel.f.password":             {en: "password", es: "contraseña"},
	"panel.telegram.on":            {en: "Connected", es: "Conectado"},
	"panel.telegram.use": {
		en: "Start a new Claude session to use it: ask, for example, \"what did Lucía say in the family group?\".",
		es: "Abre una sesión nueva de Claude para usarlo: pide, por ejemplo, «¿qué ha dicho Lucía en el grupo de la familia?».",
	},
	"panel.telegram.disconnect": {en: "Disconnect", es: "Desconectar"},
	"panel.telegram.disconnect.hint": {
		en: "Ends the session in Telegram (\"augmented brain\" leaves Settings → Devices) and takes it out of the keychain.",
		es: "Cierra la sesión en Telegram («augmented brain» sale de Ajustes → Dispositivos) y la saca del llavero.",
	},
	"panel.telegram.forget": {
		en: "Also forget the application (api_id, api_hash)",
		es: "Olvidar también la aplicación (api_id, api_hash)",
	},
	"panel.telegram.disconnect.confirm": {en: "Yes, disconnect Telegram", es: "Sí, desconectar Telegram"},
	"panel.ok.telegram.app":             {en: "Application saved.", es: "Aplicación guardada."},
	"panel.ok.telegram.install":         {en: "Telegram program installed.", es: "Programa de Telegram instalado."},
	"panel.ok.telegram.disconnect":      {en: "Telegram disconnected.", es: "Telegram desconectado."},
}
