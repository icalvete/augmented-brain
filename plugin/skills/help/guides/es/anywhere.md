# Trabajar desde el móvil u otro ordenador


Tus notas viven en un ordenador. Aun así puedes trabajar con ellas desde el móvil, una
tableta u otro ordenador. Hay dos formas, y hacen cosas distintas:

- **Hablar con Claude** en ese ordenador, desde cualquier sitio: el [acceso
  remoto](remote.md). Eliges una sesión en [claude.ai/code](https://claude.ai/code) o
  Claude Desktop y trabajas como si estuvieras sentado delante.
- **Mirar tus notas** en el panel, desde cualquier sitio: [el panel desde otro
  dispositivo](panel-remote.md). Hoy, tus tareas, el buzón, todo lo que enseña el panel.

Puedes usar una, la otra o las dos. El acceso remoto funciona en Linux, macOS y Windows; el
panel desde otro dispositivo funciona donde funcione [Tailscale](https://tailscale.com).

## Se prepara una vez

En el ordenador que guarda tus notas:

1. **Acceso remoto.** Lanza `augmented-brain remote install` y haz lo que pide: que
   Claude confíe en la carpeta y, en Linux, encender «linger» para que aguante un
   reinicio (en macOS y Windows arranca cuando entras en tu sesión). Detalles en
   [Acceso remoto](remote.md).
2. **El panel.** Instala Tailscale en el ordenador y en el móvil, con la misma cuenta, y
   lanza `augmented-brain panel remote`. Escribe una dirección y un código QR. Detalles en
   [El panel desde otro dispositivo](panel-remote.md).

`augmented-brain doctor` comprueba las dos: una línea para la puerta de cada espacio y,
en cuanto lo has usado, otra para el panel desde otro dispositivo.

## Qué sesión usar

El acceso remoto abre **una puerta por espacio**, con el nombre `<nombre del
ordenador>-<espacio>`, como `mi-portatil-personal`. Una sesión que entra por ella empieza
en la primera carpeta de trabajo de ese espacio, así que Claude carga las notas y las
reglas de ese espacio.

- En claude.ai o Claude Desktop, elige el entorno con ese nombre y empieza una sesión.
  Puedes empezar todas las que quieras.
- Para trabajar en una tarea, dilo: «retoma task-hornear-pan». Claude lee la tarea y sigue
  donde se quedó.
- Para llegar a otro espacio, usa la puerta de ese espacio. Las notas nunca se cruzan entre
  espacios.

## Cuando deja de responder

- **El ordenador está dormido.** Un portátil que se duerme corta la conexión. Vuelve cuando
  se despierta.
- **La sesión se archivó** en claude.ai, o terminó en el lado de Claude. En Linux vuelve
  sola, en unos diez minutos; en macOS y Windows son tres pasos a mano. Los dos casos
  están en [Acceso remoto](remote.md#conviene-saber).
- **Ha caducado tu sesión de Claude.** Lanza `claude` una vez en el ordenador y vuelve a
  entrar.

`augmented-brain remote status` dice cómo está cada puerta, y qué hacer. Otros problemas
están en [Si algo falla](troubleshooting.md#el-acceso-remoto-no-funciona).

## Una sesión para cada tarea

En Linux, la sesión de la puerta de un espacio puede abrir otra para una tarea («ábreme
una sesión para la tarea de la web»). En menos de un minuto sale en tu lista, con el
nombre de la tarea, en su carpeta y ya leyéndola. Abre tantas como tareas tengas. En
macOS y Windows todavía no está. Mira [Una sesión por tarea](task-sessions.md).

## Con seguridad

Las dos son puertas a tu ordenador. El acceso remoto es una puerta permanente: quien entre
en tu cuenta de Claude puede trabajar en la máquina a través de ella. Protege esa cuenta
con una contraseña fuerte y verificación en dos pasos, deja que Claude pregunte antes de
actuar (nunca `bypassPermissions`), y quita las puertas que no uses: `augmented-brain
remote uninstall` y `augmented-brain panel stop`. La dirección del panel solo funciona para
tus propios dispositivos de tu red de Tailscale, y solo mientras el panel está en marcha.
