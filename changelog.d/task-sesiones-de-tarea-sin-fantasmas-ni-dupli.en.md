- **A task's session closes when the task moves to validation, and comes back in its own
  folder.** Validation waits on you, often for hours, and Claude stops a session left
  unused that long, which then stays in claude.ai as disconnected until you archive it.
  Now moving a task to validation closes its session, as parking does. And
  `session start` without `--folder` opens where the task's last session worked, so a
  session Claude stopped comes back with its conversation instead of a second one.
- **After an update, augmented brain no longer asks you to park tasks.** Since 1.1.1
  `session restart --old` leaves busy sessions as they are, and parking would close
  them. It now names the sessions left and says to run it again when they finish.
