- **A task's session closes even when the computer goes off while it waits.** Parking,
  closing or validating a task waits for its session to end its answer; a shutdown in
  that time left the session behind until `clean` was run. Now the next session start
  finishes the job, and only one wait runs per task.
