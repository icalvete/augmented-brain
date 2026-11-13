- **Las reglas de git y gh valen se pida como se pida.** Un push desde un repositorio
  donde Claude puede hacer push pregunta ahora si va a un sitio que no es uno de sus
  remotos, y se deniega si ese sitio es un repositorio intocable. Los ajustes de git que
  ejecutan un comando o traen un alias se revisan se den como se den, y también los
  comandos que git lanza desde sus opciones. Fusionar, hacer commit de un fichero o mover
  una rama con `gh api` o `glab api` se para como `gh pr merge`; volver a lanzar un flujo,
  un `repository_dispatch` y guardar un alias de gh preguntan antes. En un repositorio
  intocable también se deniegan `merge`, `cherry-pick`, `revert`, `am` y `pull`.
