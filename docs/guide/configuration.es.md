# Configuración

[Read in English](configuration.md)

El instalador escribe la configuración por ti, y Claude le añade cosas cuando se lo pides
("`tienda-prod` es producción"). Solo la abres para cambiar algo a mano: una carpeta de
trabajo nueva, un repositorio donde Claude puede hacer commit, una excepción de root.

## Dónde está

Hay una carpeta por ordenador, la **carpeta de configuración**:

- Linux y macOS: `~/.config/augmented-brain/` (o `$XDG_CONFIG_HOME/augmented-brain/`).
- Windows: `%APPDATA%\augmented-brain\`.
- Cualquier otra, si defines `AB_CONFIG_DIR`.

Dentro, `config.toml` guarda los ajustes. Es un fichero **TOML**: líneas `nombre = valor`,
y un encabezado `[sección]` para cada espacio. Las líneas que empiezan por `#` son
comentarios; el programa los conserva cuando cambia el fichero.

## Un ejemplo

```toml
# ~/.config/augmented-brain/config.toml
language = "es"
default_space = "personal"
production_aws_profiles = ["tienda-prod", "tienda-prod-eu"]
untouchable_hosts = ["db01", "backup.example.com"]
root_ask = ["web02 until 2026-12-31"]

[spaces.trabajo]
memory = "~/augmented-brain/trabajo/notas"
work_folders = ["~/augmented-brain/trabajo", "~/code"]
backup = "git@gitlab.com:lucia/brain-trabajo.git"
git_allowed_repos = ["~/code/blog", "~/code/blog-*"]
git_ask_repos = ["~/code/tienda"]

[spaces.personal]
memory = "~/augmented-brain/personal/notas"
work_folders = ["~/augmented-brain/personal"]
backup = "https://github.com/lucia/augmented-brain-personal.git"
```

Las rutas pueden empezar por `~`, tu carpeta personal. El programa lee el fichero cada vez
que se ejecuta, así que un cambio vale al momento.

## Ajustes de todo el ordenador

Van arriba, antes de cualquier encabezado `[spaces.…]`.

- `language`: el idioma en el que te habla el programa, `en` o `es`: sus mensajes y los
  motivos del guardia. Por defecto, `en`.
- `default_space`: el espacio que se usa fuera de todas las carpetas de trabajo, como la
  carpeta provisional en la que la app de escritorio abre una sesión nueva. Su copia de
  seguridad lleva tu perfil y las notas compartidas. Es opcional; con dos espacios o más,
  lo ponen `init` y `space add`.
- `production_aws_profiles`: los perfiles de AWS que llegan a producción. El guardia te
  pregunta antes de cualquier comando que use uno, en todos los espacios. Lo tienes en
  [El guardia](guard.es.md).
- `untouchable_hosts`: máquinas que no se tocan sin preguntar. El guardia pregunta antes
  de un `ssh`, `scp`, `rsync`, `sftp` o `mosh` a ellas, en todos los espacios. Los nombres
  coinciden sin mirar mayúsculas, y un nombre corto coincide con el completo (`db01` y
  `db01.example.com`).
- `root_ask`: excepciones temporales para root, una máquina cada una, escritas
  `"<máquina> until <fecha>"`. La fecha es un día (`2026-12-31`, que dura hasta que
  acaba, en UTC) o una hora (`2026-12-31T23:59Z`). Hasta entonces, root en esa máquina te
  pregunta en vez de pararse. La fecha es obligatoria.
- `format`: la versión de la carpeta de configuración. El programa la escribe cuando una
  versión nueva lo necesita; no la cambies nunca a mano. Si no está, es 1.

## Ajustes de cada espacio

Cada espacio es una sección, `[spaces.<nombre>]`. Juntas son los `spaces` del fichero.
El nombre lleva minúsculas, cifras y guiones, y empieza por una letra.

- `memory`: la carpeta de notas del espacio. Obligatoria.
- `work_folders`: las carpetas que eligen este espacio cuando abres Claude en ellas, o en
  una carpeta dentro de ellas. Lo tienes en [Espacios](spaces.es.md).
- `backup`: la dirección del repositorio git privado del espacio. La escribe
  `backup setup`. Sin ella, el espacio no tiene copia de seguridad.
- `git_allowed_repos`: repositorios donde Claude puede hacer commit y push, escritos con
  `git -C <repo>`. La última parte de una ruta puede llevar un `*`, que vale por una
  carpeta. Un worktree de git de un repositorio de la lista cuenta como él.
- `git_ask_repos`: lo mismo, pero Claude te pregunta cada vez. Un repositorio que está en
  las dos listas pregunta.
