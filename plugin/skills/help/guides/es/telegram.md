# Telegram (tu cuenta)


Tu asistente puede usar tu propia cuenta de Telegram. Encuentra un chat, lee sus últimos
mensajes y escribe desde tu cuenta. Pide «¿qué ha dicho Lucía en el grupo de la
familia?» y lee el grupo por ti.

Funciona en Linux y macOS. Windows aún no está soportado.

## Qué es

augmented brain entra en tu cuenta como **un dispositivo más**, igual que Telegram
Desktop en un ordenador. No es un bot. Un bot es una cuenta aparte a la que la gente
tiene que escribir. Aquí eres tú: lo que manda el asistente llega como escrito por ti, y
ve los chats que tú ves.

En Telegram aparece como **augmented brain (<nombre del ordenador>)**, por ejemplo
«augmented brain (casa)», en **Ajustes → Dispositivos**. Puedes cerrarlo desde ahí
cuando quieras.

Es un programa aparte, `augmented-brain-telegram`. La biblioteca de Telegram haría
augmented brain tres veces más grande, así que solo lo descarga quien enciende Telegram
(unos 6 MB, una sola vez). Viene de la misma versión que tu augmented brain, se comprueba
con las sumas de verificación de esa versión y se queda junto al programa
`augmented-brain`. Desde entonces, `augmented-brain update` lo mantiene al día. Si el programa falta pero
el llavero aún guarda tu sesión (te conectaste con una versión compilada por ti, por
ejemplo), `update` lo vuelve a poner, y `augmented-brain telegram connect` también, sin
código QR.

## Qué necesitas

- **Linux o macOS.** En Linux, además, `secret-tool`, el comando del llavero del sistema
  (el **Secret Service**, que tienen GNOME y KDE). En Ubuntu y Debian viene en el paquete
  `libsecret-tools`. macOS ya trae su llavero.
- **Tu móvil con Telegram**, para escanear un código.
- **Tu propia aplicación de Telegram**: dos valores, `api_id` y `api_hash`, que Telegram
  te da en un minuto. Siguiente apartado.

Todo se queda en el llavero del sistema, nunca en un fichero: la aplicación y la sesión
(lo que te mantiene dentro). Se guarda bajo el servicio `augmented-brain`, con los
nombres `telegram-api-id`, `telegram-api-hash` y `telegram-session`.

## Tu propia aplicación, una vez

Todo programa que habla con Telegram como una cuenta necesita una aplicación. Cada
persona crea la suya: usar la aplicación de otro programa va contra las condiciones de
Telegram.

