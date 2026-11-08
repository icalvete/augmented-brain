# Áreas, conocimiento y lo demás

[Read in English](notes.md)

Las tareas van y vienen. Alrededor de ellas, un espacio guarda otros tipos de notas. Cada
una es un fichero Markdown en su propia carpeta, y nunca tienes que crearlas a mano: lo
dices, y Claude lo escribe.

## Áreas

"El blog", "los servidores", "la casa". Un **área** es algo que cuidas sin fecha de fin.
Agrupa las tareas y el conocimiento que le tocan.

Dices: "el boletín es un área nueva". Claude crea `areas/area-boletin.md`. Su texto dice
qué cuidas, cómo va y qué vigilar. El programa ya lista dentro las tareas y el
conocimiento del área por su cuenta.

La entrevista de la primera sesión crea tus primeras áreas, con lo que dijiste que cuidas.

## Conocimiento

Lo que merece la pena guardar más allá de una tarea. Hay dos tipos, en `knowledge/`:

- Una **referencia** dice cómo funciona algo: "cómo se publica el blog".
- Una **lección** dice lo que aprendiste, con un título que ya dice la conclusión:
  "Prueba a restaurar la copia, no solo a hacerla".

Dices: "guárdalo como lección". Las dos son de un área, y Claude las abre cuando una tarea
las necesita.

## Personas

Dices: "Diego lleva la empresa del hosting; le llamo Dieguito". Claude guarda
`people/person-diego.md`. La próxima vez que digas "dile a Dieguito que ya está", sabe
quién es.

Una **persona** tiene un nombre, quién es para ti, los otros nombres que usas para ella y
lo que merezca la pena recordar. Una tarea puede esperar a una persona
(`waiting_on: Diego`).

**La misma persona puede estar en más de un espacio.** Digamos que Diego es tu cliente y
también tu amigo. Cada espacio guarda su propia nota de él, con el mismo id, y esa nota
tiene solo lo que quieres ver allí: su papel, sus otros nombres y sus detalles. El panel
las junta en una sola ficha, para que veas todo sobre él en un sitio, y en cada detalle
marcas los espacios donde se ve. Un detalle sobre tu amigo que dejes sin marcar en
`trabajo` no se escribe nunca en un fichero de trabajo, así que nunca llega a una copia
de seguridad de trabajo. Cuando le pides a Claude que recuerde a alguien, te pregunta en
qué espacios debe verse.

## Buzón

Una idea que todavía no es una tarea. Dices: "apunta: mirar los precios nuevos del
hosting". Claude añade una línea con fecha a `inbox.md`, sin abrir tarea:

```
- 2026-10-17 — Mirar los precios nuevos del hosting
- 2026-10-17 — Llamar al seguro (remind: 2026-10-22)
```

Cada línea es una **captura**. La [revisión semanal](weekly-review.es.md) vacía el buzón:
cada captura pasa a ser una tarea, va a un área o al conocimiento, o se descarta.

"Recuérdame el jueves que llame al seguro" es una captura con fecha. Desde una semana
antes, y cada día después, sale al empezar la sesión, hasta que la revisión semanal la
resuelve (o la marcas como hecha en el panel).

## Reglas

Cómo quieres que trabaje Claude. Dices: "a partir de ahora, contesta con frases cortas".
Claude lo guarda como **regla**. Cada sesión carga todas las reglas, enteras.

Hay tres niveles:

- **reglas comunes**, que vienen con el programa: comprobar antes de aconsejar, no subir
  nunca código sin que se pida, no hacerse nunca root, no escribir nunca secretos…
- **reglas compartidas**, tuyas, para todos los espacios ("contesta con frases cortas");
- **reglas del espacio**, en `rules/` del espacio, para lo que solo vale ahí ("la base de
  datos de la tienda no se reinicia nunca en viernes").

Una regla que vale en todas partes va a las notas compartidas. Dile a Claude "esto vale
para todo".

## Tu perfil

Quién eres y cómo te gusta trabajar: tu nombre, a qué te dedicas, tu idioma, lo largas
que te gustan las respuestas, cuánto puede hacer Claude sin preguntarte. Es un solo
fichero para todos los espacios, `profile.md` en la carpeta de configuración. Lo escribe
la entrevista de la primera sesión. Para cambiarlo, díselo a Claude.

## Repositorios

Qué es cada uno de tus repositorios git, una línea cada uno: "`~/code/blog` — El blog,
hecho con Hugo". Vive en una referencia del espacio, `knowledge/ref-repositories.md`, así
que la copia de seguridad lo guarda. Escribes la línea en la página Repositorios del
panel, o te la escribe Claude.

## El índice

`MEMORY.md` es la lista que Claude lee al empezar cada sesión: la línea de tu perfil, las
reglas, las áreas y las tareas vivas con su estado y su siguiente paso. Las referencias y
lecciones compartidas salen con su título y su fichero. Al empezar, Claude la recibe más
corta: las tareas van sin su resumen, y una tarea enseña su siguiente paso (los primeros
120 caracteres) solo si te espera a ti. Claude lee una tarea antes de trabajar en ella.
El programa escribe este fichero después de cada cambio: no lo edites a mano, se
reescribiría.

## Documentos y el archivo

Hay dos carpetas que no se cargan nunca:

- `docs/<tarea>/` guarda los documentos largos que produce una tarea (un plan, un
  análisis), junto a las notas y fuera del repositorio del proyecto.
- `archive/` guarda lo antiguo: diarios largos que se sacan de la tarea, resúmenes de
  conversaciones compactadas. Ahí no se borra nunca nada.

## Editar a mano

Cada nota es Markdown normal. Ábrela con cualquier editor y cámbiala. Después, lanza
`augmented-brain check` en una carpeta del espacio: comprueba todas las notas y rehace el
índice.

## Por dónde seguir

- [Qué decirle a Claude](phrases.es.md): las frases para cada tipo de nota.
- [Formato de la memoria](../spec/memory-format.md) (en inglés): todas las carpetas y los campos
  de la cabecera.
