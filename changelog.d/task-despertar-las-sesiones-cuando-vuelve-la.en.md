- **Task sessions go on by themselves when Claude's usage limit resets.** When your
  plan's usage ran out, a task's session stopped and stayed there until you wrote to it,
  for hours if you were away. Now the watch that `remote install` sets up reads the hour
  the limit resets, and wakes each session then, one every two minutes, as the same
  session with its conversation. You are told when the wait begins and when they are
  back, and `session list` shows a stopped session as **waiting**, with the hour. Linux
  only, and only sessions opened with `session start`.
