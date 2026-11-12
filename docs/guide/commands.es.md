# Comandos

[Read in English](commands.md)

Todos los comandos del programa, con todas sus opciones. En el día a día no te hacen
falta: hablas con Claude. Sirven para preparar, comprobar y pasar a un ordenador nuevo.

Se lanzan en una terminal. Casi todos trabajan con **el espacio de la carpeta en la que
estás**. Fuera de las carpetas de tus espacios, usan el espacio por defecto.

```bash
cd ~/augmented-brain/trabajo
augmented-brain check
```

Las opciones van antes o después del resto de argumentos, con uno o dos guiones (`--all`
o `-all`). Si un comando recibe un argumento que no admite, lo dice y no hace nada.

Hay tres comandos que lanza Claude Code, no tú: `mcp`, `hook` y `panel serve`. Están aquí
para que sepas qué son.

Dentro de Claude Code hay un comando con barra: `/brain:panel` abre el panel, igual que
`augmented-brain panel`.

## augmented-brain init

Prepara todo: los espacios, la configuración y el plugin de Claude Code. El instalador lo
lanza por ti la primera vez.

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
- `--memory`: su carpeta de notas. Por defecto, `notas` dentro de la carpeta de trabajo.
- `--yes`: no pregunta; usa las opciones que le des y lo que hay por defecto.
- `--no-plugin`: no instala el plugin de Claude Code.

Con `--space`, `--work` o `--memory` se crea un solo espacio en vez de los dos.

```bash
augmented-brain init --space cliente --work ~/clientes/acme --language es
```

## augmented-brain space

Dice de qué espacio es esta carpeta, y dónde están sus notas.

```bash
augmented-brain space
```

Fuera de las carpetas de tus espacios, dice cuál es el espacio por defecto. Si no hay
espacio por defecto, lo dice y lista los espacios. No tiene opciones.

## augmented-brain space add

Añade otro espacio: un cliente, un proyecto aparte, un voluntariado.

```bash
augmented-brain space add cliente
```

El nombre va como argumento (el primero o el último) o con `--space`. Te pregunta la
carpeta de trabajo y la de notas, y te propone `~/augmented-brain/<nombre>` y su carpeta
`notas`. Si aún no había espacio por defecto, pone uno: `personal` si hay un espacio con
ese nombre, y si no, el espacio que ya estaba.

- `--space`: el nombre del espacio nuevo.
- `--work`: su carpeta de trabajo.
- `--memory`: su carpeta de notas.
- `--yes`: no pregunta; usa las opciones que le des y lo que hay por defecto.

No tiene `--language`. El idioma es el mismo para todos los espacios de este ordenador: el
ajuste `language`, que pone `init`. Lo tienes en [Configuración](configuration.es.md).

```bash
augmented-brain space add cliente --work ~/clientes/acme --yes
```

Lo tienes en [Espacios](spaces.es.md).

## augmented-brain check

Revisa las notas del espacio de esta carpeta, y rehace su índice si todas son válidas.

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
  guardaba la 0.6.0.

Lánzalo después de editar notas a mano. Acaba con error si una nota no es válida. No
tiene opciones.

## augmented-brain doctor

Dice qué falta y cómo arreglarlo.

```bash
augmented-brain doctor
```

Cada línea empieza con una marca: `✓` bien, `·` para que lo sepas, `!` un aviso, `✗`
algo que falla. Debajo de una línea que pide arreglo, `→` dice qué hacer. Revisa:

- la configuración, el perfil y en qué espacio está esta carpeta;
- la carpeta de notas de cada espacio, sus notas, su índice y si tiene copia de seguridad;
- Claude Code, git y el plugin;
- el guardia, de verdad, con un comando que tiene que parar;
- las listas del guardia, las excepciones de root en `root_ask` y las que ya han
  terminado;
