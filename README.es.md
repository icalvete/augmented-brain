<p align="center">
  <img alt="augmented brain" src="assets/brand/out/header.svg" width="320">
</p>

<p align="center"><b>Tu cerebro, aumentado con IA.</b><br>
Tu asistente recuerda en qué estabais, trabaja con método y te pregunta antes de tocar lo delicado.<br>
Todo queda en ficheros de texto tuyos, con copia en tu propio git.</p>

<p align="center"><a href="https://icalvete.github.io/augmented-brain/es/">Web</a> ·
<a href="docs/guide/how-it-works.es.md">Cómo funciona</a> · <a href="README.md">Read in English</a></p>

<p align="center"><a href="https://github.com/sponsors/icalvete"><img alt="Patrocina" src="https://img.shields.io/badge/Patrocina-EA4AAA?style=for-the-badge&logo=githubsponsors&logoColor=white"></a></p>

> [!WARNING]
> **Alfa.** Es una versión temprana: puede tener errores, su formato y sus comandos
> pueden cambiar. Probado por una persona en Ubuntu, macOS y Windows 11. Lee el [aviso legal](DISCLAIMER.md#aviso-legal)
> antes de usarlo, y haz copia de lo importante.

---

Los asistentes de IA para programar funcionan muy bien dentro de una sesión y lo
olvidan todo al cerrarla. Vuelves a explicar el mismo contexto, el trabajo a medias se
pierde y nada impide un comando arriesgado en la máquina equivocada.

**augmented brain le da a tu asistente memoria y una forma de trabajar**, guardadas en
ficheros Markdown normales que son tuyos, y lo vigila mientras trabaja.

## Qué hace en el día a día

Le hablas como siempre. Esto es lo que cambia:

| Tú | Qué pasa |
|---|---|
| "¿En qué estábamos?" | Te lista tus tareas vivas, cada una con su siguiente paso y quién la bloquea |
| "Apárcalo, mañana seguimos." | Apunta dónde lo dejáis y qué viene después; mañana, en otra sesión, arranca ahí |
| "Apunta: renovar el certificado en noviembre." | Va al buzón sin abrir tarea; lo ordenáis en la revisión semanal |
| "Recuérdame el jueves que llame al seguro." | El jueves, nada más abrir Claude, te lo recuerda |
| "¿Por qué elegimos Postgres?" | Lo busca en tus notas y te da la decisión, con fecha y porqué |
| "Guárdalo como lección." | Lo aprendido queda en una nota propia, para la próxima vez |
| "Borra el bucket viejo de producción." | Antes de ejecutarlo, te pregunta |
| Abres Claude en `~/augmented-brain/personal` | Solo ve tus notas personales; las del trabajo, no |
| "Dile a Marcos que lo dejamos para el lunes." | Sabe que Marcos es tu jefe y te prepara el mensaje |
| Se te rompe el portátil | `augmented-brain restore` en el nuevo y lo tienes todo |

## Empieza en dos pasos

Necesitas Linux, macOS (13 o posterior) o Windows (10 u 11), y una cuenta de Claude de pago
(Pro o superior), porque Claude Code no entra en el plan gratuito. augmented brain no
necesita root ni administrador. ¿Nunca has abierto una terminal? Sigue los
[primeros pasos](docs/guide/getting-started.es.md). Qué hace el instalador en cada sistema,
dónde deja cada cosa y cómo actualizar o desinstalar: [Instalación](docs/guide/install.es.md).

**1. Instálalo** con un solo comando. En Linux o en Mac, en la terminal:

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
```

En Windows, en PowerShell:

```powershell
irm https://github.com/icalvete/augmented-brain/releases/latest/download/install.ps1 | iex
```

Primero te pregunta el idioma. Si te falta git o Claude Code, te ofrece instalarlos, y
también la app de escritorio de Claude, la forma más cómoda de trabajar (en Mac, Windows,
Ubuntu y Debian). Después lo deja todo listo (dos espacios, `trabajo` y `personal`,
en `~/augmented-brain`, y el plugin de Claude Code) y te pregunta si abre Claude ya.

**2. Contesta a Claude.** La primera vez te hace seis preguntas rápidas, explicándote para
qué sirve cada una: quién eres, qué cuidas, cómo quieres las respuestas, qué no se toca,
qué herramientas usas y con quién trabajas. Con eso crea tus áreas, configura el guardia y
aprende quién es quién.

Las próximas veces, abre la app de Claude, pulsa **+ New** y escribe. Sin elegir carpeta,
estás en tu espacio personal; para el de trabajo, elige con **Choose…** la carpeta
`~/augmented-brain/trabajo`. Desde la terminal:

```bash
cd ~/augmented-brain/trabajo && claude
```

(En Windows, `cd ~\augmented-brain\trabajo; claude`.)

Si algo no va, `augmented-brain doctor` te dice qué falta y cómo arreglarlo.

## Cómo se ve

> **Tú:** Nueva tarea: añadir un formulario de contacto a la web.
>
> **Claude:** Abierta *task-formulario-de-contacto*, en definición. ¿Qué campos debe tener?
>
> *…acordáis nombre, email y mensaje, y lo dejáis por hoy…*
>
> **Tú:** Apárcala, mañana seguimos con el plan.
>
> **Claude:** Aparcada. Siguiente: escribir el plan.

Al día siguiente, en una **sesión nueva**:

> **Tú:** Seguimos con el formulario de contacto.
>
> **Claude:** Está aparcada con la definición acordada: nombre, email y mensaje, enviado
> por correo. Lo siguiente es el plan. La carpeta del proyecto sigue vacía: ¿dónde está
> el código de la web?

Sin repetir contexto. El asistente leyó la tarea y comprobó el estado real antes de
fiarse de ella.

## Qué te da

- **Tareas con ciclo de vida**: definir, analizar, planificar, ejecutar con diario,
  validar y cerrar. Aparcas una tarea y la retomas días después, en cualquier sesión.
- **Áreas** que agrupan las tareas y el conocimiento de cada responsabilidad.
- **Reglas cargadas en cada sesión**, y las importantes vigiladas por un guardia: nada
  de commit ni push salvo que lo permitas, nunca root, ningún secreto en comandos ni
  ficheros. Los perfiles de AWS de producción y los hosts que marques como intocables
  te piden confirmación, también por `ssh`.
- **Trabajo y personal, separados**: cada *espacio* tiene su memoria, sus reglas y su
  copia de seguridad. La carpeta en la que abres el asistente elige el espacio.
- **Copia de seguridad y restauración** en un repositorio git privado tuyo, uno por
  espacio. Abajo, qué te da.
- **Tus servicios** (calendario, correo, Jira…) por MCP: mira
  [Conecta tus servicios](docs/guide/connect-services.es.md).

## Pierdes el portátil, no el cerebro

La carpeta de notas de cada espacio es un repositorio git, y la copia la sube a un
repositorio privado tuyo. **Para montarla, díselo a Claude:** "monta la copia de
seguridad". Crea el repositorio privado en tu GitHub, lo conecta y hace la primera copia.
O, con otro repositorio:

```bash
augmented-brain backup setup git@gitlab.com:tu-usuario/mi-cerebro.git   # una vez
augmented-brain backup                                                   # a mano, si quieres
```

- **Una máquina nueva en minutos.** Instala augmented brain y lanza
  `augmented-brain restore <url>`: sustituye el espacio vacío por el tuyo, con notas,
  tareas, reglas, tu perfil y su configuración. Paso a paso, en
  [Cómo funciona](docs/guide/how-it-works.es.md#en-una-máquina-nueva).
- **Varias máquinas, un solo cerebro.** Antes de subir, trae lo que hizo tu otra máquina
  y lo mezcla.
- **Todo el historial.** Cada copia es un commit, con la máquina y la hora: ves qué
  cambió y vuelves a cualquier día.
- **En tu sitio.** GitHub, GitLab o tu servidor. augmented brain no tiene nube.
- **Un repositorio por espacio.** El del trabajo puede ir al git de la empresa.
- **No sube secretos.** Si una nota parece tener una contraseña o un token, se para.
- **No pisa nada.** Si dos máquinas cambiaron la misma nota, se para y te explica qué hacer.

Una vez montada, **se hace sola cada vez que terminas una sesión** de Claude; si falla,
Claude te lo dice al empezar la siguiente.

## Comandos

| Comando | Qué hace |
|---|---|
| `augmented-brain init` | Lo prepara todo: los espacios `trabajo` y `personal` y el plugin. Lo lanza el instalador; `--custom` para elegir tú |
| `augmented-brain space add` | Añade otro espacio (personal, un cliente…) |
| `augmented-brain doctor` | Dice qué falta y cómo arreglarlo |
| `augmented-brain check` | Revisa las notas del espacio de esta carpeta y rehace su índice |
| `augmented-brain backup setup <url>` | Conecta este espacio a un repositorio git privado |
| `augmented-brain backup` | Guarda los últimos cambios (`--all` para todos los espacios) |
| `augmented-brain restore <url>` | Recupera un espacio en una máquina nueva |
| `augmented-brain update` | Actualiza augmented brain y su plugin de Claude Code (Claude te avisa cuando hay versión nueva) |
| `augmented-brain uninstall` | Quita el programa y el plugin; tus notas se quedan (`--all` para empezar de cero) |

El trabajo de cada día ocurre dentro de Claude Code, con tus palabras: "nueva tarea…",
"¿qué tengo pendiente?", "apárcala", "apunta esto en el buzón".

## Cómo funciona

- **Tu memoria es Markdown**, una carpeta por espacio: `tasks/`, `areas/`,
  `knowledge/`, `rules/`. Ábrela y edítala con cualquier editor. El formato está en
  [docs/spec/memory-format.md](docs/spec/memory-format.md) (en inglés).
- **Un solo programa**, `augmented-brain`, escrito en Go. Nada más que instalar.
- **Un servidor MCP** da a tu asistente las herramientas del método, así que otros
  asistentes también pueden usarlo. Claude Code es el primero.
- **Un plugin de Claude Code** carga tu espacio al empezar cada sesión y pasa el guardia
  antes de cada comando y cada escritura de fichero. El guardia es una red contra
  errores, no una barrera de seguridad.

Las cuatro capas, una sesión paso a paso y dónde vive cada fichero:
[Cómo funciona](docs/guide/how-it-works.es.md).

Inspirado en *Building a Second Brain*, de Tiago Forte, y *Getting Things Done*, de
David Allen, adaptados a trabajar con un asistente de IA.

## Desarrollo

Necesita Go 1.25 o posterior (un `go` más antiguo lo descarga solo). ¿Trabajas con un
agente de IA? Su manual es [AGENTS.md](AGENTS.md) (en inglés): el mapa del código, las
comprobaciones y las reglas de la casa.

```bash
go test ./...
go build ./cmd/augmented-brain
```

`scripts/e2e-claude.sh` prueba la historia entera (abrir, aparcar, retomar y cerrar una
tarea, y el guardia) con un Claude Code de verdad y aislado. Gasta uso: lánzalo antes de
cada versión.

La marca (logo, iconos y paleta) está en [`assets/brand/`](assets/brand/), y se ve en su
sitio en [`docs/brand.html`](docs/brand.html).
[`scripts/privacy-check.sh`](scripts/privacy-check.sh) bloquea los commits que
publicarían datos personales o secretos.

## Apoya el proyecto

augmented brain es libre y gratis (licencia MIT). Si te ahorra tiempo, puedes
[patrocinar su desarrollo](https://github.com/sponsors/icalvete): paga el tiempo de responder incidencias y añadir funciones. También hay soporte de pago
(respuesta por correo en dos días laborables) y una sesión de puesta en marcha de una hora por videollamada.

## Licencia

MIT. Ver [LICENSE](LICENSE) y el [aviso legal](DISCLAIMER.md#aviso-legal). Sin
relación con Anthropic; "Claude" y "Claude Code" son marcas de Anthropic, PBC.
