# Conecta tus servicios

augmented brain guarda tus tareas y tus notas. Para leer tu calendario, tu correo, tus
documentos o tus tickets, tu asistente necesita además esos servicios. Claude Code habla
con ellos mediante **MCP** (Model Context Protocol): pequeños conectores que le dan
herramientas como "lista las reuniones de hoy" o "lee esta incidencia de Jira".

## La forma fácil: los conectores de claude.ai

Si entras en Claude Code con tu cuenta de claude.ai, los conectores que actives en
claude.ai también funcionan en Claude Code.

1. Abre **claude.ai → Ajustes → Conectores**.
2. Conecta los servicios que quieras: Google Calendar, Gmail, Google Drive, Atlassian
   (Jira y Confluence) y otros. Cada uno te pide entrar en ese servicio una vez.
3. **Empieza una sesión nueva de Claude Code.** Los conectores se cargan al empezar la
   sesión: una sesión que ya estaba abierta no verá un conector que acabas de añadir.
4. En Claude Code, escribe `/mcp` para verlos. Si alguno pide autenticación, se arregla
   desde claude.ai, no desde Claude Code.

Y ya está. Tu asistente los usa cuando una tarea los necesita ("¿qué reuniones tengo
mañana?", "resúmeme las incidencias abiertas de esta épica").

## Sin cuenta de claude.ai

Si usas Claude Code con una API key, los conectores de claude.ai no están disponibles.
Puedes añadir servidores MCP tú mismo:

```bash
claude mcp add --scope user <nombre> -- <comando que arranca el servidor>
```

Muchos servicios publican un servidor MCP oficial; usa esos cuando existan. Lee qué puede
hacer un servidor antes de añadirlo: actúa con tu cuenta.

## Buenos hábitos

- **Conecta solo lo que uses.** Cada conector añade herramientas entre las que tu
  asistente tiene que elegir.
- **Tus reglas de seguridad siguen valiendo.** El guardia de augmented brain vigila los
  comandos y las escrituras de ficheros. Las acciones dentro de un conector (enviar un
  correo, editar un ticket) se confirman con los avisos de permiso de Claude Code: léelos
  antes de aceptar.
- **Nunca pegues una contraseña o un token en el chat** para conectar un servicio. Cada
  conector entra por la página del propio servicio.
