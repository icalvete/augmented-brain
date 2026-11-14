# Qué decirle a Claude

[Read in English](phrases.md)

No aprendes comandos. Hablas con Claude con tus palabras, en español o en inglés, y él usa
las herramientas de augmented brain por detrás. Estas frases funcionan. Cualquier otra
forma de decir lo mismo también.

## Tareas

| Tú dices | Qué pasa |
|---|---|
| "Tarea nueva: mudar el blog a otro hosting." | Abre una tarea en definición, esperándote, y te pregunta qué es "terminado" |
| "¿En qué estábamos?" | Lista tus tareas vivas, primero las que te esperan, cada una con su siguiente paso |
| "Seguimos con lo del blog." | Lee la tarea, mira cómo están las cosas de verdad y la retoma donde la dejaste |
| "Pasa al plan." | Lleva la tarea a su siguiente fase; tú apruebas cada cambio. Si te saltas una fase, el diario dice cuál |
| "Nos quedamos con el hosting más barato." | Apunta la decisión en el diario de la tarea, con la fecha |
| "Añade un paso: cambiar el dominio." | Añade una casilla al plan de la tarea |
| "La base de datos ya está copiada." | Marca ese paso |
| "Eso depende de Diego: hazlo subtarea." | Abre una subtarea con su propio estado y su siguiente paso |
| "El día 16, mira si se ha renovado el certificado." | Pone una fecha para revisar en la tarea |
| "Apárcala, mañana seguimos." | Apunta el siguiente paso exacto y lo esencial, y pausa la tarea |
| "Déjala para algún día." | La aparca como algún día: fuera de la lista diaria, guardada en su área |
| "Cierra la tarea." | La cierra con su resultado; nunca se borra. Una tarea cerrada se queda cerrada: cerrarla otra vez se rechaza |
| "Reabre la tarea del blog: el certificado ha vuelto a fallar." | La abre otra vez; el motivo va a su diario |
| "Ábreme una sesión para la tarea del blog." | Arranca una sesión de Claude solo para esa tarea, en su carpeta; sale en Claude Desktop y en claude.ai en menos de un minuto. Por ahora, solo en Linux ([cómo](task-sessions.es.md)) |

## Buzón y recordatorios

| Tú dices | Qué pasa |
|---|---|
| "Apunta esto: mirar los precios del hosting nuevo." | Una línea con fecha en el buzón, sin abrir tarea |
| "Recuérdame el jueves que llame al seguro." | Una captura con fecha: sale al empezar la sesión |
| "Apunta esto en trabajo: renovar la tarjeta del parking." | La captura va al buzón de tu espacio "trabajo", marcada con el espacio del que viene. Sale allí al empezar cada sesión hasta que se hace |
| "Ya he llamado al seguro, eso está hecho." | Saca esa captura del buzón sin esperar a la revisión semanal |
| "Vamos con la revisión semanal." | Repasa contigo el buzón y las tareas paradas, una a una |

## Conocimiento, reglas y personas

| Tú dices | Qué pasa |
|---|---|
| "Guarda eso como lección." | Una lección, con un título que dice la conclusión |
| "Apunta cómo funciona el despliegue." | Una referencia, en el área de la tarea |
| "La newsletter es un área nueva." | Crea el área |
| "A partir de ahora, contesta con frases cortas." | Una regla, que se carga en cada sesión |
| "Esto vale para todo." | Guarda la regla o la lección como compartida, para todos los espacios |
| "Esa regla ya no vale." | La retira: pasa a `archive/`, con cuándo y por qué, y deja de llegar a tus sesiones. No se borra nada. Claude te pregunta antes |
| "Diego lleva la empresa del hosting; yo le llamo Dieguito." | Guarda a Diego como persona, con su apodo |
| "¿Por qué elegimos el hosting más barato?" | Busca la decisión en tus notas, con su fecha y su motivo |
| "Cambia mi perfil: prefiero la recomendación primero." | Reescribe tu perfil |
| "¿En qué espacio estoy?" | Dice el espacio de esta sesión, su carpeta de notas, sus áreas y cuántas tareas hay vivas |

## Seguridad

| Tú dices | Qué pasa |
|---|---|
| "El perfil de AWS tienda-prod es producción." | El guardia te pregunta antes de los comandos que lo usan |
| "No toques nunca db01 sin preguntarme." | El guardia te pregunta antes de un ssh, scp o rsync a esa máquina |
| "He guardado un secreto nuevo, telegram/alertas-bot, para los avisos al móvil." | Apunta su nombre y para qué sirve, nunca su valor |
| "Pregúntame antes de guardar una regla." | Esto no se cambia hablando: lo eliges tú en la página Guardia del panel, "Preguntarme antes", y Claude te dice dónde. Desde entonces, Claude Code te pregunta antes de guardar una regla, tu perfil o las listas del guardia ([más](guard.es.md#preguntar-antes-de-que-claude-guarde-una-regla)) |

## Copia de seguridad, sesiones y el panel

| Tú dices | Qué pasa |
|---|---|
| "Configura la copia de seguridad." | Crea un repositorio privado en tu GitHub y hace la primera copia |
| "Recupera la sesión que se cortó." | Lee lo que se dijo después de la última entrada del diario, y apunta lo que falte |
| "~/code/blog es el blog, hecho con Hugo." | Apunta esa línea para el repositorio, que sale en el panel |
| `/brain:panel` | Abre el panel en tu navegador |
| "¿Cómo veo el panel en el móvil?" | Responde cualquier duda sobre augmented brain con las guías de la versión que tienes, sin necesitar internet |
| `/brain:help` | Sin pregunta, te propone un siguiente paso: el que te falta |

## Cuando Claude habla primero

Al empezar una sesión, Claude puede sacar cosas sin que se lo pidas:

- una fecha para revisar o un recordatorio que vence en los próximos 7 días, o que ya ha
  pasado;
- la revisión semanal, cuando la última tiene una semana y hay algo que repasar;
- una sesión que se cortó, para recuperarla;
- una copia de seguridad que falló, o una que dejó ficheros fuera;
- una versión nueva, para actualizar;
- notas con problemas;
- lo que otro espacio mandó a este, hasta que se hace;
- las listas del guardia, si algo las cambió fuera del panel;
- sesiones abiertas que siguen con el programa de antes de una actualización, y que hay
  que reiniciar.

Y mientras trabajas:

- tareas que han cambiado fuera de esta sesión (otra sesión, el panel o una edición a
  mano), antes de contestar tu siguiente mensaje;
- que responde otro modelo que el que elegiste, una vez por sesión, si pusiste el modelo
  de tus sesiones en la página Guardia;
- que la ventana usa mucha memoria: a los 2 GB, y luego a los 4 y los 8 GB, una vez cada
  uno (Linux y macOS; [qué hacer](troubleshooting.es.md#la-ventana-usa-mucha-memoria)).

## Por dónde seguir

- [Herramientas MCP](mcp-tools.es.md): la herramienta que hay detrás de cada frase, y qué
  recibe.
- [El panel](panel.es.md): lo mismo, a golpe de clic.
