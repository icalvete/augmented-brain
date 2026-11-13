- **Una copia ya no puede elegir dónde caen sus ficheros, y funciona con tus ajustes de
  git.** `restore` deja las notas donde estaban solo si es dentro de tu carpeta personal
  y fuera de sus carpetas ocultas y del sistema (`~/.config`, `~/Library`, `~/AppData`,
  cualquier `.git`); si no, van a `~/augmented-brain/<espacio>` y te lo dice, para que
  una copia que alguien ha cambiado no deje ficheros donde el ordenador los ejecuta.
  Rechaza un nombre de espacio que no vale, y lee la copia de `main` aunque la rama por
  defecto del repositorio sea otra. La copia no firma, no ejecuta hooks globales ni abre
  ventanas de contraseña, que la hacían fallar a quien tiene `commit.gpgsign` o
  `core.hooksPath`. La siguiente sesión y `doctor` avisan de una copia que empezó y no
  terminó, y de cuando la última buena tiene más de 3 días y las notas han cambiado.