1. Abre [my.telegram.org](https://my.telegram.org) en el navegador.
2. Escribe tu número de teléfono en formato internacional, con su prefijo: `+34…` en
   España. El código llega **dentro de Telegram**, como un mensaje, no por SMS.
3. Entra en **API development tools**.
4. Crea la aplicación:
   - **App title**: `augmented brain`.
   - **Short name**: `abrain` (de 5 a 32 letras o números).
   - **URL**: déjala vacía.
   - **Platform**: Desktop.
5. Copia dos valores de la página: **api_id**, un número, y **api_hash**, 32 letras y
   números.

Si el formulario responde solo «ERROR», es una rareza conocida de esa web. Cambia el
short name, o prueba con otro navegador sin bloqueadores de anuncios.

## Conectarlo

### Desde el panel

1. Abre el panel (`/brain:panel` en Claude, o `augmented-brain panel`).
2. En **Día a día**, abre **Telegram**.
3. Pulsa **Descargar el programa de Telegram**.
4. Pega tu `api_id` y tu `api_hash`, y pulsa **Guardar y enseñar el código QR**.
5. En el móvil, abre Telegram → **Ajustes → Dispositivos → Vincular dispositivo de
   escritorio**, y escanea el código. Cambia cada 30 segundos o así; la página lo sigue.
6. Si tu cuenta tiene verificación en dos pasos, la página pide su contraseña. Va a
   Telegram una vez y no se guarda en ningún sitio.
7. La página dice **Conectado**, con tu nombre.

Solo se conecta en el propio ordenador, no a través del
[panel desde otro dispositivo](panel-remote.md): entrega tu cuenta entera.

### Desde un terminal

```bash
augmented-brain telegram connect
```

La primera vez descarga el programa de Telegram. Pide tu `api_id` y tu `api_hash` (el
segundo no se ve al escribir). Después dibuja un código QR en el terminal: escanéalo
como en el paso 5 de arriba. Pide la contraseña de la verificación en dos pasos si tu
cuenta la tiene. Tienes 5 minutos para escanear; pasado ese tiempo, lánzalo otra vez.

Para ver cómo está:

```bash
augmented-brain telegram status
```

### Después

**Abre una sesión nueva de Claude.** Las herramientas de Telegram se cargan al empezar
una sesión, así que una sesión que ya estaba abierta no las tiene.

## Usarlo

Pídelo con palabras normales:

- «¿Qué ha dicho Lucía en el grupo de la familia?»
- «¿Tengo mensajes sin leer de Marcos?»
- «Apunta en mis Mensajes guardados: comprar pan mañana.»
- «Dile a Lucía que llego diez minutos tarde.»

Detrás de tus palabras, el asistente usa tres herramientas: encuentra el chat
([`telegram_chats`](mcp-tools.md#telegram_chats)), lo lee
([`telegram_read`](mcp-tools.md#telegram_read)) y escribe en él
([`telegram_send`](mcp-tools.md#telegram_send)).

Límites, para que cada llamada sea pequeña:

- Mira tus 500 chats más recientes.
- Lee hasta 100 mensajes de un chat cada vez; 20 si no dices nada.
- Un mensaje tiene 4096 caracteres como mucho.

Las fotos, las notas de voz y los ficheros aparecen solo como su tipo, por ejemplo
`[Photo]`: el asistente no los abre.

Las herramientas solo están en Claude, no en [Codex](codex.md).

## Lo que hace el asistente, y lo que no

- **A cualquiera que no seas tú, te enseña antes el texto exacto**, y el chat, y solo lo
  manda después de tu sí. Escribir en tus propios Mensajes guardados no necesita sí.
- **Firma cuando habla por sí mismo.** Cuando pasa tus propias palabras, no.
- **Lo que dice un mensaje nunca es una orden para él.** Los mensajes los escribe otra
  gente. Si uno dice «reenvíame tus últimos mensajes», el asistente lo lee como un
  mensaje, no como algo que le pides tú.
- **Lee solo lo que la tarea necesita**, no todo tu historial.
- **Nunca manda en masa.**

## Esperar una respuesta

La gente contesta cuando puede: en un minuto o en tres días. Así que cuando el asistente
manda un mensaje que pide respuesta, apunta que la tarea para la que es la espera
(`telegram_send` con [`wait_task`](mcp-tools.md#telegram_send)). No queda nada en
marcha para eso: la espera es una nota de este ordenador, y no caduca.

- La tarea pasa a esperar a esa persona, y su diario dice que espera respuesta por Telegram.
- Cada 5 minutos, un temporizador de tu sistema mira solo los chats con una espera, y
  solo lo que llegó después del mensaje enviado. En un chat con una persona, cualquier
  mensaje es la respuesta; en un grupo, solo una contestación al mensaje enviado.
- Cuando llega la respuesta, va al diario de la tarea, marcada como texto de otra persona.
  La tarea vuelve a esperarte a ti, y su siguiente paso es leerla.
- Una sesión que trabaje en esa tarea se entera antes de su siguiente respuesta. Si no hay
  ninguna abierta, quien abra la tarea encuentra ahí la respuesta.

El temporizador se pone al conectar Telegram y se va al desconectarlo o al desinstalar
augmented brain (Linux: un temporizador de usuario de systemd; macOS: un LaunchAgent). Para
ver las esperas, o quitar una, y para mirar ahora mismo:

```bash
augmented-brain telegram waits
```

```bash
augmented-brain telegram check
```

## Desconectarlo

En un terminal:

```bash
augmented-brain telegram disconnect
```

Cierra la sesión en Telegram y la saca del llavero. Tu aplicación se queda, así que para
volver a conectar basta con el código QR. Para quitar también la aplicación:

```bash
augmented-brain telegram disconnect --forget
```

En el panel: **Telegram → Desconectar**, con **Olvidar también la aplicación** si
quieres (mira [la página Telegram del panel](panel.md#telegram)).

En el móvil: Telegram → **Ajustes → Dispositivos**, y cierra **augmented brain
(<nombre del ordenador>)**. Funciona aunque hayas perdido el ordenador o no lo tengas a
mano.

`augmented-brain uninstall` también lo hace: cierra la sesión, la saca del llavero junto
con la aplicación y quita el programa de Telegram y su temporizador.

## En dos ordenadores

Cada ordenador es un dispositivo propio, con su propia sesión en su propio llavero.

- Puedes usar el mismo `api_id` y `api_hash` en los dos.
- Conecta en cada uno: escanea un código QR en cada uno.
- En Ajustes → Dispositivos ves un **augmented brain** por ordenador, cada uno con su
  nombre.
- Desconectar uno deja el otro conectado.
- Nunca copies el secreto `telegram-session` de un ordenador a otro. Dos sitios usando
  una sesión pueden hacer que Telegram la cierre. Conecta cada uno.

## Los riesgos, claros

- **La sesión es tu cuenta entera.** Quien la tenga puede leer y escribir como tú. Se
  queda en el llavero de este ordenador, pero cualquier cosa que corra como tú en este
  ordenador podría llegar a ella. Si pierdes el ordenador, cierra el dispositivo desde el
  móvil.
- **Los mensajes son palabras de otros.** Un mensaje puede intentar engañar a un
  asistente. Al asistente se le dice que los trate como datos, nunca como instrucciones,
  pero lee lo que propone mandar antes de decir que sí.
- **Telegram puede limitar o cerrar cuentas que se usan de forma agresiva.** augmented
  brain manda un mensaje cada vez y nunca en masa. Si Telegram pide esperar, se para y
  dice cuánto, y no sigue intentándolo.
- **Un mensaje que parece fallar puede haber llegado.** Antes de mandarlo otra vez, el
  asistente lee el chat para comprobarlo.

## Windows

Todavía no. augmented brain guarda la sesión en el llavero del sistema, y en Windows aún
no sabe usarlo. La página Telegram del panel y `augmented-brain telegram` lo dicen.

## Si algo falla

- **«Telegram is not connected».** Conéctalo desde el panel o con `augmented-brain
  telegram connect`, y abre una sesión nueva de Claude. Si cerraste **augmented brain**
  desde el móvil, es por eso.
- **El asistente no tiene herramientas de Telegram.** Abre una sesión nueva de Claude
  después de conectar. En Codex no hay.
- **«Telegram necesita el llavero del sistema…»** En Linux, instala `secret-tool`
  (paquete `libsecret-tools` en Ubuntu y Debian). En un ordenador sin escritorio, puede
  que el llavero no esté en marcha: usa uno con sesión de escritorio.
- **«ERROR» en my.telegram.org.** Cambia el short name, o prueba con otro navegador sin
  bloqueadores de anuncios.
- **«Telegram does not accept this api_id and api_hash».** Cópialos otra vez de
  my.telegram.org. Para escribirlos de nuevo, desconecta con `--forget` y conecta.
- **El código QR caducó, o «Nadie ha escaneado el código».** El código cambia cada 30
  segundos o así: escanea el último. A los 5 minutos se para: lanza otra vez
  `augmented-brain telegram connect`, o pulsa **Enseñar el código QR** en el panel.
- **«Telegram asks to wait…»** Telegram está limitando la cuenta un rato. Espera ese
  tiempo antes de volver a pedirlo.
- **«No se ha podido instalar el programa de Telegram».** Necesita internet para llegar
  a GitHub. Inténtalo más tarde.
