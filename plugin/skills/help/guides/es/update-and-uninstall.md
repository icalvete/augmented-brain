# Actualizar y desinstalar sin perder nada


Tus notas son ficheros normales y son tuyas. Actualizar nunca toca lo que dicen, y
desinstalar las deja donde están salvo que pidas lo contrario. Esta página dice qué hace
cada paso, para que lo hagas con calma.

## Actualizar

Cuando sale una versión nueva, Claude te lo dice al empezar una sesión y se ofrece a
actualizarlo. También puedes hacerlo tú:

```bash
augmented-brain update
```

Qué hace, por orden:

1. **Revisa la instalación** y enseña lo que encuentra, para que un problema que ya estaba
   no se tome por uno de la versión nueva.
2. **Hace la copia de seguridad de cada espacio que la tenga.** Si falla una, se para y no
   actualiza nada: esa copia es tu vuelta atrás. `--force` actualiza igualmente. Si no
   hay ninguna, dice que sigue sin copia.
3. **Descarga e instala** la versión nueva, con el plugin de Claude Code que trae dentro.
   Pregunta a GitHub cuál es la última versión una sola vez: si GitHub no responde, lo
   dice y se para. La descarga se intenta tres veces, también por la API de GitHub. Si
   falla todo, o el programa no es la versión nueva al acabar, lo dice y no cambia nada:
   vuelve a lanzarlo más tarde. Si no hay versión nueva, solo actualiza el plugin.
4. **Reinicia las sesiones de Claude quietas** que siguen con la versión vieja, una a una
   y con la misma conversación. Una sesión que está respondiendo se queda como está.
   - **La sesión de una tarea** (abierta con `session start`) recibe un proceso nuevo.
   - **Una sesión del acceso remoto** (`remote install`): se reinicia su servicio. Solo
     en Linux; en macOS y Windows cuenta como una de Claude Desktop.
   - **Una sesión de Claude Desktop o de una terminal** sigue abierta, y solo se reinicia
     su augmented brain. En Windows se queda como está.
5. **Dice lo que queda:** las sesiones que no ha podido reiniciar (una que respondía, la
   desde la que lo lanzaste, en Windows cualquiera de Claude Desktop) y qué tarea está
   activa en cada espacio.

Para las que quedan, pide a la sesión que aparque su tarea, y ciérrala y ábrela. Una
sesión que estaba abierta sigue con el programa viejo hasta que se reinicia, y sus
herramientas de augmented brain pueden fallar; `/clear` no basta. `augmented-brain
doctor` y el inicio de una sesión nueva dicen cuántas siguen con la versión vieja. Más
tarde, `augmented-brain session restart --old` reinicia las que ya estén quietas.

Otras formas:

- `augmented-brain update --no-restart` actualiza sin reiniciar ninguna sesión.
- `augmented-brain update --plugin` actualiza solo el plugin de Claude Code.
- Volver a lanzar el [comando de instalación](install.md) también actualiza, sin los
  pasos 1, 2, 4 y 5.

### Antes de actualizar

- **Configura una copia de seguridad** si no tienes: dile a Claude «configura la copia de
  seguridad». `augmented-brain doctor` dice qué espacios no la tienen.
- **Actualiza todos los ordenadores que uses.** Una versión antigua nunca rompe notas que
  escribió una nueva: las rechaza y pide que la actualices.

### Cuando una versión cambia cómo se guardan las notas

A veces una versión nueva guarda las notas de otra forma. Entonces la primera sesión las
pasa a la forma nueva, después de guardar una copia completa en la carpeta `migrations/`
de la configuración, y Claude te lo dice. Nada se pasa sin esa copia.

## Desinstalar

```bash
augmented-brain uninstall
```

Enseña lo que va a quitar y pregunta. Primero para lo que seguiría funcionando sin él:
los servicios del [acceso remoto](remote.md) y las sesiones abiertas para tareas
(`session start`). Sin el plugin seguirían sin guardia, y sin el programa que sabe
quitarlos. Si alguno no se puede parar, no quita nada más: te dice por qué, para que lo
arregles y lo vuelvas a lanzar.

Después hay tres niveles:

| Comando | Quita | Deja |
|---|---|---|
| `uninstall` | El programa, el plugin de Claude Code y lo que Claude Code guarda de él (su caché, sus datos y los registros de su servidor) | Tus notas y la configuración |
| `uninstall --config` | También la configuración y tu perfil | Tus notas |
| `uninstall --all` | También las notas de todos los espacios | Tus copias de seguridad |

- **Para volver**, instala otra vez: con la configuración ahí, lo encuentra todo como
  estaba; sin ella, vuelve a crear los espacios encima de tus notas, sin perder ninguna.
- **`--all` borra tus notas.** Enseña cada carpeta de notas con cuántas notas tiene, y
  avisa de las que no tienen copia de seguridad: se perderán para siempre. Responde que no,
  configura la copia o copia las carpetas, y vuelve a lanzarlo.
- **Solo quita lo que guarda augmented brain** en una carpeta de notas (las carpetas de
  notas, `MEMORY.md`, `inbox.md`, `docs/`, `archive/`), y la carpeta solo si no queda
  nada más. Una carpeta de notas que contiene una carpeta de trabajo, tu carpeta personal
  o una carpeta sin notas se deja entera, y el plan lo dice. `--config` solo borra una
  carpeta que tenga un `config.toml`.
- **Tus copias de seguridad nunca se borran**, ni en GitHub ni en ningún sitio. Dice cuáles
  son y cómo borrar una, si quieres.
- Claude Code, su app de escritorio, git y la entrada del PATH se quedan: Claude Code
  vive en esa misma carpeta.

En Windows el fichero del programa desaparece un momento después de acabar el comando:
Windows no deja que un programa en marcha se borre a sí mismo.

## Pasarte a un ordenador nuevo

Haz una última copia en el viejo (`augmented-brain backup --all`), instala augmented
brain en el nuevo y restaura cada espacio desde su copia, también el espacio por defecto.
Los pasos, y lo que vuelves a configurar a mano, están en [Copia de seguridad y
restauración](backup.md#en-una-máquina-nueva). Lo que viaja y lo que no: [Dónde vive
cada cosa](where-things-live.md).
