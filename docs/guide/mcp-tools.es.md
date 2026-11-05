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
  [`task_journal`](#task_journal), [`task_park`](#task_park), [`task_close`](#task_close).
- **El buzón y la revisión semanal**: [`capture`](#capture),
  [`weekly_review`](#weekly_review), [`review_done`](#review_done).
- **Notas que duran**: [`area_new`](#area_new), [`knowledge_new`](#knowledge_new),
  [`rule_new`](#rule_new), [`person_save`](#person_save),
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
una tarea antes de retomarla.

- `id`: el id de la tarea, como `task-mudar-el-blog`.

## `task_update`

Cambia una tarea. Solo cambian los campos que se le dan. Un cambio de estado se apunta en
el diario.

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

Hace falta al menos un campo además de `id`.

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
- `done` (opcional): `true` para marcar el paso, `false` para desmarcarlo.

## `task_journal`

Añade una entrada con fecha al diario de una tarea. Claude escribe una en el momento en
que se hace o se decide algo.

- `id`: el id de la tarea.
- `entry`: lo que se ha hecho, lo que ha salido, lo que se ha decidido.

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
subtareas abiertas no se puede cerrar.

- `id`: el id de la tarea.
- `result`: lo que se ha conseguido, en una o dos líneas.

## `capture`

Apunta algo en el buzón sin abrir una tarea, si quieres con un día para recordártelo. La
revisión semanal lo procesa.

- `text`: la idea, el enlace o el cabo suelto, en una línea.
- `remind_on` (opcional): una fecha. Sale al empezar la sesión desde una semana antes,
  hasta que se resuelve.

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
espacio, o compartida por todos.

- `kind`: `reference` (cómo funciona algo) o `lesson` (lo aprendido).
- `title`: en una lección, un título que ya diga la conclusión.
- `summary` (opcional): una línea.
- `area` (opcional): el id de su área.
- `body` (opcional): el texto, en Markdown.
- `shared` (opcional): `true` cuando vale en todos los espacios. Entonces no tiene área.

## `rule_new`

Guarda una regla que das sobre cómo trabajar. Cada sesión la carga.

- `title`: la regla en pocas palabras.
- `summary` (opcional): una línea que dice qué hacer.
- `body`: qué hacer, y por qué, en Markdown.
- `shared` (opcional): `true` cuando vale en todos los espacios, no solo en este.

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

Guarda quién eres y cómo te gusta trabajar. Lo comparten todos los espacios, y cada
sesión empieza leyéndolo. Sustituye el perfil entero, así que Claude incluye lo que debe
quedarse.

- `name`: cómo quieres que te llamen.
- `about` (opcional): a qué te dedicas y en qué trabajas, en líneas cortas que empiezan
  por `- `.
- `work_style` (opcional): cómo te gusta trabajar con Claude: idioma, largo de las
  respuestas, cuánto preguntar antes de actuar. Líneas cortas que empiezan por `- `.

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

Monta la copia de seguridad de este espacio: un repositorio git privado que guarda sus
notas, y al que se suben solas cada vez que acaba una sesión.

- `repo_url` (opcional): un repositorio git privado al que puedas subir. Sin él, la
  herramienta crea `augmented-brain-<espacio>` en tu cuenta de GitHub con la línea de
  comandos `gh`. Si falta `gh` o no has entrado con él, devuelve los pasos para guiarte.

## `session_recover`

Lee lo que tú y Claude os dijisteis en una sesión anterior de este espacio que no acabó
bien, después de la última entrada del diario de su tarea. Solo los mensajes, sin la
salida de las herramientas, como mucho los últimos 20.000 caracteres, del registro que
guarda Claude Code en este ordenador (lo guarda 30 días). Después Claude apunta en el
diario lo que falte.

- `session_id` (opcional): la sesión que recuperar. Sin él, la que ofreció el inicio de
  la sesión.
