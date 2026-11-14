# Comandos

[Read in English](commands.md)

Todos los comandos del programa, con todas sus opciones. En el día a día no te hacen
falta: hablas con Claude. Sirven para preparar, comprobar y pasar a un ordenador nuevo.

Se lanzan en una terminal. Casi todos trabajan con **el espacio de la carpeta en la que
estás**. Fuera de las carpetas de tus espacios, usan el espacio por defecto. Si no hay
espacio por defecto (hiciste un solo espacio con `init --space`, por ejemplo), lo dicen
y no hacen nada: lánzalos desde una carpeta de ese espacio.

```bash
cd ~/augmented-brain/trabajo
augmented-brain check
```

Las opciones van antes o después del resto de argumentos, con uno o dos guiones (`--all`
o `-all`). Si un comando recibe un argumento que no admite, lo dice y no hace nada. Cada
apartado da la sintaxis del comando: `<…>` es algo que pones tú, `[…]` se puede quitar y
`|` separa opciones que se excluyen.

Algunos comandos los lanza otro, no tú: Claude Code lanza `mcp` y `hook`, `panel` lanza
`panel serve`, y en Linux un temporizador lanza `remote watch`. Están aquí para que sepas
qué son.

Dentro de Claude Code hay un comando con barra: `/brain:panel` abre el panel, igual que
`augmented-brain panel`.

## Quiero…

