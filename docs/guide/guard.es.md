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
  `git subtree push`, `git send-pack`, `gh release create`, `glab release create`, y
  lo mismo por la API (`gh api` escribiendo un fichero con `contents`, una mutación de
  GraphQL que fusiona o mueve una rama). Salvo que dejes a Claude en un repositorio (más abajo). Pedir ayuda sí pasa: `git
  commit --help`, `gh pr merge --help`, `su --version`.
- **Hacerse root.** `sudo`, `su` (y `sudo-rs`, `su-rs`), `doas`, `pkexec`, `run0`, `sudoedit`, `systemd-run`
  sin `--user`, `ssh root@…` (también `-o User=root`), un contenedor al que se le da la
  máquina entera (`docker run --privileged`, `-v /:/host`, `--pid=host`, el socket de
  Docker o uno de los discos de la máquina; también con `podman` y `nerdctl`, y `kubectl
  debug node/…`), y en Windows `gsudo` y `runas`. Claude te da el comando
  exacto para que lo ejecutes tú. Ejecutar como otro usuario (`sudo -u www-data …`) sí
  vale.
- **Secretos.** Un token o una clave escritos en un comando o en un fichero que escribe
  Claude. Los secretos se leen del llavero del sistema, de una variable de entorno o de un
  gestor de secretos. Nunca se escriben a mano.
- **Borrar tu carpeta personal o el disco entero.** `rm -r` de `~` o de `/`, también
  después de un `cd` (`cd ~ && rm -rf .`) o con un patrón que se lleva todo lo de dentro
  (`rm -rf ~/*`, `~/.*`), y en Windows se escriba como se escriba la ruta. También
  llevársela a otro sitio (`mv ~ …`) o vaciarla con `rsync --delete`.
- **PowerShell.** El guardia lee la shell, no PowerShell, así que para cualquier comando
  de PowerShell, y también `cmd` o `powershell` lanzados desde Bash. Claude lo hace en
  Bash (Git Bash en Windows). En Windows verás ese bloqueo: es lo esperado.

## Lo que te pregunta antes

- **Producción.** Un comando `aws` con un perfil que marcaste como producción. Mira todos
  los perfiles que puede usar el comando: `--profile`, `AWS_PROFILE`,
  `AWS_DEFAULT_PROFILE`, o `default` si no hay ninguno. Igual con las herramientas que
  llaman a AWS por su cuenta: `cdk`, `sam`, `serverless`, `terraform`, `tofu`, `pulumi`,
  `eksctl`, `copilot`. Y cualquier otro programa con un perfil de producción puesto en el
  propio comando (`AWS_PROFILE=prod bash deploy.sh`): lo que ejecute puede llegar a AWS
  con él.
- **Máquinas que no se tocan.** `ssh`, `scp`, `rsync`, `sftp`, `mosh`, `sshfs` y
  `autossh` a una máquina que marcaste como intocable, también cuando solo pasan por ella
  (`-J`, `ProxyJump`) o la nombran en `-o HostName`. `db01` y `db01.example.com` cuentan como la misma máquina.
- **Lanzar a mano un flujo de publicación** (`gh workflow run release`, o lo mismo por
  `gh api`). Saca una versión, así que decides tú, pero sin ir a un terminal: di que sí
  solo si has pedido tú esa versión ahora. Si dices que no, o nadie contesta, no se lanza
  nada. En el modo `bypassPermissions`, donde Claude Code puede saltarse ese diálogo, se
  para. Los demás flujos pasan; uno nombrado solo por su número también pregunta.
- **Pushes que reescriben la historia**, incluso donde Claude puede hacer push:
  `--force`, `--delete`, `--mirror`, `--prune`, una refspec que empieza por `+` o por `:`.
- **Pushes a otro sitio.** Un push desde un repositorio donde Claude puede hacer push, a
  una dirección que no es uno de sus remotos (una URL, otra carpeta). Si esa carpeta es
  un repositorio de tus listas, cuenta su propia respuesta.
