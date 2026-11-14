# Editar las notas a mano


Tus notas son ficheros Markdown normales, y son tuyas: abre cualquiera con cualquier
editor y cámbiala. Casi siempre es más fácil pedírselo a Claude («cambia el siguiente paso
de la tarea de la copia»): sus herramientas lo comprueban todo antes de escribir. A mano
está bien para una errata, una frase mejor escrita o una nota que quieres reordenar. Esta
página dice qué respetar para que el programa siga leyendo la nota.

## Antes y después

- **Antes**, mira si una sesión de Claude está trabajando en esa misma nota. Si escribe
  mientras editas, se pierde uno de los dos cambios. Una sesión abierta se entera de tu
  cambio antes de su siguiente respuesta.
- **Después**, lanza `augmented-brain check` en una carpeta del espacio. Revisa todas las
  notas y rehace el índice. Si nombra un problema, arréglalo: hasta entonces, esa nota se
  queda fuera, y Claude no la ve.

## Cómo es una nota

Una nota tiene una cabecera entre dos líneas `---`, y después el texto:

```markdown
---
id: task-web-nueva
type: task
title: Web nueva
summary: Pasar la web al hosting nuevo
area: area-casa
status: execution
waiting_on: me
next: Apuntar el dominio al servidor nuevo
created: 2026-11-02
updated: 2026-11-13
---
## Lo esencial
- …

## Diario
- 2026-11-02 — Tarea abierta.
- 2026-11-13 — Hosting elegido: el más barato con copias.
```

El texto debajo de la cabecera es tuyo: escribe ahí lo que quieras. La cabecera es lo que
lee el programa, así que tiene reglas.

## La cabecera

- **Las claves, siempre en inglés.** `title`, `status`, `next`… escribas en el idioma que
  escribas. También los valores fijos: `status: parked`, nunca `status: aparcada`.
- **Solo las claves que conoce el programa.** Una clave que no conoce deja la nota como no
  válida: el programa no distingue una errata de una clave de una versión más nueva, así
  que deja la nota fuera en vez de adivinar. No añadas claves tuyas; escríbelo en el
  texto.
- **Una línea por campo.** Un título o un siguiente paso escrito en varias líneas se lee
  como una sola.
- **Las fechas, `AAAA-MM-DD`**: `2026-11-03`, no `3-11-2026` ni `2026-11-3`.
- **`id` es el nombre del fichero** sin `.md`. Si cambias uno, cambia el otro, y cada nota
  que lo nombra en `related` o `parent`. Más fácil: pídele a Claude que lo renombre.
- **Los valores de `status`**: `definition`, `analysis`, `plan`, `execution`,
  `validation`, `closed`, `parked` o `someday`. `waiting_on` es `me`, `nobody`, el nombre
  de una persona o el id de otra tarea.

Cada clave y lo que significa están en el
[formato de la memoria](../en/memory-format.md#file-header) (en inglés).

## Las tareas

- **La primera sección es lo esencial, la última el diario.** El programa cuenta con eso:
  añade las entradas del diario al final del fichero, y lee lo esencial para retomar.
  Cambia las palabras de los títulos si quieres, pero no su orden.
- **Una entrada del diario es una línea**: un guion, la fecha, una raya larga `—` y el
  texto: `- 2026-11-13 — Lo que pasó`. Un guion corto `-` en lugar de la raya no se lee
  como entrada.
- **Los pasos son casillas** bajo el título del Plan: `- [ ]` por hacer, `- [x]` hecho.

## El buzón

`inbox.md` no tiene cabecera: una línea por captura, escrita como una entrada del diario,
`- 2026-11-13 — Llamar al seguro`. Para que salga un día, acábala con
`(remind: 2026-11-20)`.

## Lo que no se toca

- **`MEMORY.md`**, el índice: el programa lo escribe después de cada cambio, y lo tuyo se
  perdería.
- **El final `(from <espacio>)`** de una captura o de una entrada del diario: dice que la
  mandó otro espacio. No lo escribas tú.
- **La carpeta `.augmented-brain/`**: guarda la versión del formato de las notas.
- **Borrar una nota.** Nunca se borra nada: cierra una tarea (`status: closed`), y pídele a
  Claude que retire una regla, referencia o lección que ya no vale: pasa a `archive/`, que
  nunca se carga.

## Por dónde seguir

- [Tareas](tasks.md): las fases, los pasos y el diario.
- [Áreas, conocimiento y lo demás](notes.md): los otros tipos de nota.
- [Si algo falla](troubleshooting.md#una-nota-no-es-válida): cuando `check` dice que una
  nota no es válida.
