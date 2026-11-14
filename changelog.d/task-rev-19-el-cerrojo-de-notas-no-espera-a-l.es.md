- **Una sesión que hace un worktree ya no deja paradas a las demás.** `task_worktree` y
  cerrar una tarea ejecutaban git, con red, con las notas bloqueadas: con la red lenta,
  todas las herramientas de las demás sesiones esperaban, una vez 17 segundos. Ahora git
  corre después de soltar las notas, y una herramienta que las encuentra ocupadas 20
  segundos lo dice en vez de quedarse colgada.
- **Una sesión arranca a tiempo aunque otra esté escribiendo.** El arranque de sesión
  esperaba a las notas hasta tres veces, 9 de los 10 segundos que tiene; ahora espera
  una sola vez, 2 segundos como mucho, y arranca con el perfil y las reglas igualmente.
