# Instalación

[Read in English](install.md)

Un solo comando instala augmented brain y todo lo que necesita, en Linux, macOS y Windows.
Esta página cuenta qué hace ese comando en cada sistema, dónde deja cada cosa y qué cambia
en cada uno. ¿Nunca has usado la terminal? Los [primeros pasos](getting-started.es.md) te
llevan de la mano.

## En todos los sistemas

- **Un plan de Claude de pago** (Pro, Max, Team o Enterprise) o una clave de la API de
  Anthropic. augmented brain es gratis, pero trabaja con Claude Code, y el plan gratuito no
  incluye Claude Code. El instalador te lo dice antes de instalarlo. Los planes:
  [claude.com/pricing](https://claude.com/pricing).
- **Sin administrador ni root.** El programa va a tu propia carpeta. Solo si hay que
  instalar algo que falta, como git, puede pedirte la contraseña.
- **El instalador pregunta antes.** El idioma y, después, cada pieza que falte, con para
  qué sirve. Pulsa Enter para aceptar lo que propone.
- **Siempre acaba igual:** dos espacios, `trabajo` y `personal` (en inglés, `work` y
  `personal`), en `~/augmented-brain`, el plugin de Claude Code instalado y la oferta de
  abrir Claude.
- **Todo lo que descarga se comprueba.** El programa sale de la última versión publicada
  del proyecto en GitHub, y su SHA-256 tiene que coincidir con la lista de esa versión; si
  no, no se instala nada. Cuando GitHub no lo entrega, el instalador lo intenta tres
  veces, también por la API de GitHub; si falla todo, lo dice y no instala nada.

## Linux

**Necesitas:** un PC de 64 bits o un Arm (amd64 o arm64), `curl` o `wget`, y `sha256sum`
o `shasum` (las distribuciones habituales los traen).

**Instalar:** abre una terminal (en Ubuntu, `Ctrl + Alt + T`) y pega:

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
```

**Qué hace:**

1. **git**, si falta, con el gestor de paquetes de tu sistema: `apt` (Ubuntu, Debian), `dnf`
   (Fedora), `pacman` (Arch) o `zypper` (openSUSE). Te pide la contraseña.
2. **Claude Code**, si falta, con el instalador de Anthropic, en `~/.local/bin`. Si ya lo
   tienes pero va atrasado, te ofrece actualizarlo (`claude update`).
3. **La app de escritorio de Claude**, si la quieres, **solo en sistemas con `apt`** (Ubuntu,
   Debian y sus derivadas): desde el
   repositorio apt de Anthropic, que añade por ti; te pide la contraseña. En otras
   distribuciones usas Claude en la terminal.
4. **El programa**, en `~/.local/bin/augmented-brain`.
5. **El PATH:** añade `~/.local/bin` a `~/.bashrc` con bash, a `~/.zshrc` con zsh, o a
   `~/.profile` en otro caso, salvo que ya esté.
6. **La configuración**, y la oferta de abrir Claude.

**Dónde queda cada cosa:** el programa y Claude Code en `~/.local/bin`; la configuración en
`~/.config/augmented-brain` (o `$XDG_CONFIG_HOME/augmented-brain`); tus notas en
`~/augmented-brain/<espacio>/notas`.

**Conviene saber:**

- Después de instalar, **abre una terminal nueva**: la que usaste todavía no ve el PATH
  nuevo.
- Para trabajar sin la app: `cd ~/augmented-brain/trabajo && claude`.

## macOS

**Necesitas:** **macOS 13 (Ventura) o posterior**, que es lo que pide Claude Code; lo ves en
el menú de la manzana → Acerca de este Mac. Valen los Mac con Apple silicon y los Intel.
No hace falta Homebrew.

**Instalar:** abre la Terminal (`Cmd + Espacio`, escribe `Terminal`, Enter) y pega:

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
```

**Qué hace:**

1. **Primero comprueba la versión de macOS.** Si es anterior a la 13, te lo dice y para,
   antes de instalar nada que luego no arrancaría.
2. **git**, si falta, viene con las **herramientas de desarrollo de Apple**. Un Mac tiene un
   comando `git` aunque no las tenga, pero solo sirve para pedirte que las instales: el
   instalador no se deja engañar. Se abre una ventana: pulsa **Instalar** y acepta. Tarda
   unos minutos; el instalador espera y sigue solo.
3. **Claude Code**, si falta, con el instalador de Anthropic, en `~/.local/bin`. Si ya lo
   tienes pero va atrasado, te ofrece actualizarlo (`claude update`).
4. **La app de escritorio de Claude**, si la quieres: con Homebrew si lo tienes y, si no,
   directamente de Anthropic, comprobando su firma. Va a `/Applications`, o a
   `~/Applications` si no puedes escribir allí.
5. **El programa**, en `~/.local/bin/augmented-brain`.
6. **El PATH:** con zsh, la shell normal del Mac, en `~/.zshrc`. Con bash, en
   `~/.bash_profile`, o en `~/.bash_login` o `~/.profile` si ya tienes uno de ellos: la
   Terminal del Mac lee esos, no `~/.bashrc`.
7. **La configuración**, y la oferta de abrir la app de Claude.

**Dónde queda cada cosa:** como en Linux: `~/.local/bin`, `~/.config/augmented-brain` y
`~/augmented-brain`.

**Conviene saber:**

- Después de instalar, **abre una ventana nueva de la Terminal**.
- En la app: pestaña **Code**, **+ New**. Sin elegir carpeta trabajas en tu espacio
  personal; para el de trabajo, **Choose…** y elige `~/augmented-brain/trabajo`.

## Windows

**Necesitas:** **Windows 10 u 11**, PC de 64 bits o Arm. **winget**, el instalador del propio
Windows, que ya traen Windows 11 y un Windows 10 al día (si falta, viene con "Instalador de
aplicaciones" de la Microsoft Store). Basta con el PowerShell que trae Windows.

**Instalar:** abre **PowerShell** (menú Inicio, escribe `PowerShell`, Enter) y pega, con un
clic derecho:

```powershell
irm https://github.com/icalvete/augmented-brain/releases/latest/download/install.ps1 | iex
```

**Qué hace:**

1. **Git for Windows**, si falta, con winget. **En Windows es obligatorio**, y no solo por
   el historial de tus notas: trae **Git Bash**, donde Claude ejecuta los comandos con el
   guardia vigilando. Windows enseña una ventana de permiso: pulsa **Sí**.
2. **Claude Code**, si falta, con el instalador de Anthropic, en `%USERPROFILE%\.local\bin`.
   Si ya lo tienes pero va atrasado, te ofrece actualizarlo (`claude update`).
3. **La app de escritorio de Claude**, si la quieres, con winget.
4. **El programa**, en `%USERPROFILE%\.local\bin\augmented-brain.exe`.
5. **El PATH:** añade esa carpeta al PATH de tu usuario. El instalador de Claude Code lo
   deja en tus manos; este lo hace.
6. **La configuración**, y la oferta de abrir la app de Claude.

**Dónde queda cada cosa:** el programa y Claude Code en `%USERPROFILE%\.local\bin`; la
configuración en `%APPDATA%\augmented-brain`; tus notas en
`%USERPROFILE%\augmented-brain\<espacio>\notas`.

**Conviene saber:**

- Después de instalar, **abre una ventana nueva de PowerShell**: la vieja, y las que ya
  estaban abiertas, no ven el PATH nuevo.
- **Claude ejecuta los comandos en Bash, no en PowerShell.** El guardia lee la shell, no
  PowerShell, así que para cualquier comando de PowerShell y Claude usa Git Bash en su
  lugar. Verás ese bloqueo en los mensajes de Claude: es lo esperado.
- **La app puede pedir actualizarse** la primera vez ("needs the latest desktop app"):
  winget no siempre tiene la última. Pulsa **Update**.
- Para trabajar sin la app: `cd ~\augmented-brain\trabajo; claude` (el PowerShell de
  Windows no tiene `&&`).

## Actualizar

Cuando hay versión nueva, Claude te lo dice al empezar una sesión y se ofrece a
actualizarlo. También puedes hacerlo tú:

```bash
augmented-brain update
```

Actualiza el programa y el plugin de Claude Code. Antes, revisa la instalación (y enseña lo
que falla) y hace la copia de seguridad de cada espacio que la tenga; si falla una copia,
no actualiza nada (`--force` actualiza igualmente). Si ningún espacio tiene copia, te dice
que sigue sin una copia a la que volver. Después, dice qué sesiones abiertas siguen con la versión vieja y qué tarea está activa
en cada espacio: pide a esas sesiones que aparquen su tarea. Volver a lanzar el comando
de instalación también actualiza, sin esos pasos. `augmented-brain update --plugin`
actualiza solo el plugin.

Primero pregunta a GitHub cuál es la última versión, una sola vez: si GitHub no responde,
lo dice y se para. Después, la descarga se intenta tres veces, también por la API de
GitHub. Si sigue fallando, o el programa no es la versión nueva al acabar, te lo dice y
termina con error: no ha cambiado nada, vuelve a lanzarlo más tarde.

Después, cierra y abre Claude. Una sesión que estaba abierta sigue con el programa viejo
hasta que lo hagas, y sus herramientas de augmented brain pueden fallar.
`augmented-brain doctor` y el arranque de una sesión nueva te dicen cuántas sesiones
siguen con la versión vieja.

Tus notas y tus ajustes se quedan como están. A veces una versión nueva guarda las notas
de otra manera. Entonces la primera sesión las pasa a la manera nueva, después de guardar
una copia completa en la carpeta `migrations/` de la configuración, y Claude te lo dice.
Una versión más vieja nunca estropea notas escritas por una más nueva: se niega a
abrirlas y pide que la actualices. Así que actualiza todos los ordenadores que uses.

## Desinstalar

```bash
augmented-brain uninstall
```

Te enseña lo que va a quitar y te pregunta. Quita el programa, el plugin de Claude Code y
lo que Claude Code guarda de él (su caché, sus datos y los registros de su servidor).

- **Tus notas se quedan** en `~/augmented-brain/`: son tuyas. Si vuelves a instalarlo,
  lo encuentras todo como estaba.
- **La configuración también se queda.** `--config` la borra, con tu perfil.
- **Para empezar de cero**, `--all` borra además las notas de cada espacio. Antes te
  enseña cada carpeta con cuántas notas tiene, y te pregunta. Lo que tengas tuyo dentro
  de la carpeta de un espacio se queda. Tu copia de seguridad en GitHub no se borra: te
  dice cuál es y cómo borrarla. Si un espacio no tiene copia de seguridad, te avisa de que
  sus notas se perderán para siempre. Para conservarlas, responde que no y configura antes
  la copia.
- Claude Code, su app de escritorio, git y la entrada del PATH se quedan: Claude Code
  vive en esa misma carpeta.

En Windows el fichero del programa desaparece un momento después de acabar el comando,
porque Windows no deja que un programa en marcha se borre a sí mismo.

Todos los comandos y sus opciones: [Comandos](commands.es.md).

## Si algo falla

- **La terminal no encuentra una orden** (`command not found`, o "no se reconoce" en
  Windows). Abre una terminal nueva: la que usaste todavía no ve el PATH nuevo.
- **Cualquier otra cosa después de instalar.** Lanza `augmented-brain doctor`: te dice qué
  falta y cómo arreglarlo. También te avisa si un espacio no tiene copia de seguridad, si
  Claude Code lleva una segunda memoria propia para las carpetas de tus espacios, y si
  otros usuarios del ordenador podrían leer tus notas. Si la configuración está rota, te
  dice cómo volver a una versión que funcionaba.
- **"Este sistema aún no está soportado".** Funciona en Linux, macOS y Windows, en PC de 64
  bits y Arm.
- **macOS: "Claude Code necesita macOS 13".** Actualiza el Mac en Ajustes del Sistema →
  General → Actualización de software y vuelve a pegar el comando.
- **macOS: no se abrió la ventana de las herramientas de desarrollo.** Lanza
  `xcode-select --install` y luego otra vez el comando de instalación.
- **Linux: no te ofrece la app de escritorio.** Solo está para sistemas con `apt` (Ubuntu,
  Debian y sus derivadas); usa Claude en la terminal.
- **"no puedo descargar … de GitHub".** GitHub tiene problemas. No se ha instalado nada:
  vuelve a pegar el comando dentro de un rato.
- **Paraste el instalador a medias** (Ctrl+C, cerraste la ventana). Vuelve a pegar el
  comando: lo prepara todo desde el principio.
- **Windows: "Git no se ha instalado".** El instalador prueba primero con winget y luego con
  la descarga del propio Git. Si fallan las dos, instala Git desde
  [git-scm.com](https://git-scm.com/download/win) y vuelve a lanzar el comando. La app de
  escritorio necesita winget: actualiza "Instalador de aplicaciones" desde la Microsoft Store.
- **Claude pregunta si te fías de la carpeta.** La primera vez que abres Claude en una
  carpeta, Claude Code lo pregunta antes de cargar nada. Di que sí en tu carpeta de
  trabajo: hasta entonces, augmented brain no puede cargar tus notas.
- **Windows: los acentos salen rotos (`sesiÃ³n`).** Windows PowerShell 5.1 lee mal lo que
  escriben otros programas cuando lo recoge (una redirección, una sesión remota). El texto
  está bien: en Git Bash, o en una ventana, se ve bien.
- **Claude te pide un plan de pago.** Claude Code necesita Pro o superior.
- **Otra cosa.** Mira [Si algo falla](troubleshooting.es.md), o cuéntanoslo en
  [las incidencias del proyecto](https://github.com/icalvete/augmented-brain/issues), con
  lo que salió en pantalla.
