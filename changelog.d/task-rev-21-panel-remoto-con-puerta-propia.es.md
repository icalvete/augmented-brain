- **El panel remoto tiene su propia puerta.** Con el panel publicado por Tailscale,
  otra persona de tu red que tuviera la dirección podía hacerse pasar por este ordenador
  cambiando una cabecera de la petición, y desde ahí configurar una copia de seguridad
  de tus notas hacia un repositorio suyo. Ahora lo que llega por Tailscale entra por un
  puerto propio y siempre se trata como venido de la red. El enlace para el móvil solo
  abre por la red, y el del navegador de este ordenador solo aquí. Abrir el panel ya no
  retira algo que publiques tú con `tailscale serve` en el puerto 8443: augmented brain
  solo retira la dirección que publicó él. `panel stop` ya no termina otro proceso de
  augmented brain que tenga por casualidad el número de un panel que se cayó.
  `panel remote` espera a un Tailscale lento en vez de arrancar el panel de nuevo (lo
  que dejaba sin funcionar las páginas abiertas).
