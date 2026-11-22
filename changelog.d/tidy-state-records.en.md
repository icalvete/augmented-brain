- **The records of worktrees and sessions no longer pile up.** A worktree removed by hand
  stayed noted for good, and every task session left a lock file behind. Each session
  start now takes off the records of folders that are gone and the old locks.
