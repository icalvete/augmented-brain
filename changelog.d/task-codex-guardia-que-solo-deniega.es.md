- **El guardia también funciona en Codex (experimental).**
  `augmented-brain codex install` lo registra como hook `PreToolUse` de Codex. Codex no
  puede enseñar una pregunta de un hook, así que allí el guardia rechaza lo que en Claude
  Code preguntaría. Revisa los comandos, el `apply_patch` de Codex y las herramientas que
  guardan una regla. También protege los ajustes de Codex. Cómo configurarlo, y sus
  límites, están en [El guardia](https://icalvete.github.io/augmented-brain/guide/guard.es.html#en-codex-experimental).
