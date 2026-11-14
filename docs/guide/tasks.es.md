# Tareas

[Read in English](tasks.md)

Dices "nueva tarea: mudar el blog a otro hosting". Claude abre una tarea. Acordáis qué
significa "hecho" y, después, un plan. Por la tarde dices "apárcala". La semana siguiente,
en una sesión nueva, dices "seguimos con el blog", y Claude la retoma donde la dejaste.

Una **tarea** es cualquier trabajo con principio y fin. Es un fichero de texto en la
carpeta `tasks/` del espacio.

## Cómo es una tarea

```markdown
---
id: task-mudar-el-blog
type: task
title: Mudar el blog
summary: El blog se muda a un hosting más barato
area: area-blog
status: parked
waiting_on: me
next: Copiar la base de datos al hosting nuevo
created: "2026-10-15"
updated: "2026-10-16"
---
## Lo esencial
El blog se muda a un hosting más barato. Plan acordado: copiar, probar, cambiar el dominio.
Hecho: el hosting nuevo está listo. Siguiente: copiar la base de datos.

## 3. Plan
- [x] Abrir una cuenta en el hosting nuevo
- [ ] Copiar la base de datos
- [ ] Cambiar el dominio

## Diario
- 2026-10-15 — Tarea abierta.
- 2026-10-16 — Vamos con el hosting más barato: misma velocidad, la mitad de precio.
- 2026-10-16 — Aparcada. Siguiente: Copiar la base de datos al hosting nuevo
```

La parte entre las líneas `---` es la **cabecera**: los datos que lee el programa. Debajo
va texto libre, en cualquier idioma. La primera sección es siempre **lo esencial**: unas
pocas líneas que bastan para retomar la tarea. La última es siempre el **diario**: líneas
con fecha de lo que ha pasado.

Puedes abrir y editar el fichero con cualquier editor. El formato completo está en el
[formato de la memoria](../spec/memory-format.md) (en inglés).

## Las seis fases

Una tarea pasa por seis fases. **Tú apruebas cada cambio de fase.** Dentro de una fase,
Claude trabaja sin preguntarte a cada paso.

1. **Definición**: qué queremos, qué queda fuera y cómo sabremos que ha funcionado.
2. **Análisis**: cómo están de verdad las cosas, medido, no supuesto.
3. **Plan**: pasos en orden, cada uno con cómo deshacerlo.
4. **Ejecución**: hacerlo, apuntando en el diario sobre la marcha.
5. **Validación**: demostrar el criterio de "hecho".
6. **Cerrada**: terminada. Se guarda, no se borra nunca.

El fichero de la tarea tiene una sección numerada por cada fase en la que hay algo que
apuntar (1. Definición, 2. Análisis, 3. Plan, 4. Validación, 5. Cierre). La ejecución no
tiene sección propia: lo que se hace va al diario. Por eso, de la validación en adelante,
los números de las secciones y los de las fases no coinciden.

La fase es el `status` de la tarea. Dos estados más pausan una tarea sin perder nada:

- **parked**: aparcada, para retomarla pronto. `next` dice exactamente cómo.
- **someday**: algún día, no para estas semanas. Sale de la lista que Claude lee al
  empezar, y se queda en su área.

Una tarea pequeña puede saltarse fases (de la definición directa a la ejecución): se puede,
y el diario dice qué fases se saltó. Una tarea cerrada sigue cerrada: volver a cerrarla se
rechaza, para que su resultado nunca se pise, y reabrirla pide un motivo: Claude te pregunta
antes, y el motivo va al diario. En el panel, elegir otro estado para una tarea cerrada
es ese motivo.

## A quién espera, y qué viene después

Cada tarea dice dos cosas más:

- **waiting_on**: `me` (tú), `nobody` (nadie), el nombre de una persona u otra tarea.
- **next**: la siguiente acción, una sola, en una línea.

Eso es lo que te enseña "¿en qué estábamos?": cada tarea viva, primero las que te esperan
a ti, con su siguiente paso.

## Aparcar y retomar

Antes de pasar a otro trabajo, di "apárcala". Claude apunta tres cosas: el siguiente paso
exacto, a quién espera y lo esencial, reescrito para que otra sesión pueda retomarla sin
leerse todo el diario.

Para retomarla, di "seguimos con el blog". Claude lee primero la tarea. Luego comprueba
el estado real de lo que toca la tarea (los ficheros, el servidor) antes de fiarse de la
nota. Las cosas pueden haber cambiado desde entonces.

