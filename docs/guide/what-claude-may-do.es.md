# Qué puede hacer Claude sin preguntarte

[Read in English](what-claude-may-do.md)

Claude trabaja en tu cuenta, con tus ficheros. Dos cosas deciden qué puede hacer por su
cuenta: **los permisos de Claude Code** y **el guardia de augmented brain**. Esta página
los pone uno al lado del otro, para que sepas qué se para, qué te pregunta antes y qué
sigue adelante sin preguntar. La historia entera, comando a comando, está en
[El guardia](guard.es.md).

## Dos controles, uno detrás de otro

1. **El guardia** mira cada comando y cada cambio de fichero antes de que se ejecute. Para
   unas cosas, te pregunta otras, y deja pasar el resto.
2. **Los permisos de Claude Code** deciden después: el modo en que lo usas (por defecto,
   pregunta antes de actuar) y las reglas que hayas añadido. Algunos modos tienen un
   control propio que puede parar algo que el guardia dejaría pasar.

Los dos tienen que decir que sí. Un sí del guardia nunca se salta una pregunta de Claude
Code. Al revés también vale, con una excepción: en el modo que se salta las preguntas de
Claude Code (`bypassPermissions`), una pregunta del guardia puede no llegarte. No uses ese
modo con el guardia; [por qué](guard.es.md#sesiones-que-se-saltan-las-preguntas-de-claude-code).

Si haces que augmented brain se quede en tus espacios, fuera de sus carpetas el guardia
solo protege sus propios ficheros, tu carpeta personal y lo que nombran tus listas: ahí,
lo demás depende solo de los permisos de Claude Code. Mira
[Fuera de todos los espacios](spaces.es.md#fuera-de-todos-los-espacios).

## Se para

El guardia para esto. Claude te dice qué quería hacer y te da el comando, para que lo
lances tú.

- **Un commit o un push en un repositorio que no has puesto en una lista.** Por defecto
  Claude no hace commit ni push en ninguno: tú le abres un repositorio (más abajo).
- **Commit, push o merge en un repositorio que marcaste como intocable**, digan lo que
  digan las demás listas. El resto del trabajo ahí (leer, compilar, probar) sigue como en
  cualquier otro sitio.
- **Hacerse root** (el administrador de la máquina), salvo que abras una excepción para
  una máquina hasta una fecha (más abajo).
- **Escribir un secreto** en un comando o en un fichero: una contraseña, un token, una
  clave.
- **Borrar tu carpeta personal o el disco entero.**
- **PowerShell**, que el guardia no sabe leer. Claude hace lo mismo en Bash.

Las tres primeras las cambias tú, con las listas; las tres últimas se paran siempre.

## Te pregunta cada vez

- **Commit, push o merge en un repositorio de la lista «preguntar».**
- **Un push que sobrescribe lo que hay en el servidor** (`--force` y similares), incluso
  donde Claude puede hacer push. Siempre, a propósito.
- **Un push a otro sitio**: una dirección que no es uno de los remotos del repositorio.
- **Fusionar con `--admin`**, que se salta la protección de la rama.
- **Producción:** un comando `aws` con un perfil que marcaste como producción.
- **Máquinas intocables:** `ssh` y similares a una máquina que marcaste como intocable.
- **Lanzar una publicación** (`gh workflow run release`): saca una versión.
- **Root en una máquina con excepción**, cada comando, hasta la fecha de la excepción.
- **Un comando que el guardia no sabe leer hasta el final**: un programa dado como
  variable, un script que no puede abrir, `curl … | sh`.
- **Escribir los ajustes del guardia o los de Claude Code**, porque una línea ahí podría
  apagar una comprobación.
- **Guardar una regla, tu perfil o una lista del guardia**, si elegiste «preguntarte
  antes» (más abajo).

## Sin preguntar

- **Commit y push en un repositorio de la lista «permitido»**, cuando le pediste ese
  cambio. También un merge, cuando el comando nombra el repositorio: `cd ~/code/blog && gh
  pr merge 12`, o, ya en esa carpeta, `gh pr merge 12 -R owner/blog`.
- **Guardar una regla, tu perfil, o un perfil de producción o una máquina intocable** que
  le has contado. Claude Code te dice en una línea propia qué se guardó; si no lo pediste,
  dile a Claude que lo deshaga. Así es por defecto; para que te pregunte antes, mira
  [Preguntar antes de que Claude guarde una regla](guard.es.md#preguntar-antes-de-que-claude-guarde-una-regla).

Todo lo demás que hace Claude por su cuenta es leer, escribir tus notas y ejecutar
comandos normales: el guardia los deja pasar, y decide el modo de Claude Code.

## Las listas son tuyas

Las listas viven en la configuración, y las cambias en la página **Guardia** del panel.
Claude no puede abrir esa página, y el guardia pregunta antes de que Claude escriba los
ficheros de configuración. Algunas páginas avisan a propósito: añadir un repositorio donde
Claude hace commit y push sin preguntar pide marcar una casilla, por ejemplo. Claude solo
puede añadir a dos de ellas, los perfiles de producción y las máquinas intocables, y eso
solo hace que el guardia pregunte más.

## Primero aceptar, después usar

El guardia guarda un **sello** de las listas que aceptaste por última vez. Si las listas
cambian por otro lado (a mano, un script, otro programa), lo que permiten de nuevo aún no
cuenta:

- cada commit, push o merge que dejarían pasar **te pregunta antes**;
- la página Guardia enseña **qué ha cambiado** (lo añadido y lo quitado, lista por lista)
  y un botón para aceptarlo;
- `augmented-brain doctor` también lo dice.

Acepta en el panel, o empieza una sesión nueva, y las listas nuevas se aplican. Cuando
Claude solo añade un perfil de producción o una máquina intocable a unas listas que habías
aceptado, siguen aceptadas: añadir solo hace que el guardia pregunte más.

`augmented-brain config restore` vuelve a una versión anterior de la configuración, pero
no acepta sus listas por ti: ese comando también lo puede lanzar Claude, así que aceptar
se queda en el panel.

## Root en una máquina, durante un tiempo

Para un trabajo que lo necesite, añade en la página Guardia una excepción con fecha de
fin: root en esa máquina te pregunta en cada comando hasta esa fecha, y después se vuelve
a parar solo. Pon una fecha cercana. Más en
[El guardia](guard.es.md#root-en-una-máquina-durante-un-tiempo).

## Cuando quieras cambiar algo

- Para que Claude haga commit en un repositorio, añádelo en la página Guardia, primero a
  la lista «preguntar».
- Para que Claude Code pregunte menos, cambia sus propios permisos, no el guardia: el
  guardia es para lo que no debe pasar nunca.
- Si se para un comando y crees que no debería, Claude te dice por qué. Dile qué querías;
  te da el comando para que lo lances tú.
