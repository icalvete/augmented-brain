# Copia de seguridad y restauración

[Read in English](backup.md)

Se te muere el portátil. En el nuevo, instalas augmented brain, lanzas un comando y abres
Claude. Sigue donde lo dejaste: tareas, notas, reglas y tu perfil.

Eso es la **copia de seguridad**. La carpeta de notas de cada espacio es también un
repositorio git (git guarda cada versión de una carpeta y puede copiarla a un servidor).
La copia lo sube a un repositorio privado tuyo, uno por espacio.

## Configurarla: pídeselo a Claude

Dile "configura la copia de seguridad". Al final de la entrevista de la primera sesión,
Claude también te la ofrece.

Crea en tu cuenta de GitHub un repositorio privado, `augmented-brain-<espacio>`, lo
conecta y hace la primera copia. Para eso usa `gh`, la herramienta oficial de GitHub para
la terminal. Si no la tienes, Claude te dice cómo instalarla, y te guía en el único paso
que necesita tu navegador: entrar con tu cuenta.

¿Prefieres otro sitio (GitLab, el git de tu empresa)? Hazlo tú, en una carpeta del
espacio:

```bash
augmented-brain backup setup git@gitlab.com:lucia/cerebro-trabajo.git
```

Claude no acepta una dirección para esto: una que le hubiera pasado una página o un
mensaje que leyó mandaría allí todas tus notas. Usa un repositorio **privado**: dentro van
tus notas. Y una dirección sin contraseña: queda guardada en la configuración y dentro de
la propia copia, así que `backup setup` la rechaza. Deja que git guarde el acceso (en
GitHub, `gh auth setup-git`; o una dirección ssh como la de arriba). Si una versión
anterior guardó una, `augmented-brain doctor` te dice cómo quitarla.

