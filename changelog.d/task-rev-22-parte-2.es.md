- **Un perfil de producción de AWS puesto en un comando pregunta, sea cual sea el
  programa.** `AWS_PROFILE=prod bash deploy.sh` pasaba y `./deploy.sh` preguntaba: unos
  pocos programas con una regla propia se saltaban la general. Ahora la tienen todos.
  Pedir ayuda ya no se para: `git commit --help`, `gh pr merge --help`, `su --version` y
  `git -c core.pager=cat log` pasan. Y algunos motivos dicen lo que pasó de verdad: `cmd`
  lanzado desde Bash, un script descargado y ejecutado en el mismo comando, una llamada a
  una herramienta que no se pudo leer.
