- **One session making a worktree no longer stalls the others.** `task_worktree` and
  closing a task ran git, network included, with the notes locked: with a slow network,
  every tool of every other session waited, once 17 seconds. Git now runs after the
  notes are released, and a tool that finds them busy for 20 seconds says so instead
  of hanging.
- **A session starts on time while another one is writing.** The start of a session
  waited for the notes up to three times, 9 of the 10 seconds it has; it now waits once,
  2 seconds at most, and starts with the profile and the rules either way.
