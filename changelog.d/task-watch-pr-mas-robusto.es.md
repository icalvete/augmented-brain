- **`watch pr` ya no da un resultado de antes ni se rinde cuando no toca.** Justo después
  de un push o de relanzar la CI espera 90 segundos antes de creerse un verde o un rojo,
  porque GitHub aún enseña el viejo un rato. Una pull request cuyos cambios no lanzan
  ningún check obligatorio acaba diciéndolo: bien si GitHub deja mezclarla, bloqueada si
  no. Y fuera de un repositorio git saca el repositorio de las carpetas de la tarea, o
  pide `--repo` en el acto.
