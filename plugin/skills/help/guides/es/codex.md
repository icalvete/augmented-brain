# Seguir en Codex


Cuando se acaba el uso de Claude a mitad de una tarea, puedes seguir con la misma tarea en
[Codex](https://developers.openai.com/codex), el asistente de programación de OpenAI.
Codex empieza con tu espacio, como Claude: tu perfil, tus reglas y tus tareas. Puede leer
la tarea, escribir en su diario y hacerla avanzar. Las notas son los mismos ficheros, así
que cuando vuelve Claude, encuentra la tarea como la dejó Codex.

Es experimental. Se ha probado con Codex CLI 0.170.0.

## Prepararlo una vez

Necesitas Codex instalado y funcionando. Después, en un terminal:

```bash
augmented-brain codex install
```

Añade augmented brain a Codex como un plugin, con tres cosas:

- **El arranque:** cada sesión nueva de Codex recibe tu espacio, como una de Claude.
- **El guardia:** los comandos y los cambios de ficheros de Codex pasan por el
  [guardia](guard.md#en-codex-experimental).
- **Las herramientas de tareas:** where_am_i, pending y las de tareas (task_get,
  task_journal, task_update…). Las demás se quedan en Claude.

Codex solo ejecuta un hook en el que confías. Lanzar `codex install` es dar esa
confianza: confía en los hooks de augmented brain por ti, y lo dice. También apaga un
ajuste de Codex, `unified_exec_tty`, que deja al asistente escribir en un terminal donde
el guardia no ve. No cambia nada más tuyo en Codex.

Puedes volver a lanzarlo cuando quieras. `augmented-brain update` lo lanza por ti cuando
actualiza el programa.

## Seguir con una tarea

1. En Claude, aparca la tarea, o pide a Claude que apunte en su diario por dónde va. Si
   el uso se acabó antes, no se pierde nada: lo que hizo Claude está en la tarea.
2. Abre Codex en la carpeta de la tarea, o en una carpeta de su espacio.
3. Dile a Codex qué tarea: «sigue con la tarea del blog».

Codex lee la tarea, ve su siguiente paso y sigue desde ahí. Mantén la tarea al día como
con Claude: a Codex se le pide que lo haga.

Cuando vuelva Claude, abre una sesión nueva de Claude. Lee la tarea como la dejó Codex.

## Lo que cambia en Codex

- **El guardia nunca pregunta.** Codex no puede enseñarte una pregunta del guardia. Así
  que lo que en Claude te preguntaría, en Codex lo rechaza. El asistente te dice entonces
  el comando, para que lo lances tú o desde Claude.
- **Menos herramientas.** Ni buzón, ni conocimiento, ni reglas, ni cerrar una tarea, ni
  otros espacios. Codex te dice que lo hagas desde Claude o el [panel](panel.md).
- **Un espacio por sesión.** Codex trabaja en el espacio de la carpeta donde lo abres. En
  una carpeta que no es de ningún espacio, augmented brain no hace nada.
- **Sin mensajes entre sesiones** ni notificaciones: Codex te lo dice en su chat.
- **Cada asistente tiene su conversación.** Solo se comparten las notas. Lo que importe,
  escríbelo en la tarea.

## Comprobarlo

```bash
augmented-brain codex status
```

Dice si augmented brain está instalado en Codex y, si algo falla, qué. `augmented-brain
doctor` dice lo mismo, cuando Codex está en el ordenador.

El problema habitual: **una versión nueva cambió los hooks** y Codex dejó de ejecutarlos,
porque solo confía en un hook tal como era. `augmented-brain update` lo arregla solo; si
actualizaste de otra forma, vuelve a lanzar `augmented-brain codex install`.

## Quitarlo

```bash
augmented-brain codex uninstall
```

Quita el plugin y la confianza en sus hooks. `unified_exec_tty` sigue apagado: solo
protege. Tus notas no cambian. `augmented-brain uninstall` también lo quita.

Mira también: [El guardia en Codex](guard.md#en-codex-experimental) y
[Comandos](commands.md#augmented-brain-codex).
