# Qué puede hacer Claude sin preguntarte

[Read in English](what-claude-may-do.md)

Claude trabaja en tu cuenta, con tus ficheros. Dos cosas deciden qué puede hacer por su
cuenta: **los permisos de Claude Code** y **el guardia de augmented brain**. Esta página
los pone uno al lado del otro, para que sepas qué pasa sin preguntar, qué te pregunta antes
y qué no pasa nunca.

## Dos controles, uno detrás de otro

1. **El guardia** mira cada comando y cada cambio de fichero antes de que se ejecute. Para
   unas cosas, te pregunta otras, y deja pasar el resto. Mira [El guardia](guard.es.md).
2. **Los permisos de Claude Code** deciden después: el modo en que lo usas (por defecto,
   pregunta antes de actuar) y las reglas que hayas añadido. Algunos modos tienen un
   control propio que puede parar algo que el guardia dejaría pasar.

Los dos tienen que decir que sí. Un sí del guardia nunca se salta una pregunta de Claude
Code, ni al revés.

## Nunca, digas lo que digas

- Hacerse root, salvo que abras una excepción para una máquina hasta una fecha (más abajo).
- Escribir un secreto en un comando o en una nota: una contraseña, un token, una clave.
- Borrar tu carpeta de usuario o el disco entero.
- Nada en un repositorio que marcaste como **intocable**: ni commit, ni push, ni merge,
  digan lo que digan las demás listas.

Claude te dice qué quería hacer y te da el comando, para que lo lances tú.

## Te pregunta cada vez

- **Commit, push o merge en un repositorio de la lista «pregunta».**
- **Un push que reescribe lo que hay en el remoto** (`--force` y parecidos), también donde
  Claude puede hacer push. Siempre, a propósito.
- **Fusionar con `--admin`**, que se salta la protección de la rama.
- **Producción:** un comando `aws` con un perfil que marcaste como de producción, y `ssh` a
  una máquina que marcaste como intocable.
- **Root en una máquina con excepción**, cada comando, hasta la fecha de la excepción.

## Sin preguntar, porque tú lo dijiste

- **Commit y push en un repositorio de la lista «permitido»**, cuando le pediste ese
  cambio. Un merge también, cuando el comando nombra el repositorio: `cd ~/code/blog && gh
  pr merge 12`, o, si ya está en esa carpeta, `gh pr merge 12 -R usuario/blog`.

Todo lo demás que Claude hace por su cuenta es leer, escribir tus notas y lanzar comandos
corrientes: el guardia los deja pasar, y decide el modo de Claude Code.

## Las listas son tuyas

Las listas están en la configuración, y se cambian en la página **Guardia** del panel.
Claude no puede abrir esa página, y el guardia pregunta antes de que Claude escriba los
ficheros de la configuración. Algunas cosas avisan a propósito: añadir un repositorio donde
Claude hace commit y push sin preguntar pide marcar una casilla, por ejemplo.

## Primero aceptar, después usar

El guardia guarda un **sello** de las listas que aceptaste por última vez. Si las listas
cambian por otro lado (a mano, un script, otro programa), lo que permiten de nuevo aún no
cuenta:

- cada commit, push o merge que dejarían pasar **te pregunta antes**;
- la página Guardia enseña **qué ha cambiado** (lo añadido y lo quitado, lista por lista)
  y un botón para aceptarlo;
- `augmented-brain doctor` también lo dice.

Acepta en el panel, o empieza una sesión nueva, y las listas nuevas se aplican. Un comando
lo puede lanzar Claude, así que `augmented-brain config restore` nunca acepta listas por
ti.

## Root en una máquina, un tiempo

Para un trabajo que lo necesite, añade en la página Guardia una excepción con fecha de fin:
root en esa máquina te pregunta cada comando hasta esa fecha, y después se para otra vez
solo. Pon una fecha cercana.

## Cuando quieras cambiar algo

- Para dejar que Claude haga commit en un repositorio, añádelo en la página Guardia, a la
  lista «pregunta» primero.
- Para que Claude Code pregunte menos, cambia sus propios permisos, no el guardia: el
  guardia es para lo que no debe pasar nunca.
- Si un comando se para y crees que no debía, Claude dice por qué. Dile qué querías; te da
  el comando para que lo lances tú.
