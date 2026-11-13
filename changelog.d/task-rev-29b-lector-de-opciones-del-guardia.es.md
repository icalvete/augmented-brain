- **El guardia lee las opciones de los programas como los propios programas.** Entrar
  como root por ssh, llegar a una máquina intocable a través de otra y darle a un
  contenedor la máquina entera se paran ahora se escriban como se escriban sus opciones.
  `sshfs`, `autossh`, `nerdctl` y `kubectl debug node/…` tienen las mismas reglas que
  `ssh` y `docker`, y montar en un contenedor el socket de Docker o uno de los discos de
  la máquina cuenta como hacerse root.
