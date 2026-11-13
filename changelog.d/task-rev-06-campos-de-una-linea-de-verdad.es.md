- **Los campos de una línea son de una línea, siempre.** Un título, resumen, siguiente
  paso o `waiting_on` con saltos de línea (un texto copiado de un issue o de un correo,
  por ejemplo) podía escribir algo con aspecto de sección del programa, como `## Rules`,
  en el índice y en lo que lee cada sesión al empezar. Ahora esos campos quedan en una
  línea al entrar, y otra vez cada vez que se lee una nota, así que tampoco puede hacerlo
  una nota editada a mano. Cada campo tiene un límite holgado (un siguiente paso, 2.000
  caracteres; una entrada del diario, 32.000), y el error dice dónde va el texto largo.
  Una captura o una entrada del diario que acaba en `(from <espacio>)` escrito a mano ya
  no pasa por una enviada desde otro espacio. Un perfil con un nombre de varias líneas
  se puede volver a guardar. `watch pr` escribe una frase propia como siguiente paso de
  la tarea, y los nombres de los checks, que elige la pull request, solo en su diario,
  entre comillas y recortados; `--every` es de 10s como mínimo.