- `production_aws_profiles` y `untouchable_hosts`: las versiones antiguas los guardaban
  en cada espacio. Ahí siguen funcionando, y el guardia los junta con las listas de
  arriba. Los nuevos van arriba.

Las listas de repositorios de todos los espacios cuentan juntas: un repositorio de la
lista de un espacio pasa desde una sesión de cualquiera de ellos.

## Lo que puede cambiar Claude

Claude nunca edita este fichero por su cuenta: el guardia te pregunta antes. Dos
herramientas le añaden cosas, y solo añaden:

- `safety_save` añade perfiles de producción y máquinas intocables.
- `backup_setup` escribe el `backup` de un espacio, y nunca lleva a otro sitio una copia
  que ya existe.

Para quitar cualquier cosa, edita tú el fichero.

## Cuando no se puede leer

Si el fichero tiene un error, el programa dice en qué línea. Un ajuste que no conoce
también es un error, y el mensaje lo nombra. Si el fichero lo escribió una versión más
nueva, el programa te dice que actualices. Estos mensajes salen en el idioma del sistema
(o en el de `AB_LANGUAGE`): el `language` del propio fichero aún no se puede leer. Un
fichero que se lee pero dice algo mal (un espacio sin `memory`, un `default_space` que no
es un espacio) se cuenta en su `language`.

Mientras no se puede leer el fichero, el guardia no conoce tus listas. Así que te
pregunta antes de cada `aws`, `ssh`, `scp`, `rsync`, `sftp` y `mosh`.
`augmented-brain doctor` te dice qué está mal.

## `secrets.toml`

Junto a `config.toml`, `secrets.toml` nombra los secretos que guardas en el llavero del
sistema, nunca sus valores:

```toml
secrets = ["telegram/alerts-bot: avisos al móvil", "smtp/newsletter: envío del boletín"]
```

Cada entrada es `servicio/nombre: para qué es`. El servicio y el nombre son palabras
sueltas, tal como los conoce el llavero. Claude le añade entradas con la herramienta
`secret_name_save`, y la página Secretos del panel la lista. Un valor se rechaza. Es un
fichero aparte porque las versiones anteriores a la 0.6.0 rechazan un ajuste de
`config.toml` que no conocen. La 0.6.0 guardaba la lista en `config.toml` como `secrets`;
`check` y el inicio de sesión la pasan aquí.

## Otros ficheros de la carpeta

- `profile.md`: tu perfil, para todos los espacios.
- `shared/`: las notas compartidas (reglas, referencias y lecciones para todos los
  espacios).
- `state/`: lo que este ordenador sabe de las sesiones de cada espacio. Nunca va a la
  copia de seguridad.
- `migrations/`: una copia completa de las notas antes de que una versión nueva las
  cambiara de formato.
- `panel.json` y `panel.lock`: dónde escucha el panel mientras está en marcha, y su
  bloqueo.
- `review-<espacio>.draft`: una revisión semanal empezada en el panel y aún sin
  terminar.

## Variables de entorno

Defínelas en tu shell para cambiar cómo se porta el programa:

- `AB_CONFIG_DIR`: usa esta carpeta como carpeta de configuración.
- `AB_LANGUAGE`: `en` o `es`, el idioma antes de que haya configuración, o mientras no
  se puede leer (lo usan el instalador e `init`).
- `AB_NO_AUTO_BACKUP=1`: no hace la copia de seguridad cuando acaba una sesión.
- `AB_NO_UPDATE_CHECK=1`: no pregunta a GitHub, una vez al día, si hay una versión nueva.
- `AB_NO_BROWSER=1`: `augmented-brain panel` escribe la dirección en vez de abrir el
  navegador.

El instalador lee alguna más:

- `AB_YES=1`: contesta que sí a todas las preguntas.
- `AB_NO_DESKTOP=1`: no ofrece la app de escritorio de Claude.
- `AB_INSTALL_DIR`: dónde poner el programa. Por defecto, `~/.local/bin`.
- `AB_NO_INIT=1`: solo instala el programa, sin preparar nada.
- `AB_VERSION`, `AB_RELEASE_URL` y `AB_MARKETPLACE`: para pruebas: qué versión, dónde
  están los ficheros de la versión publicada y de dónde sale el plugin.

## Los ajustes de Claude Code

Aquí importa un ajuste de Claude Code. Si Claude Code guarda una memoria propia para las
carpetas de tus espacios, se desvía de la del espacio. `augmented-brain doctor` te lo
dice, y se arregla con `"autoMemoryEnabled": false` en los ajustes de Claude Code, en el
fichero que nombra `doctor`.

El formato completo de cada fichero está en el
[formato de la memoria](../spec/memory-format.md) (en inglés).
