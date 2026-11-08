# Trabajar desde el móvil u otro ordenador

[Read in English](anywhere.md)

Tus notas viven en un ordenador. Aun así puedes trabajar con ellas desde el móvil, una
tableta u otro ordenador. Hay dos formas, y hacen cosas distintas:

- **Hablar con Claude** en ese ordenador, desde cualquier sitio: el [acceso
  remoto](remote.es.md). Eliges una sesión en [claude.ai/code](https://claude.ai/code) o
  Claude Desktop y trabajas como si estuvieras sentado delante.
- **Mirar tus notas** en el panel, desde cualquier sitio: [el panel desde otro
  dispositivo](panel-remote.es.md). Hoy, tus tareas, el buzón, todo lo que enseña el panel.

Puedes usar una, la otra o las dos. El acceso remoto es solo para Linux por ahora; el panel
desde otro dispositivo funciona donde funcione [Tailscale](https://tailscale.com).

## Se prepara una vez

En el ordenador que guarda tus notas:

1. **Acceso remoto.** Lanza `augmented-brain remote install` y haz las dos cosas que pide:
   que Claude confíe en la carpeta, y encender «linger» para que aguante un reinicio.
   Detalles en [Acceso remoto](remote.es.md).
2. **El panel.** Instala Tailscale en el ordenador y en el móvil, con la misma cuenta, y
   lanza `augmented-brain panel remote`. Escribe una dirección y un código QR. Detalles en
   [El panel desde otro dispositivo](panel-remote.es.md).

`augmented-brain doctor` comprueba las dos cosas, una línea cada una.

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
- **La sesión se archivó.** Archivar una sesión en claude.ai, o dejarla mucho tiempo sin
  usar, puede terminarla en el lado de Claude mientras el servicio de tu ordenador sigue
  en marcha sin nada detrás. Vuelve a arrancar el servicio, en el ordenador:

  ```bash
  systemctl --user restart augmented-brain-remote-personal
  ```

  La sesión nueva puede tardar unos minutos en aparecer: el lado de Claude deja antes que
  caduque la vieja.
- **Ha caducado tu sesión de Claude.** `augmented-brain remote status` lo dice. Lanza
  `claude` una vez en el ordenador y vuelve a entrar.

`augmented-brain remote status` dice cómo está cada puerta, y qué hacer.

## Una sesión vieja con el mismo nombre

Cuando la puerta de un espacio se cambia de carpeta (por ejemplo, porque una versión nueva
la arranca en tu carpeta de trabajo), el lado de Claude conserva también la sesión vieja,
con el mismo nombre, en otro grupo de la lista. En la vieja no contesta nadie: lo que
escribas ahí se pierde. Usa la del grupo con el nombre de la carpeta nueva, y archiva la
vieja en claude.ai o Claude Desktop.

## Empezar sesiones nuevas desde el móvil

Abrir una sesión nueva para cada trabajo, desde el móvil, con una sesión de Claude Desktop
que se queda abierta para eso, se está probando y aún no está lista. Mientras tanto,
empieza una sesión nueva por la puerta del espacio y di qué tarea retomar.

## Con seguridad

Las dos son puertas a tu ordenador. El acceso remoto es una puerta permanente: quien entre
en tu cuenta de Claude puede trabajar en la máquina a través de ella. Protege esa cuenta
con una contraseña fuerte y verificación en dos pasos, deja que Claude pregunte antes de
actuar (nunca `bypassPermissions`), y quita las puertas que no uses: `augmented-brain
remote uninstall` y `augmented-brain panel stop`. La dirección del panel solo funciona para
tus propios dispositivos de tu red de Tailscale, y solo mientras el panel está en marcha.
