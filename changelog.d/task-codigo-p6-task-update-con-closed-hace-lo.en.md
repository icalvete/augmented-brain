- **Closing a task by changing its status now closes it fully.** `task_update` with
  status `closed` used to change only the status: no result, and the task's session and
  worktree stayed. Now it does what `task_close` does, with a new optional `result`; the
  panel's Close button also ends the task's session and removes its worktree.
