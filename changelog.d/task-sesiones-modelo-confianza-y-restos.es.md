- **`session start` abre en una carpeta en la que Claude ya confía.** Una carpeta que no
  es un repositorio, dentro de una de confianza, se rechazaba, y el comando te pedía
  aceptar una pregunta que Claude no enseña ahí. Ahora decide como Claude: un
  repositorio tiene que ser de confianza él mismo; cualquier otra carpeta lo es si lo es
  una de arriba.
- **`session start --model` elige el modelo de la sesión**, como `opus` o `sonnet`.
- **Tras `update`, las sesiones que no pudo reiniciar reciben el comando correcto.** Antes
  decía que cerraras y abrieras Claude; ahora dice que lances `augmented-brain session
  restart --old` cuando hayan terminado de responder.
- **Cerrar la sesión de una tarea dice lo que pasa de verdad.** Sale de las listas de
  Claude Desktop y claude.ai, pero la app de Claude del móvil puede seguir enseñándola,
  desconectada, hasta que la archives allí.
- **Un worktree que se queda por ficheros que git ignora los nombra.** Un programa
  compilado, el bytecode y cachés como `.next` ya no lo retienen.
