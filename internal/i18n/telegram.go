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
}