- **git y gh con algo que el comando no enseña.** Ajustes de git que ejecutan un
  comando, traen un alias o apuntan a otro remoto u otro fichero de ajustes, se den como
  se den (`-c`, `GIT_CONFIG_PARAMETERS`, `git config`, `GIT_CONFIG_GLOBAL`); una mutación
  de GraphQL con `gh api graphql`; volver a lanzar un flujo (`gh run rerun`) o lanzarlo
  con `repository_dispatch`; guardar un alias de gh o glab.
- **Lo que no puede leer.** Un programa dado como variable (`$TOOL deploy`), comandos
  anidados demasiado hondo (más de cuatro comandos, uno dentro de otro: `sudo`, `ssh`,
  `bash -c`, `find -exec`, un alias de git), un comando de más de 256 KB,
  `rm -rf "$DIR/"*` con una variable que puede estar vacía, un `rm -r .` después de un
  `cd` que no puede seguir, un script que no puede leer o de más de 64 KB, un script que
  el mismo comando escribe de una forma que no puede leer y luego ejecuta
  (`curl -o x.sh … && sh x.sh`). Y una shell que ejecuta lo que escribe un programa
  (`curl … | sh`: guárdalo en un fichero y ejecuta el fichero, y se lee), un script que
  es la salida de un comando (`source <(…)`), git, gh o glab cuando una variable dice
  qué hacen (`git $cmd`), `xargs` pasándole a git o a una shell las palabras que lee, y
  `rm -r "$(…)"`.
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

Lo más fácil es la página **Guardia** del panel. Al añadir un repositorio a la lista
donde Claude hace commit y push sin preguntar, sale un aviso y tienes que marcar una
casilla que dice que lo entiendes. Añade solo repositorios donde un error se pueda
deshacer.

**Fusionar una pull request también es un push.** `gh pr merge`, `glab mr merge` y `glab
mr accept` van por las mismas listas. gh y glab no tienen `-C`, así que el comando tiene
que decir qué repositorio es:

- `cd ~/code/blog && gh pr merge 12`: la carpeta del `cd` es el repositorio.
- En la carpeta del propio repositorio, Claude Code quita ese `cd` antes de que lo vea el
  guardia. Ahí Claude nombra el repositorio: `gh pr merge 12 -R dueño/nombre`, o la
  dirección de la pull request.
- Un repositorio nombrado así tiene que ser uno de los remotos de la carpeta, o la fusión
  se para.
- La carpeta de la sesión, sola, no cuenta.

`gh pr merge --admin`, que se salta la protección de la rama, pregunta
siempre. Solo cuenta el comando en sí: `gh pr merge` escrito en el cuerpo de una pull
request o en un mensaje de commit no es una fusión, y tampoco lo son una dirección o un
`--admin` dentro del texto de `--body` o `--subject`.

## Repositorios que Claude nunca debe tocar

Hay repositorios que no son solo tuyos, o que están a un push equivocado de producción.
Ponlos como **intocables** y el guardia deniega `git commit`, `git push` y fusionar una
pull request ahí, digan lo que digan las demás listas: gana a `git_allowed_repos` y a
`git_ask_repos`. También los comandos que hacen un commit sin decirlo: `merge`,
`cherry-pick`, `revert`, `am`, `pull` y `commit-tree` (`pull --ff-only` y `merge
--ff-only` no hacen ninguno, y pasan). Viven en `guard.toml`, una lista para todo el ordenador:

```toml
untouchable_repos = ["~/work/core", "~/work/cli-*"]
```

- Misma forma que las otras listas: `~`, un `*` en la última parte, y un worktree cuenta
  como su repositorio.
- La respuesta dice que el repositorio es intocable y que solo tú puedes cambiarlo. A
  Claude se le dice que no busque otra vía.
- Se añaden y se quitan en la página **Guardia** del panel. Quitar uno te pide escribir
  el nombre de la carpeta, y te avisa: desde ese momento deciden las otras listas, así que
  si el repositorio está en `git_allowed_repos`, Claude hará commit y push ahí.

## Preguntas que dicen de qué van

Cuando un repositorio está en `git_ask_repos`, la pregunta nombra el repositorio, la rama
y adónde lleva el remoto (la dirección, nunca una contraseña): `Rama: feat/x. Remoto:
git.example.com/acme/app`. Igual al fusionar una pull request.

## Solo tú cambias las listas del guardia

