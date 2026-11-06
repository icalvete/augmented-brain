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

Los motivos salen en tu idioma, el `language` de tu
[configuración](configuration.es.md) (el del sistema mientras no se puede leer).

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
siempre. Solo cuenta el comando en sí: `gh pr merge` escrito en el cuerpo de una pull
request o en un mensaje de commit no es una fusión, y tampoco lo son una dirección o un
`--admin` dentro del texto de `--body` o `--subject`.

## Repositorios que Claude nunca debe tocar

Hay repositorios que no son solo tuyos, o que están a un push equivocado de producción.
Ponlos como **intocables** y el guardia deniega `git commit`, `git push` y fusionar una
pull request ahí, digan lo que digan las demás listas: gana a `git_allowed_repos` y a
`git_ask_repos`. Viven en `guard.toml`, una lista para todo el ordenador:

```toml
untouchable_repos = ["~/work/core", "~/work/cli-*"]
```

- Misma forma que las otras listas: `~`, un `*` en la última parte, y un worktree cuenta
  como su repositorio.
- La respuesta dice que el repositorio es intocable y que solo tú puedes cambiarlo. A
  Claude se le dice que no busque otra vía.
- Se añaden y se quitan en la página **Guardia** del panel. Quitar uno te pide escribir
  el nombre de la carpeta.

## Preguntas que dicen de qué van

Cuando un repositorio está en `git_ask_repos`, la pregunta nombra el repositorio, la rama
y adónde lleva el remoto (la dirección, nunca una contraseña): `Rama: feat/x. Remoto:
git.example.com/acme/app`. Igual al fusionar una pull request.

## Solo tú cambias las listas del guardia

El asistente trabaja en la misma cuenta que tú, así que el guardia no puede ser un muro.
Hace tres cosas para que las listas sigan siendo tuyas:

- **Pregunta antes de escribir en los ficheros del guardia** (`config.toml`,
  `guard.toml`, `secrets.toml`, `remote.toml`, `session.toml` y los ajustes de Claude Code).
- **Deniega llegar al panel.** Leer `panel.json` (la clave del panel) con una herramienta
  o un comando, y llamar a una página del panel desde un comando, se deniega: la página
  Guardia del panel es como tú cambias las listas, y funciona con esa clave.
- **Se da cuenta de un cambio que no vio.** El guardia guarda un sello de sus listas: se
  toma al empezar una sesión y al guardar en el panel. Si las listas difieren del sello,
  algo más las cambió, y un commit, push o merge que las listas dejarían pasar pregunta
  antes, y la página Guardia y `doctor` lo dicen hasta que las aceptes.

Lo que esto no para: un script que Claude escriba en línea (`python -c`, un heredoc)
todavía puede editar los ficheros, como puede hacer cualquier cosa que tu cuenta pueda.
El guardia nota entonces el cambio y pregunta antes de un push, pero no lo impidió. Es
una red de seguridad, no una frontera de seguridad.

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
- **`guard.toml` no se puede leer.** El guardia pregunta antes de cada commit y push
  hasta que se arregle o se quite.
- `augmented-brain doctor` prueba el guardia de verdad y te dice si no está funcionando.

## Siguiente

- [Configuración](configuration.es.md): cada lista, con un ejemplo.
- [Si algo falla](troubleshooting.es.md): cuando el guardia para algo que no debería.