- las sesiones de Claude que siguen con una versión anterior después de actualizar;
- una segunda memoria que Claude Code guarda por su cuenta para tus carpetas;
- las notas que otros usuarios del ordenador podrían leer;
- si hay espacio por defecto, y si una recuperación dejó vacías las listas del guardia;
- el acceso remoto, una línea por espacio, cuando lo has puesto con `remote install`: un
  aviso si su servicio no está en marcha, Claude aún no confía en su carpeta o (en Linux)
  el linger está apagado;
- cuántas versiones de la configuración se guardan, y si cambió por fuera de ellas (a
  mano);
- las listas del guardia cambiadas desde que las aceptaste, y un repositorio a la vez
  intocable y permitido;
- una ventana de Claude que usa mucha memoria (2 GB o más);
- el panel desde otro dispositivo, si lo has usado: Tailscale, HTTPS y una dirección que
  se quedó publicada.

Acaba con error si algo falla. Si la configuración no se puede leer, te lleva a
`config history` y `config restore`. No tiene opciones. Lo tienes en
[Si algo falla](troubleshooting.es.md).

## augmented-brain import

Trae una memoria que Claude Code ya escribió. Sin `--apply`, solo enseña lo que haría.

```bash
augmented-brain import ~/.claude/projects/<carpeta>/memory
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

```bash
augmented-brain backup setup git@gitlab.com:lucia/brain-trabajo.git
```

Recibe un argumento, la dirección del repositorio, y no tiene opciones. Para que te cree
el repositorio en tu cuenta de GitHub, pídeselo a Claude: "configura la copia de seguridad".

## augmented-brain restore

Recupera un espacio de su copia de seguridad, en una máquina nueva.

```bash
augmented-brain restore https://github.com/lucia/augmented-brain-trabajo.git
```

Cambia el espacio vacío del mismo nombre por el tuyo. Un espacio que ya tiene notas lo
deja como está. Si falta el plugin de Claude Code, lo instala. Después nombra lo que no
ha traído: las listas del guardia, los repositorios donde Claude puede hacer commit y los
secretos que faltan en el llavero de esta máquina, cada uno con el comando para
guardarlo. Si no puede preguntar al llavero (no hay, o está bloqueado), nombra los
secretos que no ha podido comprobar.

- `--memory`: dónde poner las notas del espacio. Por defecto, donde estaban, dentro de la
  carpeta de trabajo.
- `--work`: la carpeta de trabajo que elige el espacio. Por defecto, la que se guardó en
  la copia.

## augmented-brain panel

Abre el panel en el navegador: el día de un vistazo, y todas las páginas de tus espacios
para verlas y cambiarlas.

```bash
augmented-brain panel
```

Arranca el panel por detrás, o encuentra el que ya está en marcha, y abre el navegador en
Hoy, con el espacio de esta carpeta. Vuelve enseguida. El panel se para solo después de
30 minutos sin usarlo.

- `--no-browser`: solo escribe la dirección, sin abrir el navegador. `AB_NO_BROWSER=1`
  hace lo mismo.

Lo tienes en [El panel](panel.es.md).

## augmented-brain panel remote

Abre el panel para tus otros dispositivos (otro ordenador, el móvil), a través de
[Tailscale](https://tailscale.com), una aplicación gratuita que une tus dispositivos en una
red privada solo suya.

```bash
augmented-brain panel remote
```

Arranca el panel y lo publica solo en tu red de Tailscale. Si el panel ya está abierto,
ese mismo panel responde también por la red, así que las páginas que tienes abiertas
siguen funcionando. Escribe la dirección y un código QR para abrirla desde el móvil. Solo
pueden abrirla tus dispositivos, con la cuenta a la que pertenece este ordenador; la dirección deja de funcionar cuando el panel se para.
Lo que falte (el propio Tailscale, HTTPS en tu red, dejar publicar a tu usuario) lo
explica, con el paso que hay que dar: nunca instala ni cambia nada de Tailscale por ti.
Lo tienes en [El panel desde otro dispositivo](panel-remote.es.md).

## augmented-brain panel stop

Para el panel ahora.

```bash
augmented-brain panel stop
```

Dice si había un panel en marcha. El enlace del pie de cada página del panel hace lo
mismo.

## augmented-brain panel serve

El panel en sí, funcionando por detrás. Lo arranca `augmented-brain panel`; nunca lo
lanzas a mano. Se para solo después de 30 minutos sin usarlo.

## augmented-brain remote

Mantiene Claude accesible desde otro ordenador o desde el móvil, también después de
reiniciar. Son tres comandos, `install`, `uninstall` y `status`, cada uno con su sección
más abajo. Usa un servicio de usuario de `systemd` en Linux, un LaunchAgent en macOS y una
tarea del Programador de tareas en Windows. En Linux hay un cuarto, `watch`, que lanza un
temporizador, no tú. Sin subcomando, o con otro, `remote` escribe su
uso. Lo tienes en [Acceso remoto](remote.es.md).

## augmented-brain remote install

Pone el acceso remoto, un servicio por espacio.

```bash
augmented-brain remote install
```

Para cada espacio que nombra `remote.toml` (por defecto, el espacio por defecto) escribe
un servicio que ejecuta `claude remote-control` en la primera carpeta de trabajo del
espacio (o en la que diga `folders` en `remote.toml`), y lo arranca: un servicio de
usuario de `systemd` en Linux, un LaunchAgent en `~/Library/LaunchAgents` en macOS, una
tarea `\augmented-brain\remote-<espacio>` del Programador de tareas en Windows (funciona
mientras tienes la sesión abierta, sin ventana). Solo un espacio sin carpetas de trabajo
recibe una carpeta propia, `remote/<espacio>` en la carpeta de configuración, que pasa a ser su carpeta de trabajo.
Necesita `claude` (en tu PATH o donde lo deja su instalador) y una carpeta que exista. El
servicio lleva un PATH con las carpetas de `claude` y de `augmented-brain`, así que sus
sesiones encuentran los dos tras reiniciar. Puedes lanzarlo otra vez cuando quieras: solo
cambia lo que es distinto.

En Linux pone también el vigilante: un temporizador de `systemd`,
`augmented-brain-watch-remote`, que lanza `augmented-brain remote watch` cada dos minutos
(mira más abajo).

Luego dice lo que aún falta, y no hace nada de ello por ti:

- Claude tiene que **confiar en cada carpeta**. Escribe `cd <carpeta> && claude` para cada
  una que aún no es de confianza (en Windows, `cd "<carpeta>"; claude`): lánzalo una vez
  y acepta la pregunta.
- En Linux, el **linger** tiene que estar activado para que los servicios arranquen tras
  reiniciar sin que nadie haya entrado. Escribe el comando, `sudo loginctl enable-linger
  <tú>`, para que lo lances tú. macOS y Windows no tienen linger: el servicio arranca
  cuando entras en tu sesión.

Sin los servicios de usuario de systemd (un contenedor, o WSL sin systemd), sin launchctl
en macOS o sin el Programador de tareas en Windows, lo dice y no cambia nada. No tiene opciones. Los ajustes están en `remote.toml` (mira [Configuración](configuration.es.md)).

## augmented-brain remote uninstall

Para el acceso remoto y lo quita.

```bash
augmented-brain remote uninstall
```

Para y quita los servicios que escribió `remote install` (en Linux, primero el vigilante), y pone `enabled = false` en
`remote.toml`. Las carpetas y sus entradas en `work_folders` se quedan: nunca borra nada
más. No tiene opciones.

## augmented-brain remote status

Dice cómo está el acceso remoto, y no cambia nada.

```bash
augmented-brain remote status
```

Para cada espacio: el nombre de la sesión (el que eliges en Claude Desktop o en
claude.ai), si el servicio está en marcha y arranca solo, y si Claude confía en su
carpeta. Luego, en Linux, si el vigilante (mira `remote watch` abajo) está en marcha y
cuánto hace que miró por última vez, y si el linger está activado; en macOS y Windows, que arranca
cuando entras en tu sesión. Debajo de cada problema, `→` dice qué hacer.
No tiene opciones.

## augmented-brain remote watch

Recupera un servicio cuando el servidor archivó su sesión. Solo en Linux. Lo lanza su
temporizador cada dos minutos; no hace falta que lo lances tú.

```bash
augmented-brain remote watch
```

Un `claude remote-control` sano tiene un proceso de Claude para su sesión. Cuando el
servidor archiva la sesión (por ejemplo, porque la archivaste en claude.ai), ese proceso
termina y el servicio sigue en marcha sin sesión: nadie puede conectarse, y systemd no ve
nada raro. El vigilante ve un servicio sin sesión en dos miradas seguidas y lo reinicia.
Si después sigue sin sesión, Claude intentaba reabrir la sesión archivada: el vigilante
cambia el nombre de su registro (`bridge-pointer.json` pasa a
`bridge-pointer.json.archived`, en el proyecto de la carpeta en `~/.claude/projects`) y lo
reinicia otra vez, así que empieza una sesión nueva, en unos tres minutos. La
conversación de la sesión archivada no vuelve. Lo que hace queda en el registro del
sistema (`journalctl --user -u augmented-brain-watch-remote`). No tiene opciones.

En cada mirada también protege las sesiones de tus tareas. Parar un servicio para todo
lo que se arrancó dentro de él, y la sesión de una tarea abierta desde una sesión remota
(con `augmented-brain session start`, o `claude --bg`) puede arrancar allí el servicio en
segundo plano de Claude, del que dependen todas las sesiones de tareas. Así que el
vigilante saca de cada servicio lo que no arrancó su propio servidor (el servicio en
segundo plano de Claude, sus sesiones, cualquier otra cosa que dejó en marcha una
sesión) a un grupo propio, `augmented-brain-kept-<número>`. `remote install` y `remote
uninstall` hacen lo mismo antes de reiniciar o parar un servicio; si no se puede, se
detienen y lo dicen, y el vigilante no reinicia ese servicio.

## augmented-brain session

Abre una sesión remota propia para una tarea, para que desde otro ordenador puedas
trabajar en varias tareas a la vez, cada una en su sesión. Son cuatro comandos, `start`,
`stop`, `restart` y `list`, cada uno con su apartado abajo; `restart` también reinicia
las demás sesiones de Claude de este ordenador. Por ahora solo en Linux: solo
se ha probado ahí. Sin subcomando, o con otro, `session` muestra su uso. Mira
[Acceso remoto](remote.es.md#una-sesión-por-tarea).

## augmented-brain session start

Abre una sesión para una tarea, en la carpeta de la tarea.

```bash
augmented-brain session start task-new-website
augmented-brain session start task-new-website --folder ~/projects/website
```

Arranca una sesión de Claude propia para la tarea, en segundo plano (`claude --bg
--remote-control`), con el título de la tarea como nombre. En menos de un minuto sale
en Claude Desktop y claude.ai. Empieza sola: su primer mensaje le pide leer su tarea,
decirte cómo está y ponerse con el siguiente paso. Si la pidió otra sesión de Claude,
el primer mensaje también le pide avisarla en una línea cuando pare: la tarea cerrada o
aparcada, o esperándote. La carpeta es la del primer repositorio de la tarea que encuentra en las carpetas
de trabajo del espacio, o la que diga `--folder`. Una tarea que no nombra ningún
repositorio se abre en la primera carpeta de trabajo del espacio, que pueden compartir
varias así.

Un repositorio bare no tiene ficheros con los que trabajar, así que la tarea recibe un
worktree de git propio a su lado: `site-new-website` para la tarea `task-new-website` de
`site`, en una rama nueva con el nombre de la tarea, desde la rama por defecto de origin
tal como está ahora (antes la trae). Dos tareas de un mismo repositorio nunca comparten
carpeta, y una sesión nunca arranca sobre código viejo. Si se pide otra vez, usa el mismo
worktree. Lo mismo vale para un repositorio bare dado con `--folder`.

Lánzalo desde una carpeta del espacio
de la tarea: la sesión remota a la que entras desde otro ordenador puede lanzarlo por ti
(«ábreme una sesión para la tarea de la web»). Toda sesión en Linux sabe hacerlo.

Dos reglas la mantienen segura. La carpeta tiene que ser de confianza ella misma: que
confíe en una carpeta de arriba no basta, así que la primera vez el comando te dice que
abras Claude una vez en esa carpeta y aceptes la pregunta. Claude no lo preguntaría, y
los ajustes y hooks de un repositorio no deben correr sin preguntar. Un worktree de git
es la excepción: Claude guarda la respuesta con su repositorio principal, así que basta
con confiar en uno de los dos. Y una tarea por carpeta: dos tareas en una carpeta se
pisarían los cambios, así que una carpeta que ya usa la sesión de otra tarea se rechaza;
dale otra con `--folder` (un worktree de git, por ejemplo).

Espera la primera respuesta de la sesión, hasta un minuto, y vuelve en cuanto llega.
Una sesión que no puede trabajar no se queda abierta: si Claude Code no tiene la sesión
iniciada, o le ha caducado, no se abre nada, y si la primera respuesta es un error (se
acabó el login, el servicio está saturado), la sesión se cierra. En los dos casos el
comando dice qué ha pasado y qué hacer, como volver a entrar con `claude auth login`.

Si Claude paró la sesión de la tarea (lo hace con una que lleva horas sin usarse, y con
todas cuando se reinicia su servicio en segundo plano), `session start` la trae de
vuelta: la misma sesión, con toda su conversación, otra vez conectada en Claude Desktop
y claude.ai, y sin un primer mensaje nuevo. claude.ai enseña esa sesión como
desconectada y no puede traerla de vuelta por sí solo. Solo si Claude se niega se abre
una sesión nueva que la sustituye.

La sesión se cierra con `session stop` o al reiniciar el ordenador. Las sesiones
lanzadas no vuelven solas tras un reinicio: pídelas otra vez.

## augmented-brain session stop

Cierra la sesión de una tarea y la quita de Claude Desktop y claude.ai. Funciona desde
cualquier carpeta.

```bash
augmented-brain session stop task-new-website
```

Si `session start` creó un worktree para la tarea, `stop` también lo quita con su rama,
siempre que no se pierda nada: con cambios sin commitear o commits sin subir, lo deja y
lo dice. Cerrar la tarea también cierra su sesión, cuando termina su respuesta, con su
worktree. Las sesiones que terminan sin `stop` (un reinicio, una sesión cerrada desde
Claude Desktop) dejan el suyo: el próximo `session start` y `doctor` quitan los que no
tienen nada propio, cerrar la tarea quita el suyo, y `doctor` avisa de cualquiera que se
quede con trabajo dentro.

También cierra una sesión que Claude retiró (lo hace con una que lleva horas sin uso).
Para entonces el servicio de fondo de Claude puede estar parado; `stop` lo arranca un
momento, y se va solo. Si aun así Claude no puede quitar la sesión de sus listas, `stop`
lo dice y da el comando para hacerlo más tarde. Una tarea sin sesión recibe «no hay nada
que cerrar», nunca un éxito callado.

- `--when-idle`: espera a que la sesión termine su respuesta y después la cierra. Cerrar
  la tarea lo lanza solo, en segundo plano: la sesión que cierra su propia tarea suele
  estar aún respondiendo, y pararla ahí cortaría esa respuesta. Una sesión que espera un
  permiso cuenta como respondiendo. Espera hasta 30 minutos; una que siga respondiendo
  entonces se queda abierta. Lo que hizo queda en `state/session-stop.log`, en la
  carpeta de la configuración.

## augmented-brain session restart

Reinicia una sesión de Claude para que use el programa de ahora, con la misma
conversación. Funciona desde cualquier carpeta. Úsalo después de actualizar, o justo
después de aparcar una tarea, para que su sesión empiece de nuevo.

```bash
augmented-brain session restart task-new-website
augmented-brain session restart --old
```

La sesión se nombra por su tarea (si la abrió `session start`), su id, los ocho
primeros caracteres de su id de sesión o su nombre; `claude agents` las lista. Cada
tipo se reinicia a su manera:

- **La sesión de una tarea** (`claude --bg`): Claude le da un proceso nuevo con la
  misma conversación.
- **Una sesión del acceso remoto** (`remote install`): se reinicia su servicio, y la
  misma sesión vuelve a claude.ai y Claude Desktop en unos diez segundos. El servicio
  lleva todas las sesiones que sirve, así que solo se reinicia si todas están quietas.
  Antes se saca de él todo lo demás (sobre todo el servicio en segundo plano de Claude),
  para que no se pare con él. Solo en Linux.
- **Una sesión de Claude Desktop o de un terminal:** su ventana sigue abierta. La abrió
  su app, y pararla desde fuera rompería esa ventana, así que solo se para su servidor
  de augmented brain; Claude Code lo arranca otra vez, con el programa de ahora, la
  próxima vez que la sesión lo usa. En Linux y macOS. En Windows se queda como está:
  ciérrala y ábrela otra vez tú.

Nunca se reinicia una sesión que está respondiendo o esperando un permiso: se vuelve a
mirar justo antes y, si no está quieta, se deja. Tampoco la sesión desde la que se
lanza la orden: esa necesita `--when-idle`. Si una falla, las demás siguen.

- `--old`: reinicia, una a una, todas las sesiones quietas que siguen con un augmented
  brain anterior, y dice qué ha pasado con cada una. `update` lo lanza solo.
- `--when-idle`: espera a que la sesión acabe su respuesta y entonces la reinicia. Así
  se reinicia una sesión a sí misma después de aparcar su tarea: la orden vuelve en el
  acto y un proceso en segundo plano espera, hasta 30 minutos. Lo que hizo va a
  `state/session-restart.log` en la carpeta de configuración.

## augmented-brain session list

Lista las sesiones de tareas abiertas en este ordenador: la tarea, si está en marcha y
su carpeta. **retirada** quiere decir que Claude la paró, tras horas sin uso o al
reiniciarse su servicio en segundo plano: claude.ai la enseña desconectada, y `session
start` con su tarea la trae de vuelta con toda su conversación. **fallida** quiere decir que su
última respuesta fue un error, como un login caducado: no hace nada hasta que alguien
le escribe. La línea dice el error y qué hacer.

```bash
augmented-brain session list
```

## augmented-brain watch

Vigila algo que tarda y dice cómo ha terminado, para que una sesión lo deje en marcha
y siga con otra cosa. Por ahora vigila una cosa, una pull request: mira `watch pr`
abajo. Sin subcomando, `watch` muestra su uso.

## augmented-brain watch pr

Vigila una pull request de GitHub hasta que hay algo que contar: se mezcla o se cierra,
tiene conflictos con su rama base (entonces no corre ninguna CI), falla un check obligatorio, terminan todos los obligatorios o pasa el tiempo de `--max`.
Entonces termina sola y dice cómo ha ido, en una línea.

```bash
augmented-brain watch pr 104 --task task-my-change
```

- `--task TAREA` apunta esa línea en la tarea: en su diario (con el siguiente paso que
  tenía) y como su siguiente paso. Las demás sesiones se enteran antes de su próxima
  respuesta, y Hoy, en el panel, lo enseña. La vigilancia la hace entonces un proceso
  propio, que sigue aunque la sesión que la lanzó se cierre, se reinicie o Claude la
  retire: la sesión solo espera su resultado. Si se pide otra vez para la misma pull
  request y tarea, espera a la que ya está en marcha en vez de lanzar otra. La tarea
  tiene que ser del espacio de esta carpeta.
- `--repo DUEÑO/NOMBRE` es el repositorio; por defecto, el de esta carpeta.
- `--max` es lo más que vigila (3h por defecto); `--every`, cada cuánto mira (30s por
  defecto).

Los checks que no son obligatorios no deciden nada, pero la línea nombra los que siguen
en marcha o han fallado. Nunca espera para siempre: si GitHub no da ningún check
obligatorio en 5 minutos, o gh falla tres veces seguidas o da una respuesta que no
entiende, para y dice qué ha dicho gh. Necesita `gh` con tu login de GitHub. El
asistente la lanza en segundo plano; su última línea le pide que te lo diga en el acto.
Termina con 0 si los checks han pasado o la pull request se ha mezclado, y con 1 si no.

## augmented-brain config

Las versiones de tu configuración: dos comandos, `history` y `restore`, cada uno con su
apartado abajo. Sin subcomando, `config` muestra cómo se usa. Mira
[Versiones de la configuración](config-history.es.md).

## augmented-brain config history

Lista las últimas versiones de tu configuración, la más reciente primero, y no cambia
nada.

```bash
augmented-brain config history
augmented-brain config history 3
```

Una versión es una copia de `config.toml`, `guard.toml`, `remote.toml`, `session.toml`,
`profile.md` y `secrets.toml` tal como estaban en un momento, con quién los cambió: el
panel, un comando, el asistente (y con qué herramienta), una vuelta a otra versión, o a
mano. Se guardan las 10 últimas, solo en este ordenador. Con un número, enseña qué cambió
esa versión respecto a la anterior, línea a línea. No tiene opciones.

## augmented-brain config restore

Vuelve a una versión de la configuración.

```bash
augmented-brain config restore 3
```

Enseña lo que cambia y lo escribe. Antes guarda como versión la configuración que
tenías, así que volver atrás se puede deshacer igual. Una versión que dejaría un fichero
que el programa no puede leer se rechaza, y no cambia nada. Si las listas del guardia
quedan distintas de las que aceptaste, el guardia pregunta antes de cada commit, push o
merge que antes dejaba pasar. Sigue preguntando hasta que aceptes las listas en la página
Guardia del panel, o empieces una sesión nueva. Un comando lo puede lanzar el asistente,
así que nunca las acepta por ti. Volver desde la página Versiones del panel sí las acepta. Funciona aunque
`config.toml` esté roto, igual que `config history`. No tiene opciones.

## augmented-brain update

Actualiza augmented brain y su plugin de Claude Code.

```bash
augmented-brain update
```

Si hay una versión más nueva, lanza el instalador publicado, que cambia el programa y
después actualiza el plugin. Si no, solo actualiza el plugin. Primero pregunta a GitHub
cuál es la última versión, una sola vez: si GitHub no responde, lo dice y se para. Si la
descarga del instalador falla (tres intentos, también por la API de GitHub) o el programa no queda en la versión nueva, lo
dice, acaba con error y no cambia nada. Alrededor de una versión nueva hace lo que de
otro modo harías a mano:

- **Antes:** la revisión de `doctor` (lo que falla se enseña, para no tomarlo por un
  problema de la versión nueva) y una copia de seguridad de cada espacio que la tenga. Si
  falla una copia, para y no actualiza nada: esa copia es la vuelta atrás. Si no hay
  ninguna copia, dice que sigue sin una copia a la que volver.
- **Después:** reinicia, una a una, las sesiones de Claude quietas que siguen con la
  versión vieja, como hace `session restart --old`: las de Claude Desktop siguen
  abiertas y solo se reinicia su augmented brain. Para las que no puede reiniciar (una
  que está respondiendo, o en Windows una de Claude Desktop), dice qué tarea está activa
  en cada espacio, para que pidas a su sesión que la aparque, y después las cierras y
  las abres.

- `--plugin`: actualiza solo el plugin de Claude Code. Lo usa el instalador.
- `--force`: actualiza aunque falle la copia de antes.
- `--no-restart`: no reinicia ninguna sesión después de actualizar; solo dice cuáles
  siguen con la versión vieja.

## augmented-brain uninstall

Quita el programa y el plugin de Claude Code. Tus notas se quedan.

```bash
augmented-brain uninstall
```

Antes enseña lo que va a quitar y te pregunta.

- `--config`: quita también la configuración y el perfil.
- `--all`: empezar de cero: también la configuración, el perfil y las notas de todos los
  espacios. Enseña cada carpeta de notas con cuántas notas tiene, y avisa de las que no
  tienen copia de seguridad, que se perderán para siempre. Tus copias de seguridad no se
  tocan.
- `--yes`: no pregunta.

Lo tienes en [Instalación](install.es.md#desinstalar).

## augmented-brain mcp

Arranca el servidor MCP: las herramientas que usa Claude para llevar tus tareas y tus
notas. **Lo arranca Claude Code**, a través del plugin; tú no.

Habla por su entrada y su salida. En cada llamada vuelve a buscar el espacio, por la
carpeta en la que trabaja Claude. Las herramientas están en
[Herramientas MCP](mcp-tools.es.md). No tiene opciones.

## augmented-brain hook

Responde a un hook de Claude Code: un aviso que Claude Code lanza en un momento concreto.
**Lo lanza Claude Code**, a través del plugin; tú no. Lee lo que Claude Code le manda por
su entrada, y nunca hace fallar a Claude: un problema va a su salida de errores.

```bash
augmented-brain hook session-start 1
```

Los eventos:

- `session-start`: empieza una sesión. Le pasa a Claude el perfil, las reglas, el índice
  de tareas y lo que vence. Llega en hasta nueve partes, de `session-start 1` a
  `session-start 9`, porque Claude Code corta el texto de cada hook a los 10.000
  caracteres. Solo se mandan las partes que tienen algo.
- `pre-tool-use`: antes de cada comando y de cada cambio de fichero. Es el
  [guardia](guard.es.md). Si no puede ejecutarse, Claude Code para el comando.
- `user-prompt-submit`: antes de que Claude responda a un mensaje. Le dice qué tareas
  han cambiado fuera de su sesión desde la última vez que miró (otra sesión cerró una, el
  panel cambió su siguiente paso), para que las vuelva a leer en vez de fiarse de lo que
  leyó antes. Nunca retiene el mensaje.
- `stop`: después de cada respuesta. Le recuerda a Claude que escriba en el diario de la
  tarea de su sesión, una vez, después de 20 minutos sin escribir. También mira cuánta memoria usa la
  ventana (el host de extensiones de VS Code, o el propio Claude en un terminal), y hace
  que Claude te avise cuando pasa de 2 GB, luego de 4 y de 8 GB, una vez cada uno. En
  Linux y macOS; tarda una fracción de milisegundo. Mira
  [Si algo falla](troubleshooting.es.md#la-ventana-usa-mucha-memoria).
- `post-compact`: después de que Claude Code compacte una conversación larga. Guarda el
  resumen en `archive/sessions/` y el diario de la tarea apunta a él.
- `session-end`: acaba la sesión. Apunta que acabó bien, y lanza la copia de seguridad
  por detrás.

## augmented-brain version

Muestra la versión del programa.

```bash
augmented-brain version
```

`--version` y `-v` hacen lo mismo.

## augmented-brain help

Muestra la lista de comandos, en tu idioma.

```bash
augmented-brain help
```

`--help` y `-h` hacen lo mismo. Lanzar `augmented-brain` sin nada también muestra la
lista, como un error (código de salida 2).