El asistente trabaja en la misma cuenta que tú, así que el guardia no puede ser un muro.
Hace tres cosas para que las listas sigan siendo tuyas:

- **Pregunta antes de escribir en los ficheros del guardia** (`config.toml`,
  `guard.toml`, `secrets.toml`, `remote.toml`, `session.toml`, el sello de abajo y los
  ajustes de Claude Code), o en lo que hace funcionar al guardia: el plugin tal como lo
  guarda Claude Code (`~/.claude/plugins/cache/augmented-brain/`,
  `installed_plugins.json`) y el programa `augmented-brain`. Copiar, desempaquetar o
  sincronizar en la carpeta que guarda uno de ellos también pregunta (`cp -t`, `tar
  -C`, `rsync`).
- **Deniega llegar al panel.** Leer `panel.json` (la llave del panel) con una herramienta
  o un comando, también con un patrón o copiando, archivando o buscando en su carpeta, y
  llamar a una página del panel desde un comando, se deniega, se escriba
  como se escriba la dirección: la página Guardia del panel es como tú cambias las
  listas, y funciona con esa llave. `augmented-brain panel` tampoco escribe nunca la
  llave: escribe un enlace que sirve una vez, y lanzado por Claude no escribe ninguno.
- **Se da cuenta de un cambio que no vio.** El guardia guarda un sello de sus listas: se
  toma cuando empiezas una sesión y al guardar en el panel. Cuando Claude solo añade un
  perfil de producción o una máquina a listas que ya habías aceptado, el sello las sigue:
  añadir solo hace que el guardia pregunte más. Si las listas difieren del
  sello, algo más las cambió, y un commit, push o merge que las listas dejarían pasar
  pregunta antes, y la página Guardia y `doctor` lo dicen, hasta que las aceptes en la
  página Guardia o empieces una sesión nueva. Mientras tanto, las listas no pueden
  ablandar el guardia: la excepción de root de esta máquina no cuenta, y todo comando
  `aws` y todo `ssh`, `scp` o `rsync` a otra máquina pregunta, porque un perfil o un
  host quitado puede ser justo lo que cambió. Una sesión nueva toma las listas como
  están, y el propio Claude Code te lo dice. Cuando Claude Code compacta una
  conversación, o después de `/clear`, el sello se queda: eso no eres tú mirando. Si el
  sello no se puede leer, el guardia también pregunta, y `doctor` dice por qué. La
  página Guardia enseña qué se ha añadido y quitado desde entonces.
- **Te dice lo que cambió el panel.** Un cambio de las listas hecho en el panel, o unas
  listas aceptadas allí, te llega como una línea del propio Claude Code, no de Claude,
  después de la siguiente respuesta o mensaje en tus sesiones abiertas.

Lo que esto no para: un script que Claude escriba en línea (`python -c`, un heredoc)
todavía puede editar los ficheros, como puede hacer cualquier cosa que tu cuenta pueda.
El guardia nota entonces el cambio y pregunta antes de un push, pero no lo impidió. Es
una red contra despistes, no un muro de seguridad.

## Preguntar antes de que Claude guarde una regla

Parte de lo que Claude guarda cambia cómo trabajan todas las sesiones desde entonces: una
regla, tu perfil y lo que añade a las listas del guardia. Tú eliges cuánto hace por su
cuenta:

- **Guardar y avisarte.** Claude lo guarda, y Claude Code te dice en una línea propia qué
  se ha guardado. Si no se lo has pedido, dile a Claude que lo deshaga. Así funciona si no
  eliges otra cosa.
- **Preguntarte antes.** Antes de guardar, cambiar o retirar una regla, antes de cambiar
  tu perfil y antes de añadir nada a las listas del guardia, Claude Code te enseña lo que
  se guardaría y dices sí o no. Si dices que no, no se escribe nada.

