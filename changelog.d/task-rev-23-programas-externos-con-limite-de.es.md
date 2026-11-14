- **Ninguna llamada a Claude Code, gh o git puede dejar colgado un comando.** `doctor`,
  `update` y la página de Ayuda del panel esperaban para siempre a un Claude Code parado
  en su pantalla de primer uso o con la red a medias; ahora cada llamada tiene un límite
  que se cumple, y `doctor` dice lo que no ha podido comprobar.
- **Esperar a que una sesión termine su respuesta dura lo que dice.** Cerrar o reiniciar
  una sesión espera 30 minutos como mucho de reloj, mira muchas menos veces y apunta en
  su log cada 5 minutos que sigue esperando.
- **`watch pr` siempre vigila.** Tras un reinicio podía unirse a una vigilancia que ya
  no existía y no decir nada durante horas; ahora arranca una siempre que no haya
  ninguna, y dos pedidas a la vez comparten una.
- **El vigilante del acceso remoto se rinde en vez de reiniciar sin fin.** Si reiniciar
  no recupera la sesión, espera cada vez más y, tras 5 intentos, deja el servicio en paz;
  `remote status` y `doctor` lo dicen.
- **`clean` respeta las carpetas temporales de las sesiones de otra carpeta de Claude**
  en Linux, y el log del servicio en macOS ya no crece sin fin.
