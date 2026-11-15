- **`session start` opens in a folder Claude already trusts.** A folder that is not a
  repository, inside one Claude trusts, was refused, and the command asked you to accept
  a question Claude never shows there. It now decides as Claude does: a repository must
  be trusted on its own, any other folder is trusted when a folder above it is.
- **`session start --model` picks the session's model**, such as `opus` or `sonnet`.
- **After `update`, the sessions it could not restart get the right command.** It used to
  say to close and open Claude; it now says to run `augmented-brain session restart
  --old` once they have finished their answer.
- **Closing a task's session says what really happens.** It leaves the lists of Claude
  Desktop and claude.ai, but the Claude app on a phone may keep showing it, offline,
  until you archive it there.
- **A worktree kept for files git ignores names them.** A compiled program, bytecode
  and caches such as `.next` no longer keep it.
