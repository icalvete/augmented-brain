- **Una respuesta por Telegram llega sola a su tarea, tarde lo que tarde.** Cuando tu
  asistente manda un mensaje que pide respuesta, la tarea para la que es pasa a esperar a
  esa persona, y la respuesta va sola al diario de la tarea, marcada como texto de otra
  persona, con leerla como siguiente paso. No queda nada en marcha: un temporizador de tu
  sistema (systemd en Linux, un LaunchAgent en macOS) mira cada 5 minutos, solo los chats
  con una espera. `augmented-brain telegram waits` lista las esperas, y `telegram check`
  mira al momento. Ahora solo hay una conexión a Telegram a la vez, así que el panel y dos
  sesiones ya no se pisan la sesión. Mira
  [Esperar una respuesta](https://icalvete.github.io/augmented-brain/guide/telegram.es.html#esperar-una-respuesta).
