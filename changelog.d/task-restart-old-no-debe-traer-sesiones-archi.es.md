- **`session restart --old`, y `update`, ya no traen de vuelta las sesiones que
  archivaste.** A Claude no se le dice que archivas la sesión de una tarea en Claude
  Desktop, así que reiniciarla tras actualizar la volvía a abrir, y reiniciar el acceso
  remoto abría sesiones nuevas en él. Ahora `--old` no reinicia ninguna: solo para su
  augmented brain anterior, y el nuevo arranca la próxima vez que la sesión lo use.
  Nombrar una sesión (`session restart <tarea>`) sigue reiniciándola.
