- **Se puede comprobar de dónde sale una descarga.** Cada fichero de una versión (los
  archivos y los dos instaladores) lleva ahora un registro de procedencia firmado por
  GitHub, que dice qué workflow lo compiló y desde qué commit (`gh attestation verify`),
  y `checksums.txt` incluye también los instaladores. El mismo commit da los mismos bytes
  en cualquier sitio. Una versión sale del commit de `main` que fijó su número, nunca de
  cambios mezclados después, y solo cuando todos sus checks exigidos han pasado ahí; se
  compila en un trabajo que no puede escribir en el repositorio. La guía de instalación
  explica cómo comprobar una descarga.
