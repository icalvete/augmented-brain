# Registro de cambios

## Unreleased

- **Abre el panel desde el móvil u otro ordenador.** Un comando nuevo,
  `augmented-brain panel remote`, publica el panel en tu propia red de
  [Tailscale](https://tailscale.com) y escribe su dirección y un código QR. Solo tus
  dispositivos llegan a él, y solo entra tu cuenta de Tailscale; la dirección deja de
  funcionar cuando el panel se para, y una que se quede colgada se quita en el siguiente
  arranque. Tailscale es opcional: sin él, el comando explica qué es y cómo conseguirlo, y
  lo mismo para HTTPS en tu red y, en Linux, para dejar publicar a tu usuario. `doctor`
  tiene una línea sobre esto en cuanto lo has usado. Guía nueva: «El panel desde otro
  dispositivo». El código QR usa `rsc.io/qr` (licencia BSD).
- **El panel en el móvil.** En una pantalla estrecha, la barra lateral pasa a ser una sola
  barra arriba (logo, un botón de menú que dice dónde estás, el espacio) y las secciones
  se abren debajo como una lista, con el buscador; el perfil y el pie van al final de la
  página. El pie dice cuándo el panel está abierto también en tus dispositivos de
  Tailscale.
## 0.10.0-alpha

Publicada el 2026-11-07.

- **El acceso remoto arranca donde trabajas.** La sesión de cada espacio arranca ahora en
  la primera carpeta de trabajo del espacio, no en una carpeta nueva dentro de la
  configuración. Si ya usas Claude ahí, ya confía en ella y no hay nada más que aceptar.
  Un ajuste nuevo, `folders` en `remote.toml`, elige otra carpeta dentro del espacio. Solo
  un espacio sin carpetas de trabajo sigue teniendo una propia. Vuelve a lanzar
  `augmented-brain remote install` para mover un servicio que ya tengas.
- **`update` hace las comprobaciones de alrededor.** Antes de una versión nueva, revisa la
  instalación (y enseña lo que encuentre) y hace la copia de seguridad de cada espacio que
  la tenga; si falla una copia, no actualiza nada (`--force` actualiza igualmente). Después, dice cuántas sesiones de
  Claude abiertas siguen con la versión vieja y qué tarea está activa en cada espacio,
  para aparcarla antes de reiniciar Claude.

## 0.9.0-alpha

Publicada el 2026-11-06.

- **Un nombre y una personalidad para tu asistente.** La página Perfil del panel tiene
  una parte nueva, **Tu asistente**, y `profile_save` dos campos nuevos,
  `assistant_name` y `assistant_personality`. Cada sesión empieza con ellos; cambian cómo
  habla Claude, nunca las reglas, el guardia ni su honestidad. Con nombre, los correos
  que Claude escribe en tu nombre van firmados con él y, debajo, una línea que añade el
  programa y que no se puede quitar, como «asistente de Ana (IA, Claude)». Viven en una
  sección nueva de `profile.md`; las versiones anteriores la conservan como texto.
  Además, `profile_save` ya no borra lo que no se le manda: una parte que no se manda
  conserva lo que tiene, y una vacía se borra.
- **Las reglas y el conocimiento se pueden cambiar y retirar, y nunca se duplican sin
  que nadie lo sepa.** Guardar una regla, referencia, lección o área con un título que
  ya existe creaba una segunda copia («-2») y se cargaban las dos; ahora no se crea
  nada, y Claude recibe el id. `rule_new` y `knowledge_new` aceptan un `id` para cambiar
  una; la versión sustituida se guarda en `archive/`. Una herramienta nueva,
  `note_retire`, pasa una regla, referencia o lección a `archive/` con el motivo y qué
  la sustituye: se conserva, pero una regla retirada deja de cargarse. Sin campos nuevos
  en las notas, así que las versiones anteriores las leen igual.
- **Saber qué modelo responde.** Un ajuste nuevo, `preferred_model` en un fichero nuevo,
  `session.toml` (o en la página Guardia del panel): si una sesión responde con otro
  modelo, Claude te lo dice, al empezar si Claude Code lo nombra y si no después de una
  respuesta (en una sesión nueva, como tarde la segunda), una vez por sesión y modelo. Nunca cambia el modelo por su
  cuenta. Vacío por defecto: no se comprueba nada.
- **Tras una compactación, el resumen no se toma como un hecho.** Antes se le pedía a
  Claude escribir el diario desde el resumen; ahora se le pide comprobar antes de afirmar
  algo de antes, y decirte que la sesión se ha compactado y proponerte aparcar la tarea y
  seguir en una sesión nueva.

- **Repositorios que Claude nunca toca, y un guardia que Claude no puede aflojar.** Una
  lista nueva de **repositorios intocables** (`untouchable_repos` en un fichero nuevo,
  `guard.toml`) donde el guardia deniega commit, push y fusionar una pull request, digan
  lo que digan `git_allowed_repos` y `git_ask_repos`. Una página **Guardia** nueva en el
  panel muestra y cambia todas las listas a las que responde el guardia (repositorios
  intocables y permitidos, excepciones de root con su fecha de fin, perfiles de
  producción, máquinas intocables), respetando tus comentarios; quitar un repositorio
  intocable o añadir una excepción de root te pide escribir su nombre. Para que esas
  listas sigan siendo tuyas: el guardia ahora también pregunta antes de escribir en
  `guard.toml`, `secrets.toml` y `remote.toml`; deniega que Claude lea `panel.json` (la
  clave del panel) o llame al panel desde un comando; y guarda un sello de sus listas, de
  modo que si algo que no es el panel ni el inicio de una sesión las cambió, un push o
  merge que las listas permiten pregunta antes, y `doctor` y la página Guardia lo dicen
  hasta que las aceptes. Las preguntas de un push o un merge ahora nombran la rama y
  adónde lleva el remoto (sin contraseñas). `doctor` avisa de un `guard.toml` ilegible y
  de un repositorio que es intocable y está también en otra lista. Una sesión que aún
  ejecuta una versión anterior no hace cumplir `guard.toml`. Un script que Claude escriba
  en línea todavía puede editar los ficheros: se nota, no se impide. El hook del plugin
  ahora también escucha `Read` y `Grep`: actualiza el plugin junto con el programa.

- **Personas compartidas entre espacios, desde el panel.** La misma persona puede estar
  ahora en varios espacios (un cliente que también es amigo). Cada espacio guarda su
  propia nota de ella, con el mismo id, y solo tiene lo que quieres que se vea allí; en
  disco no cambia nada, así que las versiones anteriores leen los ficheros como siempre.
  El panel lista a cada persona una vez, con una etiqueta por espacio, y su página es un
  solo formulario: un papel y otros nombres por espacio, y una lista de detalles con una
  casilla por espacio. Un detalle que no marcas para un espacio no se escribe nunca allí,
  así que algo personal no llega a un fichero de trabajo ni a su copia de seguridad. La
  herramienta `person_save` de Claude tiene un parámetro nuevo, `spaces`: con varios
  espacios y ninguno indicado, te pregunta dónde guardar en vez de adivinar. Dos notas
  escritas hoy con el mismo id (un párrafo cada una) salen como una sola ficha, cada
  párrafo un detalle marcado en su propio espacio.

- **Un aviso antes de que una ventana se quede sin memoria.** A veces el proceso que
  lleva las sesiones de Claude de una ventana (el host de extensiones de VS Code, o el
  propio Claude en una terminal) crece sin parar, de su medio gigabyte de siempre a
  muchos, hasta que el sistema lo mata con todas las sesiones de la ventana. Ahora,
  después de cada respuesta, el hook Stop mira su memoria y, a partir de 2 GB (luego 4 y
  8 GB, una vez cada uno), Claude te avisa y te propone aparcar la tarea y seguir en una
  ventana nueva. `doctor` también lo dice, para cada ventana de Claude del ordenador. En
  Linux y macOS; añade una fracción de milisegundo. Mira «Si algo falla», «La ventana usa
  mucha memoria».
- Las páginas de inicio enseñan el panel: una imagen de su página Hoy, para qué sirve y
  un enlace a su guía, con un enlace Panel en el menú de arriba.
- El panel dice «1 aparcada» en español, en singular, en vez de «1 aparcadas».
- **Acceso remoto: Claude accesible desde otro ordenador, también tras reiniciar.**
  `augmented-brain remote install` prepara un servicio por espacio que mantiene en marcha
  `claude remote-control`, para que puedas conectarte a esta máquina desde claude.ai/code o
  Claude Desktop (elige el entorno `<nombre del ordenador>-<espacio>`). Cada sesión empieza
  en una carpeta de su espacio, así que sus notas y reglas son las de ese espacio. `remote
  status` y `doctor` dicen si funciona, si Claude confía en su carpeta y si el «linger»
  está activado; `remote uninstall` lo quita y no borra nada más. Claude tiene que confiar
  en cada carpeta (ábrela una vez con `claude`) y el linger necesita un comando como
  administrador: `install` escribe los dos, no los lanza. Los ajustes, en el nuevo
  `remote.toml`. Solo en Linux por ahora. Mira Acceso remoto.

## 0.8.0-alpha

Publicada el 2026-11-04.

- **`space add` ya no acepta `--language`.** Nunca hizo nada: el idioma es el mismo
  para todos los espacios de un ordenador, el ajuste `language` que pone `init`. Un
  script que aún lo pase recibe ahora un error que lo nombra.
- Un sitio de documentación, en inglés y español, en `guide/` de la web: veinte páginas,
  de los primeros pasos a qué hacer si algo falla, con una referencia de cada comando y
  opción, herramienta MCP y ajuste de `config.toml`. Los tests fallan si falta alguno en
  la referencia de cualquiera de los dos idiomas.
- Una sola lista de comandos, la que usan la ayuda y esos tests. Todos los comandos
  funcionan como antes.
- **Todos los mensajes salen en tu idioma.** «Ajustes desconocidos», «lo escribió un
  augmented brain más nuevo», lo que está mal en `config.toml`, un paso fallido a un
  formato nuevo, el uso del comando `hook` y los errores del servidor MCP ahora salen
  también en español: en el `language` de la configuración, o en el del sistema mientras
  no se pueda leer la configuración.
- Los motivos del guardia, que Claude lee y tú ves cuando pregunta, siguen todos tu
  `language`. Hasta ahora solo lo hacían los de fusionar una pull request, junto a los
  demás en inglés.
- El guardia ya no toma el texto que se pasa a `gh pr merge --body` o `--subject` (o a
  `glab mr merge --message`) como parte del comando: una dirección o un `--admin`
  escritos ahí no cambian lo que es la fusión. Un `gh pr merge` escrito en el cuerpo de
  una pull request o en un mensaje de commit ya no era una fusión; ahora los tests lo
  comprueban.
- **Un agujero del guardia, cerrado.** `gh pr merge -m` (`--merge`, que no lleva valor)
  hacía que el guardia se saltara la palabra siguiente, así que `gh pr merge -m <pull
  request de otro repositorio>` pasaba en un repositorio donde fusionar está permitido.
  Ahora se rechaza, como cualquier fusión que nombre un repositorio distinto de aquel en
  el que se lanza.
- CI para una ejecución antigua cuando un push más nuevo a la misma rama o pull request
  la sustituye (nunca en `main`), y lanza las comprobaciones del instalador de dos en
  dos, así que los trabajos de macOS ya no esperan máquinas hasta que se cancelan.
- `compat-check.sh` lanza las versiones anteriores en inglés, así que las palabras que
  busca están ahí sea cual sea el idioma de la máquina.

## 0.7.0-alpha

Publicada el 2026-11-03.

- El panel dice «1 fichero cambiado» y «1 commit sin subir», no «1 ficheros», «1
  commits».
- El panel tiene una página **Repositorios**, un mapa de tus repositorios: cada
  repositorio git de las carpetas de trabajo de cada espacio (hasta tres carpetas más
  abajo), agrupados por espacio, primero los que tienen trabajo pendiente. Cada uno dice
  su rama, adónde sube (sin usuario ni contraseña), los ficheros cambiados, los commits
  sin subir, las tareas abiertas que lo nombran y una línea tuya sobre qué es, que
  escribes ahí mismo.
- Esa línea se guarda en las notas del espacio, en la referencia «Repositorios»
  (`knowledge/ref-repositories.md`), así que la copia de seguridad la guarda y las
  versiones anteriores la leen como cualquier otra referencia. La nueva herramienta MCP
  `repo_describe` deja que Claude también la escriba.
- El bloque Repositorios de Hoy enlaza con la página nueva, y pide describir un
  repositorio que aún no tiene línea.
- El panel tiene una página **Ayuda**: enlaces a las preguntas y las guías de la web, en
  tu idioma, y enlaces para avisar de un problema o proponer una idea en GitHub. El
  enlace de problema lanza la comprobación de salud al pulsarlo y abre el formulario de
  GitHub con la versión, el sistema y esa comprobación ya puestos, sin lo personal
  (carpeta de inicio, carpetas de notas y de trabajo, correos, nombres de espacios,
  máquinas intocables, perfiles de AWS de producción, líneas que parecen un secreto). No
  se envía nada hasta que pulsas el botón de enviar de GitHub, aunque el texto llega a
  GitHub en la dirección en cuanto se abre el formulario.
- Formularios de incidencia en GitHub para un problema y para una idea, y un enlace a
  las preguntas.
- **Fusionar una pull request es un push.** `gh pr merge`, `glab mr merge` y `glab mr
  accept` siguen ahora `git_allowed_repos` y `git_ask_repos`, como `git push`: pasan en
  un repositorio donde Claude puede subir, preguntan en uno donde pregunta y se bloquean
  en el resto, como antes. Con `main` protegida para que los cambios solo entren por una
  pull request, ya no hace falta lanzar la fusión a mano. gh y glab no tienen `-C`, así
  que el comando nombra el repositorio con un `cd`, igual que un push lo hace con `git
  -C`: `cd <repo> && gh pr merge 12`. La carpeta de la sesión por sí sola no cuenta. Un
  repositorio que nombre el comando (`-R`, `GH_REPO`, la dirección de la pull request)
  tiene que ser uno de los remotos de ese repositorio, o la fusión se bloquea. `gh pr
  merge --admin`, que se salta la protección de la rama, siempre pregunta. Las versiones
  (`gh release create`, …) siguen siendo cosa tuya.

## 0.6.1-alpha

Publicada el 2026-11-03.

De la 0.6.0 en una máquina real: los programas anteriores a la 0.6.0 rechazan un ajuste
de `config.toml` que no conocen, y la 0.6.0 añadió uno. Cada sesión de Claude abierta
desde antes de actualizar se quedó con el programa viejo y perdió sus herramientas de
augmented brain.

- Los nombres de los secretos viven en un fichero propio, `secrets.toml`, que los
  programas anteriores nunca leen. La lista que la 0.6.0 dejó en `config.toml` pasa ahí
  al empezar una sesión o con `check`, y la copia de seguridad lleva el fichero, como
  lleva el perfil.
- Una configuración de un programa más nuevo, con ajustes que este no conoce, dice que
  actualices, no que el fichero está mal.
- `doctor` y el inicio de sesión dicen cuántas sesiones de Claude de este ordenador
  siguen con un augmented brain anterior, y que las cierres y las vuelvas a abrir. El
  servidor de cada sesión apunta su versión al arrancar; en Linux también se encuentran
  las de versiones anteriores.
- Una restauración nombra los secretos que no tiene el llavero de la máquina nueva, cada
  uno con el comando que lo guarda; la página Secretos del panel también enseña ese
  comando.
- CI comprueba que las dos últimas versiones leen lo que escribe esta
  (`compat-check.sh`): su `doctor`, `check`, herramientas MCP, inicio de sesión y
  restauración. Habría pillado la 0.6.0.

Además, de la revisión de la 0.6.0:

- En una tabla, un `|` dentro de código ya no parte la celda.
- El panel puede vaciar las notas de una persona y el resumen de un área.

## 0.6.0-alpha

Publicada el 2026-11-03.

El panel crece y pasa a ser el sitio donde verlo todo de un vistazo:

- **Hoy** es su primera página: todos los espacios a la vez, con lo que vence en la
  próxima semana, las tareas que te esperan, el buzón, si toca la revisión semanal y las
  sesiones cortadas. Debajo, **Repositorios**: los que tus tareas abiertas nombran en
  `repos:` y tienen cambios sin commit o commits sin subir, marcados «¿olvidado?» cuando
  todas las tareas que los nombran están aparcadas o para algún día. Solo lectura; git
  se lanza sin coger bloqueos y nunca pide contraseña.
- **Las notas se leen como texto**, no como Markdown en crudo: títulos, listas, tablas,
  código y enlaces, y los `[[enlaces]]` llevan a su nota. Una tarea enseña primero lo
  esencial, el resto plegado, y su diario con lo más nuevo arriba. Un renderizador
  propio, sin dependencias: una nota nunca puede meter HTML ni scripts en la página.
- **Tareas por área**, con filtros (te esperan, en marcha, aparcadas, sin tocar, por
  persona) y una marca en las que llevan dos semanas sin tocar.
- **Personas** se pueden añadir y cambiar en el panel. La página de una persona enseña
  las tareas que la esperan y las que enlazan con ella; «espera a» lleva a la persona.
- **Buscar** en todas las notas del espacio y en las compartidas, con o sin acentos.
- **La revisión semanal** en una página: cada captura se convierte en tarea, referencia
  o lección, o se descarta; cada tarea sin tocar sigue o pasa a algún día. Lo decidido
  se guarda con la revisión. El buzón tiene un botón «Convertir en tarea».
- La página de una nota avisa cuando la nota cambió en otro sitio desde que se abrió.
  Las áreas se pueden renombrar y cambiar su resumen.
- **Secretos**, solo por nombre: los secretos que guardas en el llavero del sistema,
  para qué es cada uno, qué tareas lo nombran y, si lo pides, si el llavero de este
  ordenador lo tiene. Nunca un valor: si llega uno, se rechaza. Se guardan en
  `config.toml` como `secrets`; Claude añade uno con la nueva herramienta MCP
  `secret_name_save`; la lista viaja en la copia de seguridad del espacio por defecto, y
  una restauración la lleva a una máquina que no tiene ninguna.

Además:

- `import` nombra el conocimiento que lista personas (una sección cada una), para que
  cada una se pueda guardar como persona: en la primera migración real, las personas se
  quedaron como una referencia y la página Personas del panel salía vacía.

Del ensayo de la 0.5.10 en la máquina real:

- Las partes del inicio de sesión cuentan solo las que tienen algo: «parte 3 de 6», nunca
  «de 9» con tres que no llegan, lo que hacía pensar al asistente que se habían perdido
  algunas.
- `doctor` avisa cuando Claude Code guarda una memoria propia para una carpeta de trabajo
  (su memoria automática, una segunda memoria junto a la del espacio), y dice cómo
  quitarla.
- La nota de la restauración sobre los repositorios dice que algunos solo preguntaban
  antes de hacer commit.

## 0.5.10-alpha

Publicada el 2026-11-03.

Del ensayo de la 0.5.9:

- Una restauración deja las notas legibles solo por ti, sea cual sea la umask; `check`
  hace lo mismo con las notas escritas antes de la 0.5.9, y `doctor` avisa cuando otros
  usuarios pueden leerlas.
- Una restauración nombra los repositorios en los que la copia de seguridad dejaba hacer
  commit a Claude, que no recupera, y dónde volver a añadirlos; `doctor` y la primera
  sesión también lo dicen.

## 0.5.9-alpha

Publicada el 2026-11-03.

De una revisión a fondo del código (arquitectura, el guardia y el resto de la seguridad):

- **El guardia pilla más errores.** Una redirección pegada a una palabra (`git
  push>log`, `2>/dev/null sudo …`), `powershell -Command` o `cmd /c` desde Bash,
  `git.exe`, un script corto lanzado con `bash x.sh` o `./x.sh`, `ssh root@…`, `rm -rf
  "$DIR/"*` con una variable que puede estar vacía, `rm -rf ~/..`, `find ~ -delete`, `git
  submodule foreach`, `git subtree push`, llamadas a `gh api` que fusionan, `ssh -J`,
  `bash -o pipefail -c …`, y sudoedit, runuser, systemd-run y machinectl. Un comando
  anidado demasiado hondo pregunta en vez de pasar, y lo mismo un programa dado como
  variable. Más formas de token cuentan como secretos (Stripe, npm, Hugging Face, Docker
  Hub, SendGrid, más de GitLab).
- **Claude no puede apagar el guardia por su cuenta.** Cambiar la configuración de
  augmented brain o los ajustes de Claude Code te pregunta. `safety_save` solo añade a
  las listas del guardia, y `backup_setup` nunca mueve a otro sitio una copia de
  seguridad que ya existe.
- **Una copia de seguridad cambiada no puede salir de las notas.** Una copia que
  contiene un enlace simbólico ni se restaura ni se fusiona; una restauración ya no trae
  los repositorios en los que Claude puede hacer commit; git se lanza solo con los
  protocolos de siempre, y una dirección nunca es una opción.
- Las notas se escriben legibles solo por ti (0600, en carpetas 0700), y cada fichero
  que se reemplaza pasa por un fichero temporal con un nombre que nadie puede adivinar.
- install.sh no lanza nada si su descarga se corta; en Windows, el instalador de Git
  necesita su checksum y la firma del propio Git for Windows.
- El inicio de sesión: un plugin más nuevo que el programa recibe una parte vacía en vez
  de un error; la parte 1 espera a las notas como mucho 3 segundos, y las demás la
  esperan a ella; `check` coge el bloqueo de las notas; se nombra una excepción de root
  que no se puede leer; los recordatorios de hoy y de los próximos días conservan su
  sitio entre muchos atrasados.

## 0.5.8-alpha

Publicada el 2026-11-03.

Del ensayo de la 0.5.7:

- **Las fechas para revisar de una tarea cerrada siguen saliendo.** Al cerrar es cuando
  se apuntan las comprobaciones para más adelante, así que se mencionan al empezar una
  sesión hasta que se marcan como hechas, como las de una tarea viva. La tarea en sí ya
  no queda pendiente.
- «1 día de retraso», no «1 días».

## 0.5.7-alpha

Publicada el 2026-11-03.

Del ensayo de la 0.5.6:

- **Tus reglas llegan enteras.** En sesiones reales el asistente nunca abría el fichero
  de una regla antes de actuar, así que se perdían sus detalles (un `glab api` sin su
  `--hostname`). El inicio tiene ahora cinco partes para tus reglas, nueve partes en
  total, y cada regla va entera, sin su sección `History`. Solo una regla que no cabe va
  en una línea.
- **Nada de secretos por las herramientas.** Una tarea, una entrada del diario, una
  captura, el perfil o cualquier nota que escribe el asistente se rechaza si lleva un
  secreto (una clave de AWS, un token), para que nunca acabe en una nota que luego la
  copia de seguridad rechazaría. Una nota que ya tenía uno, como puede dejar una
  importación, se puede seguir cambiando, para arreglarla.
- **Dos máquinas, la misma tarea.** Cuando cada máquina solo añadió entradas al diario
  (líneas con fecha) de la misma nota, la copia de seguridad las junta sola, por orden de
  fecha. Cualquier otra cosa la sigue parando, con un mensaje en tu idioma.
- El inicio de sesión menciona también lo que vence en los próximos 7 días, después de
  lo atrasado.
- Una tarea se mantiene al día: `task_new` lleva el objetivo y el «hecho cuando» a la
  Definición, y aparcar o cerrar actualiza «dónde está» y «siguiente» en Lo esencial.
- `doctor`: una excepción de root ya vencida dice que la quites o le pongas otra fecha.
- Windows: borrar el fichero del panel espera a quien lo esté leyendo.

## 0.5.6-alpha

Publicada el 2026-11-03.

- **Excepciones de root temporales.** `root_ask = ["db01 until 2026-10-31"]` en la
  configuración: hasta esa fecha, un comando como root en esa máquina (aquí, o por
  `ssh`) te pregunta en vez de rechazarse, y el inicio de sesión nombra las excepciones
  vigentes. La fecha es obligatoria; pasada, root vuelve a rechazarse solo, y `doctor`
  dice que la entrada se puede quitar.
- La guía de instalación explica la pregunta de Claude Code sobre si te fías de una
  carpeta, y los acentos rotos en Windows PowerShell 5.1 (el texto está bien; es
  PowerShell el que lo lee mal).

## 0.5.5-alpha

Publicada el 2026-11-03.

Del ensayo de la 0.5.4:

- **El inicio de sesión cabe con el índice de un espacio grande.** Claude Code corta el
  texto de cada hook a los 10.000 caracteres, así que un espacio con 43 tareas empezaba
  sin ninguna en la lista. El plugin carga ahora el inicio en cinco partes, cada una
  dentro de ese límite: la cabecera con tu perfil y las reglas comunes, el índice en tres
  partes, y tus reglas.
- Un segundo `import` ya no vuelve a contar las capturas del buzón: si el buzón ya las
  tiene todas, el plan lista `inbox.md` con los ficheros que el espacio ya tiene.
- Una copia de seguridad parada por un secreto lo dice en tu idioma, con el tipo de
  secreto.
- El motivo del guardia para `rm -rf /c/` (o `/`) habla de un disco entero, no de la
  carpeta de inicio.

## 0.5.4-alpha

Publicada el 2026-11-03.

Del ensayo en Windows 11:

- **El guardia lee las rutas como las escribe Git Bash** (`/c/Users/ana`), que es donde
  Claude lanza los comandos en Windows. `rm -rf /c/Users/<tú>`, `rm -rf "$USERPROFILE"` y
  `rm -rf /c/` pasaban; ahora no se lanzan nunca. Y `git -C /c/…` de un repositorio de
  `git_allowed_repos` cuenta como ese repositorio.
- **El instalador de Windows consigue Git aunque winget esté roto** (un App Installer
  viejo cuyo origen no abre): descarga Git for Windows de su propia versión publicada,
  comprueba su checksum y su firma, y lo instala.
- Un segundo `import` ya no vuelve a copiar el archivo y `docs/`: los ficheros que el
  espacio ya tiene salen como tales, y el mensaje sobre ficheros que aparecieron después
  del plan sale en tu idioma.

## 0.5.3-alpha

Publicada el 2026-11-02.

Del segundo ensayo de una migración real:

- **Una máquina restaurada vuelve a ser la misma.** El espacio que era el de por defecto
  vuelve a serlo (antes, restaurar dos espacios no dejaba ninguno, y entonces ninguna
  copia de seguridad guardaba el perfil). Las listas del guardia siguen sin ir en ninguna
  copia, porque pueden nombrar máquinas del trabajo: `restore` y `doctor` ahora lo dicen,
  hasta que se las vuelvas a dar.
- **El instalador refresca las listas de paquetes antes de instalar git** (`apt-get
  update`): con listas viejas, apt fallaba y la instalación se paraba.
- **`import` vuelve a poner cada fichero donde está su nota.** Un segundo `--apply`
  mandaba el archivo y los documentos de notas importadas antes al espacio de `*`. Un
  secreto en un documento o en un fichero del archivo también se nombra en el informe, y
  `check` mira todos los ficheros que se lleva la copia de seguridad, no solo las notas.
- **El guardia sigue a `cd`.** `cd ~ && rm -rf .` no se lanza nunca, y `git -C` con una
  ruta relativa cuenta desde donde fue el `cd`. `rm -rf *` en una carpeta ya no se toma
  por el disco entero.
- **El guardia mira dentro de `vagrant ssh -c`, `docker exec`, `podman exec` y `kubectl
  exec`**, como hace dentro de `ssh`: hacerse root ahí es cosa tuya. Y una clave de API
  de Elasticsearch pegada sola, sin `ApiKey` delante, se pilla como secreto.
- **Las opciones van en cualquier sitio** (`import --map m carpeta --apply`), y un
  argumento de más es un error en vez de ignorarse.

## 0.5.2-alpha

Publicada el 2026-11-02.

Del primer ensayo de una migración real:

- **Se pilla una clave secreta que acaba en `/` o `+`.** El guardia dejaba pasar una
  clave secreta de AWS cuando su último carácter no era una letra ni un dígito (y una
  clave de Google acabada en `-`). Las escrituras y los comandos que llevan una se
  bloquean ahora como el resto.
- **Los repositorios en los que dejas hacer commit a Claude cuentan desde todos los
  espacios**, como los perfiles de producción: un repositorio listado en `personal` pasa
  también desde una sesión de `work`. Un repositorio que está en las dos listas pregunta.
- **Un push que reescribe el remoto pregunta, siempre** (`--force`, `--delete`,
  `--mirror`, `+rama`), incluso en un repositorio donde Claude puede subir. **`rm -r` de
  toda tu carpeta de inicio o del disco entero no se lanza nunca.**
- **Tus reglas llegan a todas las sesiones.** Cuando no caben enteras al inicio, cada una
  va con su título y su resumen, no solo con la ruta de su fichero.
- **`import` trae el resto:** las capturas de `inbox.md` van al buzón, y `docs/<tarea>/`
  se copia tal cual. Una nota con un secreto escrito se nombra en el informe, y `check`
  también la nombra.
- `docs/` en la carpeta de notas nunca se lee como notas (un documento sin cabecera salía
  como nota rota). El índice de las notas compartidas lista sus reglas y su conocimiento,
  no áreas ni tareas, que no puede tener. `init` avisa cuando el plugin no se instaló
  (sin git, por ejemplo), en vez de decir solo que está listo. `augmented-brain panel`
  se abre en el espacio de la carpeta en la que se lanza.

## 0.5.1-alpha

Publicada el 2026-11-02.

- **El panel enseña tus reglas y las notas compartidas.** Una nueva sección Reglas lista
  las reglas del espacio y las compartidas; Conocimiento lista también las referencias y
  lecciones compartidas. Cada nota compartida va marcada como tal.

## 0.5.0-alpha

Publicada el 2026-11-02.

- **Notas compartidas por todos los espacios.** Lo que aprendes que vale en todas partes
  (cómo se porta git, cómo te gustan las respuestas) se guarda una vez, en `shared/` junto
  a tu perfil, y todos los espacios lo ven: sus reglas se cargan al empezar cada sesión, y
  sus referencias y lecciones salen en la lista para que Claude las abra. Dile a Claude
  «esto vale para todo» y guarda ahí la nota. Solo se pueden compartir reglas,
  referencias y lecciones. Viajan en la copia de seguridad del espacio por defecto, como
  el perfil, e `import` puede mandar notas ahí (`lesson-x = shared` en el mapa).
- **Claude puede guardar una regla que le das** (`rule_new`), en el espacio o
  compartida, en vez de que escribas tú el fichero a mano.
- **Tus repositorios se quedan limpios.** Una nueva regla común: un repositorio solo
  recibe lo que necesita su equipo, por una rama. Los planes, análisis e informes de una
  tarea van a `docs/` en la carpeta de notas, junto a la tarea, y nunca se cargan al
  empezar.
- **Trae la memoria que Claude Code ya guarda.** `augmented-brain import <carpeta>` lee
  una memoria escrita por Claude Code (o con su misma forma) y enseña dónde iría cada
  nota en tus espacios, y lo que no ha podido decidir: ficheros que no son notas, notas
  sobre ti, enlaces que cruzarían espacios, ids ya cogidos. No se escribe nada hasta que
  lo vuelves a lanzar con `--apply`, y la carpeta de la que importas solo se lee. Con
  varios espacios, un fichero pequeño dice a qué espacio va cada área, qué notas van a
  otro sitio que no es su área y cuáles van a las notas compartidas (`--map`).
- **Al guardia se le escapa menos.** `terraform`, `tofu`, `pulumi`, `eksctl` y `copilot`
  se comprueban contra los perfiles de producción como `aws`, también el heredado. En un
  `find` con varios `-exec`, uno bloqueado ya no queda tapado por uno anterior que solo
  pregunta. `git -C a -C b` se lee como lo lee git, y un commit que nombra otro
  repositorio (`--git-dir`, `--work-tree`, `GIT_DIR`) nunca cuenta como uno permitido.
- **Un cambio en una tarea se guarda entero o no se guarda.** Actualizar una tarea con
  una fecha para revisar mal puesta guardaba el resto del cambio y luego fallaba.
- **La ayuda y los errores de los comandos hablan tu idioma**, como ya hacía el resto, y
  también el aviso del panel sobre ficheros que se quedan fuera de una copia de
  seguridad.
- Arreglos más pequeños: la configuración conserva su versión de formato al reescribirse;
  `import` se niega a escribir en notas de una versión más nueva e imprime un comando que
  se puede pegar; cada fichero pequeño que reescribe el programa (también el buzón) se
  reemplaza de una vez; un `AB_LANGUAGE` desconocido ya no para el instalador;
  `uninstall` ya no toma cualquier error que diga «not» por «ya no está».

## 0.4.0-alpha

Publicada el 2026-11-01.

- **Las notas dicen qué formato siguen.** Las notas de cada espacio llevan ahora su
  versión de formato (`.augmented-brain/format`), y también la carpeta de configuración
  (en `config.toml`, a partir de la versión 2). Cuando una versión futura escriba las
  notas de otra manera, la primera sesión después de actualizar las pone al día, tras
  una copia completa en `migrations/` de la carpeta de configuración, y Claude te lo
  dice; `augmented-brain check` hace lo mismo. Un programa más viejo que las notas se
  niega a abrirlas en vez de romperlas, y pide que lo actualices. Hoy no cambia nada en
  las notas: esto es el formato 1.

- **Commit y push donde tú digas, o después de preguntar.** Además de
  `git_allowed_repos`, un espacio puede listar repositorios en `git_ask_repos`: ahí, el
  `git -C <repo> commit` o `push` de Claude te pregunta cada vez en vez de bloquearse. En
  las dos listas, un worktree de git de un repositorio listado cuenta como él, y la
  última parte de una ruta puede llevar un `*` para una carpeta (`~/code/app-*`).

- **Más tipos de secretos.** El guardia, la comprobación de la copia de seguridad y los
  resúmenes conocen ahora también las claves de Google AI Studio, las claves personales
  de PostHog, las claves de API de Elasticsearch escritas como `ApiKey …` y los tokens de
  bot de Telegram de cualquier forma.

## 0.3.3-alpha

Publicada el 2026-11-01.

- **Repositorios que ya no están.** El `repos:` de una nota nombra carpetas dentro de la
  carpeta de trabajo del espacio. Cuando una ya no está (se movió, se renombró o se
  borró), `augmented-brain check` y la página Salud del panel ahora lo dicen, la página
  de la tarea en el panel lo enseña, y al leer la tarea se le dice a Claude que te
  pregunte dónde está ahora. No se cambia nada solo. Las tareas cerradas no se
  comprueban.

## 0.3.2-alpha

Publicada el 2026-11-01.

- **Sesiones en el panel.** Una nueva sección lista tus sesiones de Claude de cada
  espacio en este ordenador. Las que no terminaron enseñan lo que se dijo después de la
  última entrada del diario, y puedes guardarlo en las notas (en `archive/sessions/`, sin
  secretos, con una línea en el diario de la tarea) o marcarlo como revisado, sin abrir
  Claude. Las que están en marcha y las que terminaron enseñan cuánto disco ocupan sus
  transcripciones, y la transcripción de una sesión terminada se puede borrar, después
  de que confirmes que ya no se podrá retomar.

## 0.3.1-alpha

Publicada el 2026-11-01.

- **Si el guardia falla, bloquea.** Si `augmented-brain` no se puede lanzar cuando Claude
  va a lanzar un comando o cambiar un fichero (el programa no está en el PATH, se cae o
  tarda más de 10 segundos), Claude Code ahora para esa acción en vez de dejarla pasar
  sin comprobar. Usa el `onFailure: "block"` de Claude Code, desde la 2.1.295; las
  versiones anteriores lo ignoran y se portan como antes. Solo el guardia hace esto: los
  hooks que cargan el espacio, recuerdan, guardan el resumen y hacen la copia de
  seguridad nunca bloquean tu trabajo. `augmented-brain doctor` dice ahora cuál de los
  dos problemas ha encontrado: el guardia no se puede lanzar (todo se para) o se lanza
  pero deja pasar un comando arriesgado. `augmented-brain uninstall` conserva el
  programa cuando el plugin no se ha podido quitar del todo, para que Claude Code nunca
  se quede con un guardia que no puede lanzar.
- **Los comandos que se quedan en marcha también se vigilan.** La herramienta Monitor de
  Claude Code lanza un comando en segundo plano para seguir un log o una compilación;
  el guardia comprueba ahora ese comando igual que comprueba Bash. Las llamadas a
  herramientas de los subagentes y los workflows ya pasaban por el guardia.
- La comprobación de punta a punta del recordatorio de Stop ya no falla cuando el
  asistente añade una línea después de su respuesta, como «Nothing new happened on the
  task».

## 0.3.0-alpha

Publicada el 2026-11-01.

- **Subtareas y pasos.** Una parte de una tarea que espera a otra persona o va a su
  ritmo puede ser una subtarea: una tarea propia con `parent:` en su cabecera
  (`task_new` con `parent`). El índice, `pending` y el panel la listan bajo su madre;
  solo un nivel, y la madre no se puede cerrar mientras haya una subtarea abierta. Las
  partes más pequeñas son pasos: casillas (`- [ ] texto`) en el Plan de la tarea, que se
  añaden y se marcan con la nueva herramienta `task_step` o desde el panel, y se cuentan
  en el índice («2/5 pasos»). Cada cambio de un paso escribe una línea corta en el
  diario.
- **El panel**: `augmented-brain panel`, o `/brain:panel` en Claude, abre una página en
  tu navegador para ver y editar tus espacios sin el asistente: tareas (estado, a quién
  esperan, siguiente paso, diario, aparcar, cerrar, nueva), fechas para revisar y
  recordatorios, el buzón, áreas, conocimiento, personas y tu perfil, y la salud de cada
  espacio (notas no válidas, índice, última copia de seguridad, revisión semanal). Solo
  funciona en este ordenador, solo mientras lo usas (se para solo tras 30 minutos sin
  uso, o con `augmented-brain panel stop`), y solo para quien tenga la dirección que
  imprime: la dirección lleva una clave hecha para cada arranque, y no se pone ninguna
  cookie. No carga nada de internet. Cada cambio pasa por el mismo código y los mismos
  bloqueos que las herramientas de Claude, así que nunca se pisan: guardar una tarea
  guarda solo los campos que cambiaste, y un paso solo se marca si sigue diciendo lo que
  enseñaba la página. Lanzar el comando varias veces a la vez arranca un solo panel.

## 0.2.9-alpha

Publicada el 2026-11-01.

- Los perfiles de AWS de producción y las máquinas intocables son ahora para toda la
  máquina. Antes eran del espacio en el que se hizo la entrevista de la primera sesión (a
  menudo `personal`, en la app de escritorio), así que las sesiones de trabajo no
  estaban vigiladas. `safety_save` los escribe arriba del todo en la configuración; las
  listas que sigan dentro de un espacio siguen funcionando.
- El guardia comprueba el perfil de AWS que de verdad usa un comando `aws` cuando no da
  ninguno (`AWS_PROFILE`, `AWS_DEFAULT_PROFILE` o `default`); `cdk`, `sam`, `serverless`
  y `aws-vault exec` con un perfil de producción también preguntan; cuando podrían valer
  varios perfiles, pregunta si alguno es de producción. `gh pr merge`, `gh release
  create`, `glab mr merge` (y `accept`), `gh repo sync <repo>` y similares se bloquean
  como `git push`. Se comprueba lo que una tubería le pasa a un shell (`echo 'sudo …' |
  bash`, un heredoc a `bash` o a `ssh`). Los nombres de máquina coinciden en mayúsculas o
  minúsculas, y cortos o completos en cualquier sentido; `sftp host` y `mosh` quedan
  cubiertos.
- Cuando no se puede leer la configuración, el guardia pregunta antes de cada `aws`,
  `ssh`, `scp`, `rsync`, `sftp`, `mosh`, y `cdk`, `sam` o `serverless` con un perfil, en
  vez de dejarlos pasar sin comprobar. `augmented-brain doctor` pone a prueba el guardia
  de verdad y avisa si no bloquea, y enseña las listas del guardia.
- Una nota que no es válida ya no bloquea todas las herramientas: las demás siguen
  funcionando, el fichero roto nunca se sobrescribe, y al asistente se le dice qué
  fichero es y cómo arreglarlo. Una nota guardada con marca de orden de bytes (BOM, la
  ponen algunos editores de Windows) es válida.
- La copia de seguridad: el perfil viaja solo en la copia del espacio por defecto, nunca
  en un repositorio del trabajo (se quita una copia que haya dejado una versión
  anterior). `space add` pone el espacio por defecto cuando no había ninguno, para que
  siempre haya una copia que guarde el perfil, y `doctor` avisa cuando hay varios
  espacios y ninguno por defecto. Las listas del guardia no van en ninguna copia; después
  de una restauración, la siguiente sesión se ofrece a pedírtelas otra vez. Las capturas
  del buzón de dos máquinas se fusionan en vez de parar la copia. Los restos de una caída
  y los ficheros del sistema se quedan fuera (`.gitignore`), y también cualquier fichero
  de más de 50 MB, con un aviso en la siguiente sesión. Una copia nunca se hace a la vez
  que una escritura del asistente u otra copia. Las claves de ejemplo de la
  documentación de AWS ya no la paran.
- Una restauración que falla a medias deja como estaba el espacio que iba a reemplazar.
- Sin conexión, las sesiones ya no esperan cada vez a la comprobación de versión nueva.
- Las personas cuyos nombres no tienen letras latinas ya no se pisan unas a otras; a una
  persona se la encuentra por su nombre o por otro de sus nombres.
- `augmented-brain check` (y `space`, `backup`) fuera de la carpeta de cualquier espacio
  usan el espacio por defecto, como hace Claude. Las carpetas coinciden en mayúsculas o
  minúsculas en macOS y Windows.
- El trabajo se apunta mientras pasa, así que una sesión que acaba mal pierde poco. El
  método dice ahora que cada decisión o paso terminado vaya al diario en el momento. Un
  nuevo hook Stop se lo recuerda a Claude, una vez, cuando hay una tarea abierta y no se
  ha escrito nada en 20 minutos. Cuando Claude Code compacta una conversación, su
  resumen se guarda en `archive/sessions/` (sin secretos), el diario de la tarea apunta a
  él, y se le pide a Claude que apunte lo decidido. Una sesión que no terminó bien se
  ofrece para recuperar en el siguiente inicio: la nueva herramienta `session_recover`
  lee, de la transcripción de Claude Code en esta máquina, lo que se dijo después de la
  última entrada del diario de la tarea.

## 0.2.8-alpha

Publicada el 2026-10-19.

- `augmented-brain uninstall` no deja nada del plugin: también quita lo que Claude Code
  guarda de él después de desinstalarlo (su copia en caché, su carpeta de datos y los
  logs de su servidor MCP), que una desinstalación real dejaba en la máquina.
- `augmented-brain uninstall --all` empieza de cero: también se van la configuración, el
  perfil y las notas de todos los espacios, después de enseñar qué (con cuántas notas
  tiene cada carpeta) y preguntar. La carpeta de un espacio solo se va cuando no queda
  nada del usuario en ella. Las copias de seguridad en GitHub no se borran: dice cuáles y
  cómo. La comprobación de la instalación en Linux y macOS acaba con él y se asegura de
  que no queda nada.

## 0.2.7-alpha

Publicada el 2026-10-19.

- Instalar la app de escritorio ya no se queda callado ni se equivoca. Antes de pedir la
  contraseña (Linux), el instalador dice que no se verá mientras la escribes; la pide al
  principio y luego enseña un punto cada pocos segundos mientras se instala la app. Si
  ha funcionado se juzga por que la app esté ahí, no por el código de salida: en una
  máquina real, apt falló por unos paquetes del kernel a medio instalar después de
  instalar bien la app, y el instalador dijo que la app no se había instalado. Si algo
  falla, solo se enseñan las últimas líneas de la salida, con el fichero que la tiene
  entera. Lo mismo en macOS y en Windows (donde avisa de que Windows puede pedir permiso
  en una ventana), y al instalar git con sudo.
- Cuando ya hay una app de Claude abierta, el instalador lo dice (ciérrala y vuelve a
  abrirla, para que cargue augmented brain) en vez de hacer como que abre una: una
  segunda app de Claude solo le pasa el testigo a la que ya está en marcha y se cierra,
  así que no aparecía nada.

## 0.2.6-alpha

Publicada el 2026-10-19.

- El instalador, en Linux, macOS y Windows, comprueba un Claude Code que ya esté
  instalado: si es más viejo que la versión estable de Anthropic lo dice y se ofrece a
  actualizarlo con `claude update` (sí por defecto). A uno viejo le puede faltar lo que
  usa el plugin, como el hook de fin de sesión que lanza la copia de seguridad. Si la
  actualización falla, dice cómo hacerla más tarde y sigue. Las comprobaciones de la
  instalación cubren un Claude Code viejo y uno actual.

## 0.2.5-alpha

Publicada el 2026-10-18.

- Al final de la instalación, el mensaje sobre la app de Claude dice qué hacer: abrir la
  pestaña Code y pulsar + New, que empieza en el espacio personal; para el espacio de
  trabajo, elegir su carpeta con Choose…. Antes mandaba a elegir la carpeta de trabajo.

## 0.2.4-alpha

Publicada el 2026-10-18. La revisión semanal y los recordatorios con fecha, comprobados
de punta a punta con un Claude Code real (30 de 30).

- La revisión semanal, que el buzón siempre prometió: cuando la última tiene una semana
  y hay algo que repasar, el inicio de sesión la ofrece. La nueva herramienta
  `weekly_review` lista el buzón, las tareas que llevan dos semanas sin tocar y las
  fechas que vencen esta semana; `review_done` guarda lo decidido en `reviews/` y vacía
  del buzón las capturas procesadas.
- Recordatorios: «recuérdamelo el jueves» es una captura con fecha (`capture` acepta
  `remind_on`), y `task_update` puede ahora añadir a una tarea una fecha para revisar o
  marcarla como hecha. Lo que vence hoy o mañana, o va con retraso, se dice al empezar la
  sesión.
- El instalador en Linux y macOS dice, antes de instalar Claude Code, que necesita un
  plan de pago, como ya hacía el de Windows, con un enlace a los planes, y pregunta si
  tienes uno. Un «no» no lo para: Claude te pide que inicies sesión la primera vez que lo
  abres. Sin terminal, o con `AB_YES=1`, solo informa.

## 0.2.3-alpha

- En un móvil, el menú de la web enseña todos los enlaces, que bajan de línea bajo el
  nombre en vez de esconder las secciones, y los comandos del guardia bajan de línea en
  vez de cortarse.
- Una guía, «Instalación», que dice lo que hace el comando de instalación en Linux, macOS
  y Windows, dónde pone las cosas, qué cambia en cada uno y cómo actualizar y
  desinstalar. El README, la web y «Primeros pasos» dan también el comando de Windows, y
  ya no dicen que Windows no está soportado. install.sh, lanzado desde Git Bash en
  Windows, remite a install.ps1.
- El instalador en un Mac: avisa al principio cuando macOS es anterior a la 13 (Ventura),
  que Claude Code necesita, en vez de instalar lo que no arrancaría. Sin las herramientas
  de desarrollo de Apple ya no toma el /usr/bin/git de pega por git; abre su instalador
  y, con terminal, lo espera y sigue, sin tener que lanzarlo otra vez. Con bash, la línea
  del PATH va a ~/.bash_profile (o al .bash_login o .profile que ya haya), que es el que
  lee el Terminal del Mac, no a ~/.bashrc.
- `scripts/install-check-mac.sh` comprueba esas tres cosas en cada fila de Mac de CI, y
  CI tiene ahora también un Mac Intel (macos-15-intel).
- El guardia bloquea los comandos de la herramienta PowerShell de Claude Code, la
  principal en Windows, y le pide al asistente que use Bash (Git Bash) en su lugar. El
  guardia solo lee shell POSIX: en Windows, un `git push` por PowerShell pasaba sin
  comprobar.
- En Windows, desinstalar quita también el programa, y la consola enseña bien los
  acentos.
- Un instalador para Windows, `install.ps1`, y archivos de Windows en cada versión. En
  Windows, `update` lo lanza, e init dice `cd …; claude`.
- El idioma se detecta también en Windows (su idioma de pantalla), y `AB_LANGUAGE` manda
  sobre el locale.
- En Windows funciona ya el bloqueo que mantiene separados a dos escritores, y la copia
  de seguridad al final de una sesión se hace sola.
- El plugin arranca el programa directamente, con `~/.local/bin` añadido al PATH que
  recibe, en vez de a través de `sh`, que Windows no tiene: ahí el servidor no
  arrancaba.
- Una guía de «Primeros pasos» para quien nunca ha abierto una terminal: qué necesitas,
  cómo abrir la terminal, las preguntas del instalador, iniciar sesión, la entrevista,
  frases para la primera semana, las palabras que vas a oír y qué hacer cuando algo
  falla.
- La web, el README y la guía dicen desde el principio que Claude Code necesita un plan
  de Claude de pago (Pro o superior). «¿Es gratis?» solo decía que augmented brain lo es.
- Las guías tienen un botón Copiar en cada bloque de código.
- La página de inicio reproduce una conversación recreada y corta, dos días de trabajo en
  cuarenta segundos: retomar una tarea, aparcarla, recogerla en una sesión nueva y el
  guardia preguntando antes de tocar producción. Tiene un botón de pausa, y se ve quieta
  con movimiento reducido o sin JavaScript. El aviso de alfa va centrado.

## 0.2.2-alpha

Publicada el 2026-10-16. Más fácil de tener al día, de copiar y de quitar. Comprobada con
el instalador en macOS y Ubuntu, y de punta a punta con un Claude Code real (26 de 26).

- `augmented-brain update`: actualiza el programa, con el instalador publicado, y luego
  el plugin de Claude Code, que hasta ahora se quedaba en la versión que se instaló la
  primera vez. Lanzar otra vez el instalador también actualiza el plugin.
- La copia de seguridad se hace sola: cuando termina una sesión de Claude, un hook nuevo
  hace la copia de su espacio en segundo plano, si tiene una configurada. Ahí git nunca
  espera una contraseña; si falla, la siguiente sesión te lo dice, con el motivo de git.
  Las copias se hacen de una en una. `augmented-brain backup --space NOMBRE` hace la
  copia de un espacio desde cualquier sitio. `AB_NO_AUTO_BACKUP=1` la apaga.
- Configura la copia de seguridad pidiéndoselo a Claude: la nueva herramienta
  `backup_setup` crea un repositorio privado `augmented-brain-<espacio>` en la cuenta de
  GitHub del usuario con `gh`, deja que git use el inicio de sesión de gh y hace la
  primera copia; sin gh, devuelve los pasos para que Claude guíe al usuario. La
  entrevista de la primera sesión termina ofreciéndola.
- `restore` funciona en una máquina recién instalada: reemplaza un espacio que sigue tal
  como lo dejó el instalador (y un perfil en blanco) en vez de negarse; un espacio con
  notas del usuario nunca se reemplaza. La guía explica la restauración paso a paso.
- `augmented-brain uninstall`: quita el plugin de Claude Code y el programa, después de
  preguntar (`--yes` no pregunta). Las notas se quedan siempre, y la configuración
  también salvo con `--config`; Claude Code, su app, git y la línea del PATH (Claude Code
  vive en la misma carpeta) no se tocan. Instalar otra vez lo encuentra todo como estaba:
  el instalador ahora instala el plugin cuando falta en vez de intentar actualizarlo.
- Como mucho una vez al día, la primera sesión te avisa cuando hay una versión más
  nueva, y Claude se ofrece a hacer la actualización por ti. `AB_NO_UPDATE_CHECK=1` lo
  apaga.

## 0.2.1-alpha

Publicada el 2026-10-16. La probó una persona en un Mac real (macOS 26, Apple M1, sin
Homebrew): un comando instaló Claude Code, la app de escritorio y augmented brain, y la
primera sesión empezó en el espacio personal con la entrevista, en español.

- En un Mac sin Homebrew, lo normal, el instalador instala ahora también la app de
  escritorio de Claude: la última versión del canal de actualizaciones de Anthropic (el
  que lee Homebrew), con su firma comprobada, en /Applications (o ~/Applications).
- La comprobación de la instalación en GitHub se hace también en un Mac sin Homebrew.

## 0.2.0-alpha

Publicada el 2026-10-15. **Alfa:** un primer arranque que puede seguir alguien que
empieza, encontrado probándolo en un escritorio Ubuntu limpio. Lo probó una persona en
Ubuntu 24.04; en macOS, la instalación se prueba sin intervención en las máquinas de
GitHub, y la parte visual (la app de escritorio, iniciar sesión) aún tiene que probarla
una persona.

- El instalador pregunta primero el idioma (el del sistema por defecto; `AB_LANGUAGE`
  lo responde), y todo lo que viene después lo sigue: sus mensajes, los nombres de los
  espacios, el programa y la entrevista de la primera sesión. Luego la entrevista sigue
  el idioma en el que escribe el usuario.
- El instalador comprueba primero git y Claude Code y, si falta alguno, da el comando
  exacto para instalarlo. Añade `~/.local/bin` al PATH en el fichero de arranque del
  shell, lanza `init` y acaba con un único paso siguiente.
- `init` no pregunta nada: el idioma sale del sistema, y crea dos espacios, `work` y
  `personal` (`trabajo` y `personal` en español), cada uno en
  `~/augmented-brain/<espacio>` con sus notas dentro. `init --custom` pregunta, como
  antes.
- Una carpeta de trabajo que no existe se crea en vez de rechazarse.
- El instalador se ofrece a instalar lo que falta (git con el gestor de paquetes del
  sistema, Claude Code con su instalador oficial y la app de escritorio), enseñando solo
  una línea mientras trabaja y un ✓ al acabar (su propia salida solo si fallan), y, al
  final, a abrir Claude en el espacio de trabajo, con el PATH bien puesto: un comando de
  la nada a la primera sesión.
- Una primera sesión guiada: mientras el perfil está en blanco, Claude hace seis
  preguntas rápidas, diciendo por qué pregunta cada una, y con las respuestas guarda el
  perfil (`profile_save`), crea áreas, configura el guardia (`safety_save`: perfiles de
  AWS de producción y máquinas que no se tocan) y recuerda a las personas
  (`person_save`).
- Un espacio por defecto, `default_space` (`init` lo pone en `personal`): una sesión
  fuera de todas las carpetas de trabajo, como la carpeta temporal donde la app de
  escritorio de Claude empieza una sesión nueva, lo usa, y Claude lo dice en una línea.
  Sin él, Claude dice enseguida que no hay memoria cargada y qué carpetas abrir.
- Personas: un nuevo tipo de nota en `people/`, listado en el índice con quiénes son y
  los nombres con los que las llama el usuario, para que «Marcos» lleve a la persona
  correcta.
- `doctor` falla cuando faltan git, Claude Code o el plugin, con el comando para
  arreglarlo, y enseña lo que solo es información (un perfil en blanco, una carpeta fuera
  de tus espacios) con `·` en vez de un aviso.
- `restore` vuelve a poner las notas en el mismo sitio dentro de la carpeta de trabajo,
  instala el plugin cuando falta y acaba con el paso siguiente.
- El idioma se lee de `LC_ALL`, `LC_MESSAGES` y `LANG`, en ese orden.

## 0.1.0-alpha

Publicada el 2026-10-15. **Alfa:** la primera versión que se puede usar, probada solo en
Linux. Puede tener fallos y su formato puede cambiar; lee
[DISCLAIMER.md](DISCLAIMER.md).

- Formato de la memoria v0.1 (`docs/spec/memory-format.md`): una carpeta por tipo, un
  perfil para la persona y otro opcional por espacio; importar otros formatos está
  previsto para la 0.2.
- Núcleo en Go: `internal/config` (espacios, elegidos por la carpeta en la que trabajas)
  e `internal/memory` (lee y comprueba la cabecera de cada nota; el cuerpo puede ir en
  cualquier idioma).
- `augmented-brain version`, `space` y `check`.
- `augmented-brain init` y `space add`: crean la configuración, el perfil y los espacios
  a partir de plantillas en inglés o español. No se sobrescribe nada, y no se tocan los
  ajustes propios del asistente.
- `augmented-brain doctor`: dice lo que falta y cómo arreglarlo.
- El índice de la sesión, `MEMORY.md`, se genera: las tareas vivas agrupadas por área.
- Servidor MCP (`augmented-brain mcp`), para cualquier asistente que hable MCP: abrir,
  actualizar, apuntar en el diario, aparcar y cerrar tareas; ver lo pendiente; capturar
  al buzón; crear áreas, referencias y lecciones. Las llamadas van de una en una, también
  entre sesiones.
- Plugin de Claude Code `brain` (con su marketplace en este repositorio): carga el
  perfil, las reglas y el índice del espacio al empezar una sesión, vigila los comandos y
  las escrituras de ficheros (nada de commit ni push, nada de root, nada de secretos; los
  perfiles de AWS de producción y las máquinas intocables preguntan antes) y registra el
  servidor MCP. Comprobado de punta a punta con un Claude Code limpio: abrir una tarea,
  apuntar una decisión, aparcarla, retomarla en una sesión nueva, cerrarla.
- `init` se ofrece a instalar el plugin de Claude Code; `doctor` lo comprueba.
- Copia de seguridad y restauración: `augmented-brain backup setup <url>`, `backup` (o
  `backup --all`) y `restore <url>`. Cada espacio va a su propio repositorio privado con
  su configuración y el perfil; una copia se rechaza mientras una nota tenga un secreto.
- Versiones publicadas: `install.sh` (Linux y macOS, sin root, con el checksum
  comprobado) y archivos para amd64 y arm64, compilados y publicados por GitHub Actions
  con una etiqueta de versión.
- Una revisión del código antes de esta versión: el guardia entiende los comandos
  compuestos y lo que leen los shells; aparcar nunca pierde un diario; editar la
  configuración conserva los comentarios; las copias de seguridad se fusionan entre dos
  máquinas. `scripts/e2e-claude.sh` comprueba toda la historia con un Claude real.
- La web: una página de producto en inglés y español, con un aviso legal y este aviso de
  alfa.

## 0.0.1

Primera instantánea pública: el objetivo, el diseño y la marca del proyecto. Aún no hay
nada que instalar.

- README en inglés y español.
- Borrador del formato de la memoria (`docs/spec/memory-format.md`).
- Marca: logo, símbolo, iconos, cabecera del README, tarjeta para redes y paleta, con los
  scripts que los generan (`assets/brand/`) y una hoja que los enseña en su sitio
  (`docs/brand.html`). `logo-dark` y `symbol-dark` añaden un borde claro para fondos
  oscuros. La cabecera del README es el logo sobre una tarjeta blanca redondeada, para
  que se lea en los temas claro y oscuro de GitHub.
- `scripts/privacy-check.sh` y un hook de pre-commit que bloquean datos personales y
  secretos.
- Licencia MIT.
