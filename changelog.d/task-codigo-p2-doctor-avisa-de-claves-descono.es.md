- **`doctor` y el inicio de sesión nombran los ajustes de `guard.toml` y `session.toml`
  que el programa no conoce.** Se ignoraban sin decir nada, así que un `untouchable_repo`
  mal escrito no protegía ningún repositorio. Si parece una errata, el mensaje dice el
  nombre correcto. Los ficheros se siguen leyendo, así que un ajuste de una versión más
  nueva no rompe a una anterior.
