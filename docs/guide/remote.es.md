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
defecto tiene puerta. Para abrir otra para trabajo, pon los dos espacios en `remote.toml`,
en la carpeta de configuración, y vuelve a lanzar `augmented-brain remote install`:

```toml
spaces = ["personal", "work"]
```

Esa carpeta es la primera carpeta de trabajo del espacio, la que pusiste al crearlo: donde
ya trabajas, así que una sesión abierta desde el móvil empieza donde la abrirías delante
del ordenador. Para que empiece en otra carpeta dentro del espacio, nómbrala en
`remote.toml`:

```toml
folders = { work = "~/code/clients" }
```

Solo un espacio sin carpetas de trabajo tiene una carpeta propia, `remote/<espacio>`
dentro de la carpeta de configuración (mira [Dónde vive cada cosa](where-things-live.es.md)).
Todos los ajustes de `remote.toml` están en [Configuración](configuration.es.md#remotetoml).

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

## Compruébalo

```bash
augmented-brain remote status
```

Para cada espacio dice si el servicio está en marcha, si Claude confía en su carpeta y,
en Linux, si el linger está activado, con lo que hacer con cada problema.

En Linux, `install` prepara también un **vigilante**: un temporizador pequeño que mira
los servicios cada dos minutos y trae de vuelta uno cuya sesión archivó el servidor (mira
«Conviene saber», abajo). `status` dice cómo está: si está en marcha y cuánto hace que
miró por última vez. `augmented-brain doctor` enseña lo mismo, una línea por espacio, y
una más para el vigilante solo cuando falla.

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

Las sesiones de la puerta empiezan todas en la carpeta del espacio. En Linux, una tarea
puede tener además una sesión propia, en su propia carpeta: pídeselo a la sesión en la
que estás, «ábreme una sesión para la tarea de la web». Mira
[Una sesión por tarea](task-sessions.es.md).

## Conviene saber

- **Un portátil que se duerme corta la conexión.** La sesión vuelve cuando despierta. El
  programa no cambia tus ajustes de suspensión.
- **En macOS y Windows, tras reiniciar queda apagado hasta que entras.** Si el ordenador
  se reinicia solo (una actualización, un corte de luz), nadie puede conectarse hasta que
  alguien entra en la sesión.
- Si caduca tu acceso a Claude, el servicio sigue en marcha pero nadie puede conectarse.
  `status` y `doctor` lo dicen; lanza `claude` una vez e inicia sesión otra vez.
- **Si el servidor archiva la sesión** (porque la archivaste en claude.ai, por ejemplo),
  el servicio sigue en marcha sin nadie detrás. En Linux el vigilante lo trae de vuelta
  solo, con una sesión nueva y vacía del mismo nombre, en unos diez minutos (mira
  [`remote watch`](commands.es.md#augmented-brain-remote-watch)).
- Las sesiones que empieces desde el móvil son del espacio de su carpeta: la puerta de
  trabajo abre las notas de trabajo. Para llegar a otro espacio, usa la puerta de ese
  espacio.

### Una sesión archivada en macOS y Windows

Ahí no hay vigilante, y volver a instalar no basta: Claude guarda una nota de la sesión
archivada, `bridge-pointer.json`, y un servicio nuevo la vuelve a coger y la pierde al
momento. Aparta esa nota a mano:

1. Para las puertas:

   ```bash
   augmented-brain remote uninstall
   ```

2. En `.claude/projects`, dentro de tu carpeta personal, abre la carpeta cuyo nombre
   acaba en el nombre de la carpeta de la puerta (Claude la nombra con la ruta entera,
   con `-` en lugar de cada separador). Cambia el nombre de `bridge-pointer.json` a
   `bridge-pointer.json.archived`. No lo borres.
3. Arranca otra vez las puertas:

   ```bash
   augmented-brain remote install
   ```

En pocos minutos sale una sesión nueva y vacía del mismo nombre: antes, el lado de Claude
suelta la vieja. La conversación de la sesión archivada no vuelve.

## Quitarlo

```bash
augmented-brain remote uninstall
```

Para los servicios y los quita, y en Linux también el vigilante. En Windows, además,
cierra los programas que esos servicios habían arrancado. Las carpetas, y su sitio entre
las carpetas de trabajo del espacio, se quedan, y no se borra nada más. Bórralas a mano
si quieres.

## Los ajustes

`remote.toml`, en la carpeta de configuración: `enabled`, `spaces`, `name_prefix`,
`permission_mode` y `folders`. Mira [Configuración](configuration.es.md#remotetoml). Los
comandos están en [Comandos](commands.es.md).
