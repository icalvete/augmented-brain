# El panel

[Read in English](panel.md)

El **panel** es una página en tu navegador para ver y cambiar tus espacios sin pasar por
Claude. Ábrelo con `/brain:panel` en Claude, o desde una terminal:

```bash
augmented-brain panel
```

Abre tu navegador en **Hoy**, con el espacio de la carpeta desde la que lo lanzas. Desde
una terminal también escribe una dirección, por si el navegador no se abrió (en un
ordenador sin pantalla, por ejemplo). Esa dirección abre el panel **una vez**, en los
próximos 2 minutos; vuelve a lanzar el comando para tener otra. Cuando lo lanza Claude
por ti, solo abre el navegador: Claude nunca ve una dirección que abra el panel. Si el
navegador no puede abrirse (Claude trabajando en segundo plano, un ordenador sin
pantalla), tampoco escribe la dirección y te pide que lances el comando en una terminal
tuya.

El panel cambia tus notas igual que Claude, con el mismo programa, y se esperan el uno al
otro: puedes tener los dos abiertos a la vez.

Cada página empieza con unas palabras sobre qué es y para qué sirve, y un enlace a su
parte de esta guía. Cuando ya te manejas, **Ocultar esta ayuda** la oculta en esa página,
y **Ocultarla en todas las páginas**, en todas. El panel lo recuerda en este ordenador;
**¿Qué es esta página?** la trae de vuelta.

## Espacios y páginas

A un lado, bajo **Perfil**, la lista **Espacio** elige el espacio que estás mirando.
Unas páginas enseñan solo ese espacio; otras, todos, o este ordenador entero. En un
móvil, el menú se pliega bajo el nombre de la página en la que estás.

