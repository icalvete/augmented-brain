# Actualizar y desinstalar sin perder nada

[Read in English](update-and-uninstall.md)

Tus notas son ficheros normales y son tuyas. Actualizar nunca toca lo que dicen, y
desinstalar las deja donde están salvo que pidas lo contrario. Esta página dice qué hace
cada paso, para que lo hagas con calma.

## Actualizar

Cuando sale una versión nueva, Claude te lo dice al empezar una sesión. Para actualizar:

```bash
augmented-brain update
```

Qué hace, por orden:

1. **Revisa la instalación** y enseña lo que encuentra, para que un problema que ya estaba
   no se tome por uno de la versión nueva.
2. **Hace la copia de seguridad de cada espacio que la tenga.** Si falla una, se para y no
   actualiza nada: esa copia es tu vuelta atrás. Si no hay ninguna, dice que sigue sin
   copia.
3. **Descarga e instala** la versión nueva y el plugin de Claude Code. Si GitHub no entrega
   los ficheros, lo intenta tres veces, también por la API de GitHub; si falla todo, lo
   dice y no cambia nada.
4. **Comprueba que la versión nueva está puesta**, y dice qué sesiones de Claude abiertas
   siguen con la vieja, y qué tarea está activa en cada espacio.

Después pide a esas sesiones que aparquen su tarea, y cierra y abre Claude. Una sesión que
estaba abierta sigue con el programa viejo hasta que lo hagas.

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

Enseña lo que va a quitar y pregunta. Hay tres niveles:

| Comando | Quita | Deja |
|---|---|---|
| `uninstall` | El programa y el plugin de Claude Code | Tus notas y la configuración |
| `uninstall --config` | También la configuración y tu perfil | Tus notas |
| `uninstall --all` | También las notas de todos los espacios | Tus copias de seguridad |

- **Para volver**, instala otra vez: con la configuración ahí, lo encuentra todo como
  estaba; sin ella, vuelve a crear los espacios encima de tus notas, sin perder ninguna.
- **`--all` borra tus notas.** Enseña cada carpeta de notas con cuántas notas tiene, y
  avisa de las que no tienen copia de seguridad: se perderán para siempre. Responde que no,
  configura la copia o copia las carpetas, y vuelve a lanzarlo.
- **Tus copias de seguridad nunca se borran**, ni en GitHub ni en ningún sitio. Dice cuáles
  son y cómo borrar una, si quieres.
- Claude Code, su app de escritorio y git se quedan.

En Windows el fichero del programa desaparece un momento después de acabar el comando:
Windows no deja que un programa en marcha se borre a sí mismo.

## Pasarte a un ordenador nuevo

1. En el viejo, haz una última copia: `augmented-brain backup --all`.
2. En el nuevo, instala augmented brain y restaura cada espacio:
   `augmented-brain restore <dirección de su copia>`.
3. Da otra vez lo que no viaja: las listas del guardia, y el valor de los secretos
   (`restore` nombra los que le faltan a su llavero, con el comando para guardar cada uno).

Mira [Copia de seguridad y restauración](backup.es.md), y [Dónde vive cada
cosa](where-things-live.es.md) para lo que viaja y lo que no.
