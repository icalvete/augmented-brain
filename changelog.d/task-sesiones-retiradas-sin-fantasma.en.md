- **A task's session no longer goes disconnected after hours unused.** `session start`
  pins it in `claude agents`, so Claude does not retire it after 8 idle hours and leave
  a "disconnected" entry in claude.ai. `session stop`, parking and closing the task
  take the pin out; your own pins stay.
