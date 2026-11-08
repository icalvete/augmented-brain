# Versiones de la configuración

[Read in English](config-history.md)

Tu configuración es lo que augmented brain sabe de este ordenador y de ti: tus espacios,
las listas del guardia, el acceso remoto, tu perfil y los nombres de tus secretos. Un
cambio equivocado ahí (una lista vaciada sin querer, un espacio que apunta a la carpeta
que no es) no tenía vuelta atrás. Ahora cada cambio se guarda como una **versión**, y
puedes ver qué cambió y volver atrás.

## Qué se guarda

Una versión es una copia de estos ficheros de la carpeta de configuración, tal como
estaban en un momento: `config.toml`, `guard.toml`, `remote.toml`, `session.toml`,
`profile.md` y `secrets.toml` (solo los nombres de tus secretos, nunca un valor). Dice
también quién hizo el cambio: el panel, un comando, el asistente (y con qué herramienta),
una vuelta a una versión anterior, o a mano.

- Se guarda una versión nueva cada vez que cambia la configuración. Un cambio hecho a
  mano, o con un programa anterior, se guarda la próxima vez que augmented brain mira:
  al empezar una sesión, al lanzar un comando, cuando el asistente usa una herramienta o
  cuando guardas algo en el panel.
- Se guardan las **10** últimas. Cuando hay una más, se va la más antigua.
- Se quedan **solo en este ordenador**, en la carpeta `history` de la carpeta de
  configuración. Nunca van a la copia de seguridad: las listas del guardia pueden nombrar
  máquinas del trabajo, y la copia las deja fuera a propósito.
- Tus notas no están aquí. Tareas, reglas, conocimiento y personas (también las
  compartidas) tienen su propia historia en la [copia de seguridad](backup.es.md).

## En el panel

La página **Versiones** del panel las lista, la más reciente primero, con cuándo y quién.
Debajo de cada una, **Qué cambió** enseña las líneas que cambiaron respecto a la anterior:
en verde lo que se añadió y en rojo lo que se quitó. **Volver a esta** la convierte otra
vez en la configuración.

Volver guarda antes como versión la configuración que tenías, así que se puede deshacer
igual. Una versión que dejaría un fichero que el programa no puede leer se rechaza, y no
cambia nada. Si cambian las listas del guardia, volver desde el panel cuenta como que las
aceptas, igual que cualquier cambio en la página [Guardia](guard.es.md).

Desde otro dispositivo ([el panel desde otro dispositivo](panel-remote.es.md)) la página
solo enseña las versiones: volver a una solo se puede desde el panel de este ordenador,
porque puede cambiar lo que el guardia deja pasar.

## Con comandos

```bash
augmented-brain config history
augmented-brain config history 3
augmented-brain config restore 3
```

`config history` lista las versiones; con un número, enseña qué cambió esa. `config
restore` vuelve a una. Un comando lo puede lanzar el asistente, así que volver de esta
forma nunca acepta por ti listas nuevas del guardia: hasta que las aceptes en la página
Guardia del panel, o empieces una sesión nueva, el guardia pregunta antes de cada commit,
push o merge que antes dejaba pasar. Mira [Comandos](commands.es.md#augmented-brain-config-history).

Los dos funcionan aunque `config.toml` esté roto y no arranque nada más: es justo cuando
más ayuda volver atrás. `augmented-brain doctor` lo dice entonces, y te lleva a ellos.

`augmented-brain doctor` tiene una línea con cuántas versiones se guardan, y avisa cuando
la configuración ha cambiado después de la más reciente.
