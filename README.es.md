<p align="center">
  <img alt="augmented brain" src="assets/brand/out/header.svg" width="320">
</p>

<p align="center"><b>Un segundo cerebro para trabajar con asistentes de IA.</b><br>
Tareas, áreas y reglas que tu asistente recuerda de una sesión a otra.</p>

<p align="center"><a href="README.md">Read in English</a></p>

---

> **Estado: 0.0.1, en diseño.** Todavía no hay nada que instalar. Este repositorio
> recoge el objetivo del proyecto, su diseño y su marca. Síguelo o abre un issue con ideas.

## El problema

Los asistentes de IA para programar funcionan muy bien dentro de una sesión y lo
olvidan todo al cerrarla. Vuelves a explicar el mismo contexto, el trabajo a medias se
pierde entre sesiones y nada impide que el asistente lance un comando arriesgado en la
máquina equivocada.

## Qué hace augmented brain

Le da a tu asistente memoria y una forma de trabajar, guardadas en ficheros Markdown
normales que son tuyos.

- **Tareas con ciclo de vida.** Todo trabajo pasa por los mismos pasos: definir,
  analizar, planificar, ejecutar (con diario), validar y cerrar. Puedes aparcar una
  tarea y retomarla días después, en otra sesión, justo donde la dejaste.
- **Áreas.** Las responsabilidades que no terminan (tu plataforma, tu proyecto propio)
  agrupan sus tareas y lo aprendido, para que empieces siempre por el mapa correcto.
- **Reglas que se cumplen.** Cómo te gusta trabajar se carga al empezar cada sesión. Las
  importantes se hacen cumplir, no solo se piden: nada de commit ni push sin que lo
  digas, nada de root, ningún secreto escrito en comandos ni ficheros.
- **Seguro en servidores y en la nube.** Las mismas protecciones valen por `ssh`. Los
  perfiles de AWS de producción piden confirmación antes.
- **Trabajo y personal, separados.** Cada *espacio* tiene su memoria, sus reglas y su
  copia de seguridad. La carpeta en la que abres el asistente elige el espacio.
- **Copia de seguridad y restauración.** Si pierdes el portátil, lo recuperas todo en
  uno nuevo.
- **Tus herramientas.** Guía para conectar Google Calendar, Gmail, Drive, Jira y otros
  servicios por MCP.

## Cómo funcionará

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
augmented-brain init
```

`init` hace tres preguntas (idioma, dónde guardar tu memoria y qué asistente usas) y lo
deja todo listo. Luego abres tu asistente y dices *"nueva tarea: …"*.

- **Un solo binario**, escrito en Go. Sin Python, Node ni nada más que instalar.
- **Linux y macOS** primero; Windows, después.
- **Claude Code** primero. El método vive en un servidor MCP, así que otros asistentes
  también podrán usarlo.
- **En inglés y en español.**

## Hoja de ruta

| Versión | Qué |
|---|---|
| 0.0.1 | Objetivo, diseño y marca (esta versión) |
| 0.1 | Formato de la memoria, `init` y `doctor`, plugin de Claude Code con los hooks de seguridad |
| 0.2 | Servidor MCP con el ciclo de tarea, revisión semanal y copia / restauración |
| 0.3 | Web de documentación, instalador y primeros binarios públicos |

## Inspiración

*Building a Second Brain*, de Tiago Forte (el método PARA: proyectos, áreas, recursos y
archivo), y *Getting Things Done*, de David Allen (el buzón y la revisión semanal),
adaptados a trabajar con un asistente de IA.

## Desarrollo

Necesita Go 1.25 o posterior (un `go` más antiguo lo descarga solo).

```bash
go test ./...
go build ./cmd/augmented-brain
```

## El repositorio

- [`docs/spec/memory-format.md`](docs/spec/memory-format.md): cómo se guarda la memoria (borrador, en inglés).
- [`docs/brand.html`](docs/brand.html): logo, iconos y paleta, vistos en su sitio.
- [`assets/brand/`](assets/brand/): las fuentes de la marca y el script que genera cada recurso.
- [`scripts/privacy-check.sh`](scripts/privacy-check.sh): bloquea los commits que publicarían datos personales o secretos.

## Licencia

MIT. Ver [LICENSE](LICENSE).
