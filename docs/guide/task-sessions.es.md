# Una sesión por tarea

[Read in English](task-sessions.md)

Cada tarea puede tener una sesión de Claude propia, en su propia carpeta. La pides con
palabras normales, y en menos de un minuto sale en claude.ai y Claude Desktop, ya leyendo
su tarea. Con varias, trabajas en varias tareas a la vez, desde el ordenador o desde el
móvil, sin que ninguna pise a otra.

## Dónde funciona

| Comando | Linux | macOS | Windows |
|---|---|---|---|
| `session start`, `stop`, `list` | sí | todavía no | todavía no |
| `session restart` | sí | sí | no |
| `watch pr`, `clean` | sí | sí | sí |

`session start`, `stop` y `list` solo se han probado en Linux. En macOS y Windows
`session start` dice que no está disponible, y no crea nada.

## Abre una sesión para una tarea

Pídeselo a la sesión en la que estás: «ábreme una sesión para la tarea de la web». Toda
sesión en Linux sabe hacerlo, en el ordenador o [desde otro](remote.es.md), porque el
inicio de cada sesión se lo cuenta. Lanza

```bash
augmented-brain session start task-new-website
```

y sale en claude.ai y Claude Desktop una sesión nueva, con el nombre de la tarea. Empieza
sola: lee su tarea, te dice en dos o tres líneas cómo está y se pone con el siguiente
paso. Solo oye lo que es de su tarea: las fechas de esa tarea, no el resto de avisos del
espacio, la revisión semanal ni las sesiones por recuperar, que salen en tus otras
sesiones. Cuando para (la tarea cerrada o aparcada, o esperándote), avisa en una línea a
la sesión que la pidió.

El comando espera la primera respuesta de la sesión, hasta un minuto. Una sesión que no
puede trabajar no se queda abierta: si Claude Code no tiene la sesión iniciada, no se
abre nada, y si la primera respuesta es un error (el login caducó, el servicio está
saturado), la sesión se cierra. En los dos casos el comando dice qué hacer, como `claude
auth login`.

## En qué carpeta se abre

La carpeta es la del primer repositorio de la tarea que encuentra en las carpetas de
trabajo del espacio. `--folder` da otra, siempre que esté dentro de las carpetas de
trabajo del espacio: una sesión fuera no encontraría su tarea. Una tarea que no nombra
ningún repositorio se abre en la primera carpeta de trabajo del espacio.

```bash
augmented-brain session start task-new-website --folder ~/projects/website
```

Dos reglas la mantienen segura:

- **Claude tiene que confiar en esa carpeta ella misma.** Que confíe en una carpeta de
  más arriba no basta: los ajustes y hooks propios de un repositorio no deben ejecutarse
  sin preguntar. La primera vez, el comando dice qué hacer: en el ordenador (o por
  `ssh`), entra en la carpeta, lanza `claude`, acepta la pregunta y ciérralo con `/exit`.
  Luego pídelo otra vez.
- **Una tarea por carpeta.** Dos tareas en una carpeta se pisarían los cambios, así que
  una carpeta que ya usa la sesión de otra tarea se rechaza. Dale otra con `--folder`,
  como un worktree (abajo).

## Worktrees, para un repositorio bare

Un **worktree** es una segunda copia de trabajo de un repositorio git: otra carpeta, en
otra rama, con la misma historia. Un **repositorio bare** es uno con historia pero sin
copia de trabajo: hay quien guarda así cada proyecto y hace un worktree para cada
trabajo.

Cuando el repositorio de la tarea es bare, `session start` le hace a la tarea un worktree
propio a su lado: `site-new-website` para la tarea `task-new-website` de `site`, en una
rama nueva con el nombre de la tarea, desde la rama por defecto de origin. Antes la
trae, así que la rama empieza con el código de ahora; si no llega a origin, empieza con
lo último que este ordenador sabía, y lo dice. Si una sesión anterior de la tarea dejó
su rama, el worktree coge esa rama tal como estaba. Así dos tareas de un repositorio
nunca comparten carpeta. Si se pide otra vez, usa el mismo worktree.
Claude guarda la confianza de un worktree con su repositorio principal: basta con
confiar en uno de los dos.

El worktree se va solo cuando no se pierde nada: con `session stop`, al cerrar la tarea
o, si la sesión terminó de otra forma (un reinicio, cerrarla desde Claude Desktop), en el
próximo `session start`. Uno con cambios sin commitear, commits sin subir o ficheros que
git ignora y ninguna compilación rehace (un `.env`, una clave) se queda, y
`augmented-brain doctor` lo nombra.

## Verlas y cerrarlas

```bash
augmented-brain session list
```

lista las sesiones de tarea abiertas en este ordenador: la tarea, si corre y su carpeta.

- **retirada**: Claude para una sesión que lleva horas sin uso. claude.ai la enseña
  desconectada. `session start` con su tarea la trae de vuelta, con toda su
  conversación.
- **fallida**: su última respuesta fue un error, como un login caducado. La línea dice
  qué hacer.

```bash
augmented-brain session stop task-new-website
```

cierra la sesión y la quita de Claude Desktop y claude.ai, con su worktree. Cerrar la
tarea hace lo mismo, cuando la sesión termina su respuesta. Una sesión también termina
al reiniciar el ordenador, y no vuelve sola: pídela otra vez.

## Reinicia una sesión tras actualizar

Tras `augmented-brain update`, las sesiones abiertas siguen con el programa con el que
empezaron.

```bash
augmented-brain session restart task-new-website
augmented-brain session restart --old
```

El primero reinicia una sesión, con la misma conversación; `--old` reinicia todas las
que están paradas y siguen con una versión anterior, y `update` lo lanza solo. Una
sesión que está respondiendo nunca se corta. Funciona en Linux y macOS; en Windows,
cierra la sesión y ábrela otra vez tú. Los detalles, en
[`session restart`](commands.es.md#augmented-brain-session-restart).

## Esperar a una pull request

Una sesión que abrió una pull request no se queda esperando sus checks. Deja esto en
marcha en segundo plano, con su tarea:

```bash
augmented-brain watch pr 104 --task task-new-website
```

Cuando los checks pasan o fallan, la pull request se mezcla o se cierra, o choca con su
rama base, apunta el resultado en la tarea, para que tus otras sesiones se enteren
aunque esta ya no esté, y la sesión te avisa en el acto. Nunca espera para siempre: 3
horas como mucho. Necesita `gh`, el comando de GitHub, con tu login. Todas las opciones,
en [`watch pr`](commands.es.md#augmented-brain-watch-pr).

## Limpia lo que queda

Claude Code hace worktrees y carpetas temporales propias, y deja algunas atrás.

```bash
augmented-brain clean
```

enseña lo que queda de sesiones que ya terminaron, y lo quita solo si dices que sí:
nunca un worktree con trabajo propio, que solo nombra. `doctor` avisa cuando hay algo
que limpiar. Los detalles, en [`clean`](commands.es.md#augmented-brain-clean).
