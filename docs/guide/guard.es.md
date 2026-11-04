# El guardia

[Read in English](guard.md)

Claude va a ejecutar `git push origin main`. Antes, augmented brain lee el comando y lo
para: subir código es cosa tuya. Claude te dice que el cambio está listo, y lo subes tú.

Quien hace esa comprobación es el **guardia**. Lee cada comando que ejecuta Claude, y cada fichero
que escribe, antes de que pase. Tiene tres respuestas:

- **Adelante**: casi siempre. `ls`, `go test`, `aws --profile dev s3 ls`.
- **Te pregunta**: Claude Code te enseña el comando y espera tu sí. `aws --profile prod
  …`, `ssh db01 …`.
- **Lo para**: el comando no se ejecuta. Claude recibe el motivo, y qué hacer en su lugar.

**El guardia es una red contra despistes, no un muro de seguridad.** Lee los comandos
como texto. Pilla los errores que comete un asistente, también escondidos dentro de otros
comandos. Un programa empeñado en saltárselo podría hacerlo. Lee el
[aviso legal](../../DISCLAIMER.md#aviso-legal).

## Lo que para

- **Commits y pushes.** `git commit`, `git push`, y lo que publica como un push:
  `git subtree push`, `git send-pack`, `gh release create`, `glab release create`.
  Salvo que dejes a Claude en un repositorio (más abajo).
- **Hacerse root.** `sudo`, `su`, `doas`, `pkexec`, `run0`, `sudoedit`, `systemd-run`
  sin `--user`, `ssh root@…`, y en Windows `gsudo` y `runas`. Claude te da el comando
  exacto para que lo ejecutes tú. Ejecutar como otro usuario (`sudo -u www-data …`) sí
  vale.
- **Secretos.** Un token o una clave escritos en un comando o en un fichero que escribe
  Claude. Los secretos se leen del llavero del sistema, de una variable de entorno o de un
  gestor de secretos. Nunca se escriben a mano.
- **Borrar tu carpeta personal o el disco entero.** `rm -r` de `~` o de `/`, también
  después de un `cd` (`cd ~ && rm -rf .`), y en Windows se escriba como se escriba la
  ruta.
- **PowerShell.** El guardia lee la shell, no PowerShell, así que para cualquier comando
  de PowerShell. Claude lo hace en Bash (Git Bash en Windows). En Windows verás ese
  bloqueo: es lo esperado.

## Lo que te pregunta antes

- **Producción.** Un comando `aws` con un perfil que marcaste como producción. Mira todos
  los perfiles que puede usar el comando: `--profile`, `AWS_PROFILE`,
  `AWS_DEFAULT_PROFILE`, o `default` si no hay ninguno. Igual con las herramientas que
  llaman a AWS por su cuenta: `cdk`, `sam`, `serverless`, `terraform`, `tofu`, `pulumi`,
  `eksctl`, `copilot`.
- **Máquinas que no se tocan.** `ssh`, `scp`, `rsync`, `sftp` y `mosh` a una máquina que
  marcaste como intocable. `db01` y `db01.example.com` cuentan como la misma máquina.
- **Pushes que reescriben la historia**, incluso donde Claude puede hacer push:
  `--force`, `--delete`, `--mirror`, `--prune`, una refspec que empieza por `+` o por `:`.
- **Lo que no puede leer.** Un programa dado como variable (`$TOOL deploy`), comandos
  anidados demasiado hondo, `rm -rf "$DIR/"*` con una variable que puede estar vacía, un
  `rm -r .` después de un `cd` que no puede seguir.
- **Los ajustes del propio guardia.** Claude nunca cambia por su cuenta la configuración
  de augmented brain ni los ajustes de Claude Code, porque una línea ahí podría apagar una
  comprobación.

## Deja que Claude haga commit en un repositorio

Por defecto, Claude nunca hace commit ni push. Si tienes un repositorio donde quieres que
lo haga, ponlo en la configuración, en cualquier espacio:

```toml
[spaces.trabajo]
git_allowed_repos = ["~/code/blog", "~/code/blog-*"]   # commit y push sin preguntar
git_ask_repos = ["~/code/tienda"]                      # pregúntame cada vez
```

- Claude tiene que nombrar el repositorio en el comando: `git -C ~/code/blog push`.
- Un worktree de git de un repositorio de la lista cuenta como él.
- `~/code/blog-*` vale para cada carpeta `blog-<nombre>` junto a `blog`.
- Un repositorio que está en las dos listas pregunta.
- Las listas de todos los espacios cuentan juntas: un repositorio es tuyo, no del espacio.

**Fusionar una pull request también es un push.** `gh pr merge`, `glab mr merge` y `glab
mr accept` van por las mismas listas. gh y glab no tienen `-C`, así que Claude escribe
`cd ~/code/blog && gh pr merge 12`. La carpeta de ese `cd` es el repositorio. La carpeta
de la sesión, sola, no cuenta. Un repositorio que nombre el comando (`-R dueño/nombre`,
la dirección de la pull request) tiene que ser uno de los remotos de ese repositorio, o la
fusión se para. `gh pr merge --admin`, que se salta la protección de la rama, pregunta
siempre.

## Perfiles de producción y máquinas intocables

Se los das en la entrevista de la primera sesión, o cuando quieras: "el perfil de AWS
`tienda-prod` es producción", "no toques nunca `db01` sin preguntarme". Claude los guarda
con la herramienta `safety_save`. Valen en toda la máquina, en todos los espacios.

Claude solo puede añadir a estas listas. Para quitar uno, edita tú la configuración:
`production_aws_profiles` y `untouchable_hosts` en [Configuración](configuration.es.md).

Estas listas no van en ninguna copia de seguridad, porque pueden nombrar máquinas de tu
trabajo. Después de restaurar en una máquina nueva, la siguiente sesión te ofrece volver
a pedírtelas.

## Root en una máquina, durante un tiempo

A veces Claude necesita de verdad ser root en una máquina, para una migración, por
ejemplo. Añade una excepción con fecha de fin en la configuración:

```toml
root_ask = ["db01 until 2026-12-31"]
```

Hasta esa fecha, root en `db01` te pregunta cada vez en vez de pararse. El comando que
ejecuta se sigue mirando. Pasada la fecha, root vuelve a pararse solo, y
`augmented-brain doctor` te dice que puedes quitar la línea. La fecha es obligatoria, así
que ninguna excepción se queda olvidada.

## Mira por dentro

El guardia lee lo que se ejecuta dentro de otros comandos: `ssh máquina '…'`,
`vagrant ssh -c`, `docker exec`, `podman exec`, `kubectl exec`, `bash -c`, los bucles, lo
que una tubería le pasa a una shell (`echo 'sudo …' | bash`) y los scripts cortos que
lanza Claude (`bash deploy.sh`, `./deploy.sh`). Mira los comandos que Claude deja en
marcha para vigilar algo, y los de los ayudantes que Claude lanza (subagentes y flujos de
trabajo).

## Cuando algo le pasa

- **No se puede leer la configuración.** Entonces el guardia no puede saber tus listas.
  Te pregunta antes de cada `aws`, `ssh`, `scp`, `rsync`, `sftp` y `mosh`, y antes de las
  herramientas que llaman a AWS.
- **Falta el programa o está roto.** Claude Code para cada comando y cada cambio de
  fichero en vez de dejarlos pasar (Claude Code 2.1.295 o posterior).
- `augmented-brain doctor` prueba el guardia de verdad y te dice si no está funcionando.

## Siguiente

- [Configuración](configuration.es.md): cada lista, con un ejemplo.
- [Si algo falla](troubleshooting.es.md): cuando el guardia para algo que no debería.