La página [Copia de seguridad](panel.es.md#copia-de-seguridad) del panel hace lo mismo con
dos botones, y enseña las últimas copias de cada espacio y un **Hacer una copia ahora**.

Mientras un espacio no tenga copia, augmented brain lo dice: `augmented-brain doctor`
nombra cada espacio sin copia, `update` avisa cuando sigue sin una copia a la que volver,
y `uninstall --all` avisa de que esas notas se perderán para siempre.

## Se hace sola

Desde entonces, **la copia se hace sola cada vez que termina una sesión de Claude**, en
segundo plano. No esperas a nada. Un espacio en el que esta sesión escribió desde otro
(una tarea de otro espacio) también se copia entonces. Si falla (sin conexión, sin permiso en el repositorio),
Claude te lo dice al empezar la siguiente sesión.

También puedes lanzarla a mano:

```bash
augmented-brain backup          # el espacio de esta carpeta
augmented-brain backup --all    # todos los espacios
```

Para apagar la copia automática, pon `AB_NO_AUTO_BACKUP=1` en tu entorno.

## En una máquina nueva

1. Instala augmented brain con el comando de siempre. Te crea los espacios vacíos. Si al
   acabar se abre Claude, ciérralo sin hacer la entrevista: tu perfil viene en la copia.
2. Si el repositorio está en GitHub, entra una vez con `gh auth login`, para que git
   tenga permiso.
3. Recupera cada espacio desde su copia:
   ```bash
   augmented-brain restore https://github.com/lucia/augmented-brain-trabajo.git
   ```
   Sustituye el espacio vacío por el tuyo: notas, tareas, reglas y la configuración del
   espacio, que solo puedes leer tú. Si algo falla a medias, el espacio vacío se queda
   como estaba. Si ese espacio ya tenía notas tuyas, no lo toca.
4. Abre Claude. Sigue donde lo dejaste, y la copia sigue haciéndose sola.

Recupera también el espacio por defecto (`personal`): con él vienen tu perfil, tus notas
compartidas y la lista de los nombres de tus secretos.

## Volver a una copia

¿Se ha estropeado una nota, o la has borrado sin querer? Abre la página **Copia de
seguridad** del panel. Cada copia enseña **Qué cambió**: las notas que añadió, cambió y
quitó, con sus líneas.

Para tener tus notas como estaban en una copia, usa **Volver a esta copia**:

1. Una página enseña qué cambiaría en tus notas tal como están ahora, nota a nota.
   También salen las notas que aún no están en ninguna copia, como quitadas.
2. Marca la casilla y escribe el nombre del espacio.
3. Primero, lo que tienes ahora se guarda como una copia nueva. Después tus notas pasan a
   ser las de la copia que elegiste, y eso también se guarda como copia. La siguiente
   copia de seguridad lo sube.

Para deshacerlo, vuelve a la copia que se hizo justo antes. Volver solo toca las notas de
ese espacio, nunca tu configuración (eso es [Versiones](config-history.es.md)). Se hace
solo desde el panel de este ordenador: desde otro dispositivo la página solo enseña.

## Lo que no trae una restauración

Algunas cosas se quedan fuera a propósito:

- **Las listas del guardia**: los perfiles de AWS de producción y las máquinas
  intocables. Pueden nombrar máquinas de tu trabajo, así que no van en ninguna copia.
  `restore` y `doctor` lo dicen, y la siguiente sesión te ofrece volver a pedírtelas.
  Hasta entonces, nada pregunta antes de tocar producción.
- **Los repositorios donde Claude puede hacer commit y push** (`git_allowed_repos` y
  `git_ask_repos`). Alguien podría haber cambiado la copia, así que `restore` los nombra,
  y los vuelves a añadir si lo sigues queriendo.
- **El valor de tus secretos.** Solo viajan sus nombres. `restore` nombra los secretos
  que el llavero de esta máquina aún no tiene, cada uno con el comando para guardarlo, y
  los que no ha podido comprobar cuando el llavero no responde, o en Windows, donde no se
  le puede preguntar.

## Lo que te da

- **Varias máquinas, un solo cerebro.** Antes de subir, la copia trae lo que hizo tu otra
  máquina y lo mezcla. Una tarea en la que cada máquina solo escribió en el diario, o en la
  que una movió entradas antiguas a `archive/journals/` mientras la otra escribía, se une
  sola.
- **Todo el historial.** Cada copia es un commit de git, con la máquina y la hora. El
  panel enseña qué cambió cada una y vuelve a ella; con `git log` las ves todas.
- **En tu sitio.** GitHub, GitLab o tu servidor: el repositorio lo eliges tú. No hay nube
  de augmented brain.
- **Un repositorio por espacio.** El del trabajo puede ir al git de la empresa, y el
  personal al tuyo. Tu perfil solo viaja en la copia del espacio por defecto, nunca en la
  del trabajo.
- **No sube secretos.** Las herramientas de Claude se niegan a escribir una contraseña o
  un token en una nota. Si aun así entra uno (a mano, por ejemplo), la copia se para y te
  dice en qué fichero.
- **No pisa nada.** Si dos máquinas cambiaron la misma nota, la copia se para sin tocar
  nada y te explica cómo resolverlo. Si cada máquina solo le añadió entradas de diario,
  se juntan solas, por fecha. El índice se rehace solo, y las capturas del buzón hechas en
  dos máquinas se guardan las dos.
- **Ni basura ni gigantes.** Los restos de un cuelgue y los ficheros que dejan los
  sistemas en las carpetas (`.DS_Store`, `Thumbs.db`) se quedan fuera. Un fichero de más
  de 50 MB también (GitHub los rechaza), y Claude te dice cuál.

## Cuando falla

Claude te lo dice al empezar la siguiente sesión, con el motivo. Si lanzas
`augmented-brain backup` en una carpeta del espacio, ves el problema. Los más comunes
están en [Si algo falla](troubleshooting.es.md#ha-fallado-la-copia-de-seguridad).

Una copia nunca se queda colgada: cada paso se para tras 3 minutos sin respuesta (o una
descarga de menos de 1 KB por segundo durante un minuto), y una copia espera como mucho
10 minutos a que acabe otra. En los dos casos dice por qué.
