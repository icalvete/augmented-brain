# Si algo falla

[Read in English](troubleshooting.md)

Algo no va bien. Empieza por aquí: abre una terminal y escribe

```bash
augmented-brain doctor
```

Revisa toda la instalación y te dice qué falta y cómo arreglarlo. Cada línea empieza con
una marca: `✓` bien, `·` para que lo sepas, `!` un aviso, `✗` algo que falla. Debajo de
una línea que pide arreglo, `→` dice qué hacer.

Si tienes el panel abierto, su página **Ayuda** tiene enlaces a estas respuestas y un
botón para avisar de un problema en GitHub, con la revisión de salud ya puesta y sin lo
que es personal. Lo tienes en [El panel](panel.es.md#ayuda).

## Problemas y arreglos

### Claude no conoce mis tareas, o se salta la entrevista

- **Claude Code te preguntó si confías en la carpeta.** La primera vez que abres Claude
  en una carpeta, Claude Code lo pregunta antes de cargar nada. Di que sí en tus carpetas
  de trabajo. Hasta entonces, augmented brain no puede cargar tus notas.
- **La sesión empezó antes de instalar.** Empieza una sesión nueva.
- **Estás en otro espacio.** Escribe `augmented-brain space` en esa carpeta. Fuera de las
  carpetas de tus espacios, se usa el espacio por defecto. Para que una carpeta cuente
  para un espacio, añádela a sus `work_folders` (lo tienes en [Espacios](spaces.es.md)).

### La terminal no encuentra un comando

`command not found`, o "no se reconoce" en Windows. Cierra la terminal y abre otra: la que
usabas aún no ve el PATH nuevo.

### Claude Code para todos los comandos y cambios de ficheros

El guardia no puede ejecutarse, así que Claude Code lo para todo en vez de dejarlo pasar.
Falta el programa o está roto. Escribe `augmented-brain doctor`. Comprueba que
`augmented-brain` está en tu PATH o en `~/.local/bin`. Volver a instalar lo arregla.

### Todos los comandos de PowerShell se paran

Es lo esperado, en Windows. El guardia lee la shell, no PowerShell, así que para los
comandos de PowerShell y Claude los lanza en Git Bash.

### Claude pregunta antes de cada comando aws o ssh

No se puede leer la configuración, así que el guardia no conoce tus listas y pregunta por
si acaso. `augmented-brain doctor` nombra el problema. Si dice "ajustes desconocidos", el
fichero tiene una línea que esta versión no conoce: una errata, o un ajuste de una versión
más nueva. Si es un ajuste de una versión más nueva, actualiza con
`augmented-brain update`. Si es una errata, o un cambio que lo rompió, vuelve a una
versión que funcionaba: `augmented-brain config history` y después
`augmented-brain config restore <número>`. Los dos funcionan aunque el fichero esté roto.
Lo tienes en [Configuración](configuration.es.md) y en
[Versiones de la configuración](config-history.es.md).

### Claude no puede hacer push, aunque se lo pida

Si dice que un repositorio es **intocable**, es a propósito: solo tú puedes cambiarlo, en
la página Guardia del panel (mira [El guardia](guard.es.md#repositorios-que-claude-nunca-debe-tocar)).

El guardia para `git commit` y `git push` salvo que el repositorio esté en la lista. Haz
tú el push, o añade el repositorio a `git_allowed_repos` (o a `git_ask_repos`) y pide a
Claude que use `git -C <repo>`. Lo tienes en
[El guardia](guard.es.md#deja-que-claude-haga-commit-en-un-repositorio).

### Claude pregunta antes de un push que antes hacía

Las listas del guardia cambiaron fuera del panel desde que las aceptaste (a mano, con un
comando o al volver a una versión). Hasta que las aceptes en la página Guardia o empiece
una sesión nueva, un commit, push o merge que permiten pregunta antes. `doctor` lo dice.
Abre la página Guardia del panel, mira qué cambió y acéptalo. Lo tienes en
[El guardia](guard.es.md#solo-tú-cambias-las-listas-del-guardia).

### Claude necesita root en una máquina

Claude te da el comando exacto para que lo lances tú. Para una migración que necesita
muchos comandos, añade una excepción con fecha, `root_ask = ["db01 until 2026-12-31"]`.
Así root en esa máquina te pregunta cada vez, hasta esa fecha.

### Las herramientas de augmented brain fallan después de actualizar

Una sesión que estaba abierta sigue con el programa antiguo. `augmented-brain session
restart --old` reinicia las que están quietas, con la misma conversación; una de Claude
Desktop sigue abierta y solo se reinicia su augmented brain (en Windows, ciérrala y vuelve
a abrirla). `augmented-brain doctor` dice cuántas sesiones
siguen con la versión antigua.

### El instalador o la actualización no pueden descargar

A veces GitHub deja de entregar los ficheros de una versión durante un rato. El
instalador y `augmented-brain update` lo intentan tres veces, y también por la API de
GitHub, que suele seguir funcionando cuando la dirección de siempre no. Si falla todo,
lo dicen y no cambia nada: se queda el programa que tenías. Vuelve a probar dentro de un
rato.

Cómo puede acabar `augmented-brain update`:

- **"no puedo llegar a GitHub para ver la última versión"**: `update` pregunta la última
  versión una vez, antes de descargar. Vuelve a probar dentro de un rato.
- **"el programa sigue siendo la X, no la Y"**: el instalador se ejecutó pero el programa
  no cambió. No ha cambiado nada más; vuelve a lanzar `augmented-brain update` más tarde.
- **"La copia de seguridad ha fallado, así que no se ha actualizado nada"**: arregla la
  copia (más abajo) y vuelve a actualizar, o lanza `augmented-brain update --force`.

Lo tienes en [Instalación](install.es.md#actualizar).

### La instalación se paró a medias

Vuelve a lanzar el comando de instalación (o `augmented-brain init`): hasta que acaba no
hay configuración, así que empieza de nuevo. Lo tienes en [Instalación](install.es.md).

### "Lo escribió un augmented brain más nuevo"

Una versión más nueva, en otro ordenador, pasó las notas a un formato nuevo. Actualiza
también este ordenador: `augmented-brain update`. El programa antiguo rechaza las notas
en vez de romperlas.

### Ha fallado la copia de seguridad

Claude te lo dice al empezar la siguiente sesión. Escribe `augmented-brain backup` en una
carpeta del espacio para ver por qué:

- **No hay red, o no hay acceso.** Comprueba que llegas al repositorio. En GitHub, entra
  con `gh auth login`.
- **"Stopped after 3m0s without an answer"** (se paró tras 3 minutos sin respuesta). La
  red o el repositorio no respondían, así que la copia se rindió en vez de esperar para
  siempre. Vuelve a probar cuando haya red; la siguiente copia no espera detrás de esta.
- **"Another backup has been running for over 10m0s"** (otra copia lleva más de 10
  minutos). Estorba una copia que ya no responde. Vuelve a probar más tarde.
- **Un secreto en una nota.** Dice qué fichero es. Quita el secreto y vuelve a hacer la
  copia. `augmented-brain check` también nombra esos ficheros.
- **Dos máquinas cambiaron la misma nota.** Nombra las notas, y la carpeta donde juntarlas
  a mano. Cuando cada máquina solo añadió entradas al diario, se juntan solas.
- **Un fichero de más de 50 MB.** Se queda fuera, y el resto se copia. Sácalo de la
  carpeta de notas.

### `doctor` dice que un espacio no tiene copia de seguridad

Si pierdes este ordenador, pierdes las notas de ese espacio. Pide a Claude «configura la
copia de seguridad» en una carpeta del espacio. O lanza allí `augmented-brain backup setup
<dirección de un repositorio privado>`, y después `augmented-brain backup`. Lo tienes en
[Copia de seguridad y restauración](backup.es.md).

### Una nota no es válida

`augmented-brain check` en una carpeta del espacio lista cada problema con su fichero.
Arregla a mano la cabecera (entre las líneas `---`) y vuelve a lanzar `check`. Hasta
entonces, el índice no se rehace, pero las herramientas siguen funcionando.

Un fichero que no se puede leer (por sus permisos, o un enlace roto que dejó una
sincronización) sale de la misma forma: arregla sus permisos o sácalo de la carpeta de
notas. Un fichero cuyo nombre empieza por punto, como el de bloqueo que deja Emacs
mientras editas una nota, se salta.

### Un repositorio "no está en este ordenador"

Una tarea nombra en `repos:` un repositorio que no está en ninguna carpeta de trabajo del
espacio. Se movió, se cambió de nombre o se borró, o este ordenador nunca lo tuvo. Dile a
Claude dónde está ahora, o quítalo de la tarea.

### Claude Code guarda una segunda memoria

`doctor` avisa de que Claude Code guarda una memoria propia para tus carpetas. Se desvía
de la del espacio. Añade `"autoMemoryEnabled": false` al fichero de ajustes que nombra
`doctor`. Lo que ya tiene se queda, y puedes [traerlo](import.es.md).

### Otros usuarios pueden leer mis notas

`doctor` te avisa. Escribe `augmented-brain check` en una carpeta del espacio: deja las
notas solo para ti.

### El panel no se abre, o su dirección ha dejado de funcionar

En un ordenador sin pantalla, `augmented-brain panel` escribe la dirección para que la
abras tú. Esa dirección sirve una vez, en los próximos 2 minutos. Las páginas que abre
llevan una llave que cambia cada vez que arranca el panel, y el panel se para después
de 30 minutos sin usarlo. Vuelve a lanzar `augmented-brain panel` para tener una
dirección nueva. Si tu navegador dice «Este enlace ya se usó» y no lo has abierto tú, lo
ha abierto otra cosa: mira la página Guardia del panel.

### El panel no se abre desde otro dispositivo

`doctor` dice qué falta: Tailscale parado, HTTPS apagado en tu red, o una dirección que se
quedó publicada después de parar el panel (te da el comando para quitarla). Lo tienes en
[El panel desde otro dispositivo](panel-remote.es.md).

### El acceso remoto no funciona

Lanza `augmented-brain remote status`: debajo de cada problema, `→` dice qué hacer. Lo
normal es que Claude aún no confíe en la carpeta (lanza una vez `cd <carpeta> && claude`)
o que el linger esté apagado. `remote install` se para, sin cambiar nada, en una máquina
sin servicios de usuario de systemd (un contenedor, WSL sin systemd). Lo tienes en
[Acceso remoto](remote.es.md).

### Una sesión se cortó

La ventana se cerró a la fuerza, o el ordenador se apagó. La siguiente sesión te ofrece
recuperarla. Di que sí, o usa el panel: Hoy la enseña. Claude Code guarda cada
conversación 30 días.

### La ventana usa mucha memoria

Claude te dice que la ventana usa mucha memoria, o `doctor` avisa de una ventana de
Claude de varios GB. El proceso que lleva las sesiones de la ventana ha crecido mucho
más de lo normal: el host de extensiones de VS Code (el programa dentro de VS Code que
ejecuta Claude y las demás extensiones), o el propio Claude en un terminal. Lo normal es
que se quede cerca de 0,5 GB durante días. De vez en cuando uno crece sin parar, y el
sistema acaba cerrándolo, con todas las sesiones de Claude de la ventana. La longitud de
la conversación no tiene nada que ver.

Qué hacer, en cuanto lo veas:

1. Pídele a Claude que aparque la tarea ("apárcala"). Se guarda dónde estabas y lo que
   viene después.
2. Abre una ventana nueva de VS Code (o una sesión nueva en el terminal) y retoma la
   tarea allí.
3. Cierra la ventana vieja cuando acabes con ella: así se libera la memoria.

Claude avisa a los 2 GB, luego a los 4 y a los 8 GB, una vez cada uno. En Windows no
lo mira.

### Se acumulan worktrees viejos y carpetas temporales

`doctor` dice que ha quedado algo del trabajo con Claude Code. Cuando Claude Code trabaja
en un worktree (una segunda copia de trabajo de un repositorio, en `.claude/worktrees`), el
worktree se queda al terminar la sesión. Sus sesiones dejan además carpetas en la carpeta
temporal del sistema, que un ordenador que casi no se reinicia nunca vacía.

Ejecuta `augmented-brain clean`. Enseña lo que quitaría y pregunta antes. Un worktree con
cambios, commits o ficheros ignorados (un `.env`) propios nunca se quita: solo lo nombra,
para que lo mires tú.

### Después de recuperar una copia, nada pregunta antes de tocar producción

Las copias de seguridad no llevan las listas del guardia. Abre una sesión de Claude: se
ofrece a preguntártelas. O añade tú `production_aws_profiles` y `untouchable_hosts` a la
configuración.

### Después de recuperar, hay secretos que "no se ha podido comprobar"

No se pudo preguntar al llavero: un Linux sin sesión de escritorio, un llavero de macOS
bloqueado, o Windows, donde no se le puede preguntar. Puede que los secretos estén. Abre
una sesión de escritorio o desbloquea el llavero, y mira la página Secretos del panel.
Guarda los que falten con el comando que sale. Lo tienes en
[Copia de seguridad y restauración](backup.es.md).

### Windows: los acentos salen mal (`sesiÃ³n`)

Windows PowerShell 5.1 lee mal lo que escriben otros programas cuando lo captura. El texto
está bien: en Git Bash, o en una ventana, se ve bien.

### macOS: no se abrió la ventana de las herramientas de desarrollo

Escribe `xcode-select --install`, y después otra vez el comando de instalación.

Más problemas del instalador, en cada sistema: [Instalación](install.es.md#si-algo-falla).

## Preguntas

### ¿Es gratis?

augmented brain sí: código abierto, licencia MIT, sin cuenta y sin suscripción. Pero
trabaja con Claude Code, que necesita un plan de pago de Claude (Pro o superior).

### ¿Mis notas salen de mi ordenador?

augmented brain no las manda a ningún sitio. Pero tu asistente las lee para trabajar,
como cualquier fichero que abre, así que viajan con la conversación a su proveedor. Con
copia de seguridad, van también al repositorio privado que elegiste.

### ¿Puedo editar las notas a mano?

Sí, son ficheros Markdown. Después, escribe `augmented-brain check`.

### ¿Con qué asistentes funciona?

Hoy, con Claude Code. Las herramientas son un servidor MCP, un estándar abierto, así que
pueden venir otros asistentes.

### ¿Y si dejo de usarlo?

Tus notas se quedan donde están, como ficheros normales. No hay nada que exportar. Lo
tienes en [Instalación](install.es.md#desinstalar).

## ¿Sigues atascado?

Cuéntalo en [las incidencias del proyecto](https://github.com/icalvete/augmented-brain/issues),
con lo que salió en pantalla y lo que dice `augmented-brain doctor`. La página Ayuda del
panel lo rellena por ti.

Lo que cambia en cada versión está en el
[registro de cambios](https://github.com/icalvete/augmented-brain/blob/main/CHANGELOG.md)
(en inglés).
