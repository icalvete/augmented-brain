- **Un corte de luz ya no deja una nota vacía.** Cada nota queda en el disco antes de que
  el programa siga (antes se quedaba un rato solo en memoria), y los ficheros temporales
  que deja una escritura cortada se limpian en la sesión siguiente. Una nota que es un
  enlace simbólico nunca se escribe a través de él: augmented brain lo dice en vez de
  escribir en otro sitio. Una migración que se para a medias conserva la única copia
  hecha antes de empezar, con los enlaces copiados como enlaces, en vez de hacer otra, y
  los cambios en `config.toml` se comprueban antes de llegar al fichero, no después.
