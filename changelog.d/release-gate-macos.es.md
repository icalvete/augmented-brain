- **Una versión espera a las pruebas de macOS, aunque una pull request ya no.** Las pull
  requests se mezclan sin esperar a macOS, que es lento, y solo lanzan allí las pruebas de
  Go y compat-check; las del instalador, fallos, Safari y acceso remoto en macOS corren en
  `main` tras cada mezcla, y no se publica ninguna versión hasta que pasan allí sus pruebas.
