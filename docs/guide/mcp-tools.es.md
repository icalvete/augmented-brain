# Herramientas MCP

[Read in English](mcp-tools.md)

Dices "apárcala". Claude llama a la herramienta `task_park`, con el id de la tarea y el
siguiente paso. El programa escribe el fichero de la tarea. Tú nunca llamas a estas
herramientas: las llama Claude, detrás de tus palabras.

**MCP** (Model Context Protocol) es un estándar abierto para dar herramientas a un
asistente. El programa las ofrece con `augmented-brain mcp`, que Claude Code arranca a
través del plugin. En Claude Code aparecen como
`mcp__plugin_brain_augmented-brain__<herramienta>`.

Hay cosas que valen para todas:

- Trabajan con **el espacio de la carpeta en la que está Claude**, que buscan de nuevo en
  cada llamada.
- Van de una en una, y esperan al panel, así que nunca escriben dos a la vez.
- **Nada de secretos.** Una herramienta se niega a escribir una contraseña, un token o
  una clave en una nota.
- **No se borra nada.** Las tareas se cierran, no se quitan.
- Las fechas se escriben `AAAA-MM-DD`.
- Un parámetro es obligatorio salvo que diga opcional.

## Herramientas por uso

- **Dónde está cada cosa**: [`where_am_i`](#where_am_i), [`pending`](#pending).
- **Tareas**: [`task_new`](#task_new), [`task_get`](#task_get),
  [`task_update`](#task_update), [`task_step`](#task_step),
  [`task_journal`](#task_journal), [`task_park`](#task_park), [`task_close`](#task_close),
  [`task_worktree`](#task_worktree).
- **El buzón y la revisión semanal**: [`capture`](#capture),
  [`capture_done`](#capture_done), [`weekly_review`](#weekly_review), [`review_done`](#review_done).
- **Notas que duran**: [`area_new`](#area_new), [`knowledge_new`](#knowledge_new),
  [`rule_new`](#rule_new), [`note_retire`](#note_retire), [`person_save`](#person_save),
  [`repo_describe`](#repo_describe).
- **Tú y esta máquina**: [`profile_save`](#profile_save),
  [`safety_save`](#safety_save), [`secret_name_save`](#secret_name_save),
  [`backup_setup`](#backup_setup), [`session_recover`](#session_recover).

## `where_am_i`

En qué espacio trabaja esta sesión, su carpeta de notas, sus áreas y cuántas tareas
están vivas. No tiene parámetros.

## `pending`

Las tareas vivas, primero las que esperan por ti, cada una con su estado, los pasos
hechos, a quién espera y su siguiente paso. Después, las fechas para revisar que han
llegado. Claude la usa para "¿en qué estábamos?".

- `within_days` (opcional): lista también las fechas para revisar que llegan en estos
  días. Por defecto, 7.
- `space` (opcional): otro de tus espacios, para listar sus tareas vivas en su lugar,
  solo para leer: para encontrar allí la tarea de la que hablas.

## `task_new`

Abre una tarea nueva, en la fase de definición y esperando por ti. Con una tarea madre,
es una subtarea.

- `title`: un nombre corto para la tarea.
- `summary` (opcional): una línea sobre qué es.
- `area` (opcional): el id de su área. Por defecto, el área de la tarea madre, o
  `area-general`.
- `parent` (opcional): el id de la tarea de la que esta es una parte. Un solo nivel: la
  tarea madre no puede ser una subtarea, ni estar cerrada.
- `goal` (opcional): una línea sobre para qué es la tarea. Va a la Definición.
- `done_when` (opcional): una línea sobre cómo sabremos que ha salido bien. Va a la
  Definición.

## `task_get`

Lee una tarea: su cabecera, todo su cuerpo con el diario, y sus subtareas. También dice
de qué tarea es parte, y si falta en este ordenador un repositorio que nombra. Claude lee
una tarea antes de retomarla. Las entradas antiguas del diario que se movieron a
`archive/journals/` las nombra una línea del diario (mira
[Un diario largo](tasks.es.md#un-diario-largo)); una tarea aún demasiado larga para una
respuesta, escrita a mano o por una versión anterior, se lee sin sus entradas más antiguas,
y una línea dice dónde están.

- `id`: el id de la tarea, como `task-mudar-el-blog`.
- `space` (opcional): otro de tus espacios, del que es la tarea. Claude solo lo usa
  cuando le pides trabajar desde aquí en una tarea de ese espacio. Mira
  [Una tarea de otro espacio](spaces.es.md#una-tarea-de-otro-espacio).

## `task_update`

Cambia una tarea. Solo cambian los campos que se le dan. Un cambio de estado se apunta en
el diario, y si se salta fases (de `definition` directa a `execution`, por ejemplo), la
línea dice cuáles: saltarlas se puede, y se ve. Una tarea cerrada solo se reabre con
`reopen`.

- `id`: el id de la tarea.
- `status` (opcional): `definition`, `analysis`, `plan`, `execution`, `validation`,
  `closed`, `parked` o `someday`.
- `waiting_on` (opcional): `me`, `nobody`, una persona, o el id de otra tarea.
- `next` (opcional): la única acción siguiente, en una línea.
- `summary` (opcional): una línea sobre qué es.
- `area` (opcional): el id de su área.
- `watch_date` (opcional): añade una fecha para volver a mirar algo. Necesita
  `watch_what`.
- `watch_what` (opcional): qué mirar en `watch_date`, en una línea.
- `watch_done` (opcional): marca como hecha la fecha para revisar de este día.
- `reopen` (opcional): solo para volver a abrir una tarea cerrada, con un `status` nuevo:
  por qué, en pocas palabras. Va al diario. Claude te lo pregunta antes.
- `space` (opcional): otro de tus espacios, del que es la tarea. Claude solo lo usa
  cuando le pides trabajar desde aquí en una tarea de ese espacio. Cerrarla se deja a una
  sesión de ese espacio. Mira
  [Una tarea de otro espacio](spaces.es.md#una-tarea-de-otro-espacio).

Hace falta al menos un campo además de `id` y `space`.

## `task_step`

Los pasos son las partes pequeñas de una tarea: casillas (`- [ ] texto`) en su Plan. Esta
herramienta añade una, o la marca o la desmarca. Cada cambio escribe una línea corta en
el diario, y la respuesta lista los pasos.

- `id`: el id de la tarea.
- `add` (opcional): añade este paso, sin marcar, al final del Plan, en vez de marcar
  uno.
- `number` (opcional): el paso que marcar o desmarcar, 1 para la primera casilla.
- `text` (opcional): el paso que marcar o desmarcar, por su texto (o un trozo que solo
  tenga él), cuando no se da `number`.
- `done` (opcional): `true` para marcar el paso, `false` para desmarcarlo. Si no se da,
  lo desmarca.
- `space` (opcional): otro de tus espacios, del que es la tarea. Claude solo lo usa
  cuando le pides trabajar desde aquí en una tarea de ese espacio. Mira
  [Una tarea de otro espacio](spaces.es.md#una-tarea-de-otro-espacio).

## `task_journal`

Añade una entrada con fecha al diario de una tarea. Claude escribe una en el momento en
que se hace o se decide algo.

- `id`: el id de la tarea.
- `entry`: lo que se ha hecho, lo que ha salido, lo que se ha decidido.
- `space` (opcional): otro de tus espacios, del que es la tarea. Claude solo lo usa
  cuando le pides trabajar desde aquí en una tarea de ese espacio. La entrada acaba en
  `(from <este espacio>)`. Mira
  [Una tarea de otro espacio](spaces.es.md#una-tarea-de-otro-espacio).

## `task_park`

Deja una tarea en pausa sin perder nada, antes de pasar a otra cosa.

- `id`: el id de la tarea.
- `next`: cómo retomarla exactamente: un comando, un fichero o una decisión.
- `waiting_on` (opcional): a quién espera ahora.
- `essentials` (opcional, pero Claude debería pasarlo): de tres a seis líneas que basten
  para retomarla sin el diario. Sustituyen a la primera sección de la tarea. Sin ellas,
  allí solo se actualizan "dónde está" y "siguiente".
- `someday` (opcional): `true` cuando no se va a tocar en semanas. La tarea sale de la
  lista del día.

## `task_close`

Cierra una tarea terminada con su resultado. Se guarda, nunca se borra. Una tarea con
subtareas abiertas no se puede cerrar, y una ya cerrada se rechaza, para que su resultado
nunca se pise. Si la tarea tiene fechas para revisar sin marcar
como hechas, lo dice: siguen saliendo al empezar la sesión en su día, porque cerrar suele
ser cuando se apunta lo que hay que mirar más adelante. Si `session start` creó un
worktree de git para la tarea y su sesión ya no corre, al cerrarla lo quita con su rama
cuando no tiene nada que no esté en el remoto, y lo dice; si no, dice por qué se queda.
Si la sesión propia de la tarea sigue en marcha (a menudo la que la cierra), un proceso
aparte cierra esa sesión cuando termina su respuesta, la quita de Claude Desktop y
claude.ai y quita su worktree de la misma forma. Una sesión que Claude retiró tras horas
sin uso no responde nada: se cierra en el acto.

- `id`: el id de la tarea.
- `result`: lo que se ha conseguido, en una o dos líneas.

## `task_worktree`

Crea un worktree de git para una tarea, o apunta uno ya hecho, para que cerrar la tarea
lo quite. Un worktree es una segunda carpeta de un repositorio, en una rama propia, para
que una tarea cambie código sin tocar la carpeta principal. Creado aquí, es
`<repo>-<tarea>`, al lado del repositorio, en una rama nueva con el nombre de la tarea,
desde la rama por defecto de origin tal como está ahora; si se pide otra vez, devuelve
el mismo. Al cerrar la tarea se quita cada worktree apuntado que no tenga nada sin
commitear ni sin subir, con la rama creada para él (uno apuntado conserva su rama); uno
con trabajo propio se queda, y también el de la sesión que cierra, y la respuesta dice
por qué. Las dos rutas tienen que estar dentro de las carpetas de trabajo del espacio.

- `id`: el id de la tarea.
- `repo` (opcional): un repositorio de git: crea el worktree de la tarea en él.
- `folder` (opcional): un worktree ya hecho para la tarea: lo apunta. Da `repo` o
  `folder`, uno de los dos.

## `capture`

Apunta algo en el buzón sin abrir una tarea, si quieres con un día para recordártelo. La
revisión semanal lo procesa.

- `text`: la idea, el enlace o el cabo suelto, en una línea.
- `remind_on` (opcional): una fecha. Sale al empezar la sesión desde una semana antes,
  hasta que se resuelve.
- `space` (opcional): otro de tus espacios, para mandarlo allí. Llega al buzón de ese
  espacio marcado con su origen, `(from personal)`, y sale al empezar cada sesión allí
  hasta que se hace. Sirve para algo que se decide en un espacio y se hace en otro.
  Claude solo lo hace cuando se lo pides: lo que va a un espacio se ve allí y entra en
  su copia de seguridad.

## `capture_done`

Quita una captura del buzón cuando ya está hecha o se ha convertido en tarea, sin
esperar a la revisión semanal. Sobre todo para lo que ha mandado otro espacio, que si no
sale al empezar cada sesión.

- `date`: el día en que se capturó, tal como sale.
- `text`: su texto tal como sale, con o sin el origen. Otra captura con el mismo texto
  se queda.

## `weekly_review`

Lista lo que repasa la revisión semanal: el buzón, las tareas vivas que nadie ha tocado
en 14 días o más, y las fechas que llegan en los próximos 7 días. Después Claude lo
repasa contigo, una cosa cada vez. No tiene parámetros.

## `review_done`

Termina la revisión semanal. Guarda lo decidido en `reviews/`, y saca del buzón las
capturas procesadas.

- `summary`: lo que se ha decidido, en unas pocas líneas.
- `processed` (opcional): el texto de cada captura resuelta, tal como la listó
  `weekly_review`. Salen del buzón. Una que no se encuentra se queda como estaba, y se
  nombra.

## `area_new`

Crea un área: una responsabilidad sin fecha de fin que agrupa tareas y conocimiento.

- `title`: el nombre del área.
- `summary` (opcional): una línea sobre qué cuida.

## `knowledge_new`

Guarda una referencia o una lección que merece la pena más allá de una tarea: en este
espacio, o compartida por todos. Un título que ya existe no crea nada: Claude recibe su
id, para cambiar esa o elegir otro título. Nada se guarda dos veces sin que nadie lo sepa.

- `id` (opcional): para cambiar una referencia o lección que existe. Lo que se manda
  sustituye a lo que tenía, lo que no se manda se queda, y la versión sustituida se
  guarda en `archive/`.
- `kind`: `reference` (cómo funciona algo) o `lesson` (lo aprendido).
- `title`: en una lección, un título que ya diga la conclusión.
- `summary` (opcional): una línea.
- `area` (opcional): el id de su área.
- `body` (opcional): el texto, en Markdown.
- `shared` (opcional): `true` cuando vale en todos los espacios. Entonces no tiene área.

## `rule_new`

Guarda una regla que das sobre cómo trabajar. Cada sesión la carga. Un título que ya
existe no crea nada: Claude recibe su id.

- `id` (opcional): para cambiar una regla que existe. Lo que se manda sustituye a lo que
  tenía, lo que no se manda se queda, y la versión sustituida se guarda en `archive/`.
- `title`: la regla en pocas palabras.
- `summary` (opcional): una línea que dice qué hacer.
- `body`: qué hacer, y por qué, en Markdown.
- `shared` (opcional): `true` cuando vale en todos los espacios, no solo en este.

## `note_retire`

Retira una regla, referencia o lección que ya no vale, sin borrarla. Pasa a `archive/`,
que nunca se carga, así que una regla retirada deja de llegar al inicio de la sesión.
Una línea al final dice cuándo, por qué y qué la sustituye. Para un duplicado o una
versión vieja; Claude te pregunta antes.

- `id`: la regla, referencia o lección.
- `reason`: por qué, en pocas palabras. Se queda con la nota.
- `replaced_by` (opcional): el id de la nota que la sustituye.
- `shared` (opcional): `true` cuando la nota está en la capa compartida.

## `person_save`

Recuerda a alguien con quien trabajas o tratas, para que un nombre de pila lleve a la
persona correcta. Guardar un nombre que ya existe actualiza a esa persona.

Una persona puede estar en varios espacios: la misma persona, con el rol y las notas que
cada espacio puede mostrar. Claude te pregunta dónde debe verse (por ejemplo, una nota
sobre un amigo no debe acabar en tu espacio de trabajo) y pasa tu respuesta en `spaces`.
Con un solo espacio no hay nada que preguntar. Sin `spaces` y con varios, no se guarda
nada: la respuesta lista tus espacios y pregunta. Lo que se guarda en un espacio solo se
ve en ese espacio.

- `name`: el nombre completo, o como la conoces.
- `aliases` (opcional): otros nombres que usas para ella, como un mote.
- `role` (opcional): quién es para ti, en pocas palabras: "mi jefa", "lleva la empresa
  del hosting".
- `notes` (opcional): cualquier otra cosa que merezca la pena recordar.
- `spaces` (opcional con un espacio, necesario con varios): los nombres de los espacios
  donde se ve. El rol, los otros nombres y las notas se escriben solo en esos espacios.

## `repo_describe`

Apunta, en una línea, qué es uno de tus repositorios git, para la página Repositorios del
panel. Solo repositorios dentro de las carpetas de trabajo de este espacio. La línea va a
la referencia `ref-repositories`, una línea por repositorio; una nueva sustituye a la
anterior.

- `repo`: el repositorio: su ruta (`~/…` o absoluta), o su nombre tal como lo escribe el
  `repos:` de una tarea, respecto a una carpeta de trabajo.
- `text`: qué es, en una línea, con tus palabras. Vacío, quita la línea.

## `profile_save`

Guarda quién eres, cómo te gusta trabajar, y el nombre y la personalidad que le das a
Claude. Lo comparten todos los espacios, y cada sesión empieza leyéndolo. Claude manda
solo las partes que cambian: una parte que no se manda conserva lo que tiene, y una
vacía se borra.

- `name` (hace falta la primera vez): cómo quieres que te llamen.
- `about` (opcional): a qué te dedicas y en qué trabajas, en líneas cortas que empiezan
  por `- `.
- `work_style` (opcional): cómo te gusta trabajar con Claude: idioma, largo de las
  respuestas, cuánto preguntar antes de actuar. Líneas cortas que empiezan por `- `.
- `assistant_name` (opcional): el nombre que le das a Claude. Firma con él los correos
  que escribe en tu nombre, siempre seguido de una línea que dice que es una IA (ver
  [Panel, Perfil](panel.es.md#perfil)).
- `assistant_personality` (opcional): cómo quieres que sea Claude, con tus palabras.
  Solo estilo: nunca cambia las reglas, el guardia ni su honestidad.

## `safety_save`

Añade cosas a lo que el guardia te pregunta. Las listas valen en toda esta máquina, en
todos los espacios. Solo añade: para quitar algo, editas tú la configuración, y la
respuesta dice dónde.

- `production_aws_profiles` (opcional): perfiles de AWS CLI que llegan a producción. El
  guardia pregunta antes de los comandos que los usan.
- `untouchable_hosts` (opcional): máquinas que no se tocan sin preguntar, tal como las
  escribes para ssh.

## `secret_name_save`

Apunta el nombre de un secreto que guardas en el llavero del sistema, y para qué es.
Nunca su valor: un valor se rechaza. La lista es la misma en todos los espacios; el panel
la enseña. Guardar un nombre que ya está en la lista cambia para qué es.

- `service`: el servicio bajo el que se guarda el secreto en el llavero, como
  `telegram`.
- `name`: su nombre dentro de ese servicio, como `alerts-bot`.
- `what` (opcional): para qué es, en pocas palabras.

## `backup_setup`

Configura la copia de seguridad de este espacio: un repositorio git privado que guarda sus
notas, y al que se suben solas cada vez que acaba una sesión.

- `repo_url` (opcional): un repositorio git privado al que puedas subir. Sin él, la
  herramienta crea `augmented-brain-<espacio>` en tu cuenta de GitHub con la línea de
  comandos `gh`. Si falta `gh` o no has entrado con él, devuelve los pasos para guiarte.

Hace la primera copia en el momento. Si el espacio ya tiene copia, dice adónde va y cómo
fue la última; nunca la lleva a otro sitio: eso solo lo haces tú, con
`augmented-brain backup setup <url>`.

## `session_recover`

Lee lo que tú y Claude os dijisteis en una sesión anterior de este espacio que no acabó
bien, después de la última entrada del diario de su tarea. Solo los mensajes, sin la
salida de las herramientas, como mucho los últimos 20.000 caracteres, del registro que
guarda Claude Code en este ordenador (lo guarda 30 días). Después Claude apunta en el
diario lo que falte.

- `session_id` (opcional): la sesión que recuperar. Sin él, la que ofreció el inicio de
  la sesión.
