- **The guard works in Codex too (experimental).** Register
  `augmented-brain hook pre-tool-use codex` as Codex's `PreToolUse` hook. Codex cannot
  show a question from a hook, so there the guard refuses whatever it would ask about in
  Claude Code. It checks commands, Codex's `apply_patch` and the tools that save a rule.
  It also protects Codex's settings. How to set it up, and its limits, are in
  [The guard](https://icalvete.github.io/augmented-brain/guide/guard.html#in-codex-experimental).