## El diario, sobre la marcha

Cada decisión o paso terminado va al diario en el momento, no solo al aparcar. Una sesión
puede acabar en cualquier momento, y lo que no está en la nota se pierde.

Si la sesión tiene una tarea abierta y en 20 minutos no se ha escrito nada en ella, Claude
recibe un recordatorio al final de su respuesta, una vez.

Cada sesión lleva la cuenta de su propia tarea. Con dos ventanas en tareas distintas, a
cada una se le recuerda la suya.

Una conversación larga se puede compactar: Claude Code cambia los detalles por un resumen.
Ese resumen se guarda en `archive/sessions/`, y el diario apunta a él. El resumen puede
estar mal, así que a Claude se le piden tres cosas: que no lo tome como un hecho, que
compruebe antes de afirmar algo de antes y que te diga que la sesión se ha compactado.
También te propone aparcar la tarea y seguir en una sesión nueva, que empieza desde las
notas.

### Un diario largo

Una tarea que dura semanas puede juntar un diario largo, demasiado para que Claude lo lea de
una vez. Cuando una tarea pasa de 24 KB, sus entradas más antiguas se mueven a un fichero
propio en `archive/journals/` (con el nombre de la tarea y las fechas de la primera y la
última entrada), y una línea del diario dice cuántas se movieron y adónde. En la tarea se
quedan los últimos 8 KB de entradas, más o menos. No se pierde nada: las entradas se mueven
tal como se escribieron, y Claude lee ese fichero cuando necesita algo de entonces.

## Pasos y subtareas

Una tarea grande se puede partir de dos maneras.

Un **paso** es una parte pequeña: una casilla del Plan de la tarea, como `- [ ] Copiar la
base de datos`. Claude la marca cuando está hecha. La lista de tareas dice cuántos van
hechos ("1/3 pasos").

Una **subtarea** es una parte con vida propia, porque espera a otra persona o lleva su
propio ritmo. Por ejemplo, "Pedirle a Diego la contraseña del dominio" espera a Diego,
mientras tú sigues con lo demás. Una subtarea es una tarea entera, con una línea más en
su cabecera: `parent: task-mudar-el-blog`.

- Un solo nivel: una subtarea no tiene subtareas.
- La tarea grande se cierra la última, cuando todas sus subtareas están cerradas.
- Aparcar la tarea grande deja sus subtareas como están.

Regla práctica: si espera a alguien o lleva su propio ritmo, es una subtarea. Si no, es un
paso.

## Una sesión propia

Una tarea puede tener una sesión de Claude propia, abierta en segundo plano en la carpeta
de la tarea: dices "abre una sesión para la tarea del blog" y sigues con otra cosa. La
sesión lee la tarea y se pone con ella. En Linux lo hace `augmented-brain session start`;
mira [Una sesión por tarea](task-sessions.es.md).

## Fechas para volver a mirar algo

Algunas comprobaciones son de un día: "el 16, ¿se ha renovado el certificado?". Díselo a
Claude y añade una **fecha para revisar** a la tarea. Desde 7 días antes, y cada día
después, Claude te lo dice al empezar la sesión, hasta que se marque como hecha.

También vale en una tarea cerrada. Al cerrar es cuando muchas veces apuntas lo que hay que
comprobar después, así que cuando Claude cierra una tarea que aún tiene alguna, te avisa de
que seguirá saliendo.

Para algo que no es una tarea ("recuérdame el jueves que llame al seguro"), Claude usa el
buzón: mira [Áreas, conocimiento y lo demás](notes.es.md#buzón).

## Cerrar

Di "cierra la tarea". Claude apunta el resultado, y la tarea se queda en su carpeta,
marcada como cerrada. Nunca se borra nada.

- Una tarea con subtareas abiertas no se puede cerrar: ciérralas antes.
- Si la tarea tiene una sesión propia, cerrar la tarea también cierra esa sesión, cuando
  termina su respuesta.
- Si se hizo un worktree de git para la tarea (una segunda carpeta de trabajo del mismo
  repositorio), cerrar lo quita, pero solo si no guarda nada que no esté en otro sitio.
  Si no, se queda, y Claude te dice por qué.

Una tarea que nadie ha cambiado en 14 días sale en la
[revisión semanal](weekly-review.es.md), para que nada se pare en silencio.

## Por dónde seguir

- [Qué decirle a Claude](phrases.es.md): las frases para todo esto.
- [Herramientas MCP](mcp-tools.es.md): las herramientas que usa Claude detrás de cada
  frase.
