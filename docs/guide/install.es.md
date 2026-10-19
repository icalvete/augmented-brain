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
  no, no se instala nada.

## Linux

**Necesitas:** un PC de 64 bits o un Arm (amd64 o arm64), `curl` o `wget`, y `sha256sum`
(las distribuciones habituales los traen).

**Instalar:** abre una terminal (en Ubuntu, `Ctrl + Alt + T`) y pega:

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
```

**Qué hace:**

1. **git**, si falta, con el gestor de paquetes de tu sistema: `apt` (Ubuntu, Debian), `dnf`
   (Fedora), `pacman` (Arch) o `zypper` (openSUSE). Te pide la contraseña.
2. **Claude Code**, si falta, con el instalador de Anthropic, en `~/.local/bin`. Si ya lo
   tienes pero va atrasado, te ofrece actualizarlo (`claude update`).
3. **La app de escritorio de Claude**, si la quieres, **solo en Ubuntu y Debian**: desde el
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

```bash
augmented-brain update
```

Actualiza el programa y el plugin de Claude Code; tus notas y tus ajustes se quedan. Claude
te avisa cuando hay una versión nueva. Volver a lanzar el comando de instalación hace lo
mismo.

## Desinstalar

```bash
augmented-brain uninstall
```

Quita el programa, el plugin de Claude Code y lo que Claude Code guarda de él (su caché y
los registros de su servidor). **Tus notas se quedan**, y la configuración también, salvo que
añadas `--config`. Para empezar de cero, `--all` borra también la configuración, tu perfil y
las notas de cada espacio, después de enseñarte qué y preguntarte. Claude Code, su app de escritorio, git
y la entrada del PATH también se quedan: Claude Code vive en esa misma carpeta. En Windows
el programa desaparece un momento después de acabar el comando, porque Windows no deja que
un programa en marcha se borre a sí mismo.

## Si algo falla

- **La terminal no encuentra una orden** (`command not found`, o "no se reconoce" en
  Windows). Abre una terminal nueva: la que usaste todavía no ve el PATH nuevo.
- **Cualquier otra cosa después de instalar.** Lanza `augmented-brain doctor`: te dice qué
  falta y cómo arreglarlo.
- **"Este sistema aún no está soportado".** Funciona en Linux, macOS y Windows, en PC de 64
  bits y Arm.
- **macOS: "Claude Code necesita macOS 13".** Actualiza el Mac en Ajustes del Sistema →
  General → Actualización de software y vuelve a pegar el comando.
- **macOS: no se abrió la ventana de las herramientas de desarrollo.** Lanza
  `xcode-select --install` y luego otra vez el comando de instalación.
- **Linux: no te ofrece la app de escritorio.** Solo está para Ubuntu y Debian; usa Claude en
  la terminal.
- **Windows: "winget no se reconoce".** Instala "Instalador de aplicaciones" desde la
  Microsoft Store, o instala Git desde [git-scm.com](https://git-scm.com/download/win), y
  vuelve a lanzar el comando.
- **Claude te pide un plan de pago.** Claude Code necesita Pro o superior.
- **Otra cosa.** Cuéntanoslo en [las incidencias del proyecto](https://github.com/icalvete/augmented-brain/issues),
  con lo que salió en pantalla.
