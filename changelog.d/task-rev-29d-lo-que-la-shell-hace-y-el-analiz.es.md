- **El guardia lee más de lo que hace la shell antes de ejecutar un comando.** Expande
  las llaves y `$IFS` como la shell, lee el fichero que se le pasa a una shell
  (`bash < x.sh`) y pregunta antes de que una shell ejecute lo que le escribe otro
  programa (`curl … | sh`): guárdalo en un fichero y ejecuta el fichero, y el guardia lo
  lee. También lee los programas que ejecutan el resto de su línea (`strace`, `screen`,
  `tmux`, `at`, las shells `fish` y `csh`). Llevarse la carpeta personal a otro sitio,
  vaciarla con `rsync --delete` o borrarla con un patrón se paran, como `rm -rf ~`.
