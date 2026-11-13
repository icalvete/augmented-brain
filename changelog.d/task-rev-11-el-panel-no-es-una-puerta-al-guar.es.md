- **El panel ya no es un rodeo al guardia.** `augmented-brain panel` escribía la dirección
  del panel con su llave, y `/brain:panel` hace que la lance Claude: con esa dirección y
  dos comandos que el guardia dejaba pasar (`127.1` en vez de `127.0.0.1`), Claude podía
  añadir un repositorio a los que puede subir, y el panel sellaba el cambio para que el
  guardia no notara nada. Ahora el comando escribe un enlace que abre el panel una vez,
  en los próximos 2 minutos (10 con `panel remote`), y lanzado por Claude con el
  navegador abierto no escribe ninguno; si otra cosa usó tu enlace antes, el navegador lo
  dice. Un cambio de las listas del guardia hecho en el panel, o unas listas aceptadas
  allí, te llega como una línea del propio Claude Code, no de Claude. El guardia deniega
  una dirección del panel se escriba como se escriba el host. Compactar una
  conversación, `/clear` o retomarla ya no toma como vistas unas listas cambiadas (solo
  una sesión que empiezas tú), y un sello que no se puede leer pregunta en vez de dejar
  pasar un push. Un `*` suelto en un comando que solo nombra la carpeta de configuración
  ya no se deniega.
