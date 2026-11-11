# Acceso remoto

[Read in English](remote.md)

A Claude Code se puede llegar desde otro ordenador o desde el móvil: desde
[claude.ai/code](https://claude.ai/code) o Claude Desktop eliges una sesión que corre en
tu máquina y trabajas en ella como si estuvieras delante. Para eso, un programa pequeño,
`claude remote-control`, tiene que estar funcionando en la máquina.

`augmented-brain remote` lo mantiene funcionando por ti. Arranca solo, y vuelve a
arrancar si se para. Se prepara una vez. Funciona en los tres sistemas, cada uno con su
forma de arrancar programas solos:

- **Linux**: un servicio de usuario de systemd (la parte de Linux que arranca y vigila los
  servicios). Puede arrancar al encender el ordenador, antes de que nadie entre. No en un
  contenedor, ni en WSL sin systemd.
- **macOS**: un LaunchAgent, la forma que tiene macOS de arrancar un programa al entrar.
- **Windows**: una tarea del Programador de tareas que arranca al entrar, sin ventana.

En macOS y Windows arranca **cuando entras en tu sesión**, no antes: tras reiniciar, entra
una vez y vuelve. Ninguno de los dos deja a un usuario normal arrancar algo sin que nadie
haya entrado, y augmented brain nunca pide un administrador.

## Una puerta por espacio

Cada espacio (trabajo, personal…) tiene su propia puerta, con su propio nombre:
`<nombre del ordenador>-<espacio>`, como `mi-portatil-personal`. Una sesión abierta por esa
puerta empieza en una carpeta que es de ese espacio, así que Claude carga las notas y las
reglas de ese espacio, y nada se cruza entre espacios. Por defecto solo tu espacio por
defecto tiene puerta. Para abrir otra para trabajo, mira [Configuración](configuration.es.md).

Esa carpeta es la primera carpeta de trabajo del espacio, la que pusiste al crearlo: donde
ya trabajas, así que una sesión abierta desde el móvil empieza donde la abrirías delante
del ordenador. Para que empiece en otra carpeta dentro del espacio, usa `folders` en
[`remote.toml`](configuration.es.md). Solo un espacio sin carpetas de trabajo tiene una
carpeta propia, `~/.config/augmented-brain/remote/<espacio>`.

## Instálalo

```bash
augmented-brain remote install
```

Comprueba que el sistema puede mantener el servicio, lo escribe y lo arranca. Si no
puede, dice por qué y no cambia nada. Luego te dice lo que falta, si falta algo: una cosa
que solo puedes hacer tú, dos en Linux.

### 1. Que Claude confíe en la carpeta

Claude se niega a trabajar en una carpeta de la que no le han dicho que se fíe, y cuando
arranca solo no hay nadie al teclado para contestar. Si ya usas Claude en tu carpeta de
trabajo, ya confía en ella y no hay nada que hacer. Si no, una vez por cada carpeta que
nombre `install`, abre Claude ahí y acepta la pregunta:

```bash
cd ~/proyectos && claude
```

En Windows, en PowerShell, `install` lo escribe como `cd "C:\Users\tu\proyectos"; claude`.
Acepta «confiar en esta carpeta» y cierra Claude (`/exit`). augmented brain nunca toca por
ti los ajustes de Claude. El servicio lo reintenta solo (cada 30 segundos, cada minuto en
Windows), así que en menos de un minuto está en marcha.

### 2. En Linux, activa el «linger»

Un servicio de usuario normalmente solo funciona mientras tienes la sesión abierta. Para
que esté tras reiniciar, sin que nadie haya entrado, hay que decirle al sistema que
conserve tus servicios: se llama **linger** y lo tiene que hacer un administrador.
`install` escribe el comando exacto y no lo lanza:

```bash
sudo loginctl enable-linger tu-usuario
```

macOS y Windows no tienen linger: ahí el servicio arranca cuando entras en tu sesión.

## Compruébalo

```bash
augmented-brain remote status
```

Para cada espacio dice si el servicio está en marcha, si Claude confía en su carpeta y,
en Linux, si el linger está activado, con lo que hacer con cada problema.
`augmented-brain doctor` enseña lo mismo, una línea por espacio.

Cuando un servicio falla, dice por qué. En Linux, en el registro del sistema (el journal):

```bash
journalctl --user -u augmented-brain-remote-personal -n 20
```

En macOS, en `~/Library/Logs/augmented-brain/augmented-brain-remote-<espacio>.log`; en
Windows, en `augmented-brain-remote-<espacio>.log` dentro de
`%LOCALAPPDATA%\augmented-brain\remote`. `status` dice el fichero.

Un servicio instalado por una versión anterior no guardaba sus errores en ninguna parte;
`status` y `doctor` te dicen que vuelvas a lanzar `augmented-brain remote install`, que lo
pone al día.

## Conéctate desde otro ordenador

1. Abre [claude.ai/code](https://claude.ai/code) o Claude Desktop, con la misma cuenta de
   Claude que en la máquina.
2. Elige el entorno que se llama `<nombre del ordenador>-<espacio>`, el nombre que enseña
   `status`.
3. Empieza una sesión. Empieza las que quieras: cada una arranca en la carpeta de ese
   espacio.

## Una sesión por tarea

Esta parte es solo para Linux por ahora: solo se ha probado ahí. En macOS y Windows
`session start` dice que no está disponible.

Las sesiones de la puerta empiezan todas en la carpeta del espacio. Para trabajar en una
tarea en su propia sesión, en su carpeta, pídeselo a la sesión en la que estás: «ábreme una
sesión para la tarea de la web». Toda sesión en Linux sabe hacerlo, aquí o desde otro
ordenador, porque el inicio de cada una se lo cuenta. Lanza

```bash
augmented-brain session start task-new-website
```

y en menos de un minuto sale en claude.ai y Claude Desktop una sesión nueva, con el nombre
de la tarea. Empieza sola: lee su tarea, te dice cómo está y se pone con el siguiente
paso. Cuando para, avisa en una línea a la sesión que la pidió. La carpeta es la del primer repositorio de la tarea que encuentra en
las carpetas de trabajo del espacio, o la que diga `--folder`; sin repositorio, la carpeta
del espacio. Para un repositorio bare, la
tarea recibe un worktree de git propio, en una rama nueva desde la rama por defecto de
origin. Se va solo cuando no se pierde nada: con `session stop`, al cerrar la tarea o, si
la sesión terminó de otra forma (un reinicio, cerrarla desde Claude Desktop), en el
próximo `session start` o `doctor`. Uno con cambios sin commitear o sin subir se queda, y
`doctor` lo nombra.

- **Claude tiene que confiar en esa carpeta ella misma.** Que confíe en la carpeta del
  espacio no basta. La primera vez, el comando dice qué hacer: en la máquina (o por
  `ssh`), entra en la carpeta, lanza `claude`, acepta la pregunta y ciérralo con `/exit`.
  Luego pídelo otra vez. En un worktree de git, Claude guarda la respuesta con su
  repositorio principal: basta con confiar en uno de los dos.
- **Una tarea por carpeta.** Dos tareas en una carpeta se pisarían los cambios, así que
  una carpeta que ya usa la sesión de otra tarea se rechaza. Dale otra con `--folder`
  (un worktree de git del repositorio, por ejemplo).
- **Termina con `augmented-brain session stop <tarea>`**, que también la quita de Claude
  Desktop y claude.ai, al cerrar su tarea (cuando termina su respuesta) o al reiniciar la
  máquina. Tras un reinicio solo vuelven las
  puertas: pide otra vez las sesiones de las tareas. `augmented-brain session list`
  enseña las abiertas.
- **Claude retira una sesión que lleva horas sin uso.** Deja de correr, pero sigue en
  Claude Desktop y claude.ai, y vuelve al abrirla allí. `session list` la llama
  retirada; `session stop` y cerrar la tarea la cierran igual.
- **Una sesión que no puede trabajar no se queda abierta.** `session start` comprueba
  antes que Claude Code tiene la sesión iniciada, y luego espera la primera respuesta
  (hasta un minuto). Si esa respuesta es un error, como un login caducado, cierra la
  sesión y dice qué hacer. Una sesión cuya respuesta falla más tarde sale como fallida
  en `session list`.

## Mantenlo seguro

**Es una puerta permanente a tu ordenador.** Quien entre en tu cuenta de Claude puede
trabajar en la máquina por ella.

- Claude pregunta antes de actuar, como siempre (modo de permisos `default`), y el
  guardia sigue vigente: frena los push, root y los secretos, y pregunta antes de
  producción.
- **No pongas `permission_mode` en `bypassPermissions`.** Deja que quien se conecte
  ejecute lo que quiera sin preguntar, y deja la máquina abierta.
- Protege tu cuenta de Claude: una contraseña fuerte y verificación en dos pasos.
- Quita la puerta cuando no la necesites: `augmented-brain remote uninstall`.

## Conviene saber

- **Un portátil que se duerme corta la conexión.** La sesión vuelve cuando despierta. El
  programa no cambia tus ajustes de suspensión.
- **En macOS y Windows, tras reiniciar queda apagado hasta que entras.** Si el ordenador
  se reinicia solo (una actualización, un corte de luz), nadie puede conectarse hasta que
  alguien entra en la sesión.
- Si caduca tu acceso a Claude, el servicio sigue en marcha pero nadie puede conectarse.
  `status` y `doctor` lo dicen; lanza `claude` una vez e inicia sesión otra vez.
- Las sesiones que empieces desde el móvil son del espacio de su carpeta: la puerta de
  trabajo abre las notas de trabajo. Para llegar a otro espacio, usa la puerta de ese
  espacio.

## Quitarlo

```bash
augmented-brain remote uninstall
```

Para y quita los servicios (en Windows, también el programa que tenían en marcha). Las
carpetas, y su sitio entre las carpetas de trabajo del
espacio, se quedan, y no se borra nada más. Bórralas a mano si quieres.

## Los ajustes

`remote.toml`, en la carpeta de configuración: `enabled`, `spaces`, `name_prefix`,
`permission_mode` y `folders`. Mira [Configuración](configuration.es.md). Los comandos están en
[Comandos](commands.es.md).
