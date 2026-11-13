- **`update` solo toma la versión de GitHub, y un programa nuevo que no arranca nunca
  sustituye al que funciona.** Una variable como `AB_RELEASE_URL` en el entorno de una
  sesión de Claude podía mandar la actualización a otro sitio; ahora `update` no le hace
  caso y lo dice (`--from` es para pruebas). El instalador prueba el programa nuevo antes
  de cambiar el viejo y devuelve el viejo si falla, y se niega a correr como root
  (`sudo`), que dejaba todo a nombre de root.
