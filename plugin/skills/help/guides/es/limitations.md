# Limitaciones conocidas


Lo que augmented brain todavía no sabe hacer, o hace solo en parte, y qué hacer en su
lugar cuando hay algo que hacer. Cuando algo falla y quieres arreglarlo, mira
[Si algo falla](troubleshooting.md). La lista crece cuando sale algo nuevo: si
encuentras algo, cuéntanoslo desde la página de Ayuda del panel.

## Solo Claude Code

augmented brain funciona con Claude Code: la terminal, la aplicación de escritorio y las
extensiones de los editores. Otros asistentes podrían usar las herramientas de notas por
MCP, pero el inicio de sesión y el guardia dependen de los hooks de Claude Code, así que
no tendrían ninguno de los dos.

## Windows

- **Claude ejecuta los comandos en Git Bash, no en PowerShell.** El guardia lee los
  comandos de una shell POSIX (Bash), no de PowerShell, así que para cada comando de
  PowerShell, y Claude usa Git Bash en su lugar. Lo que solo funciona en PowerShell lo
  lanzas tú: Claude te da el comando. Si Claude prueba PowerShell de todos modos, ves en
  rojo `PreToolUse:PowerShell hook error`: es el guardia parándolo, y Claude sigue en
  Bash. No pasa nada.
- **Instalar por ssh no pregunta nada.** En una sesión ssh, Windows dice que no hay
  nadie delante, así que el instalador no puede preguntar: imprime los comandos que hay
  que lanzar, y winget suele fallar ahí. Instala desde PowerShell en el escritorio, o con
  `AB_YES=1` para contestar que sí a todo.
- **Los acentos pueden salir rotos** (`sesiÃ³n`) en Windows PowerShell 5.1. El arreglo
  está en [Si algo falla](troubleshooting.md).
- **Sin aviso de la memoria de la ventana.** En Linux y macOS, Claude avisa cuando la
  ventana que lleva tus sesiones pasa de 2 GB. En Windows no lo mira: si Claude va
  lento, aparca la tarea y abre una ventana nueva.
- **No se puede preguntar al llavero.** Después de recuperar una copia, augmented brain
  no sabe qué secretos tiene ya el llavero de Windows. Mira la página Secretos del panel
  y guarda los que falten.

## El primer arranque de Claude Code

La primera vez que lanzas `claude`, Claude Code enseña sus propias pantallas antes de que
augmented brain diga nada: un tema de colores, tu inicio de sesión y si confías en la
carpeta. En tu espacio, contesta **Yes, I trust this folder**: Enter a secas elige «No,
exit». Ctrl-C no sale de la pantalla del tema; elige uno con Enter. Esas pantallas son de
Claude Code, y augmented brain no puede cambiarlas.

## Acceso remoto y una sesión por tarea

`augmented-brain remote` (llegar a tu ordenador desde claude.ai o Claude Desktop)
funciona en Linux, macOS y Windows. `augmented-brain session start` (una sesión remota
para una tarea) por ahora solo funciona en Linux: en macOS y Windows lo dice y no cambia
nada.

- **En un Mac y en Windows, el acceso remoto arranca cuando entras en el ordenador**, no
  antes. Tras reiniciar, también cuando el ordenador se reinicia solo (una actualización,
  un corte de luz), nadie puede conectarse hasta que entres una vez. En Linux puede
  arrancar sin que nadie entre.
- **En Windows solo funciona mientras tienes la sesión abierta.** Si la cierras, se para.
- **En Windows, una carpeta con `%` en el nombre** puede impedir que el acceso remoto
  arranque en ella. Elige otra con `folders` en `remote.toml`.
- **Una conexión real solo se ha probado en Linux.** En un Mac y en Windows, instalar,
  ver el estado y quitarlo se prueban en la CI con un Claude de mentira; conectarse
  desde claude.ai todavía no se ha probado.
- Una sesión abierta con `session start` no vuelve tras reiniciar: pídela otra vez. La
  sesión remota del espacio sí vuelve.
- Claude admite una sesión remota por carpeta, y la carpeta tiene que ser de confianza
  ella misma. `session start` te dice qué hacer la primera vez.

## macOS

Funciona en un Mac, y cada cambio se prueba en Macs en la CI: el instalador, el guardia,
el panel en Safari y un llavero bloqueado. Pero ninguna persona lo ha usado todavía en un
Mac de verdad, así que nadie ha mirado lo que solo se nota delante de la pantalla: abrir
el programa desde el Finder («Abrir igualmente»), una sesión real de Claude, Claude
Desktop, o Tailscale con `panel remote`. Si a ti te falla algo, nadie lo ha visto
todavía: cuéntalo, por favor.

## El guardia

El guardia lee cada comando como texto antes de que Claude lo ejecute. Atrapa los errores
que comete un asistente; no es un muro, y no sustituye a los permisos, las copias de
seguridad ni tu propia revisión. Lo que no puede ver:

