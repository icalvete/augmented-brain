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

La app de escritorio empieza cada sesión nueva en una carpeta temporal suya. Esa carpeta
no es de ningún espacio. Entonces se usa el **espacio por defecto**, `personal` salvo que
lo cambies, y Claude te lo dice en una línea, con cómo pasar al de trabajo.

El espacio por defecto importa para una cosa más: su copia de seguridad lleva tu perfil y
las notas compartidas. Se elige con `default_space` en la configuración. Si tienes un solo
espacio, es ese.

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
  tocan y los repositorios donde Claude puede hacer commit. Valen en toda la máquina. Un
  perfil de producción guardado desde el espacio de trabajo también se vigila en el
  personal.

Un espacio nunca lee las notas de otro.

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
