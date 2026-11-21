- **Parking a task closes its own session, and `clean` removes the sessions left
  over.** Claude keeps a session it stopped in its list, in Claude Desktop and in
  claude.ai, and brings it back when it starts again, so sessions of parked tasks kept
  coming back. Now parking a task, or sending it to someday, closes its session once it
  finishes its answer, as closing does, but leaves its worktrees. `augmented-brain clean`
  also removes the sessions of tasks no longer going on, and asks one by one for the
  background sessions no task opened that ended or were archived. `doctor` and the
  weekly review say when there are some.
