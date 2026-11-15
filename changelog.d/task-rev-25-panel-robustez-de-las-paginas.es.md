- **El panel ya no ejecuta los filtros de un repositorio al enseñar tus repositorios.**
  Una carpeta descomprimida de un archivo trae su `.git/config`, y un filtro que nombre
  podía arrancar un programa con solo abrir Hoy o Repositorios. Tus propios filtros,
  como git-lfs, siguen funcionando.
- **Una nota con una cita sin fin ya no para el panel.** Una línea de un millón de `>`
  lo tumbaba; ahora las citas paran en 16 niveles y el resto sale como texto.
- **Los textos largos se guardan.** El panel aceptaba como mucho 64 KiB de un
  formulario, menos de lo que cabe en un perfil escrito en ruso o en japonés; ahora
  acepta 1 MiB y, si se pasa, dice que el texto es demasiado largo en vez de un «bad
  request» sin más.
- **El informe de un problema ya no lleva el nombre de tu ordenador.** El diagnóstico
  deja de nombrar la máquina en tu red de Tailscale, y el informe quita el nombre de
  este ordenador y cualquier nombre `*.ts.net`.
- **Escribir en el panel cuenta como usarlo.** Se para tras 30 minutos sin uso; escribir
  un texto largo sin cambiar de página ya no te deja sin panel donde guardarlo.
