# Trae una memoria que ya tienes

[Read in English](import.md)

Si Claude Code ya te guarda una memoria, no empiezas de cero. Claude Code la escribe en
`~/.claude/projects/<carpeta>/memory/`. Puedes traerla a tus espacios.

## Primero, mira

```bash
augmented-brain import ~/.claude/projects/<carpeta>/memory
```

Eso no escribe nada. Enseña adónde iría cada nota, y lo que no ha podido decidir:

- ficheros que no son notas (un índice, por ejemplo);
- notas sobre ti: copia lo que importe a tu perfil;
- enlaces que cruzarían de un espacio a otro;
- ids que tu espacio ya tiene.

La carpeta de la que importas solo se lee, ahora y después.

## Luego, escribe

Cuando la lista esté bien, vuelve a lanzarlo con `--apply`:

```bash
augmented-brain import ~/.claude/projects/<carpeta>/memory --apply
```

Nunca escribe encima de una nota que ya existe.

## En qué se convierte cada fichero

Cada fichero pasa a ser una nota de su tipo: por su nombre (`task-…`, `area-…`, `ref-…`,
`lesson-…`, `rule-…`, `person-…`) o, si no, por lo que Claude Code dice que es. Una tarea
sin estado entra aparcada, esperándote, con «revisar esta tarea importada» como siguiente
paso.

Lo demás también viene:

- Las capturas de tu `inbox.md` antiguo (cada línea `- …`) van al buzón del espacio de
  `*` (más abajo), con su fecha y nunca dos veces.
- `archive/` y `docs/<tarea>/` se copian tal cual, junto a su nota, también si vuelves a
  importar más tarde.
- Un fichero con un secreto escrito (una clave, un token) se importa, pero sale en el
  informe. La copia de seguridad no lo sube hasta que se quite el secreto.
  `augmented-brain check` también lo nombra.
- Una referencia o lección que es una lista de personas (una sección `## Nombre` por cada
  una, con una palabra como "personas" o "equipo" en su id o su título) entra tal cual, y
  el informe la nombra. Las personas son notas propias, así que pide a Claude que guarde
  cada una como persona.

## Más de un espacio

Un fichero pequeño dice a qué espacio va cada área, qué notas van a otro sitio que su
área, y cuáles van a las notas compartidas:

```text
area-familia = personal
task-curso-de-tarde = trabajo
lesson-git-commits-pequenos = shared
* = trabajo
```

`*` es para todo lo demás. Pasa el fichero con `--map`:

```bash
augmented-brain import ~/.claude/projects/<carpeta>/memory --map mapa.txt
```

Si las notas nombran repositorios en `repos:`, `--repos-base <carpeta>` dice respecto a
qué carpeta están esas rutas. Entonces se reescriben respecto a la carpeta de trabajo del
espacio.

## Apaga la memoria antigua

¿Claude Code llevaba esa memoria por su cuenta? Entonces la sigue llevando para las
carpetas de tus espacios: una segunda memoria, junto a la del espacio, que se va
separando de ella. `augmented-brain doctor` te avisa, y te dice cómo apagarla:
`"autoMemoryEnabled": false` en los ajustes de Claude Code. Lo que ya tiene se queda,
listo para importar.

Todas las opciones: [Comandos](commands.es.md#augmented-brain-import).
