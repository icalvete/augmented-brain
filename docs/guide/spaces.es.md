# Espacios

[Read in English](spaces.md)

Abres Claude en `~/augmented-brain/trabajo/web`. Ve tus tareas del trabajo, tus reglas del
trabajo y a la gente del trabajo. Lo abres en `~/augmented-brain/personal/blog`. Ahora ve
tus notas personales, y nada del trabajo.

Eso es un **espacio**: un juego completo de notas, reglas y copia de seguridad, separado
de los demás. El instalador crea dos, `trabajo` y `personal` (en inglés, `work` y
`personal`).

## La carpeta elige el espacio

Cada espacio tiene una o más **carpetas de trabajo**. Cuando abres Claude dentro de una de
ellas, estás en ese espacio. Las notas del espacio viven en una carpeta `notas/` dentro de
ella:

```
~/augmented-brain/
├── trabajo/       una carpeta de trabajo: abre Claude aquí para lo del trabajo
│   ├── notas/     las notas del trabajo
│   └── web/       un proyecto tuyo
└── personal/      y aquí para lo personal
    └── notas/
```

El fichero de configuración dice qué carpetas son de qué espacio:

```toml
# ~/.config/augmented-brain/config.toml
default_space = "personal"

[spaces.trabajo]
memory = "~/augmented-brain/trabajo/notas"
work_folders = ["~/augmented-brain/trabajo", "~/code"]

[spaces.personal]
memory = "~/augmented-brain/personal/notas"
work_folders = ["~/augmented-brain/personal"]
```

`memory` es donde están las notas. `work_folders` son las carpetas que eligen el espacio.

Para saber de qué espacio es una carpeta, lanza esto en ella:

```bash
augmented-brain space
```

## Fuera de todos los espacios

Una carpeta que no está en ninguna carpeta de trabajo está fuera de todos los espacios.
Lo que hace augmented brain ahí depende de una opción, en la página Guardia del panel:
**dónde actúa augmented brain**.

- **En todo el ordenador** (como viene). augmented brain actúa en todas las sesiones de
  Claude Code de este ordenador, sea cual sea la carpeta: su plugin está instalado para
  todo tu usuario. Fuera de tus espacios usa el **espacio por defecto**, `personal` salvo
  que lo cambies, y Claude te lo dice en una línea, con cómo pasar al de trabajo.
- **Solo en mis espacios.** Una sesión fuera de las carpetas de tus espacios no recibe
  nada de augmented brain: ni memoria, ni avisos, ni copia. Ahí el guardia solo protege
  sus propios ficheros, tu carpeta personal y lo que nombran tus listas. Lo demás lo
  siguen decidiendo los permisos de Claude Code, como siempre. Es para ti si ya usas
  Claude Code en otros proyectos y quieres que sigan como estaban.

Lo que cambia, al lado de una sesión en un espacio:

| | En un espacio | Fuera, en todo el ordenador | Fuera, solo en mis espacios |
|---|---|---|---|
| Perfil, reglas, tareas y el resto de las notas | Las del espacio | Las del espacio por defecto, y Claude te lo dice | Ninguna |
| Lo que Claude escribe en tus notas | Va al espacio | Va al espacio por defecto | Nada: ahí Claude no puede escribir notas |
| Tareas cambiadas por otra sesión, recordatorio del diario, una sesión que recuperar | Sí | Las del espacio por defecto | No |
| Commit y push | Según tus listas | Según tus listas | Sin vigilar, salvo en repositorios de tus listas |
| Root, secretos en las órdenes, PowerShell | Parado | Parado | Sin vigilar |
| Borrar tu carpeta personal | Parado | Parado | Parado |
| Perfiles de producción, máquinas y repositorios intocables | Pregunta, o parado | Pregunta, o parado | Pregunta, o parado |
| Los ficheros del guardia, el plugin y el programa | Protegidos | Protegidos | Protegidos |
| Copia de seguridad al cerrar la sesión | La del espacio | La del espacio por defecto | Ninguna |
| Avisos de la memoria de la ventana y del modelo | Sí | Sí | No |

Tres cosas que saber:

- **Dentro de un espacio no cambia nada**, elijas lo que elijas.
- **Una orden que entra en un espacio se mira entera.** Lanzada desde fuera,
  `cd ~/work/app && git commit` o `git -C ~/work/app commit` recibe todas las reglas, como
  en el espacio. Igual un clon de un repositorio de tus listas, que se reconoce por su
  dirección.
- **Ojo con la app de escritorio** si haces que augmented brain se quede en tus espacios.
  Empieza cada sesión nueva en una carpeta temporal suya, que no es de ningún espacio: esa
  sesión no tiene memoria hasta que eliges carpeta. Por eso «en todo el ordenador» es lo
  que viene.

