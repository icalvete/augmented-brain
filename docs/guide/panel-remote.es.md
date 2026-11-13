# El panel desde otro dispositivo

[Read in English](panel-remote.md)

El [panel](panel.es.md) se abre en el ordenador donde funciona augmented brain, y solo
ahí. Para abrirlo desde otro de tus dispositivos, el móvil u otro ordenador, augmented
brain usa **[Tailscale](https://tailscale.com)**: una aplicación gratuita que une tus
dispositivos en una red privada solo suya. En ella solo están tus dispositivos; nada
queda abierto a internet.

```bash
augmented-brain panel remote
```

Arranca el panel, lo publica en tu red de Tailscale y escribe su dirección, como
`https://mi-portatil.ejemplo.ts.net:8443/t/…`, y un código QR. Apunta la cámara del
móvil al código, o abre la dirección en el otro ordenador. La dirección abre el panel
una vez, en los próximos 10 minutos: vuelve a lanzar el comando para tener otra.

## Lo que hace falta, una vez

El comando comprueba cada cosa y, si falta alguna, dice exactamente qué hacer. Nunca
instala ni cambia nada de Tailscale por ti.

1. **Tailscale en los dos dispositivos**, con la misma cuenta:
   [tailscale.com/download](https://tailscale.com/download). Funciona en Linux, macOS,
   Windows, Android e iPhone.
2. **HTTPS en tu red de Tailscale**, para que el panel viaje cifrado con un certificado de
   verdad. Actívalo en la página de administración de Tailscale,
   [login.tailscale.com/admin/dns](https://login.tailscale.com/admin/dns): **HTTPS
   Certificates → Enable HTTPS**. Una cosa que conviene saber: los nombres de tus máquinas
   (como `mi-portatil.ejemplo.ts.net`) pasan a registros públicos de certificados.
   Cualquiera puede ver que ese nombre existe; nadie fuera de tus dispositivos puede
   llegar a él.
3. **En Linux, deja publicar a tu usuario.** Tailscale solo deja publicar en el ordenador
   a un administrador. Una vez, con tu contraseña de administrador:

   ```bash
   sudo tailscale set --operator=tu-usuario
   ```

   augmented brain escribe esta línea con tu nombre y no la lanza. En macOS y Windows no
   hace falta.

## Quién puede abrirlo

Cuatro cosas hacen que sea solo tuyo:

- **Solo tu red de Tailscale llega a la dirección.** En internet no existe.
- **Solo entra tu cuenta.** Tailscale le dice al panel quién manda cada petición (la
  cuenta con la que entraste, con su contraseña y su segundo factor), y el panel solo
  deja entrar al dueño del ordenador. Otra persona de tu red, o alguien con quien
  compartes un dispositivo, recibe «forbidden» aunque tenga la dirección.
- **Lo que llega de otro dispositivo se trata siempre como tal.** Tailscale le pasa
  al panel sus visitas por una entrada propia, y el panel las reconoce por esa entrada,
  no por lo que la visita diga de sí misma. Así, una visita no puede hacerse pasar por
  este ordenador, y lo que solo puede hacer este ordenador se queda aquí (mira abajo).
- **La dirección sirve una vez**, y las páginas que abre llevan una llave que cambia
  cada vez que arranca el panel. Esa llave es solo del otro dispositivo: no abre el
  panel en este ordenador, y la llave de las páginas de este ordenador no lo abre desde
  otro dispositivo. Así, quien se hiciera con la dirección del otro dispositivo seguiría
  sin poder hacer aquí lo que solo puede este ordenador.
  La dirección es para ti, no para Claude: lanzado por Claude, el comando no publica
  nada y te pide que lo lances en una terminal tuya.

No hay otra contraseña que teclear: tu cuenta de Tailscale es la entrada.

## Lo que solo puede hacer este ordenador

Desde otro dispositivo puedes leer todas las páginas y hacer casi todo: tareas, buzón,
recordatorios, personas, tu perfil. Unas pocas cosas cambian tu copia de seguridad o lo
que Claude puede hacer, así que el panel solo las hace en este ordenador, y desde otro
dispositivo la página solo enseña:

- hacer una copia ahora;
- configurar una copia de seguridad;
- volver a una copia de tus notas;
- volver a una configuración anterior, en la página Versiones;
- cambiar cualquier cosa de la página Guardia: sus listas, el modelo de tus sesiones y
  lo que el asistente guarda solo.

## Siempre es temporal

La dirección funciona mientras el panel está en marcha. El panel se para solo tras 30
minutos sin uso, o ya con:

```bash
augmented-brain panel stop
```

Al pararse, le quita la dirección a Tailscale. Si el ordenador se colgó o se reinició con
el panel publicado, el siguiente panel que arranque la quita, y
`augmented-brain doctor` avisa mientras siga ahí. El puerto 8443 de tu red de Tailscale es
del panel: si ya publicas otra cosa ahí (aunque sea otro programa de este ordenador),
augmented brain no la toca y te lo dice. Solo retira la dirección que publicó él mismo.

## Compruébalo

`augmented-brain doctor` tiene una línea sobre el panel remoto en cuanto lo has usado: si
Tailscale sigue instalado, conectado y con HTTPS, y si quedó publicada una dirección del
panel.

## Si el panel ya estaba abierto

Si el panel ya está abierto en este ordenador cuando lanzas
`augmented-brain panel remote`, no se para: el mismo panel empieza a responder también por la red, y las páginas
que tienes abiertas siguen funcionando. Si ese panel era de una versión anterior, tiene
que arrancar de nuevo: el comando lo dice, y recargas esas páginas. Si publicarlo falla,
el panel sigue solo en este ordenador.
