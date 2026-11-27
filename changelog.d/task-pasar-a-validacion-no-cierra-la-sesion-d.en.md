- **Moving a task to validation no longer closes its session.** The proof is run there
  with you, and a failed one goes back to execution in the same conversation; before,
  the session was cut the moment the task went to validation. Closing, parking and
  someday still close it. Since 1.1.4 a task's session is pinned, so Claude does not
  retire it while it waits.
