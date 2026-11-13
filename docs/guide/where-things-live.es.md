# Dónde vive cada cosa, y cómo se protege

[Read in English](where-things-live.md)

augmented brain guarda todo en ficheros normales en tu ordenador. Esta página dice dónde
está cada cosa, qué la protege, y qué viaja en la copia de seguridad y qué no viaja nunca.

## Tus notas

**Dónde:** una carpeta por espacio, dentro de su carpeta de trabajo:
`~/augmented-brain/trabajo/notas/`, por ejemplo (`notes` en inglés). Tareas, áreas,
conocimiento, personas, el buzón, el índice `MEMORY.md`. Ficheros Markdown que puedes
abrir y leer.

**Cómo se protegen:**

- **Solo tú puedes leerlas.** La carpeta es privada de tu usuario; `augmented-brain check`
  la vuelve a cerrar si alguien la abrió.
- **Sin secretos dentro.** Las herramientas de Claude se niegan a escribir una contraseña,
  un token o una clave en una nota.
- **Nunca a medio escribir.** Cada cambio se comprueba antes y se escribe de una vez, así
  que un fallo deja la nota como estaba.
- **La copia de seguridad**, si la configuras: la carpeta de notas es un repositorio git,
  que se copia a un repositorio privado tuyo al final de cada sesión. Mira [Copia de
  seguridad y restauración](backup.es.md).

## La configuración

**Dónde:** una carpeta por ordenador, la carpeta de configuración:
`~/.config/augmented-brain/` en Linux y macOS, `%APPDATA%\augmented-brain\` en Windows.

| Fichero | Qué guarda |
|---|---|
| `config.toml` | Los espacios y las listas del guardia de todo el ordenador. |
| `guard.toml` | Los repositorios que nada puede tocar. |
| `secrets.toml` | Los **nombres** de tus secretos y para qué es cada uno. Nunca un valor. |
| `session.toml` | El modelo que deben usar tus sesiones. |
| `remote.toml` | El acceso remoto: qué espacios tienen puerta. |
| `profile.md` | Quién eres y cómo te gusta trabajar. |

**Cómo se protege:**

- **El guardia pregunta** antes de que Claude escriba cualquiera de estos ficheros.
- **Versiones.** Se guardan las 10 últimas, con quién hizo cada cambio: volver atrás es un
  comando, aunque `config.toml` se haya quedado roto. Mira [Versiones de la
  configuración](config-history.es.md). Solo en este ordenador.
- **El sello.** El guardia nota cuándo sus listas cambiaron fuera del panel, y pregunta
  antes de usarlas. Mira [Qué puede hacer Claude sin preguntarte](what-claude-may-do.es.md).

## Lo que sabe este ordenador

También en la carpeta de configuración, nunca en la copia:

- `state/`: lo que hicieron las sesiones de cada espacio (la tarea activa de cada una, una
  sesión que no acabó bien), el sello del guardia, las sesiones de Claude en marcha.
- `history/`: las versiones de la configuración.
- `panel.json`: dónde escucha el panel mientras está en marcha, y su clave. Claude nunca
  puede leerlo.
- `migrations/`: una copia completa de las notas, guardada antes de que una versión nueva
  las pasara a su formato.
- `marketplace/`: el plugin de Claude Code, tal como lo lleva el programa (abajo).

## Tus secretos

Los valores se quedan en el llavero de tu sistema (o donde los guardes): augmented brain
nunca los guarda. Solo se apuntan sus nombres, en `secrets.toml`, para que tú y un
ordenador nuevo sepáis qué secretos hay. La página **Secretos** del panel dice cuáles tiene
el llavero de este ordenador.

## Qué lleva la copia de seguridad, y qué deja fuera

| Va en la copia | No va nunca |
|---|---|
| Todas las notas del espacio | El valor de tus secretos |
| Los ajustes del propio espacio: sus carpetas y sus listas | Las listas de todo el ordenador: perfiles de producción, máquinas y repositorios intocables, excepciones de root |
| Solo en el espacio por defecto: tu perfil, las notas compartidas y los nombres de tus secretos | Las versiones de la configuración, `state/` y `panel.json` |

Las listas de todo el ordenador se quedan fuera porque pueden nombrar las máquinas de tu
trabajo. Los repositorios donde Claude puede hacer commit o push viajan con el espacio,
pero restaurar no los usa, porque alguien podría haber cambiado la copia: los nombra, y los
vuelves a añadir si los sigues queriendo.

## El programa y el plugin

- El programa: `~/.local/bin/augmented-brain` (`augmented-brain.exe` en Windows).
- El plugin de Claude Code viaja dentro del programa: lo escribe en `marketplace/`, en la
  carpeta de configuración, y Claude Code lo lee de ahí. Así sus hooks son siempre los de
  la versión que instalaste, y el guardia pregunta antes de que Claude los cambie.

Desinstalar quita estos dos y deja tus notas. Mira [Actualizar y desinstalar sin perder
nada](update-and-uninstall.es.md).
