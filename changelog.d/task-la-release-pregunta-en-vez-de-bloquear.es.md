- **Lanzar un flujo de publicación te pregunta en vez de pararse.** Cuando un asistente
  lanza `gh workflow run release`, el guardia ahora te enseña el diálogo de permiso. Di
  que sí y la versión sale, sin ir a un terminal; di que no, o no contestes, y no se lanza
  nada. En el modo `bypassPermissions`, donde ese diálogo puede no salir, se sigue
  parando.
