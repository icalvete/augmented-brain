# Limitaciones conocidas

[Read in English](limitations.md)

Lo que augmented brain todavía no sabe hacer, o hace solo en parte. Cada punto dice qué
hacer en su lugar. La lista crece cuando sale algo nuevo: si encuentras algo, cuéntanoslo
desde la página de Ayuda del panel.

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
  está en [Si algo falla](troubleshooting.es.md).

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

Cada pull request pasa las pruebas en un Mac en la CI, también el instalador, Safari y un
llavero bloqueado. Unas pocas cosas solo las ve una persona delante de un Mac de verdad,
y de momento nadie las comprueba:

1. Abrir desde el Finder el programa descargado con un navegador («Abrir igualmente»).
2. Una sesión real de Claude Code: el guardia para un push y el inicio de sesión carga.
3. Claude Desktop instalado por el instalador, y Tailscale con `panel remote`.
4. Sin red: `backup`, `update` y `panel remote` lo dicen y no dejan nada a medias.
5. El panel en Safari con la consola abierta, sin errores.

Si a ti te falla alguna, nadie lo ha visto todavía: cuéntalo, por favor.

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
  push, pero no lo impide.
- **Alias de git en ficheros incluidos.** Tus alias de git cuentan como lo que quieren
  decir (`git ci` por `git commit`) cuando están en tus ajustes de git habituales. Un
  alias en un fichero que esos ajustes incluyen con `include` no se ve.
- **Commits locales de merge, revert y cherry-pick.** El guardia para `git commit` donde
  Claude no puede hacer commits, pero `git merge`, `git revert` y `git cherry-pick`
  también hacen commits en tu ordenador. No se publican: un push sigue parado.
- **Preguntas en los modos automáticos.** Cuando el guardia pregunta, Claude Code te
  enseña la pregunta. Qué hace Claude Code con ella en sus modos totalmente automáticos
  depende de Claude Code, no de augmented brain.

Más en [El guardia](guard.es.md).

## Probado por poca gente

augmented brain está en alfa. Una persona lo usa a diario en Ubuntu, y lo ha probado en
macOS y Windows 11. Lo que tienes sigue funcionando de una versión a otra (mira el
[aviso legal](https://github.com/icalvete/augmented-brain/blob/main/DISCLAIMER.md#aviso-legal)),
pero haz copia de lo importante.
