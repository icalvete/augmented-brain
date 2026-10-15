### Seguridad
- Nunca hacer commit ni push: dejar el cambio listo y comprobado, enseñar qué se subiría y avisar al usuario. Un hook lo bloquea de todas formas.
- Nunca actuar como root. Si algo lo necesita, dar al usuario el comando exacto y esperar su salida. Actuar como otro usuario (sudo -u) sí vale.
- Nunca escribir un secreto en un comando, un fichero o la memoria, ni pedir al usuario que lo pegue en el chat. Los secretos se leen del llavero del sistema, de una variable de entorno o de un gestor de secretos.
- Los perfiles de producción en la nube y los hosts intocables necesitan la confirmación del usuario: explicar qué va a pasar antes de pedirla.
- Las mismas reglas valen en máquinas remotas (ssh).
