# Acceso remoto

[Read in English](remote.md)

A Claude Code se puede llegar desde otro ordenador o desde el móvil: desde
[claude.ai/code](https://claude.ai/code) o Claude Desktop eliges una sesión que corre en
tu máquina y trabajas en ella como si estuvieras delante. Para eso, un programa pequeño,
`claude remote-control`, tiene que estar funcionando en la máquina.

`augmented-brain remote` lo mantiene funcionando por ti. Arranca solo cuando arranca el
ordenador, y vuelve a arrancar si se para. Se prepara una vez.

Por ahora es para Linux con systemd (la parte de Linux que arranca y vigila los
servicios): no un contenedor, ni WSL sin systemd. macOS y Windows quedan pendientes.

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

Comprueba que systemd puede mantener el servicio, lo escribe y lo arranca. Si systemd no
puede, dice por qué y no cambia nada. Luego te dice lo que falta, si falta algo: dos cosas que
solo puedes hacer tú.

### 1. Que Claude confíe en la carpeta

Claude se niega a trabajar en una carpeta de la que no le han dicho que se fíe, y cuando
arranca solo no hay nadie al teclado para contestar. Si ya usas Claude en tu carpeta de
trabajo, ya confía en ella y no hay nada que hacer. Si no, una vez por cada carpeta que
nombre `install`, abre Claude ahí y acepta la pregunta:

```bash
cd ~/proyectos && claude
```

Acepta «confiar en esta carpeta» y cierra Claude (`/exit`). augmented brain nunca toca por
ti los ajustes de Claude. El servicio lo reintenta cada 30 segundos, así que en medio
minuto está en marcha.

### 2. Activa el «linger»

Un servicio de usuario normalmente solo funciona mientras tienes la sesión abierta. Para
que esté tras reiniciar, sin que nadie haya entrado, hay que decirle al sistema que
conserve tus servicios: se llama **linger** y lo tiene que hacer un administrador.
`install` escribe el comando exacto y no lo lanza:

```bash
sudo loginctl enable-linger tu-usuario
```

## Compruébalo

```bash
augmented-brain remote status
```

Para cada espacio dice si el servicio está en marcha, si Claude confía en su carpeta y si
el linger está activado, con lo que hacer con cada problema. `augmented-brain doctor`
enseña lo mismo, una línea por espacio.

Cuando un servicio falla, dice por qué en el registro del sistema (el journal):

```bash
journalctl --user -u augmented-brain-remote-personal -n 20
```

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
- Si caduca tu acceso a Claude, el servicio sigue en marcha pero nadie puede conectarse.
  `status` y `doctor` lo dicen; lanza `claude` una vez e inicia sesión otra vez.
- Las sesiones que empieces desde el móvil son del espacio de su carpeta: la puerta de
  trabajo abre las notas de trabajo. Para llegar a otro espacio, usa la puerta de ese
  espacio.

## Quitarlo

```bash
augmented-brain remote uninstall
```

Para y quita los servicios. Las carpetas, y su sitio entre las carpetas de trabajo del
espacio, se quedan, y no se borra nada más. Bórralas a mano si quieres.

## Los ajustes

`remote.toml`, en la carpeta de configuración: `enabled`, `spaces`, `name_prefix`,
`permission_mode` y `folders`. Mira [Configuración](configuration.es.md). Los comandos están en
[Comandos](commands.es.md).
