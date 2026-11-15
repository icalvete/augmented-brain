- **Sessions start sooner.** The nine parts of the session start no longer queue for 50
  ms at a time on the notes lock, the inbox is read once and without a slow pattern, a
  new session reuses what another already read of the tasks, and the list of processes
  is read only after an update. The check for a new version no longer makes the session
  wait for GitHub: it runs apart, and a new version is told at the next session.