Si Claude Code ya estaba en tu ordenador, el instalador te lo pregunta una vez, la
primera; si solo pulsas Intro, queda «en todo el ordenador». Después se cambia en la
página Guardia del panel; un cambio a mano en `guard.toml` cuenta solo cuando lo aceptas
allí ([`reach`](configuration.es.md#guardtoml)).

El espacio por defecto importa para una cosa más: su copia de seguridad lleva tu perfil,
las notas compartidas y los nombres de tus secretos (nunca sus valores). Se elige con
`default_space` en la configuración. Si tienes un solo espacio, es ese.

Para meter una carpeta en un espacio, añádela a los `work_folders` del espacio (lo
siguiente).

## ¿Tus proyectos viven en otro sitio?

Si tu código está en `~/code`, añade esa carpeta a `work_folders` del espacio que toque,
como en el ejemplo de arriba. Así, abrir Claude en `~/code/tienda` usa el espacio de
trabajo.

## Añade un espacio

Para un cliente, un proyecto aparte o un voluntariado:

```bash
augmented-brain space add cliente
```

Te pide la carpeta de trabajo y la carpeta de notas, y te propone
`~/augmented-brain/cliente` y `~/augmented-brain/cliente/notas`. Pulsa Enter para
aceptar. Si aún no había espacio por defecto, lo fija. Las opciones están en
[Comandos](commands.es.md#augmented-brain-space-add).

Después, dos cosas:

- **Dale una copia de seguridad.** Abre Claude en la carpeta de trabajo nueva y di
  "configura la copia de seguridad"; cada espacio tiene la suya. Mira
  [Copia de seguridad y restauración](backup.es.md).
- **Trabaja en él.** Abre Claude en su carpeta de trabajo, o en una carpeta dentro de
  ella, y la sesión usa el espacio nuevo.

## Qué tiene cada espacio, y qué comparten

Cada espacio tiene lo suyo:

- notas: tareas, áreas, conocimiento, personas, buzón, revisiones;
- reglas, además de las comunes;
- repositorio de copia de seguridad.

Todos los espacios comparten:

- **tu perfil**: quién eres y cómo te gusta trabajar. Eres la misma persona en el trabajo
  y en casa.
- **las notas compartidas**: reglas, referencias y lecciones que valen en todas partes
  (abajo).
- **las listas del guardia**: los perfiles de AWS de producción, las máquinas que no se
  tocan, los repositorios donde Claude puede hacer commit y los que no debe tocar nunca, y
  las excepciones de root. Valen en toda la máquina, las escribas desde el espacio que
  sea. Un perfil de producción guardado desde el espacio de trabajo también se vigila en
  el personal, y un repositorio donde Claude puede hacer commit vale desde todos los
  espacios.

Un espacio nunca lee las notas de otro por su cuenta.

## Una tarea de otro espacio

A veces decides, en un espacio, algo de una tarea de otro: en una sesión personal dejas
claro cómo sigue una tarea del trabajo. Díselo a Claude y leerá o escribirá esa tarea
donde vive, en el otro espacio. Puede leerla, listar las tareas vivas de ese espacio para
encontrarla, añadir a su diario, marcar un paso y cambiar su siguiente paso, su estado o
a quién espera.

- Lo que Claude escribe allí acaba en `(from personal)`, para que quien lo lea en el otro
  espacio sepa dónde se decidió.
- Nada se copia en las notas de este espacio: lo que Claude lee se queda en la
  conversación.
- Claude solo lo hace cuando se lo pides, igual que al
  [mandar una captura a otro espacio](notes.es.md#mandar-una-captura-a-otro-espacio).
- Esa tarea se cierra desde una sesión de su propio espacio, porque cerrarla también
  cierra la sesión de la tarea y su worktree.
- Ese espacio se copia cuando termina esta sesión, si tiene copia de seguridad: no espera
  a una sesión suya.

## Notas compartidas

Algunas cosas que aprendes valen en todas partes: cómo se porta git, cómo te gustan las
respuestas, una lección sobre una herramienta. Guardarlas una vez por espacio dejaría dos
copias que acaban siendo distintas. Por eso van una sola vez en las **notas
compartidas**, junto a tu perfil, en `shared/` dentro de la carpeta de configuración.

Solo se pueden compartir reglas, referencias y lecciones. Las tareas y las áreas son siempre
de un espacio. Una persona puede estar en varios, cada uno con su nota y solo lo que
quieres ver allí (mira Personas en Notas).

No mueves ficheros a mano. Dile a Claude "esto vale para todo" y guarda la nota como
compartida. Cada sesión carga las reglas compartidas, y lista las referencias y lecciones
compartidas para que Claude abra la que necesite. Viajan en la copia de seguridad del
espacio por defecto, con tu perfil.

## Por dónde seguir

- [Tareas](tasks.es.md): lo que va dentro de un espacio.
- [Configuración](configuration.es.md): todos los ajustes de un espacio.
