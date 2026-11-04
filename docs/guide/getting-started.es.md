# Primeros pasos

[Read in English](getting-started.md)

## Qué es

augmented brain le da a tu asistente de IA una memoria, un método y un guardia. Hoy el
asistente es Claude Code.

- **Una memoria.** Tus tareas, decisiones, personas y reglas se guardan como ficheros de
  texto normal en tu ordenador. Cada sesión nueva empieza leyéndolos, así que nunca
  explicas dos veces lo mismo.
- **Un método.** Cada trabajo es una tarea que va de la idea a cerrada en seis fases.
  Puedes parar una tarea hoy y retomarla la semana que viene, en cualquier sesión.
- **Un guardia.** Antes de que Claude ejecute un comando, augmented brain lo lee. Subir
  código, hacerse root o escribir una contraseña se para. Tocar producción te pregunta
  antes.

No aprendes comandos nuevos. Hablas con Claude como siempre: "nueva tarea…",
"apárcala", "¿en qué estábamos?".

Es gratis y de código abierto (MIT). También es alfa: una versión temprana que puede
cambiar.

Esta guía es para empezar de cero, aunque nunca hayas abierto una terminal. Son unos diez
minutos.

## Lo que necesitas

- **Un ordenador con Linux, macOS o Windows.** En Mac, macOS 13 (Ventura) o posterior: lo
  ves en el menú de la manzana → Acerca de este Mac. En Windows, Windows 10 u 11. Lo que
  cambia en cada sistema está en [Instalación](install.es.md).
- **Una cuenta de Claude de pago**: Pro o superior. augmented brain es gratis, pero
  trabaja con Claude Code, y Claude Code no entra en el plan gratuito. Los planes, en
  [claude.com/pricing](https://claude.com/pricing).
- **La contraseña de tu ordenador**, por si hay que instalar algo que falta.

## 1. Abre la terminal

La terminal es una ventana en la que escribes órdenes en vez de pulsar botones. Solo la
necesitas para instalar.

- **En Mac:** pulsa `Cmd + Espacio`, escribe `Terminal` y pulsa Enter.
- **En Ubuntu:** pulsa `Ctrl + Alt + T`.
- **En Windows:** pulsa la tecla de Windows, escribe `PowerShell` y pulsa Enter.

## 2. Pega el comando de instalación

Copia esta línea, en Mac o Linux:

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
```

O esta, en Windows:

```powershell
irm https://github.com/icalvete/augmented-brain/releases/latest/download/install.ps1 | iex
```

Pégala en la terminal (`Cmd + V` en Mac, `Ctrl + Shift + V` en Ubuntu, clic derecho en
Windows) y pulsa Enter.

## 3. Contesta al instalador

Te hace unas pocas preguntas. Para aceptar lo que te propone, basta con pulsar Enter.

1. **El idioma.** Español o inglés.
2. **Lo que falte**, si falta algo: git (guarda el historial de tus notas), Claude Code
   y la app de escritorio de Claude. Te dice para qué es cada cosa antes de instalarla. Si
   ya tenías Claude Code pero está atrasado, te ofrece actualizarlo.
3. **¿Abro ahora la app de Claude?** Di que sí.

Si te pide la contraseña, escríbela aunque no veas nada: la terminal no la enseña, ni
siquiera con asteriscos. Pulsa Enter al acabar.

En Mac, si falta git, se abre una ventana para instalar las herramientas de desarrollo de
Apple. Pulsa **Instalar** y acepta. Tarda unos minutos; el instalador espera y sigue solo.

En Windows no hay contraseña que escribir: al instalar Git, Windows enseña una ventana que
pide permiso. Pulsa **Sí**.

## 4. Entra en Claude

En la app de Claude:

1. Entra con tu cuenta.
2. Abre la pestaña **Code**.
3. Pulsa **+ New** y escribe "Hola".

Sin elegir carpeta, trabajas en tu espacio personal. Para el de trabajo, elige con
**Choose…** la carpeta `~/augmented-brain/trabajo`. Un **espacio** es un cajón separado
de notas: te lo cuenta [Espacios](spaces.es.md).

## 5. Contesta a la entrevista

La primera vez, Claude se presenta y te hace seis preguntas, una cada vez, diciéndote
para qué sirve cada una: quién eres, qué cuidas, cómo quieres las respuestas, qué no se
toca, qué herramientas usas y con quién trabajas. Puedes saltarte cualquiera.

Al final te enseña lo que ha guardado y te ofrece montar la copia de seguridad. Dile que
sí: tus notas quedan en un repositorio privado tuyo, y si pierdes el ordenador no pierdes
nada.

## Tu primera semana

Háblale como siempre. Estas frases te sacan partido desde el primer día:

| Dices | Qué hace |
|---|---|
| "Nueva tarea: preparar la reunión del jueves." | Abre una tarea y te pregunta lo necesario para definirla |
| "Apárcala, mañana seguimos." | Apunta dónde lo dejáis y qué viene después |
| "¿En qué estábamos?" | Te lista tus tareas vivas con su siguiente paso |
| "Seguimos con la reunión del jueves." | Retoma la tarea donde la dejaste, en cualquier sesión |
| "Apunta: llamar al seguro." | Lo deja en el buzón, sin abrir tarea |
| "Vamos con el proveedor B, que es más barato." | Lo apunta en el diario de la tarea, con fecha |
| "Guárdalo como lección." | Lo aprendido queda en una nota propia |
| "Cierra la tarea." | La cierra; no se borra nunca |
| "Lucía es mi jefa, la llamo Lu." | La guarda, y la próxima vez sabe quién es |
| "¿Por qué elegimos el proveedor B?" | Lo busca en tus notas y te da la decisión, con fecha y porqué |

Más frases, agrupadas por lo que quieres hacer: [Qué decirle a Claude](phrases.es.md).

## Palabras que vas a oír

- **Espacio:** un cajón separado, con sus notas y sus reglas. Vienen dos, `trabajo` y
  `personal`. La carpeta en la que abres Claude decide en cuál estás.
- **Tarea:** algo con principio y fin. Pasa por seis fases, de la idea a cerrada.
- **Área:** algo que cuidas siempre (la web, la casa, un equipo). Agrupa sus tareas.
- **Buzón:** donde van las ideas sueltas hasta que las ordenas.
- **Guardia:** el control que para o te pregunta antes de un comando arriesgado.
- **Copia de seguridad:** tus notas, subidas a un repositorio git privado tuyo después
  de cada sesión.
- **Plugin:** la pieza que conecta augmented brain con Claude Code. La instala el
  instalador.
- **Panel:** una página en tu navegador para ver y cambiar todo sin pedírselo a Claude.

## Si algo falla

- **Claude no se presenta ni te hace la entrevista.** Empieza una sesión nueva. Si sigue
  igual, abre la terminal y escribe `augmented-brain doctor`: te dice qué falta y cómo
  arreglarlo.
- **La terminal dice que no encuentra una orden** (`command not found`, o "no se reconoce"
  en Windows). Cierra la terminal y abre otra: la nueva ya encuentra los programas recién
  instalados.
- **Claude te pide un plan de pago.** Claude Code necesita Pro o superior.
- **Otra cosa.** Mira [Si algo falla](troubleshooting.es.md).

## Por dónde seguir

- [Tareas](tasks.es.md): cómo avanza una tarea, y cómo aparcarla y retomarla.
- [El guardia](guard.es.md): qué para, qué pregunta y por qué.
- [El panel](panel.es.md): todo de un vistazo, en tu navegador.
- [Copia de seguridad y restauración](backup.es.md): para que perder el portátil no te
  haga perder nada.