| Quiero… | Comando |
|---|---|
| Prepararlo todo la primera vez | [`init`](#augmented-brain-init) |
| Saber de qué espacio es esta carpeta | [`space`](#augmented-brain-space) |
| Añadir un espacio | [`space add`](#augmented-brain-space-add) |
| Revisar mis notas después de editarlas a mano | [`check`](#augmented-brain-check) |
| Saber qué falla | [`doctor`](#augmented-brain-doctor) |
| Traer una memoria que Claude Code ya escribió | [`import`](#augmented-brain-import) |
| Hacer ya la copia de un espacio | [`backup`](#augmented-brain-backup) |
| Conectar un espacio a su repositorio de copia | [`backup setup`](#augmented-brain-backup-setup) |
| Recuperar un espacio en un ordenador nuevo | [`restore`](#augmented-brain-restore) |
| Verlo y cambiarlo todo en el navegador | [`panel`](#augmented-brain-panel) |
| Abrir el panel desde el móvil | [`panel remote`](#augmented-brain-panel-remote) |
| Llegar a Claude desde el móvil | [`remote install`](#augmented-brain-remote-install) |
| Ver cómo está el acceso remoto | [`remote status`](#augmented-brain-remote-status) |
| Abrir una sesión propia para una tarea | [`session start`](#augmented-brain-session-start) |
| Reiniciar sesiones tras actualizar | [`session restart`](#augmented-brain-session-restart) |
| Enterarme de cuándo acaba una pull request | [`watch pr`](#augmented-brain-watch-pr) |
| Volver a una configuración anterior | [`config restore`](#augmented-brain-config-restore) |
| Limpiar lo que dejó Claude Code | [`clean`](#augmented-brain-clean) |
| Pasar a la última versión | [`update`](#augmented-brain-update) |
| Quitar augmented brain | [`uninstall`](#augmented-brain-uninstall) |

Qué significa cada código de salida está al final, en
[Códigos de salida](#códigos-de-salida).

## augmented-brain init

Prepara todo: los espacios, la configuración y el plugin de Claude Code. El instalador lo
lanza por ti la primera vez.

Sintaxis: `augmented-brain init [--custom] [--language en|es] [--space <nombre>]
[--work <carpeta>] [--memory <carpeta>] [--yes] [--no-plugin]`

```bash
augmented-brain init
```

Sin opciones no pregunta nada. Toma el idioma del sistema y crea dos espacios en
`~/augmented-brain`: `trabajo` y `personal` (`work` y `personal` en inglés), con
`personal` como espacio por defecto. Después instala el plugin. Si no hay Claude Code,
escribe los dos comandos para instalar el plugin más tarde.

Si ya hay configuración, se para y no cambia nada. Para añadir un espacio, usa
`space add`. Escribe la configuración al final: si se para a medias, vuelve a lanzarlo.

- `--custom`: te pregunta el idioma, el espacio y sus carpetas, para hacer un solo espacio
  a tu manera.
- `--language`: el idioma, `en` o `es`. Sin ella, el del sistema (o `AB_LANGUAGE`).
- `--space`: el nombre del único espacio que se crea (minúsculas, cifras y guiones,
  empezando por una letra).
- `--work`: su carpeta de trabajo. Por defecto, `~/augmented-brain/<espacio>`.
- `--memory`: su carpeta de notas. Por defecto, `~/augmented-brain/<espacio>/notas`,
  también si `--work` dice otra carpeta.
- `--yes`: no pregunta; usa las opciones que le des y lo que hay por defecto.
- `--no-plugin`: no instala el plugin de Claude Code.

Con `--custom`, `--space`, `--work` o `--memory` se crea un solo espacio en vez de los
dos, y no queda espacio por defecto: los comandos lanzados fuera de sus carpetas lo dicen
y no hacen nada. Con un segundo espacio (`space add`) vuelve a haber espacio por defecto.

```bash
augmented-brain init --space cliente --work ~/clientes/acme --language es
```

Eso crea el espacio `cliente`, con su carpeta de trabajo en `~/clientes/acme` y sus notas
en `~/augmented-brain/cliente/notas`.

## augmented-brain space

Dice de qué espacio es esta carpeta, y dónde están sus notas.

Sintaxis: `augmented-brain space`

```bash
augmented-brain space
```

Fuera de las carpetas de tus espacios, nombra el espacio por defecto. Si no hay espacio
por defecto, lo dice y lista los espacios.

## augmented-brain space add

Añade otro espacio: un cliente, un proyecto propio, un voluntariado.

Sintaxis: `augmented-brain space add [<nombre>] [--space <nombre>] [--work <carpeta>]
[--memory <carpeta>] [--yes]`

```bash
augmented-brain space add cliente
```

El nombre va como argumento (primero o último) o con `--space`. Pregunta la carpeta de
trabajo y la de notas, y propone `~/augmented-brain/<nombre>` y
`~/augmented-brain/<nombre>/notas`. Si aún no había espacio por defecto, pone uno:
`personal` si hay un espacio con ese nombre, y si no, el primero por orden alfabético de
los que ya había.

- `--space`: el nombre del espacio nuevo.
- `--work`: su carpeta de trabajo.
- `--memory`: su carpeta de notas. Por defecto, `~/augmented-brain/<nombre>/notas`,
  también si `--work` dice otra carpeta.
- `--yes`: no pregunta; usa las opciones que le des y lo que hay por defecto.

No tiene `--language`. El idioma es el mismo para todos los espacios de este ordenador:
el ajuste `language`, que pone `init`. Mira [Configuración](configuration.es.md).

```bash
augmented-brain space add cliente --work ~/clientes/acme --yes
```

Lo tienes en [Espacios](spaces.es.md).

## augmented-brain check

Revisa las notas del espacio de esta carpeta, y rehace su índice si todas son válidas.

Sintaxis: `augmented-brain check`

```bash
augmented-brain check
```

Además:

- pasa las notas a un formato nuevo cuando una versión nueva lo necesita, después de hacer
  una copia completa;
- revisa las notas compartidas, y rehace su índice;
- nombra cada fichero que guarda un secreto (la copia de seguridad lo rechaza hasta que
  lo quites);
- nombra los repositorios a los que apuntan las notas y que no están en este ordenador;
- deja las notas solo para ti, si otros usuarios podían leerlas;
- pasa la lista de nombres de secretos a `secrets.toml`, desde `config.toml`, donde la
  guardaba la 0.6.0;
- mueve a `archive/journals/` las entradas más antiguas del diario de una tarea que pasa de
  24 KB, como hacen las herramientas al escribirla (mira
  [Un diario largo](tasks.es.md#un-diario-largo)).

Lánzalo después de editar notas a mano. Acaba con error si una nota no es válida.

## augmented-brain doctor

Dice qué falta y cómo arreglarlo.

Sintaxis: `augmented-brain doctor`

```bash
augmented-brain doctor
```

Cada línea empieza con una marca: `✓` bien, `·` para que lo sepas, `!` un aviso, `✗`
algo que falla. Debajo de una línea que pide arreglo, `→` dice qué hacer. Revisa:

- **Tu instalación:** la configuración, el perfil y en qué espacio está esta carpeta; si
  hay espacio por defecto; cuántas versiones de la configuración se guardan, y si cambió
  por fuera de ellas (a mano).
- **Tus notas:** la carpeta de notas de cada espacio, sus notas, su índice y si tiene
  copia de seguridad; las tareas demasiado largas para leerlas de una vez, que `check`
  acorta; cuánto ocupan tus reglas en cada arranque de sesión, con las tres más largas
  (van enteras a cada sesión, así que una regla más corta cuesta menos cada vez); las
  notas que otros usuarios del ordenador podrían leer; una segunda memoria que Claude
  Code guarda por su cuenta para tus carpetas.
- **Claude Code y el guardia:** Claude Code, git y el plugin, y si Claude Code es lo
  bastante reciente para pararlo todo cuando el guardia falla (2.1.295 o posterior); el
  guardia, de verdad, con un comando que tiene que parar; las listas del guardia, las
  excepciones de root en `root_ask` y las que ya han terminado; las listas cambiadas
  desde que las aceptaste, un repositorio a la vez intocable y permitido, y listas que
  una recuperación dejó vacías.
- **Sesiones:** las sesiones de Claude que siguen con una versión anterior después de
  actualizar; una ventana de Claude que usa mucha memoria (2 GB o más); las sesiones de
  fondo que conservan su proceso sin que nadie las use (terminadas o archivadas, o sin
  tarea y paradas más de 12 horas: archivarla en Claude Desktop no la para), cada una con
  el `claude stop` que la termina.
- **Restos:** los worktrees que hizo Claude Code con la sesión terminada, y las carpetas
  temporales de sesiones que acabaron (mira `clean`).
- **Desde otro dispositivo:** el acceso remoto, una línea por espacio, cuando lo has
  puesto con `remote install` (un aviso si su servicio no está en marcha, Claude aún no
  confía en su carpeta o, en Linux, el linger está apagado); el panel desde otro
  dispositivo, si lo has usado (Tailscale, HTTPS y una dirección que se quedó publicada).

Acaba con error si algo falla. Si la configuración no se puede leer, te lleva a
`config history` y `config restore`. Lo tienes en [Si algo falla](troubleshooting.es.md).

## augmented-brain import

Trae una memoria que Claude Code ya escribió. Sin `--apply`, solo enseña lo que haría.

Sintaxis: `augmented-brain import <carpeta> [--map <fichero>] [--repos-base <carpeta>]
[--apply]`

Con un solo espacio, todo va a él. Con dos o más (como los que crea `init`), necesita
`--map` para saber dónde va cada nota. Un mapa con una línea, `* = trabajo`, lo manda todo
a `trabajo`:

```bash
augmented-brain import ~/.claude/projects/<carpeta>/memory --map ~/mapa-import.txt
```

- `--apply`: escribe las notas. Sin ella, no se escribe nada.
- `--map`: un fichero que dice a qué espacio va cada área o nota: líneas `area-x =
  espacio`, `task-y = espacio`, `lesson-z = shared`, y `* = espacio` para el resto.
- `--repos-base`: la carpeta respecto a la que van las rutas `repos:` del origen.

Nunca escribe encima de una nota que ya existe, y el origen solo lo lee. Lo tienes en
[Traer una memoria](import.es.md).

## augmented-brain backup

Guarda los últimos cambios del espacio de esta carpeta en su repositorio privado. También
se lanza solo cuando acaba una sesión de Claude.

Sintaxis: `augmented-brain backup [--all | --space <nombre>]`

```bash
augmented-brain backup
```

- `--all`: copia todos los espacios, no solo el de esta carpeta.
- `--space`: copia este espacio, estés donde estés: `--space trabajo`.
- `--auto`: así lo lanza el final de una sesión, por detrás. No dice nada y nunca espera
  una contraseña. No te hace falta.

Se para sin subir nada si una nota guarda un secreto, o si otra máquina cambió la misma
nota de otra forma, y dice cuál. Lo tienes en
[Copia de seguridad y recuperación](backup.es.md).

## augmented-brain backup setup

Conecta el espacio de esta carpeta a un repositorio git privado. Todavía no copia nada:
lanza después `augmented-brain backup` para hacer la primera copia (te lo dice).

Sintaxis: `augmented-brain backup setup <dirección>`

```bash
augmented-brain backup setup git@gitlab.com:lucia/brain-trabajo.git
```

Para que te cree el repositorio en tu cuenta de GitHub, pídeselo a Claude: "configura la
copia de seguridad".

## augmented-brain restore

Recupera un espacio de su copia de seguridad, en una máquina nueva.

Sintaxis: `augmented-brain restore <dirección> [--memory <carpeta>] [--work <carpeta>]`

```bash
augmented-brain restore https://github.com/lucia/augmented-brain-trabajo.git
```

Cambia el espacio vacío del mismo nombre por el tuyo. Un espacio que ya tiene notas lo
deja como está. Si falta el plugin de Claude Code, lo instala.

Con el espacio llegan, si esta máquina aún no tiene los suyos: el perfil (solo lo lleva la
copia del espacio por defecto), las notas compartidas y los nombres de tus secretos (nunca
sus valores). Después nombra lo que no ha traído: las listas del guardia, los repositorios
donde Claude puede hacer commit y los secretos que faltan en el llavero de esta máquina,
cada uno con el comando para guardarlo. Si no puede preguntar al llavero (no hay, o está
bloqueado), nombra los secretos que no ha podido comprobar.

Una carpeta que nombra la copia solo se usa si está dentro de tu home: no el home mismo,
ni una carpeta oculta en su primer nivel (`~/.config`), ni las del sistema (`~/Library`,
`~/AppData`), ni dentro de un `.git`. En lugar de una así usa `~/augmented-brain/<espacio>`,
y lo dice.

- `--memory`: dónde poner las notas del espacio. Por defecto, el mismo sitio dentro de la
  carpeta de trabajo que antes, o `~/augmented-brain/<espacio>` si estaban fuera de ella.
- `--work`: la carpeta de trabajo que elige el espacio. Por defecto, la que se guardó en
  la copia.

## augmented-brain panel

Abre el panel en el navegador: el día de un vistazo, y todas las páginas de tus espacios
para verlas y cambiarlas.

Sintaxis: `augmented-brain panel [--no-browser]`

```bash
augmented-brain panel
```

Arranca el panel por detrás, o encuentra el que ya está en marcha, y abre el navegador en
Hoy, con el espacio de esta carpeta. Vuelve en el acto. El panel se para solo después de
30 minutos sin usarse. La dirección que escribe abre el panel una vez, en 2 minutos; si
lo lanza Claude, con el navegador abierto, no escribe ninguna.

- `--no-browser`: solo escribe la dirección, no abre el navegador. `AB_NO_BROWSER=1` hace
  lo mismo.

Lo tienes en [El panel](panel.es.md).

## augmented-brain panel remote

Abre el panel para tus otros dispositivos (otro ordenador, el móvil), a través de
[Tailscale](https://tailscale.com), una app gratuita que une tus dispositivos en una red
privada propia.

Sintaxis: `augmented-brain panel remote`

```bash
augmented-brain panel remote
```

Arranca el panel y lo publica solo en tu red de Tailscale. Si el panel ya está abierto,
ese mismo panel contesta también por la red, así que las páginas que tienes abiertas
siguen funcionando. Escribe la dirección y un código QR para abrirlo desde el móvil.

- Solo pueden abrirlo tus propios dispositivos, con la sesión iniciada en la cuenta de
  la que es este ordenador.
- La dirección lo abre una vez, en 10 minutos. Las páginas dejan de funcionar cuando el
  panel se para.
- Lo que falte (el propio Tailscale, HTTPS en tu red, dejar publicar a tu usuario) lo
  explica, con el paso que hay que dar. Nunca instala ni cambia nada de Tailscale por ti.

Lo tienes en [El panel desde otro dispositivo](panel-remote.es.md).

## augmented-brain panel stop

Para el panel ya.

Sintaxis: `augmented-brain panel stop`

```bash
augmented-brain panel stop
```

Dice si había un panel en marcha. El enlace al pie de cada página del panel hace lo mismo.

## augmented-brain panel serve

El propio panel, en marcha por detrás. Lo arranca `augmented-brain panel`; nunca lo
lanzas a mano. Se para solo después de 30 minutos sin usarse.

## augmented-brain remote

Mantiene a Claude al alcance desde otro ordenador o el móvil, también después de
reiniciar. Son tres comandos, `install`, `uninstall` y `status`, cada uno con su apartado
abajo. Usa un servicio de usuario de `systemd` en Linux, un LaunchAgent en macOS y una
tarea del Programador de tareas en Windows. En Linux hay un cuarto, `watch`, que lanza un
temporizador, no tú. Sin subcomando, o con otro, `remote` muestra su uso. Mira
[Acceso remoto](remote.es.md).

## augmented-brain remote install

Pone en marcha el acceso remoto, un servicio por espacio.

Sintaxis: `augmented-brain remote install`

```bash
augmented-brain remote install
```

Para cada espacio que nombra `remote.toml` escribe un servicio que lanza `claude
remote-control`, y lo arranca. Por defecto es el espacio por defecto, o el único espacio
si solo hay uno.

- **Dónde:** un servicio de usuario de `systemd` en Linux, un LaunchAgent en
  `~/Library/LaunchAgents` en macOS, una tarea `\augmented-brain\remote-<espacio>` del
  Programador de tareas en Windows (corre mientras tienes la sesión iniciada, sin
  ventana).
- **En qué carpeta:** la primera carpeta de trabajo del espacio, o la que diga `folders`
  en `remote.toml`. Solo un espacio sin carpeta de trabajo recibe una carpeta propia,
  `remote/<espacio>` en la carpeta de la configuración, que pasa a ser su carpeta de
  trabajo.
- **Qué necesita:** `claude` (en tu PATH, o donde lo pone su instalador) y una carpeta
  que exista. El servicio recibe un PATH con las carpetas de `claude` y
  `augmented-brain`, para que sus sesiones encuentren los dos después de reiniciar.
- **En Linux** instala además el vigilante: un temporizador de `systemd`,
  `augmented-brain-watch-remote`, que lanza `augmented-brain remote watch` cada dos
  minutos (mira abajo).

Puedes lanzarlo otra vez cuando quieras: solo cambia lo que sea distinto. Después dice
lo que aún falta, y no hace nada de ello por ti:

- Claude tiene que **confiar en cada carpeta**. Escribe `cd <carpeta> && claude` para
  cada una en la que aún no confía (en Windows, `cd "<carpeta>"; claude`): lánzalo una
  vez y acepta la pregunta.
- En Linux, el **linger** tiene que estar encendido para que los servicios arranquen
  después de reiniciar sin nadie con la sesión iniciada. Escribe el comando,
  `sudo loginctl enable-linger <tú>`, para que lo lances tú. macOS y Windows no tienen
  linger: el servicio arranca cuando inicias sesión.

Sin los servicios de usuario de systemd (un contenedor, o WSL sin systemd), launchctl en
macOS o el Programador de tareas en Windows, lo dice y no cambia nada. Los ajustes están
en `remote.toml` (mira [Configuración](configuration.es.md)).

## augmented-brain remote uninstall

Para el acceso remoto y lo quita.

Sintaxis: `augmented-brain remote uninstall`

```bash
augmented-brain remote uninstall
```

Para y quita los servicios que escribió `remote install` (en Linux, primero el
vigilante), y pone `enabled = false` en `remote.toml`. Las carpetas y sus entradas en
`work_folders` se quedan: nunca borra nada más.

## augmented-brain remote status

Dice cómo está el acceso remoto, y no cambia nada.

Sintaxis: `augmented-brain remote status`

```bash
augmented-brain remote status
```

Para cada espacio: el nombre de la sesión (lo que eliges en Claude Desktop o claude.ai),
si el servicio está en marcha y arranca solo, y si Claude confía en su carpeta. Después,
en Linux, si el vigilante (mira `remote watch` abajo) está en marcha y hace cuánto miró
por última vez, y si el linger está encendido; en macOS y Windows, que arranca cuando
inicias sesión. Debajo de cada problema, `→` dice qué hacer.

## augmented-brain remote watch

Recupera un servicio cuando el servidor archivó su sesión. Solo en Linux. Lo lanza su
temporizador cada dos minutos; no hace falta que lo lances tú.

Sintaxis: `augmented-brain remote watch`

```bash
augmented-brain remote watch
```

Lo que hace en cada vuelta:

- **Un servicio sin sesión.** Cuando el servidor archiva una sesión (por ejemplo, la
  archivaste en claude.ai), su proceso de Claude termina y el servicio sigue en marcha
  sin sesión: nadie puede conectarse, y systemd no ve nada raro. Tras dos vueltas
  seguidas así, el vigilante reinicia el servicio.
- **Sigue sin sesión.** Entonces Claude intentaba reabrir la sesión archivada. El
  vigilante renombra lo que Claude apunta de ella (`bridge-pointer.json` pasa a
  `bridge-pointer.json.archived`, en el proyecto de la carpeta en `~/.claude/projects`) y
  reinicia otra vez el servicio, para que empiece una sesión nueva: unos diez minutos en total.
  La conversación de la sesión archivada no vuelve.
- **Las sesiones de tareas, a salvo.** Parar un servicio para todo lo que se arrancó
  dentro de él. La sesión de una tarea abierta desde una sesión remota puede arrancar
  allí el servicio en segundo plano de Claude, del que dependen todas las sesiones de
  tareas. Así que el vigilante saca de cada servicio lo que no arrancó su propio
  servidor a un grupo propio, `augmented-brain-kept-<número>`. `remote install` y
  `remote uninstall` hacen lo mismo antes de reiniciar o parar un servicio; si no se
  puede, se detienen y lo dicen, y el vigilante no reinicia ese servicio.

Lo que hace queda en el diario del sistema
(`journalctl --user -u augmented-brain-watch-remote`).

También despierta las sesiones de tarea que paró el límite de uso de Claude. Cuando se
acaba el uso de tu plan, Claude Code rechaza el turno y dice a qué hora vuelve el
límite. Una sesión que tienes abierta en un terminal sigue sola a esa hora; una sesión
de tarea corre en segundo plano y no: se queda ahí, sin hacer nada, hasta que alguien
le escribe. El vigilante lee esa hora de la última respuesta de la sesión y, cuando ha
pasado, para la sesión y la reanuda con un mensaje que le dice que lea su tarea y siga.
Vuelve como la misma sesión, con su conversación, en Claude Desktop y claude.ai.
Despierta una sesión en cada mirada, así que vuelven con dos minutos entre ellas y no
gastan el uso nuevo de golpe. Solo se despiertan las sesiones abiertas con
`augmented-brain session start`, nunca otras. Te avisa dos veces, en la siguiente
sesión que uses: cuando empieza la espera, con la hora, y cuando ha vuelto la última.
Cada tarea despertada recibe una línea en su diario.

## augmented-brain session

Abre una sesión propia para una tarea, para que desde otro ordenador puedas trabajar en
varias tareas a la vez, cada una en su sesión. Son cuatro comandos, `start`, `stop`,
`restart` y `list`, cada uno con su apartado abajo. `start`, `stop` y `list` funcionan
por ahora solo en Linux: solo se han probado ahí. `restart` funciona en Linux y macOS.
Sin subcomando, o con otro, `session` muestra su uso. Para qué sirven y cómo usarlos:
[Una sesión por tarea](task-sessions.es.md).

## augmented-brain session start

Abre una sesión para una tarea, en la carpeta de la tarea.

Sintaxis: `augmented-brain session start <tarea> [--folder <carpeta>]`

```bash
augmented-brain session start task-new-website
augmented-brain session start task-new-website --folder ~/projects/website
```

Arranca una sesión de Claude propia para la tarea, en segundo plano (`claude --bg
--remote-control`), llamada `<name_prefix> · <título de la tarea>` (`name_prefix` se pone
en `remote.toml`; por defecto, el nombre del ordenador). En menos de un minuto sale en
Claude Desktop y claude.ai. Empieza sola: su primer mensaje le pide leer su tarea,
decirte cómo está y ponerse con el siguiente paso.

- `--folder`: la carpeta donde abrir la sesión. Tiene que estar dentro de las carpetas de
  trabajo del espacio de la tarea: una sesión fuera de ellas no encontraría su tarea.

**Qué carpeta.** Sin `--folder`, la del primer repositorio de la tarea que encuentra en
las carpetas de trabajo del espacio. Una tarea que no nombra ningún repositorio se abre
en la primera carpeta de trabajo del espacio, que pueden compartir varias así.

**Un repositorio bare** (uno que solo tiene la historia de git, sin ficheros con los que
trabajar) recibe un worktree de git para la tarea: una segunda copia de trabajo del
repositorio, en una carpeta a su lado. Para la tarea `task-new-website` de `site` es
`site-new-website`, en una rama con el nombre de la tarea. Una rama nueva sale de la rama
por defecto de origin tal como está ahora: antes la trae, y si no llega a origin parte de
lo último que este ordenador sabía de ella, y lo dice. Si una sesión anterior de la tarea
dejó su rama, el worktree la retoma. Si se pide otra vez, usa el mismo worktree. Lo
mismo vale para un repositorio bare dado con `--folder`. Mira
[Worktrees, para un repositorio bare](task-sessions.es.md#worktrees-para-un-repositorio-bare).

**Reglas:**

- Lánzalo desde una carpeta del espacio de la tarea. La sesión remota a la que entras
  desde otro ordenador puede lanzarlo por ti («ábreme una sesión para la tarea de la
  web»).
- La carpeta tiene que ser de confianza ella misma: que confíe en una carpeta de arriba
  no basta, así que la primera vez el comando te dice que abras Claude una vez en esa
  carpeta y aceptes la pregunta. Un worktree de git es la excepción: basta con confiar en
  su repositorio principal.
- Una tarea por carpeta: una carpeta que ya usa la sesión de otra tarea se rechaza. Dale
  otra con `--folder` (un worktree de git, por ejemplo).
- Si Claude Code no tiene la sesión iniciada, o le ha caducado, no se abre nada. Espera
  hasta un minuto la primera respuesta de la sesión; si esa respuesta es un error (se
  acabó el login, el servicio está saturado), la sesión se cierra. En los dos casos dice
  qué hacer, como `claude auth login`.
- Si Claude paró la sesión de la tarea (lo hace con una que lleva horas sin usarse, y con
  todas cuando se reinicia su servicio en segundo plano), `session start` la trae de
  vuelta, con toda su conversación y sin un primer mensaje nuevo. Solo si Claude se niega
  se abre una sesión nueva que la sustituye.
- En un sistema donde las sesiones de tarea aún no funcionan, lo dice antes de crear nada.
- La sesión se cierra con `session stop` o al reiniciar el ordenador. Las sesiones no
  vuelven solas tras un reinicio: pídelas otra vez.

## augmented-brain session stop

Cierra la sesión de una tarea y la quita de Claude Desktop y claude.ai. Funciona desde
cualquier carpeta.

Sintaxis: `augmented-brain session stop <tarea> [--when-idle]`

```bash
augmented-brain session stop task-new-website
```

- `--when-idle`: espera a que la sesión termine su respuesta y después la cierra. La
  orden vuelve en el acto, y un proceso en segundo plano espera, hasta 30 minutos; una
  sesión que espera un permiso cuenta como respondiendo, y una que siga respondiendo
  entonces se queda abierta. Cerrar la tarea lo lanza así solo, para que una sesión pueda
  cerrar su propia tarea sin cortar su respuesta. Lo que hizo queda en
  `state/session-stop.log`, en la carpeta de la configuración.

**El worktree de la tarea** (el que creó `session start`, mira arriba) también se va, con
su rama, siempre que no se pierda nada. Se queda, y el comando lo dice, si tiene:

- cambios sin commitear (también los ficheros que tus ajustes de git esconden);
- commits sin subir, en la rama de la tarea, aunque el worktree se quedara en otro
  commit;
- ficheros que git ignora y que ningún build vuelve a hacer (un `.env`, una clave;
  `node_modules`, `build`, `dist` y parecidos no cuentan).

Las sesiones que terminan sin `stop` (un reinicio, una sesión cerrada desde Claude
Desktop) dejan su worktree. El próximo `session start` quita los que no tienen nada
propio (no uno creado en los últimos diez minutos), cerrar la tarea quita el suyo, y
`doctor` solo dice cuántos se irían.

**Otros casos:**

- Claude no sabe decir si la sesión sigue en marcha (no se puede leer su lista): lo dice
  y no cambia nada; la sesión sigue apuntada, y su worktree también.
- También cierra una sesión que Claude retiró (una que lleva horas sin uso). Si aun así
  Claude no puede quitarla de sus listas, lo dice y da el comando para hacerlo más tarde.
- Una tarea sin sesión recibe «no hay nada que cerrar», nunca un éxito callado.

## augmented-brain session restart

Reinicia una sesión de Claude para que use la versión de augmented brain instalada
ahora, con la misma conversación. Funciona desde cualquier carpeta. Úsalo después de
actualizar, o justo después de aparcar una tarea, para que su sesión empiece de nuevo.

Sintaxis: `augmented-brain session restart <sesión> [--when-idle]`, o
`augmented-brain session restart --old` (sin sesión y sin `--when-idle`).

```bash
augmented-brain session restart task-new-website
augmented-brain session restart --old
```

- `<sesión>`: su tarea (si la abrió `session start`), su id, los ocho primeros caracteres
  de su id de sesión o su nombre; `claude agents` las lista.
- `--old`: reinicia una a una todas las sesiones quietas que siguen con un augmented
  brain anterior, y dice qué ha pasado con cada una. `update` lo lanza solo. Aquí solo la
  sesión propia de una tarea (abierta por `session start`) recibe un proceso nuevo:
  cualquier otra sesión de fondo puede estar archivada en Claude Desktop, cosa que a
  Claude no se le dice, así que solo se para su servidor de augmented brain.
- `--when-idle`: espera a que la sesión termine su respuesta y después la reinicia. Así
  se reinicia una sesión a sí misma después de aparcar su tarea: la orden vuelve en el
  acto, y un proceso en segundo plano espera, hasta 30 minutos. Lo que hizo queda en
  `state/session-restart.log`, en la carpeta de la configuración.

Cada tipo de sesión se reinicia a su manera:

- **La sesión de una tarea** (`claude --bg`): Claude le da un proceso nuevo con la misma
  conversación. Una que archivaste, o que terminó, nunca se relanza: solo se para su
  servidor de augmented brain, y arranca la versión nueva si algún día la abres.
- **Una sesión del acceso remoto** (`remote install`): se reinicia su servicio, y la
  misma sesión vuelve a claude.ai y Claude Desktop en unos diez segundos. El servicio
  lleva todas las sesiones que sirve, así que solo se reinicia si todas están quietas.
  Antes se saca de él todo lo demás (sobre todo el servicio en segundo plano de Claude),
  para que no se pare con él. Solo en Linux.
- **Una sesión de Claude Desktop o de un terminal:** su ventana sigue abierta, y solo se
  para su servidor de augmented brain; Claude Code arranca la versión nueva la próxima
  vez que la sesión lo use. En Linux y macOS. En Windows se deja como está: ciérrala y
  ábrela tú.

Una sesión que está respondiendo, o esperando un permiso, nunca se reinicia: se vuelve a
mirar justo antes. Tampoco la sesión desde la que se lanza el comando: esa necesita
`--when-idle`. Una que falla no para las demás.

## augmented-brain session list

Lista las sesiones de tareas abiertas en este ordenador: la tarea, si está en marcha y su
carpeta.

Sintaxis: `augmented-brain session list`

```bash
augmented-brain session list
```

- **retirada** quiere decir que Claude la paró, tras horas sin uso o al reiniciarse su
  servicio en segundo plano. claude.ai la enseña como desconectada, y `session start` con
  su tarea la trae de vuelta con toda su conversación.
- **fallida** quiere decir que su última respuesta fue un error, como un login caducado:
  no hace nada hasta que alguien le escriba. La línea da el error y qué hacer.
- **en espera** quiere decir que la paró el límite de uso de Claude. La línea da la hora
  a la que vuelve el límite, que es cuando el vigilante la despierta (mira `remote
  watch` arriba).

## augmented-brain watch

Sigue algo que tarda y dice cómo acabó, para que una sesión pueda dejarlo en marcha y
seguir con otra cosa. Por ahora sigue una sola cosa, una pull request: mira `watch pr`
abajo. Sin subcomando, `watch` muestra su uso.

## augmented-brain watch pr

Sigue una pull request de GitHub hasta que hay algo que contar, y entonces termina sola y
dice cómo fue, en una línea.

Sintaxis: `augmented-brain watch pr <número> [--task <tarea>] [--repo <dueño/nombre>]
[--max <tiempo>] [--every <tiempo>]`

```bash
augmented-brain watch pr 104 --task task-my-change
```

Termina cuando la pull request se mezcla o se cierra, choca con su rama base (entonces no
corre la CI), falla un check obligatorio, terminan todos los obligatorios o pasa `--max`.

- `--task`: escribe esa línea en el diario de la tarea, y lo que pasó como su siguiente
  paso. Todas las demás sesiones se enteran antes de su próxima respuesta, y lo enseña
  Hoy, en el panel. La vigilancia la hace entonces un proceso propio, que sigue aunque la
  sesión que la empezó se cierre o se reinicie. Si se pide otra vez para la misma pull
  request y tarea, espera a la que ya está en marcha. La tarea tiene que ser del espacio
  de esta carpeta.
- `--repo`: el repositorio, como `dueño/nombre`. Por defecto, el de esta carpeta.
- `--max`: lo más que vigila. Por defecto, `3h`.
- `--every`: cada cuánto mira. Por defecto, `30s`; `10s` como poco.

`--max` y `--every` llevan un número con su unidad: `s` segundos, `m` minutos, `h` horas,
como en `90m` o `1h30m`.

**Qué dice la línea.** Los checks no obligatorios no deciden nada, pero la línea nombra
los que siguen en marcha o han fallado (cinco como mucho, cada uno cortado a 60
caracteres). Sus nombres van solo al diario, nunca al siguiente paso, que todas las
sesiones leen al empezar: esos nombres los elige quien escribe los workflows.

**Cuándo se rinde.** Si GitHub no lista ningún check obligatorio a los 5 minutos, o `gh`
falla tres veces seguidas o da una respuesta que no entiende, se para y dice lo que dijo
`gh`.

Necesita `gh` con tu login de GitHub. El asistente lo lanza en segundo plano; su última
línea le pide avisarte en el acto. Acaba con 0 si los checks pasaron o la pull request se
mezcló, y con 1 en cualquier otro caso.

## augmented-brain config

Las versiones de tu configuración: dos comandos, `history` y `restore`, cada uno con su
apartado abajo. Sin subcomando, `config` muestra su uso. Mira
[Versiones de la configuración](config-history.es.md).

## augmented-brain config history

Lista las últimas versiones de tu configuración, de la más nueva a la más vieja, y no
cambia nada.

Sintaxis: `augmented-brain config history [<número>]`

```bash
augmented-brain config history
augmented-brain config history 3
```

Una versión es una copia de `config.toml`, `guard.toml`, `remote.toml`, `session.toml`,
`profile.md` y `secrets.toml` tal como estaban en un momento, con quién los cambió: el
panel, un comando, el asistente (y con qué herramienta), una vuelta a una versión, o a
mano. Se guardan las 10 últimas, solo en este ordenador. Con un número, enseña qué cambió
esa versión respecto a la anterior, línea a línea. Funciona aunque `config.toml` esté
roto.

## augmented-brain config restore

Vuelve a una versión de la configuración.

Sintaxis: `augmented-brain config restore <número>`

```bash
augmented-brain config restore 3
```

Enseña qué cambia y lo escribe. La configuración que tenías se guarda antes como una
versión, así que la vuelta atrás se deshace igual. Una versión que dejaría un fichero que
el programa no puede leer se rechaza, y no cambia nada. Funciona aunque `config.toml` esté
roto, igual que `config history`.

Si las listas del guardia quedan distintas de las que aceptaste, el guardia pregunta
antes de cada commit, push o mezcla que antes dejaba pasar. Sigue preguntando hasta que
aceptas las listas en la página Guardia del panel, o empiezas una sesión nueva. Un
comando lo puede lanzar el asistente, así que nunca las acepta por ti. Volver atrás desde
la página Versiones del panel sí las acepta.

## augmented-brain clean

Quita lo que deja el trabajo con Claude Code y nadie limpia. Primero enseña la lista, y
solo lo quita si dices que sí.

Sintaxis: `augmented-brain clean [--yes]`

```bash
augmented-brain clean
```

Mira en cada carpeta en la que se abrió Claude Code, y en su carpeta temporal:

- **Worktrees que hizo Claude Code** (en `.claude/worktrees`) con la sesión terminada.
  Uno se va solo si no tiene cambios sin commitear, ni commits que no estén en el remoto,
  ni ficheros que git ignora y que ningún build vuelve a hacer (un `.env`, una clave): se
  quita con `git worktree remove`, nunca borrando la carpeta. Su rama se va con él si se
  la puso Claude Code (`worktree-<nombre>`); una rama tuya se queda. Un worktree con
  trabajo propio solo se nombra, para que lo mires.
- **Worktrees que git recuerda y cuya carpeta ya no está**: `git worktree prune` los
  olvida. Sus ramas se quedan.
- **Carpetas temporales de sesiones que acabaron** (`/tmp/claude-<número>` en Linux), sin
  tocar desde hace más de un día. Las de sesiones en marcha se quedan, y todo lo demás
  que haya ahí también.

Antes de quitar cada cosa vuelve a mirar, así que una sesión que haya empezado entretanto
conserva sus carpetas. `doctor` avisa cuando hay algo que limpiar.

- `--yes`: no pregunta.

## augmented-brain update

Actualiza augmented brain y su plugin de Claude Code.

Sintaxis: `augmented-brain update [--plugin] [--force] [--no-restart]`

```bash
augmented-brain update
```

Primero pregunta a GitHub por la última versión, una vez: si GitHub no contesta, lo dice
y se para. Si no hay una versión más nueva, solo actualiza el plugin. Si la hay, lanza el
instalador publicado, que cambia el programa y después actualiza el plugin. Alrededor
hace lo que si no harías a mano:

- **Antes:** la revisión de `doctor` (lo que falla se enseña, para que no se tome por un
  problema de la versión nueva), y una copia de seguridad de cada espacio que la tenga.
  Si una copia falla, se para y no actualiza nada: esa copia es el camino de vuelta. Sin
  ninguna copia, avisa de que sigue sin copia a la que volver.
- **El instalador:** si falla su descarga (tres intentos, también por la API de GitHub),
  `update` acaba con error y no ha cambiado nada. El instalador prueba el programa nuevo
  antes de cambiar el viejo: si no arranca, el viejo se queda. Si después el programa no
  es la versión nueva, `update` lo dice y acaba con error.
- **Después:** las sesiones de Claude quietas que siguen con la versión vieja se
  reinician, una a una, como hace `session restart --old`: las de Claude Desktop siguen
  abiertas y solo se reinicia su augmented brain. Para las que no puede reiniciar (una
  que está respondiendo, o en Windows una de Claude Desktop), dice qué tarea está activa
  en cada espacio, para que pidas a su sesión que la aparque, y después las cierras y las
  abres.

Opciones:

- `--plugin`: actualiza solo el plugin de Claude Code. Lo usa el instalador.
- `--force`: actualiza aunque falle la copia de antes.
- `--no-restart`: no reinicia ninguna sesión después de actualizar; solo dice cuáles
  siguen con la versión vieja.
- `--from <dirección>`: para pruebas: instala la versión de esa dirección en vez de la
  última de GitHub. Tiene que ser `https://`, `file://`, o `http://` a este ordenador o al
  host de una VM de pruebas (`10.0.2.2`).

`update` no hace caso de `AB_RELEASE_URL` ni de `AB_MARKETPLACE`, y lo dice: un programa
podría ponerlas para que la actualización venga de otro sitio. De las demás variables
`AB_`, solo pasa `AB_CONFIG_DIR` y `AB_INSTALL_DIR` al instalador.

## augmented-brain uninstall

Quita el programa y el plugin de Claude Code. Tus notas se quedan.

Sintaxis: `augmented-brain uninstall [--config | --all] [--yes]`

```bash
augmented-brain uninstall
```

Enseña lo que va a quitar y pregunta antes. Primero se van los servicios del acceso
remoto y las sesiones abiertas para tareas; si uno no se puede parar, no se quita nada
más.

- `--config`: quita también la configuración y el perfil.
- `--all`: empezar de cero: también la configuración, el perfil y las notas de todos los
  espacios. Enseña cada carpeta de notas con cuántas notas tiene, y avisa de las que no
  tienen copia de seguridad, que se perderán para siempre. Solo se va lo que augmented
  brain guarda en una carpeta de notas; una carpeta que contiene una carpeta de trabajo,
  o que no tiene notas, se queda entera. Tus copias de seguridad no se tocan.
- `--yes`: no pregunta.

Lo tienes en [Instalar](install.es.md#desinstalar).

## augmented-brain mcp

Lanza el servidor MCP: las herramientas con las que Claude lleva tus tareas y notas.
**Lo arranca Claude Code**, a través del plugin; tú no.

Habla por su entrada y su salida. Vuelve a buscar el espacio en cada llamada, a partir de
la carpeta en la que trabaja Claude. Las herramientas están en
[Herramientas MCP](mcp-tools.es.md).

## augmented-brain hook

Contesta a un hook de Claude Code: una llamada que hace Claude Code en un momento dado.
**Lo lanza Claude Code**, a través del plugin; tú no. Lee lo que Claude Code le manda por
su entrada.

Sintaxis: `augmented-brain hook <evento> [<parte>]`

```bash
augmented-brain hook session-start 1
```

Un problema va a su salida de errores y el hook acaba igual con 0, así que nunca para a
Claude. La única excepción es el guardia (`pre-tool-use`): si no puede dar su respuesta,
acaba con 2, y Claude Code para el comando.

Los eventos:

- `session-start`: empieza una sesión. Le da a Claude el perfil, las reglas, el índice de
  tareas y lo que toca. Va en hasta nueve partes, de `session-start 1` a
  `session-start 9`, porque Claude Code corta el texto de cada hook a los 10.000
  caracteres. Solo se mandan las partes que tienen algo.
- `pre-tool-use`: antes de cada comando y cada cambio de un fichero. Es el
  [guardia](guard.es.md).
- `user-prompt-submit`: antes de que Claude conteste a un mensaje. Le dice a Claude qué
  tareas han cambiado fuera de su sesión desde la última vez que miró (otra sesión cerró
  una, el panel cambió su siguiente paso), para que las vuelva a leer en vez de fiarse de
  lo que leyó antes. Nunca retiene el mensaje.
- `stop`: después de cada respuesta. Tarda milisegundos, y:
  - recuerda a Claude que escriba en el diario de la tarea de su sesión, una vez,
    después de 20 minutos sin escribir;
  - con `preferred_model` puesto (mira [Configuración](configuration.es.md)), hace que
    Claude te diga cuándo contestó otro modelo, una vez por sesión y modelo;
  - te enseña, una vez, un cambio de las listas del guardia o algo que el asistente
    guardó para todas las sesiones;
  - en Linux y macOS, mira cuánta memoria usa la ventana (el host de extensiones de VS
    Code, o el propio Claude en una terminal), y hace que Claude te avise cuando pasa de
    2 GB, y después de 4 y 8 GB, una vez cada uno (una fracción de milisegundo). Mira
    [Si algo falla](troubleshooting.es.md#la-ventana-usa-mucha-memoria).
- `post-compact`: después de que Claude Code compacte una conversación larga. Guarda el
  resumen en `archive/sessions/` y lo enlaza desde el diario de la tarea.
- `session-end`: acaba la sesión. Apunta que acabó bien, y arranca la copia de seguridad
  por detrás.

## augmented-brain version

Escribe la versión del programa.

Sintaxis: `augmented-brain version`

```bash
augmented-brain version
```

`--version` y `-v` hacen lo mismo.

## augmented-brain help

Escribe la lista de comandos, en tu idioma.

Sintaxis: `augmented-brain help`

```bash
augmented-brain help
```

`--help` y `-h` hacen lo mismo. Lanzar `augmented-brain` sin nada también escribe la
lista, como error (código de salida 2).

## Códigos de salida

Cada comando acaba con uno de estos códigos. Un script puede fiarse de ellos: no cambian
de una versión a otra (mira [Compatibilidad](compatibility.es.md)). Los mensajes sí
pueden cambiar.

| Código | Qué quiere decir |
|---|---|
| `0` | Hizo lo que se le pidió. |
| `1` | Algo falló, o no acabó bien. El mensaje dice qué. |
| `2` | No se lanzó: la línea de comando está mal. |

Una línea de comando mal es un comando o una opción que no existen, un argumento que
falta o que sobra, o un valor que no puede usar (`--language fr`).

Algunos comandos dan a sus códigos un sentido propio:

- `check` y `doctor`: `1` si una nota no es válida, o algo falla.
- `watch pr`: `0` solo si los checks pasaron o la pull request se mezcló; `1` en todo lo
  demás (cerrada, un conflicto, un check fallido, `--max` alcanzado).
- `hook`: `0` aunque algo falle dentro, para no parar nunca a Claude. La excepción es el
  guardia (`pre-tool-use`), que acaba con `2` para parar el comando si no puede dar su
  respuesta.