- **Scripts escritos por otro programa.** Si Claude escribe un script con un programa
  (Python, una herramienta de compilación) y lo ejecuta más tarde, el guardia solo ve lo
  que el script tiene cuando se ejecuta. Un script escrito y ejecutado en el mismo
  comando sí se revisa; uno escrito de una forma que el guardia no puede leer hace que te
  pregunte antes.
- **Valores que solo se conocen al ejecutar.** Cuando un programa o una carpeta salen de
  una variable o de la salida de otro comando, el guardia no puede saberlo antes: te
  pregunta, pero no puede decirte exactamente qué se ejecutará.
- **Ajustes cambiados con un programa.** El guardia pregunta antes de que Claude cambie
  los ajustes de augmented brain o de Claude Code con sus herramientas de ficheros o con
  comandos habituales. Un programa pequeño escrito en el momento (en Python, por ejemplo)
  aún podría cambiarlos; el guardia lo nota después y pregunta antes de un commit o un
  push, pero no lo impide. Un programa así también podría usar el panel con un enlace
  que haya visto: el enlace sirve una vez, así que tu navegador diría que ya se usó, y
  Claude Code te dice lo que cambió el panel.
- **Alias de git en ficheros incluidos.** Tus alias de git cuentan como lo que quieren
  decir (`git ci` por `git commit`) cuando están en tus ajustes de git habituales. Un
  alias en un fichero que esos ajustes incluyen con `include` no se ve.
- **Commits locales de merge, revert y cherry-pick.** El guardia para `git commit` donde
  Claude no puede hacer commits, pero `git merge`, `git pull`, `git revert`,
  `git cherry-pick` y `git am` también hacen commits en tu ordenador. En un repositorio
  intocable también se paran; en los demás pasan. No se publican: un push sigue parado.
- **Reglas que guarda Claude.** Si no cambias nada, Claude guarda una regla, tu perfil o
  una máquina para el guardia sin preguntarte antes, y todas las sesiones siguientes los
  cargan. Un texto que Claude haya leído (una web, un correo) podría convencerle de
  guardar uno. Te enteras: Claude Code te enseña, una vez, lo que se ha guardado. Si no
  se lo has pedido, dile a Claude que lo deshaga, o míralo en el panel. Para que te
  pregunte antes, elige «Preguntarme antes» en la página Guardia del panel (mira
  [El guardia](guard.md#preguntar-antes-de-que-claude-guarde-una-regla)).
- **Sin reglas en las sesiones que se saltan las preguntas.** Con «Preguntarme antes»,
  una sesión que se salta las preguntas de permiso de Claude Code (`bypassPermissions`)
  no tiene a quién preguntar, así que ahí Claude no puede guardar reglas, tu perfil ni
  máquinas para el guardia. Te dice qué quería guardar.
- **Solo en tus espacios: un script que entra en uno.** Cuando augmented brain se queda
  en tus espacios, una orden lanzada fuera de ellos que entra en uno (`cd ~/work/app`,
  `git -C ~/work/app`) se mira con todas las reglas. Un script que cambia de carpeta por
  dentro, no: lanzado desde fuera, un `bash build.sh` que entra en un espacio se mira
  como de fuera. Mira [Fuera de todos los espacios](spaces.md#fuera-de-todos-los-espacios).
- **Solo en tus espacios, con una versión anterior aún abierta.** Una sesión de Claude
  abierta antes de actualizar sigue con el programa anterior, que no conoce esta opción.
  Su arranque acepta las listas del guardia como las cuenta ese programa, así que hasta
  que vuelve a arrancar una sesión de la versión nueva, augmented brain actúa en todo el
  ordenador. Es el lado seguro, y Claude Code te avisa cada vez que se aceptan las listas.
- **Preguntas en los modos automáticos.** Cuando el guardia pregunta, Claude Code te
  enseña la pregunta. Qué hace Claude Code con ella en sus modos totalmente automáticos
  depende de Claude Code, no de augmented brain.

Más en [El guardia](guard.md).

## En otros sitios

- **`watch pr` solo sigue pull requests de GitHub.** Usa la herramienta de GitHub, `gh`.
  Una merge request de GitLab o de otro sitio no se sigue: mírala allí.
- **Un fichero de más de 50 MB se queda fuera de la copia.** El resto se copia, y la
  copia dice qué fichero es. Guarda los ficheros grandes fuera de la carpeta de notas.

## Probado por poca gente

augmented brain está en beta. Una persona lo usa a diario en Ubuntu, y lo ha probado en
Windows 11; en macOS solo se prueba en la CI. Hasta ahora, cada versión ha mantenido
funcionando lo que tienes, y desde la 1.0 será una promesa (mira
[Lo que no cambia](compatibility.md)). Haz copia de lo importante.
