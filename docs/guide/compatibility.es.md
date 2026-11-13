# Lo que no cambia

[Read in English](compatibility.md)

augmented brain numera sus versiones de la forma habitual: **1.x** son todas las
versiones que empiezan por 1. Desde la 1.0, nada de esta página se rompe dentro de las
1.x. Si algún día hace falta romper algo de aquí, será la **2.0**, y el registro de
cambios dirá qué cambia y cómo pasar.

Las versiones beta (0.x) ya siguen estas reglas, pero todavía no son una promesa.

## Lo que se promete

- **Tus notas.** El formato que describe el [formato de la memoria](../spec/memory-format.md):
  las carpetas, los tipos de nota, los campos de la cabecera y lo que significa cada uno.
  Si una 1.x escribe algo de otra forma, trae el paso que pone al día tus notas y lo da
  sola, después de una copia completa. Las notas de cualquier beta (0.x) se ponen al día
  igual.
- **Tu configuración.** `config.toml`, `secrets.toml`, `guard.toml`, `session.toml`,
  `remote.toml` y `profile.md`: cada ajuste de la
  [referencia de configuración](configuration.es.md) conserva su nombre y lo que hace.
- **Los comandos.** Cada comando y opción de la [referencia de comandos](commands.es.md)
  sigue existiendo y haciendo lo mismo. También los códigos de salida que da esa página
  (por ejemplo, `watch pr` acaba en 0 o 1).
- **Las herramientas del asistente.** Cada herramienta MCP de la
  [referencia de herramientas](mcp-tools.es.md) conserva su nombre, sus parámetros y lo
  que hace. Una versión puede añadir herramientas, o parámetros que se pueden omitir;
  nunca un parámetro nuevo que una llamada tenga que dar. Importa porque tus reglas y tus
  notas nombran estas herramientas.
- **Las variables de entorno** de [Configuración](configuration.es.md#variables-de-entorno)
  (`AB_CONFIG_DIR`, `AB_LANGUAGE`, `AB_NO_AUTO_BACKUP`…) y las del instalador.
- **Las copias de seguridad.** Una copia hecha con cualquier 1.x se restaura con
  cualquier 1.x posterior.
- **Instalar y actualizar.** La línea de instalación de la portada sigue funcionando, y
  `augmented-brain update` lleva de cualquier 1.x a la última.

**Un nombre que cambia sigue funcionando.** Si un comando, una opción, un ajuste, una
herramienta o un parámetro cambia de nombre, el viejo sigue valiendo hasta la 2.0 y dice
cuál lo sustituye. Nada de la lista desaparece dentro de las 1.x.

**Una versión anterior nunca estropea tus notas.** Con dos ordenadores, o con una sesión
que sigue abierta después de actualizar, puede haber un programa más viejo leyendo notas
que escribió uno más nuevo. Las dos versiones anteriores a la última las leen; si el
formato cambió, las rechazan y piden actualizar, sin escribir nada.

## Lo que no se promete

- **Las palabras.** Los mensajes del programa, lo que contestan las herramientas, lo que
  el asistente lee al empezar una sesión y la redacción de las reglas comunes cambian
  para explicarse mejor. No los leas desde un script; usa los códigos de salida.
- **El panel.** Sus páginas, su aspecto y sus direcciones.
- **Lo de dentro.** Las carpetas `state/`, `history/`, `migrations/` y `marketplace/` de
  la carpeta de configuración, los comandos ocultos (`panel serve`, `remote watch`), lo
  que escriben `hook` y `mcp` (los nombres se quedan: el plugin los llama), y el código
  Go (no es una biblioteca).
- **Lo que deja pasar el guardia.** Puede volverse más estricto en cualquier versión: un
  comando que hoy pasa puede preguntar mañana. Tus listas (`git_allowed_repos` y las
  demás) siguen valiendo.
- **Los sistemas y las versiones de otros programas.** Una 1.x puede necesitar un Claude
  Code, un Git o un sistema operativo más nuevos cuando los viejos dejen de funcionar; lo
  dice el registro de cambios.
- **Las variables de prueba** que no da la documentación.

## Una excepción: la seguridad

Si arreglar un problema de seguridad obliga a romper algo de la lista, se arregla en una
1.x, y el registro de cambios lo dice lo primero, con lo que tienes que hacer.

## Cómo se comprueba

Lo que se puede comprobar está congelado en ficheros dentro del repositorio, y las
pruebas fallan si una versión quita o cambia algo de ellos: las herramientas MCP con sus
parámetros y cuáles tiene que dar una llamada, los comandos con sus opciones y los
eventos de `hook`, y los ajustes de cada fichero de configuración. Añadir sí se puede.
Para las notas, cada cambio de formato trae su paso y una prueba que lo da sobre notas
escritas a la manera vieja, y en cada cambio las dos versiones anteriores tienen que leer
todo lo que escribe la nueva.
