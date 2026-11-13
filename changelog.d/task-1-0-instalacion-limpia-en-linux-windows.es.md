- **El instalador de Windows ya no enseña el error de winget cuando consigue Git por
  otro camino.** En un Windows recién instalado, winget a menudo aún no puede instalar
  Git, y el instalador lo descarga de la página de Git for Windows. Hasta ahora enseñaba
  antes las últimas líneas del error de winget, que parecían un fallo; ahora solo salen
  si el otro camino también falla. Lo mismo con la app de escritorio: la salida de winget
  solo sale si la app no queda instalada.