Lo eliges en la página Guardia del panel, en «Lo que el asistente guarda solo»
(`memory_trust` en [`session.toml`](configuration.es.md#sessiontoml)). Claude no puede
cambiarlo por su cuenta: el guardia pregunta antes de escribir en ese fichero. Las
sesiones ya abiertas siguen la nueva elección desde lo siguiente que guarden.

Las tareas, el buzón, las referencias y las lecciones se guardan siempre sin preguntar:
son el trabajo en sí, y no le dicen a Claude cómo comportarse.

Una sesión que se salta las preguntas de permiso (`bypassPermissions`) no puede
preguntarte. Con «preguntarte antes», el guardia deniega ahí el guardado, y Claude te
dice lo que quería guardar para que lo guardes desde otra sesión.

## Perfiles de producción y máquinas intocables

Se los das en la entrevista de la primera sesión, o cuando quieras: "el perfil de AWS
`tienda-prod` es producción", "no toques nunca `db01` sin preguntarme". Claude los guarda
con la herramienta `safety_save`. Valen en toda la máquina, en todos los espacios.

Claude solo puede añadir a estas listas. Para quitar uno, usa la página **Guardia** del
panel, o edita tú `production_aws_profiles` y `untouchable_hosts`
([Configuración](configuration.es.md)). Añadir no deja las listas esperando a que las
aceptes: si ya las habías aceptado, siguen aceptadas, y los commits y los push no empiezan
a preguntar. Si otra cosa las había cambiado antes, siguen esperándote.

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

También puedes añadirla en la página **Guardia** del panel. Te pide escribir otra vez el
nombre de la máquina y te recuerda que hasta esa fecha Claude podrá lanzar comandos como
root allí, después de que confirmes cada uno. Pon una fecha cercana.

## Mira por dentro

El guardia lee lo que se ejecuta dentro de otros comandos: `ssh máquina '…'`,
`vagrant ssh -c` (y tras `--`), `docker exec`, `podman exec`, `kubectl exec`, `bash -c`, los bucles, lo
que una tubería le pasa a una shell (`echo 'sudo …' | bash`), los scripts cortos que
lanza Claude (`bash deploy.sh`, `./deploy.sh`), también cuando el mismo comando los
escribe antes (`echo '…' > x.sh && bash x.sh`), un fichero que lee una shell
(`bash < x.sh`, `cat x.sh | bash`), `env -S '…'`, las llaves y `$IFS` como los expande la
shell, los programas que ejecutan el resto de su línea (`strace`, `screen`, `tmux`,
`setpriv`, `sg`, `at`, y las shells `fish`, `tcsh`, `csh` y `mksh`), tus alias de git
(`git ci` cuando `ci` quiere decir `commit`) y los de gh y glab, y los comandos que git
lanza desde sus opciones y variables (`--upload-pack`, `difftool -x`, `GIT_SSH_COMMAND`,
`GIT_EDITOR`). Una shell que solo comprueba la sintaxis de un script
(`sh -n install.sh`, `bash -o noexec`) no ejecuta nada, así que pasa. Mira los comandos que Claude deja en
marcha para vigilar algo, y los de los ayudantes que Claude lanza (subagentes y flujos de
trabajo).

## Cuando algo le pasa

- **No se puede leer la configuración.** Entonces el guardia no puede saber tus listas.
  Te pregunta antes de cada `aws`, `ssh`, `scp`, `rsync`, `sftp` y `mosh`, y antes de las
  herramientas que llaman a AWS. Para volver a una versión que funcionaba, lanza
  `augmented-brain config history` y después `augmented-brain config restore <número>`
  ([Versiones de la configuración](config-history.es.md)).
- **Falta el programa o está roto.** Claude Code para cada comando y cada cambio de
  fichero en vez de dejarlos pasar (Claude Code 2.1.295 o posterior; `doctor` avisa si
  es anterior). Si el propio guardia falla al revisar, rechaza esa acción y Claude te lo
  dice: es un error de augmented brain, del que puedes informar desde la página Ayuda del
  panel.
- **No se sabe cuál es la carpeta de configuración** (sin `HOME`, como en algunos
  servicios y contenedores), **o falta `config.toml` pero queda lo demás.** El guardia
  pregunta, como cuando no se puede leer la configuración.
- **`guard.toml` no se puede leer.** El guardia pregunta antes de cada commit y push
  hasta que se arregle o se quite.
- `augmented-brain doctor` prueba el guardia de verdad y te dice si no está funcionando.

## Por dónde seguir

- [Configuración](configuration.es.md): cada lista, con un ejemplo.
- [Si algo falla](troubleshooting.es.md): cuando el guardia para algo que no debería.
