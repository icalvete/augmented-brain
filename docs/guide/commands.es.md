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
`space add`.

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
- la carpeta de notas de cada espacio, sus notas y su índice;
- Claude Code, git y el plugin;
- el guardia, de verdad, con un comando que tiene que parar;
- las listas del guardia, las excepciones de root en `root_ask` y las que ya han
  terminado;
- las sesiones de Claude que siguen con una versión anterior después de actualizar;
- una segunda memoria que Claude Code guarda por su cuenta para tus carpetas;
- las notas que otros usuarios del ordenador podrían leer;
- si hay espacio por defecto, y si una recuperación dejó vacías las listas del guardia;
- el acceso remoto, una línea por espacio, cuando lo has puesto con `remote install`: un
  aviso si su servicio no está en marcha, Claude aún no confía en su carpeta o el linger
  está apagado.

Acaba con error si algo falla. No tiene opciones. Lo tienes en
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

Conecta el espacio de esta carpeta a un repositorio git privado, y hace la primera copia.

```bash
augmented-brain backup setup git@gitlab.com:lucia/brain-trabajo.git
```

Recibe un argumento, la dirección del repositorio, y no tiene opciones. Para que te cree
el repositorio en tu cuenta de GitHub, pídeselo a Claude: "monta la copia de seguridad".

## augmented-brain restore

Recupera un espacio de su copia de seguridad, en una máquina nueva.

```bash
augmented-brain restore https://github.com/lucia/augmented-brain-trabajo.git
```

Cambia el espacio vacío del mismo nombre por el tuyo. Un espacio que ya tiene notas lo
deja como está. Si falta el plugin de Claude Code, lo instala. Después nombra lo que no
ha traído: las listas del guardia, los repositorios donde Claude puede hacer commit y los
secretos que faltan en el llavero de esta máquina, cada uno con el comando para
guardarlo.

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
más abajo. Solo en Linux por ahora (usa un servicio de usuario de `systemd`); en macOS y
Windows dicen que aún no está disponible. Sin subcomando, o con otro, `remote` escribe su
uso. Lo tienes en [Acceso remoto](remote.es.md).

## augmented-brain remote install

Pone el acceso remoto, un servicio por espacio.

```bash
augmented-brain remote install
```

Para cada espacio que nombra `remote.toml` (por defecto, el espacio por defecto) crea una
carpeta dentro de la carpeta de configuración, la hace carpeta de trabajo de ese espacio,
escribe un servicio de usuario de `systemd` que ejecuta `claude remote-control` ahí y lo
arranca. Puedes lanzarlo otra vez cuando quieras: solo cambia lo que es distinto.

Luego dice lo que aún falta, y no hace nada de ello por ti:

- Claude tiene que **confiar en cada carpeta**. Escribe `cd <carpeta> && claude` para cada
  una que aún no es de confianza: lánzalo una vez y acepta la pregunta.
- El **linger** tiene que estar activado para que los servicios arranquen tras reiniciar
  sin que nadie haya entrado. Escribe el comando, `sudo loginctl enable-linger <tú>`,
  para que lo lances tú.

No tiene opciones. Los ajustes están en `remote.toml` (mira [Configuración](configuration.es.md)).

## augmented-brain remote uninstall

Para el acceso remoto y lo quita.

```bash
augmented-brain remote uninstall
```

Para y quita los servicios que escribió `remote install`, y pone `enabled = false` en
`remote.toml`. Las carpetas y sus entradas en `work_folders` se quedan: nunca borra nada
más. No tiene opciones.

## augmented-brain remote status

Dice cómo está el acceso remoto, y no cambia nada.

```bash
augmented-brain remote status
```

Para cada espacio: el nombre de la sesión (el que eliges en Claude Desktop o en
claude.ai), si el servicio está en marcha y arranca solo, y si Claude confía en su
carpeta. Luego, si el linger está activado. Debajo de cada problema, `→` dice qué hacer.
No tiene opciones.

## augmented-brain update

Actualiza augmented brain y su plugin de Claude Code.

```bash
augmented-brain update
```

Si hay una versión más nueva, lanza el instalador publicado, que cambia el programa y
después actualiza el plugin. Si no, solo actualiza el plugin. Después cierra Claude y
vuelve a abrirlo, para que las sesiones usen la versión nueva.

- `--plugin`: actualiza solo el plugin de Claude Code. Lo usa el instalador.

## augmented-brain uninstall

Quita el programa y el plugin de Claude Code. Tus notas se quedan.

```bash
augmented-brain uninstall
```

Antes enseña lo que va a quitar y te pregunta.

- `--config`: quita también la configuración y el perfil.
- `--all`: empezar de cero: también la configuración, el perfil y las notas de todos los
  espacios. Enseña cada carpeta de notas con cuántas notas tiene. Tus copias de seguridad
  no se tocan.
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
- `stop`: después de cada respuesta. Le recuerda a Claude que escriba en el diario de la
  tarea, una vez, después de 20 minutos sin escribir. También mira cuánta memoria usa la
  ventana (el host de extensiones de VS Code, o el propio Claude en un terminal), y hace
  que Claude te avise cuando pasa de 2 GB, luego de 4 y de 8 GB, una vez cada uno. En
  Linux y macOS; tarda una fracción de milisegundo. Mira
  [Problemas](troubleshooting.es.md#la-ventana-usa-mucha-memoria).
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
lista.
