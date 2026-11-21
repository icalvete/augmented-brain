- **`session restart --old`, and `update`, no longer bring back sessions you archived.**
  Claude is not told when you archive a task's session in Claude Desktop, so restarting
  it after an update opened it again, and restarting the remote access opened new
  sessions in it. Now `--old` restarts none of them: it only stops their earlier
  augmented brain, and the new one starts the next time the session uses it. Naming one
  session (`session restart <task>`) still restarts it.