| Página | Para qué sirve | Enseña |
|---|---|---|
| [Hoy](#hoy) | Lo que vence y lo que te espera | todos los espacios |
| [Tareas](#tareas) | Ver, cambiar, abrir y cerrar tareas | un espacio |
| [Recordatorios](#recordatorios) | Fechas para revisar y recordatorios del buzón | un espacio |
| [Buzón](#buzón) | Ideas sueltas, para más tarde | un espacio |
| [Revisión semanal](#revisión-semanal) | Vaciar el buzón, mirar las tareas paradas | un espacio |
| [Áreas, conocimiento, reglas](#áreas-conocimiento-reglas-y-personas) | Leerlas; crear áreas | un espacio, más las notas compartidas |
| [Personas](#áreas-conocimiento-reglas-y-personas) | Quién es quién | todos los espacios |
| [Repositorios](#repositorios) | Trabajo que solo está en este ordenador | todos los espacios |
| [Secretos](#secretos) | Los nombres de tus secretos | la misma lista en todos los espacios |
| [Copia de seguridad](#copia-de-seguridad) | Copias de tus notas | todos los espacios |
| [Salud](#salud) | Si el espacio está en orden | un espacio |
| [Guardia](#guardia) | Lo que Claude puede hacer | este ordenador |
| [Versiones](#versiones) | Configuraciones anteriores | este ordenador |
| [Ayuda](#ayuda) | Respuestas, y avisar de un problema | — |
| [Perfil](#perfil) | Quién eres, tu asistente | todos los espacios |

## Hoy

La primera página: todos los espacios a la vez. De cada uno:

- lo que vence en los próximos 7 días;
- las tareas que te esperan (las aparcadas, en una línea);
- cuántas capturas esperan en el buzón;
- si toca la revisión semanal;
- las sesiones que se cortaron, cada una con **Guardarlo en las notas** y **Dar por
  revisada** (mira [Sesiones](#sesiones));
- las notas con problemas.

Debajo, **Repositorios**: los que tus tareas abiertas nombran en `repos:` y tienen
cambios sin commit o commits sin subir. Es trabajo que solo está en este ordenador. Si
todas sus tareas están aparcadas (o para algún día), sale marcado «¿olvidado?», aquí y
en el mapa completo; uno limpio, nunca. Una rama
que nunca se subió solo cuenta si el repositorio tiene un remoto (una copia en un
servidor, como GitHub) al que subirla. **Ver
todos los repositorios** lleva al mapa completo.

## Tareas

Agrupadas por área, primero las que te esperan a ti, con su estado y su siguiente paso.
Los filtros enseñan solo las que te esperan, las que están en marcha, las aparcadas, las
paradas, o las que esperan a una persona. Una tarea que nadie ha cambiado en 14 días sale
marcada «parada N días». Al final, plegadas, las de **Algún día** y las **Cerradas**.

Abre una. Primero lo esencial, las demás secciones plegadas, y el diario con lo último
arriba. Escribe en su diario, cambia su estado o su siguiente paso, marca sus pasos,
apárcala o ciérrala. También puedes abrir tareas nuevas. Si un estado se salta fases, el
diario las nombra. Si una tarea espera a una persona de la que tienes nota, «espera a» te
lleva a ella.

Cerrar una tarea aquí hace lo mismo que cerrarla en una sesión: si la tarea tiene una
sesión propia, esa sesión acaba cuando termina su respuesta, y su worktree se quita cuando
no tiene nada suyo.

Para reabrir una tarea cerrada, búscala en **Cerradas** y elígele un estado nuevo; su
diario dice que se reabrió desde el panel.

Al guardar una tarea, solo se guarda lo que cambiaste en la página. Si Claude cambió otra
cosa mientras tanto, se queda. Un paso solo se marca si sigue diciendo lo que mostraba la
página.

## Recordatorios

Las fechas para revisar de tus tareas (también de las cerradas) y los recordatorios del
buzón: primero lo atrasado, luego los próximos 30 días. Marca uno como hecho, o añade a una tarea una fecha para revisar. Un recordatorio que no es
de ninguna tarea es una captura con fecha: se crea en el [Buzón](#buzón).

## Buzón

Apunta una idea, con una fecha para que te la recuerde si quieres. Márcala como resuelta,
o conviértela en tarea.

## Revisión semanal

La [revisión semanal](weekly-review.es.md), en una página. Cada captura pasa a tarea, a
referencia o a lección, o se descarta. Cada tarea sin tocar dos semanas sigue como está, o
pasa a algún día (o la abres para cerrarla). Lo que decides se va apuntando, y se guarda
en el registro de la revisión al terminar.

## Áreas, conocimiento, reglas y personas

Conocimiento y reglas enseñan las del espacio y las compartidas, marcadas «compartida».
Aquí se leen; no se escriben. Una referencia o una lección la crea Claude, o sale de una
captura en la [revisión semanal](#revisión-semanal); una regla, Claude. Puedes crear áreas y cambiar el título y el resumen de un área.

Las personas salen una sola vez, con una etiqueta por cada espacio en el que están. Una
persona nueva empieza en el espacio en el que estás. La página de una persona es un solo
formulario: el nombre se queda (un nombre nuevo sería otra persona), luego un papel y
otros nombres por cada espacio, y una lista de detalles. Cada detalle tiene una casilla
por espacio; marca dónde debe verse. Vacía el texto de un detalle para quitarlo. Un
detalle sin ninguna casilla marcada no se guarda, y la página lo dice junto a él. Siempre
hay una fila vacía al final para añadir uno, y «Otro detalle» recarga la página con una
más (guarda antes: no conserva lo que escribiste). La página también enseña las tareas
que le esperan y las que la enlazan.

## Repositorios

Tu mapa de repositorios: todos los repositorios git de las carpetas de trabajo de cada
espacio, hasta tres carpetas por debajo. Se salta las carpetas que están dentro de otro
repositorio, las carpetas ocultas, `node_modules` y `vendor`. Salen agrupados por espacio,
primero los que tienen trabajo pendiente.

De cada uno: su rama, adónde sube (como `servidor/dueño/nombre`, nunca con contraseña),
los ficheros cambiados, los commits sin subir, las tareas abiertas que lo nombran, y una
línea tuya sobre qué es. Esa línea se escribe ahí mismo. Se guarda en las notas del
espacio, en la referencia «Repositorios» (`knowledge/ref-repositories.md`), así que la
copia de seguridad la guarda. Claude también puede escribirla.

El panel solo lee tus repositorios. Git funciona sin coger bloqueos, así que nunca te
estorba, y nunca pide una contraseña. Solo se ejecuta dentro de las carpetas de trabajo,
y los enlaces a carpetas de fuera no se siguen.

## Secretos

Los nombres de los secretos que guardas en el llavero del sistema (donde tu ordenador
guarda las contraseñas), nunca su valor. Cada uno como `servicio/nombre`, para qué sirve,
y qué tareas lo nombran. Así sabes cuáles hay, y una máquina nueva sabe cuáles pedirte.

**Comprobar en el llavero** dice si cada uno está en este ordenador. Se lo pregunta a
`secret-tool` en Linux y a `security` en macOS; en otros sistemas no lo puede saber. Solo
pregunta cuando pulsas, porque un llavero bloqueado te pide su contraseña. Si al llavero
le falta uno, la página enseña el comando que lo guarda (ese comando te pide el valor). Si no se
puede preguntar al llavero (en Linux sin sesión de escritorio, o con el llavero de macOS
bloqueado), el secreto sale «aquí no se puede saber», no como que falta: compruébalo
antes de guardarlo otra vez.

Puedes apuntar un nombre, o quitar uno de la lista (el secreto sigue en el llavero). Lo
que parezca un valor se rechaza. La lista es la misma en todos los espacios, y vive en
`secrets.toml` (mira [Configuración](configuration.es.md#secretstoml)).

## Copia de seguridad

Si las notas de cada espacio tienen una copia en otro sitio, en un repositorio privado,
para que sobrevivan si este ordenador se rompe. Un espacio sin copia lo dice en rojo.

- En un espacio con copia: cómo fue la última, adónde va, las últimas copias con la
  fecha y el ordenador que las hizo, y **Hacer una copia ahora**. La copia se hace por
  detrás y tarda unos segundos; recarga la página para ver cómo fue. También se hace
  sola cada vez que termina una sesión.
- Cada copia enseña **Qué cambió**, nota a nota, con las líneas. **Volver a esta copia**
  enseña antes qué cambiaría en tus notas de ahora, y pide una casilla y el nombre del
  espacio. Lo que tenías se guarda antes como copia, así que se puede deshacer. Mira
  [Volver a una copia](backup.es.md#volver-a-una-copia).
- En un espacio sin copia: **Configurarla en mi GitHub** crea un repositorio privado
  llamado `augmented-brain-<espacio>` en tu cuenta de GitHub y hace la primera copia.
  Necesita la línea de comandos de GitHub, `gh`, instalada y conectada; si no lo está, la
  página dice cómo. **Ya tengo un repositorio privado** toma su dirección en su lugar.
- **Recuperar las notas en otro ordenador** enseña el comando que hay que ejecutar allí.
  Recuperar nunca se hace desde el panel: es para un ordenador que aún no tiene las notas.

Llevar a otro sitio una copia que ya existe se hace en una terminal, con
`augmented-brain backup setup`. Desde otro dispositivo la página solo enseña. Mira
[Copia de seguridad](backup.es.md).

## Salud

La carpeta de notas y las carpetas de trabajo del espacio, si es el espacio por defecto,
notas que no son válidas, repositorios a los que apuntan las notas y que ya no están en
este ordenador, el índice, la última copia de seguridad, la última revisión semanal y un
enlace a las [Sesiones](#sesiones) del espacio.

## Sesiones

Tus sesiones de Claude en este ordenador, de los últimos 30 días. No está en el menú:
ábrela desde **Salud**, o desde una sesión cortada en **Hoy**.

- Las que no terminaron (cerradas a la fuerza, o cortadas al apagarse el ordenador)
  muestran lo dicho después de la última entrada del diario. Guárdalo en las notas (va a
  `archive/sessions/` con una línea en el diario de la tarea) o dalas por revisadas.
- Las que están en marcha y las terminadas, con lo que ocupan en disco sus
  transcripciones (los ficheros donde Claude Code guarda cada conversación). Puedes borrar
  la transcripción de una sesión terminada; esa sesión ya no se podrá retomar.

## Guardia

Todas las listas a las que responde el guardia, en una página, para leerlas y cambiarlas:

- **Repositorios intocables**, donde Claude nunca hace commit, push ni merge. Quitar uno
  avisa de que las demás listas vuelven a decidir sobre él (si una deja a Claude hacer
  commit y push ahí, lo hará), y te pide escribir el nombre de la carpeta.
- **Repositorios** donde cada espacio deja que Claude haga commit, o te pregunte, con un
  formulario para añadir o quitar. La lista donde Claude hace commit y push sin preguntar
  lleva un aviso, y añadir a ella te pide marcar que lo entiendes.
- **Excepciones de root**, con la fecha en que acaba cada una (las que ya acabaron se
  marcan). Añadir una avisa de lo que permite y te pide escribir otra vez el nombre de la
  máquina.
- **Perfiles de producción** y **máquinas intocables**.
- **El modelo de tus sesiones** (`preferred_model` en `session.toml`): si una sesión
  responde con otro, Claude te lo dice. Debajo, lo que Claude Code dijo de su modelo en el
  último inicio de sesión.
- **Lo que el asistente guarda solo** (`memory_trust` en `session.toml`): si Claude guarda
  una regla, tu perfil y lo que añade a las listas del guardia y se te avisa, o si te
  pregunta antes. Lo cuenta [el guardia](guard.es.md#preguntar-antes-de-que-claude-guarde-una-regla).
- Los ficheros que Claude no puede cambiar por su cuenta.

Cómo se comporta:

- Tus comentarios de `config.toml` se quedan donde están.
- Si las listas no son las que viste por última vez (algo las cambió fuera de esta
  página), un aviso dice lo añadido y lo quitado, lista por lista, frente a las listas
  que aceptaste por última vez. Si esas ya no están entre las versiones guardadas, lo dice.
- Después, un botón, **Las he visto, aceptar**. Hasta que lo pulses, o empieces una sesión
  nueva, un commit, push o merge que las listas permiten te pregunta antes.
- Lo que cambies o aceptes aquí te lo dice Claude Code en tus sesiones abiertas, después
  de la siguiente respuesta o mensaje, en una línea que empieza por «augmented brain · las
  listas del guardia». Si no has sido tú, mira aquí.
- Claude no puede abrir esta página: nunca ve la dirección del panel, y el guardia le
  niega la llave del panel.
- Desde otro dispositivo la página solo enseña. Las listas deciden lo que Claude puede
  hacer en este ordenador, así que solo se cambian desde el panel de este ordenador.

Cómo funciona cada lista lo cuenta [El guardia](guard.es.md).

## Versiones

Las 10 últimas versiones de tu configuración (espacios, guardia, acceso remoto,
ajustes de sesión, perfil y los nombres de tus secretos),
guardadas solo en este ordenador, la más reciente primero, con cuándo y quién la cambió.
**Qué cambió** enseña las líneas de cada una; **Volver a esta** la convierte otra vez en
la configuración, guardando antes como versión la que tenías. Pide marcar una casilla, y
avisa antes si cambiarían las listas del guardia, con qué. Nunca van en la copia de
seguridad: tus notas sí, con su propia historia. Desde otro dispositivo la
página solo enseña. Mira [Versiones de la configuración](config-history.es.md).

## Ayuda

Enlaces a las respuestas de esta web, en tu idioma. Y una forma de contarnos un problema o
proponer una idea en GitHub (la web donde vive el proyecto; necesitas una cuenta gratuita
allí).

Pulsa **Avisa de un problema en GitHub**. El panel hace la revisión de salud (lo que dice
`augmented-brain doctor`) en ese momento. Luego abre el formulario de GitHub en otra
pestaña, con la versión, tu sistema y esa revisión ya escritos.

Antes de abrirlo, el panel quita lo personal: tu carpeta personal (sale como `~`), tus
carpetas de notas y de trabajo, las direcciones de correo, los nombres de tus espacios,
las máquinas intocables, los perfiles de AWS de producción, y cualquier línea que parezca
una contraseña o una clave. Tú cuentas qué pasó y lees el resto: cambia o borra lo que
prefieras no compartir. Nada se envía hasta que pulsas el botón de enviar de GitHub. Para
ser claros: el texto viaja a GitHub dentro de la dirección de la página en cuanto se abre,
así que GitHub puede verlo entonces, aunque no lo envíes.

## Perfil

Arriba del todo a un lado, encima de la lista de espacios (en un móvil, la última del
menú). Quién eres y cómo te gusta trabajar: el mismo perfil para todos los espacios.

En **Tu asistente** puedes darle a Claude un nombre y una personalidad. Cada sesión
empieza con ellos. Cambian cómo habla, nunca las reglas, el guardia ni su honestidad.
Con nombre, cada sesión le pide a Claude que firme los correos que escribe en tu nombre
con ese nombre y, debajo, una línea como «asistente de Ana (IA, Claude)», para que quien
lo lea sepa que no lo escribió una persona. Esa línea la escribe el programa, no tú, así
que no se puede dejar fuera; quien la pone en el correo es Claude. La página enseña la
firma.
Deja los dos campos vacíos si no quieres ni nombre ni personalidad.

## Búsqueda y lectura de notas

El cuadro de **búsqueda** del lateral busca en las notas del espacio y en las
compartidas: títulos, resúmenes y texto. Sale una nota cuando tiene todas las palabras,
en mayúsculas o minúsculas, con tildes o sin ellas.

Las notas se leen como texto, no como Markdown en crudo (el formato de texto
en que se escriben las notas): títulos, listas, tablas, código y
enlaces. Un `[[enlace]]` a otra nota te lleva a ella. Si una nota cambia en el disco
mientras tienes abierta su página (Claude escribió en ella, por ejemplo), la página te lo
dice, con un enlace para verla como está ahora.

## Hecho para que siga siendo tuyo

- Funciona **solo mientras lo usas**. Tras 30 minutos sin uso se para solo. Para pararlo
  ya: `augmented-brain panel stop`, o el enlace del pie de la página.
- Funciona **solo en este ordenador**. Las otras máquinas de tu red no llegan a él.
  Para abrirlo desde el móvil u otro ordenador tuyo, usa `augmented-brain panel remote`:
  mira [El panel desde otro dispositivo](panel-remote.es.md).
- La dirección que recibes lo abre **una vez**. Después el panel lleva tu navegador a sus
  páginas, cuya dirección lleva una **llave**, nueva cada vez que el panel arranca. Sin
  ella el panel no responde nada, así que ningún otro programa ni página web de tu
  ordenador puede usarlo. Trata la dirección de las páginas como una contraseña: no la
  compartas. Deja de valer cuando el panel se para. Si otra cosa usó tu enlace antes, tu
  navegador dice «Este enlace ya se usó» en vez de abrirse: te enteras.
- En Linux el navegador recibe un pequeño fichero, que solo tú puedes leer, que lo lleva
  a la dirección, en vez de la dirección misma: con qué se arranca un programa lo puede
  leer cualquier usuario del ordenador. Si tu navegador por defecto es un snap o un
  flatpak (el Firefox que trae Ubuntu es un snap), no puede leer ese fichero, y recibe la
  dirección como antes; igual que los navegadores de macOS.
- No carga **nada de internet**: ni fuentes ni estadísticas. Funciona sin conexión. Una
  nota tampoco puede meter HTML ni scripts propios en la página. Los enlaces de la página
  Ayuda van a la web solo cuando los pulsas.
- Mirar si una nota ha cambiado no cuenta como usar el panel: una pestaña abierta no lo
  mantiene en marcha.
