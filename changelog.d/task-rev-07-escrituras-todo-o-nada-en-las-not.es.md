- **Cerrar la revisión semanal ya no puede perder capturas.** `review_done` sacaba del
  buzón las capturas procesadas antes de guardar la revisión, así que una revisión que
  luego rechazaba (un resumen vacío, un token citado en él) las dejaba perdidas sin nada
  guardado. Ahora primero se guarda la revisión, con las capturas que saca escritas en
  ella, y solo después salen del buzón. Otras escrituras también son todo o nada: un
  cambio rechazado ya no deja una copia en `archive/`, aparcar una tarea cerrada o tocar
  sus pasos se rechaza, una captura tras una última línea sin salto de línea queda en una
  línea propia, y un nombre de secreto escrito a mano con comillas ya no vacía la lista.
