# Instalación


Un solo comando instala augmented brain y todo lo que necesita, en Linux, macOS y Windows.
Esta página cuenta qué hace ese comando en cada sistema, dónde deja cada cosa y qué cambia
en cada uno. ¿Nunca has usado la terminal? Los [primeros pasos](getting-started.md) te
llevan de la mano.

## En todos los sistemas

- **Un plan de Claude de pago** (Pro o superior: Pro, Max, Team o Enterprise), o una clave
  de la API de Anthropic. augmented brain es gratis, pero trabaja con Claude Code, y el plan gratuito no
  incluye Claude Code. El instalador te lo dice antes de instalarlo. Los planes:
  [claude.com/pricing](https://claude.com/pricing).
- **Sin administrador ni root.** El programa va a tu propia carpeta. Solo si hay que
  instalar algo que falta, como git, puede pedirte la contraseña.
- **El instalador pregunta antes.** El idioma (te propone el del sistema) y, después, cada
  pieza que falte, con para qué sirve. Pulsa Enter para aceptar lo que propone. Para cambiar
  el idioma más tarde, pon `language` en tu [configuración](configuration.md#ajustes-de-todo-el-ordenador).
- **Sin cuenta de GitHub.** El programa se descarga de GitHub sin ella. La copia de
  seguridad sí la usa, u otro servidor de git, y se prepara después: mira
  [Copia de seguridad y restauración](backup.md).
- **Una instalación nueva acaba siempre igual:** dos espacios, `trabajo` y `personal` (en
  inglés, `work` y `personal`), en `~/augmented-brain`, el plugin de Claude Code instalado
  y la oferta de abrir Claude. Si lo lanzas sobre una que ya tienes, conserva tus espacios
  y tus ajustes, y actualiza el programa y el plugin.
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
- En la app: pestaña **Code**, **+ New**. Sin elegir carpeta trabajas en tu espacio
  personal; para el de trabajo, **Choose…** y elige `~/augmented-brain/trabajo`.
- Para trabajar sin la app: `cd ~/augmented-brain/trabajo && claude`.
- La primera vez que abres Claude en una carpeta, te pregunta si confías en ella: di que
  sí, o augmented brain no podrá cargar tus notas.

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

**Necesitas:** **Windows 10 u 11**, PC de 64 bits o Arm. Basta con el PowerShell que trae
Windows. **winget**, el instalador del propio Windows, ayuda pero no es imprescindible: ya
lo traen Windows 11 y un Windows 10 al día (si falta, viene con "Instalador de
aplicaciones" de la Microsoft Store). Sin él, Git se descarga de su propia web; solo la
app de escritorio necesita winget.

**Instalar:** abre **PowerShell** (menú Inicio, escribe `PowerShell`, Enter) y pega, con un
clic derecho:

```powershell
irm https://github.com/icalvete/augmented-brain/releases/latest/download/install.ps1 | iex
```

**Qué hace:**

1. **Git for Windows**, si falta, con winget, o de la web de Git si no lo hay.
   **En Windows es obligatorio**, y no solo por
   el historial de tus notas: trae **Git Bash**, donde Claude ejecuta los comandos con el
   guardia vigilando. Windows enseña una ventana de permiso: pulsa **Sí**.
2. **Claude Code**, si falta, con el instalador de Anthropic, en `%USERPROFILE%\.local\bin`.
   Si ya lo tienes pero va atrasado, te ofrece actualizarlo (`claude update`).
3. **La app de escritorio de Claude**, si la quieres, con winget (sin winget no te la
   ofrece: descárgala de [claude.ai/download](https://claude.ai/download)).
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

Primero hace la copia de seguridad de tus espacios, actualiza el programa y el plugin de
Claude Code, y reinicia las sesiones que siguen con la versión vieja. Volver a lanzar el
comando de instalación también actualiza. Qué hace cada paso, y qué hacer con las
sesiones que deja: [Actualizar y desinstalar](update-and-uninstall.md#actualizar).

## Desinstalar

```bash
augmented-brain uninstall
```

Quita el programa y el plugin de Claude Code, después de enseñar lo que va a quitar y
preguntar. Tus notas y la configuración se quedan; `--config` y `--all` van más lejos.
Qué quita y qué deja cada nivel: [Actualizar y desinstalar](update-and-uninstall.md#desinstalar).

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
- **Otra cosa.** Mira [Si algo falla](troubleshooting.md), o cuéntanoslo en
  [las incidencias del proyecto](https://github.com/icalvete/augmented-brain/issues), con
  lo que salió en pantalla.

## Comprobar una descarga

Para instalar no hace falta. Es para quien quiera estar seguro de que lo que ha bajado es
lo que publicó el proyecto, hecho a partir de su código público.

Cada versión la compila GitHub a partir de un commit de `main`, nunca en el ordenador de
nadie, y GitHub firma un registro de ello: qué workflow hizo cada fichero y desde qué
commit. Con la [CLI de GitHub](https://cli.github.com/) (`gh`) se comprueba cualquier
fichero de una versión, también el instalador:

```bash
gh attestation verify augmented-brain_<versión>_linux_amd64.tar.gz -R icalvete/augmented-brain
```

con el nombre del fichero que has descargado.

Termina con `Verification succeeded!` y nombra el workflow y el commit. Un fichero
cambiado después de compilarse no pasa.

`checksums.txt`, en cada versión, tiene el SHA-256 de cada archivo y de los dos
instaladores. Los instaladores comprueban el archivo con él; tú puedes comprobar
cualquier fichero también:

```bash
sha256sum --ignore-missing -c checksums.txt
```

La compilación es reproducible: el mismo commit, compilado con la versión de Go de
`go.mod`, da los mismos bytes. `scripts/release-build.sh <versión>` desde un clon de esa
etiqueta escribe el mismo `checksums.txt` que la versión publicada.
