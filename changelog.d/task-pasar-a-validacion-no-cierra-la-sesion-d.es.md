- **Pasar una tarea a validación ya no cierra su sesión.** La prueba se hace ahí contigo,
  y si falla la tarea vuelve a ejecución en la misma conversación; antes la sesión se
  cortaba en cuanto la tarea pasaba a validación. Cerrar, aparcar y algún día la siguen
  cerrando. Desde la 1.1.4 la sesión de una tarea está fijada, así que Claude no la
  retira mientras espera.
