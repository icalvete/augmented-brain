- **Una tarea que ya existe puede hacerse subtarea, cambiar de madre o volver a ser una
  tarea propia.** `task_update` admite `parent`: el id de la nueva madre, o `none`.
  Comprueba lo mismo que con una subtarea nueva (un solo nivel, una madre que no esté
  cerrada), rechaza una tarea con subtareas propias, pasa al área de la madre salvo que se
  dé `area`, y apunta el cambio en el diario.
