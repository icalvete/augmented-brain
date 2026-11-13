- **Una sesión parada por el límite de uso vuelve al primer intento.** En la 0.26.0-beta
  el vigilante la reanudaba un momento antes de tiempo: Claude abría una copia, el
  vigilante la quitaba, y la sesión volvía dos minutos después, en la siguiente mirada.
  Ahora espera a que Claude la haya soltado de verdad.
