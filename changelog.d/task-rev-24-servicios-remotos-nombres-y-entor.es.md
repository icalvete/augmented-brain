- **Se rechaza un nombre de espacio o una carpeta que podría colarse en un servicio.** Un
  nombre escrito a mano con un salto de línea, una barra, un espacio o un signo como `&` ya
  no carga, y el error dice cómo cambiarlo; `remote install` rechaza además una ruta con un
  salto de línea o, en Windows, con `%` o comillas, en vez de escribir un servicio que
  ejecuta otra cosa.
- **Las sesiones en segundo plano reciben solo las variables que necesitan.** El programa
  en segundo plano de Claude guardaba todo lo que tenía la terminal que lo arrancó, tokens
  e interruptores de pruebas incluidos, para todas las sesiones posteriores; ahora recibe
  la carpeta personal, `PATH`, el idioma, el agente de ssh, los proxies y los ajustes del
  propio Claude.
- **Desinstalar en Windows nombra cada programa por su ruta completa**, así que un
  `ping.bat` en la carpeta desde la que lo lanzas no se ejecuta.
