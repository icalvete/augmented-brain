- **The guard reads more of what the shell does before it runs a command.** It expands
  braces and `$IFS` as the shell does, reads the file a shell is fed (`bash < x.sh`),
  and asks before a shell runs what another program writes to it (`curl … | sh`): save
  it to a file and run the file, and the guard reads it. Programs that run the rest of
  their line (`strace`, `screen`, `tmux`, `at`, the `fish` and `csh` shells) are read
  too. Moving the home folder away, emptying it with `rsync --delete` or deleting it
  with a pattern are stopped, as `rm -rf ~` is.
