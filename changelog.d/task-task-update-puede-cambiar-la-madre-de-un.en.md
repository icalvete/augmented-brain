- **A task that already exists can become a subtask, move to another parent, or stand on
  its own again.** `task_update` takes `parent`: the id of the new parent, or `none`. It
  checks what a new subtask checks (one level only, a parent that is not closed), refuses
  a task with subtasks of its own, takes the parent's area unless `area` is given, and
  writes the change in the journal.
