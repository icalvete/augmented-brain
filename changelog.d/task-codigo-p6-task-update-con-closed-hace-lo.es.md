- **Cerrar una tarea cambiando su estado ahora la cierra del todo.** `task_update` con
  estado `closed` solo cambiaba el estado: sin resultado, y la sesión y el worktree de la
  tarea se quedaban. Ahora hace lo mismo que `task_close`, con un `result` opcional nuevo;
  el botón Cerrar del panel también acaba la sesión de la tarea y quita su worktree.
