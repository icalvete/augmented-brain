# Configuración

[Read in English](configuration.md)

El instalador escribe la configuración por ti, y Claude le añade cosas cuando se lo pides
("`tienda-prod` es producción"). Para cambiar lo que deja pasar el guardia (un
repositorio donde Claude puede hacer commit, una excepción de root, una máquina que no se
toca), usa la página **Guardia** del panel: explica cada lista y mantiene al guardia al
tanto. Abre los ficheros a mano solo para lo demás, como una carpeta de trabajo nueva.
Lo tienes en [Lo que puede cambiar Claude](#lo-que-puede-cambiar-claude).

## Dónde está

Hay una carpeta por ordenador, la **carpeta de configuración**:

- Linux y macOS: `~/.config/augmented-brain/` (o `$XDG_CONFIG_HOME/augmented-brain/`).
- Windows: `%APPDATA%\augmented-brain\`.
- Cualquier otra, si defines `AB_CONFIG_DIR`.

Dentro, `config.toml` guarda los ajustes. Es un fichero **TOML**: líneas `nombre = valor`,
y un encabezado `[sección]` para cada espacio. Las líneas que empiezan por `#` son
comentarios; el programa los conserva cuando cambia el fichero.

## Los ficheros

| Fichero | Qué guarda | Quién lo escribe | En la copia de seguridad |
|---|---|---|---|
| `config.toml` | Tus espacios, el idioma y las listas del guardia | El instalador, `space add`, `backup setup`, la página Guardia, `safety_save` (solo añade) | La sección de cada espacio, en la copia de ese espacio. Las listas de todo el ordenador, nunca |
| `guard.toml` | Los repositorios que Claude nunca toca | La página Guardia | No |
| `session.toml` | El modelo que esperas y si Claude pregunta antes de guardar una regla | La página Guardia | No |
| `secrets.toml` | Los nombres de tus secretos, nunca sus valores | `secret_name_save`, la página Secretos | Sí, en la copia del espacio por defecto |
| `remote.toml` | El acceso remoto | `remote install` y `remote uninstall` | No |
| `profile.md` | Tu perfil | `profile_save`, la página Perfil | Sí, en la copia del espacio por defecto |

Por qué tantos ficheros: un programa anterior al que añadió un ajuste rechaza un ajuste
de `config.toml` que no conoce, pero no hace caso de un fichero que no conoce. Así que
cada tipo de ajuste nuevo tiene su propio fichero, y una versión anterior sigue leyendo
tu configuración.

Las listas de todo el ordenador (perfiles de producción, máquinas intocables,
excepciones de root) no van en las copias de seguridad, porque pueden nombrar máquinas
de tu trabajo. Tras restaurar en un ordenador nuevo, la
siguiente sesión se ofrece a preguntártelas otra vez.

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
  motivos del guardia. Por defecto, `en`. Lo pregunta el instalador; cámbialo aquí cuando
  quieras. Cuenta al momento. Las carpetas y notas ya creadas no cambian de nombre ni de idioma.
- `default_space`: el espacio que se usa fuera de todas las carpetas de trabajo, como la
  carpeta provisional en la que la app de escritorio abre una sesión nueva. Su copia de
  seguridad lleva tu perfil y las notas compartidas. Es opcional; con dos espacios o más,
  lo ponen `init` y `space add`.
- `production_aws_profiles`: los perfiles de AWS que llegan a producción. El guardia te
  pregunta antes de cualquier comando que use uno, en todos los espacios. Lo tienes en
  [El guardia](guard.es.md).
- `untouchable_hosts`: máquinas que no se tocan sin preguntar. El guardia pregunta antes
  de un `ssh`, `scp`, `rsync`, `sftp`, `mosh`, `sshfs` o `autossh` a ellas, en todos los
  espacios. Los nombres
  coinciden sin mirar mayúsculas, y un nombre corto coincide con el completo (`db01` y
  `db01.example.com`).
- `root_ask`: excepciones temporales para root, una máquina cada una, escritas
  `"<máquina> until <fecha>"`. La fecha es un día (`2026-12-31`, que dura hasta que
  acaba, en UTC) o una hora (`2026-12-31T23:59Z`). Hasta entonces, el guardia te pregunta
  antes de un comando como root en esa máquina, en vez de pararlo. La fecha es
  obligatoria. Lo tienes en
  [Root en una máquina, durante un tiempo](guard.es.md#root-en-una-máquina-durante-un-tiempo).
- `format`: la versión de la carpeta de configuración. El programa la escribe cuando una
  versión nueva lo necesita; no la cambies nunca a mano. Si no está, es 1.

## Ajustes de cada espacio

Cada espacio es una sección, `[spaces.<nombre>]`. Juntas son los `spaces` del fichero.
El nombre lleva minúsculas, cifras y guiones, y empieza por una letra. Un fichero sin
ningún espacio es un error: tiene que haber uno.

- `memory`: la carpeta de notas del espacio. Obligatoria.
- `work_folders`: las carpetas que eligen este espacio cuando abres Claude en ellas, o en
  una carpeta dentro de ellas. Si hay carpetas de trabajo una dentro de otra (`~/code` en
  un espacio, `~/code/tienda` en otro), gana la más profunda. Lo tienes en
  [Espacios](spaces.es.md).
- `backup`: la dirección del repositorio git privado del espacio. La escribe
  `backup setup`. Sin ella, el espacio no tiene copia de seguridad.
- `git_allowed_repos`: repositorios donde Claude puede hacer commit, push y merge de una
  pull request sin preguntar. Claude tiene que nombrar el repositorio en el comando
  (`git -C ~/code/blog push`, `cd ~/code/blog && gh pr merge 12`): la carpeta en la que
  está la sesión no cuenta por sí sola. La última parte de una ruta puede llevar un `*`,
  que vale por una carpeta. Un worktree de git de un repositorio de la lista cuenta como
  él. Lo tienes en
  [Deja que Claude haga commit en un repositorio](guard.es.md#deja-que-claude-haga-commit-en-un-repositorio).
- `git_ask_repos`: lo mismo, pero el guardia te pregunta cada vez. Un repositorio que
  está en las dos listas pregunta.
- `production_aws_profiles` y `untouchable_hosts`: las versiones antiguas los guardaban
  en cada espacio. Ahí siguen funcionando, y el guardia los junta con las listas de
  arriba. Los nuevos van arriba.

Las listas de repositorios de todos los espacios cuentan juntas: un repositorio de la
lista de un espacio pasa desde una sesión de cualquiera de ellos.

## Lo que puede cambiar Claude

Claude nunca edita estos ficheros por su cuenta: el guardia te pregunta antes. Dos
herramientas le añaden cosas a `config.toml`, y solo añaden:

- `safety_save` añade perfiles de producción y máquinas intocables. Añadir solo hace que
  el guardia pregunte más, así que unas listas que habías aceptado siguen aceptadas.
- `backup_setup` escribe el `backup` de un espacio, solo para el repositorio que crea en
  tu cuenta de GitHub, y nunca lleva a otro sitio una copia que ya existe.

Para quitar algo de las listas del guardia, o para dejar que Claude haga más, usa la
página **Guardia** del panel. También puedes editar los ficheros a mano, pero entonces el
guardia ve unas listas que no has aceptado: te pregunta antes de cada commit, push o
merge que dejarían pasar, y también antes de cada comando `aws` y de cada conexión a otra
máquina, hasta que las aceptes en la página Guardia o empieces una sesión nueva. Lo
tienes en
[Solo tú cambias las listas del guardia](guard.es.md#solo-tú-cambias-las-listas-del-guardia).

## Cuando no se puede leer

Si el fichero tiene un error, el programa dice en qué línea. Un ajuste que no conoce
también es un error, y el mensaje lo nombra. Si el fichero lo escribió una versión más
nueva, el programa te dice que actualices. Un fichero con un error se cuenta en el idioma
del sistema (o en el de `AB_LANGUAGE`), porque su propio `language` no se puede leer. Todo
lo demás se cuenta en el `language` del fichero: un ajuste que no conoce, una versión más
nueva, un espacio sin `memory`, un `default_space` que no es un espacio. Si el propio
`language` está mal, en el del sistema.

Mientras no se puede leer el fichero, el guardia no conoce tus listas. Así que te
pregunta antes de cada comando `aws` y de cada conexión a otra máquina (`ssh`, `scp`,
`rsync`, `sftp`, `mosh`, `sshfs`, `autossh`).
`augmented-brain doctor` te dice qué está mal.

Para volver a una versión que funcionaba, lanza `augmented-brain config history` y
después `augmented-brain config restore <número>`: los dos funcionan aunque el fichero
esté roto. Lo tienes en [Versiones de la configuración](config-history.es.md).

## `secrets.toml`

Junto a `config.toml`, `secrets.toml` nombra los secretos que guardas en el llavero del
sistema, nunca sus valores:

```toml
secrets = ["telegram/alerts-bot: avisos al móvil", "smtp/newsletter: envío del boletín"]
```

Cada entrada es `servicio/nombre: para qué es`. El servicio y el nombre son palabras
sueltas, tal como los conoce el llavero. Claude le añade entradas con la herramienta
`secret_name_save`, y la página Secretos del panel la lista. Un valor se rechaza. La
0.6.0 guardaba la lista en `config.toml` como `secrets`;
`check` y el inicio de sesión la pasan aquí.

## `guard.toml`

Junto a `config.toml`, `guard.toml` guarda lo que el guardia nunca permite:

```toml
untouchable_repos = ["~/work/core", "~/work/cli-*"]
```

- `untouchable_repos`: repositorios donde Claude nunca hace commit, push ni merge, digan
  lo que digan `git_allowed_repos` y `git_ask_repos`. Para todo el ordenador. Misma forma
  que esas listas (`~`, un `*` en la última parte, los worktrees cuentan).

La página Guardia del panel lo escribe. Si el fichero falta, nada es intocable. Si no se
puede leer, el guardia no sabe qué repositorios son intocables, ni si lo demás está como
lo dejaste: te pregunta antes de cada commit, push o merge que las otras listas dejarían
pasar, y también antes de cada comando `aws` y de cada conexión a otra máquina, hasta que
se arregle el fichero. Una sesión que aún ejecuta una versión anterior a `guard.toml` no
hace cumplir la lista.

## `session.toml`

Junto a `config.toml`, `session.toml` guarda cómo quieres tus sesiones:

```toml
preferred_model = "opus"
memory_trust = "ask"
```

- `preferred_model`: el modelo que esperas que responda (`opus`, `sonnet`, `haiku` o un
  nombre completo como `claude-opus-5-5`). Si una sesión responde con otro modelo, Claude
  te lo dice. Lo mira al empezar, si Claude Code dice con qué modelo arranca. Si no, lo
  mira después de una respuesta: en una sesión nueva, como tarde la segunda. Y otra vez si
  el modelo cambia. Una vez por sesión y modelo. Nunca cambia el modelo por su cuenta: lo cambias tú, con `/model` o con el
  selector de modelo.
- `memory_trust`: qué hace Claude antes de guardar algo que siguen todas las sesiones: una
  regla (nueva, cambiada o retirada), tu perfil, o algo que se añade a las listas del
  guardia. `"tell"`: Claude lo guarda y Claude Code te dice qué se ha guardado. Es lo que
  pasa si el ajuste falta. `"ask"`: Claude Code te pregunta antes, enseñando lo que se
  guardaría, y si dices que no, no se escribe nada. Un valor que el programa no conoce
  cuenta como `"ask"`. Las tareas, el buzón, las referencias y las lecciones se guardan
  siempre sin preguntar. En una sesión que se salta las preguntas de permiso
  (`bypassPermissions`), `"ask"` deniega en vez de preguntar: Claude te dice lo que quería
  guardar. Más en [el guardia](guard.es.md#preguntar-antes-de-que-claude-guarde-una-regla).

La página Guardia del panel lo escribe, y el guardia pregunta antes de que Claude lo
escriba. Si el fichero falta o `preferred_model` está vacío, el modelo no se comprueba.

Una línea de `guard.toml` o `session.toml` que el programa no conoce no se tiene en
cuenta, pero no se rechaza: una versión más nueva puede añadir ajustes ahí, y una anterior
sigue funcionando. `doctor` y el inicio de una sesión la nombran, con el nombre correcto
si parece una errata, porque un ajuste mal escrito ahí no hace nada.

## `remote.toml`

Junto a `config.toml`, `remote.toml` guarda los ajustes del [acceso remoto](remote.es.md),
que mantiene Claude accesible desde otro ordenador. Lo crea `augmented-brain remote
install`.

```toml
enabled = true
spaces = ["personal", "trabajo"]
name_prefix = "mi-portatil"
permission_mode = "default"
folders = { trabajo = "~/code/clientes" }
```

Todos son opcionales:

- `enabled`: si el acceso remoto está puesto. `remote install` escribe `true` y `remote
  uninstall` escribe `false`. Solo cuando es `true` revisa `doctor` los
  servicios; un fichero que no puede leer es un aviso siempre.
- `spaces`: los espacios que tienen una sesión, cada uno en un servicio propio. Por
  defecto, el `default_space`, o el único espacio si solo hay uno.
- `name_prefix`: las sesiones se llaman `<name_prefix>-<espacio>`, como
  `mi-portatil-personal`. Una palabra sencilla: letras, cifras, punto, guion y guion
  bajo. Por defecto, el nombre del ordenador.
- `permission_mode`: cuánto puede hacer Claude sin preguntar. Uno de `default`,
  `acceptEdits`, `plan`, `auto`, `dontAsk` y `bypassPermissions`. Por defecto `default`,
  donde Claude pregunta como siempre y el guardia sigue vigente. **No uses
  `bypassPermissions`**: deja que quien se conecte ejecute lo que quiera, sin preguntas, y
  deja el ordenador abierto.
- `folders`: dónde arranca la sesión de un espacio, cuando no es su primera carpeta de
  trabajo (lo normal). La carpeta tiene que existir y estar dentro de una carpeta de
  trabajo de ese espacio, para que la sesión siga cargando ese espacio.

Vuelve a lanzar `augmented-brain remote install` después de cambiarlo, para que los
servicios lo recojan.

## Otros ficheros de la carpeta

- `profile.md`: tu perfil, para todos los espacios.
- `shared/`: las notas compartidas (reglas, referencias y lecciones para todos los
  espacios).
- `state/`: lo que este ordenador sabe de las sesiones de cada espacio. Nunca va a la
  copia de seguridad.
- `migrations/`: una copia completa de las notas antes de que una versión nueva las
  cambiara de formato.
- `marketplace/`: el plugin de Claude Code de la versión instalada, que Claude Code lee
  de aquí. El programa lo escribe cuando instala o actualiza el plugin; el guardia
  pregunta antes de que otra cosa escriba ahí.
- `panel.json` y `panel.lock`: dónde escucha el panel mientras está en marcha, y su
  bloqueo. El guardia nunca deja que Claude lea `panel.json`: guarda la llave del panel.
- `remote/<espacio>/`: la carpeta donde arranca el servicio de acceso remoto de un espacio,
  solo si el espacio no tiene carpetas de trabajo. Pasa a ser carpeta de trabajo de ese
  espacio.
- `review-<espacio>.draft`: una revisión semanal empezada en el panel y aún sin
  terminar.
- `history/`: las 10 últimas versiones de tu configuración, con quién las cambió. Solo en
  este ordenador, nunca va a la copia de seguridad. Lo tienes en
  [Versiones de la configuración](config-history.es.md).

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
- `AB_DESKTOP_TIMEOUT`: en Linux, cuántos segundos puede tardar la descarga de la app de
  escritorio antes de que el instalador la deje y siga sin ella. Por defecto, 480 (8 minutos).
- `AB_INSTALL_DIR`: dónde poner el programa. Por defecto, `~/.local/bin`, y en Windows
  `%USERPROFILE%\.local\bin`.
- `AB_NO_INIT=1`: solo instala el programa, sin preparar nada.
- `AB_VERSION`, `AB_RELEASE_URL`, `AB_API_URL`, `AB_RETRY_DELAY`, `AB_MARKETPLACE`,
  `AB_DESKTOP_URL` y `AB_CLAUDE_STABLE_URL`: para pruebas: qué versión, dónde están los
  ficheros de la versión publicada, la API de GitHub a la que recurrir, la espera entre
  intentos, una carpeta o un repositorio del que tomar el plugin en vez del que lleva el
  programa, dónde está el repositorio apt de la app de escritorio, y dónde leer la última
  versión estable de Claude Code (el instalador se ofrece a actualizar un Claude Code
  antiguo). `augmented-brain update` no hace caso de
  ninguna: para sus pruebas usa `--from`.
- `AB_ALLOW_ROOT=1`: en Linux y macOS, instala aunque sea como root (`sudo`). Sin ella, el
  instalador se niega: no necesita root, y como root el programa y sus ajustes acabarían
  siendo de root, o en la carpeta de root, no en la tuya.

## Los ajustes de Claude Code

Aquí importa un ajuste de Claude Code. Si Claude Code guarda una memoria propia para las
carpetas de tus espacios, se desvía de la del espacio. `augmented-brain doctor` te lo
dice, y se arregla con `"autoMemoryEnabled": false` en los ajustes de Claude Code, en el
fichero que nombra `doctor`.

El formato completo de cada fichero está en el
[formato de la memoria](../spec/memory-format.md) (en inglés).
