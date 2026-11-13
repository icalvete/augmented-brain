- **El guardia ya no se cuelga ni falla en silencio.** Unos pocos comandos raros (un
  `find` dentro de otro muchas veces, miles de `$(`, un alias de git que se llama a sí
  mismo) lo tenían ocupado más de los 10 segundos del hook; ahora cuenta cada comando
  anidado en otro y, pasados cuatro niveles o 256 KB, te pregunta. Si el guardia falla por
  dentro, rechaza esa acción con palabras llanas en vez de enseñar la traza de Go. Una
  llamada con una forma inesperada, una carpeta de configuración que no encuentra, un
  `config.toml` que falta, un script de shell que no puede leer o una respuesta que no
  pudo escribir ya no dejan pasar un comando sin revisar. `doctor` avisa cuando Claude
  Code es anterior a la 2.1.295, que lo deja pasar todo cuando el guardia falla.
