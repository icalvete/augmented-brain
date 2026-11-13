- **The guard reads programs' options as the programs do.** Root over ssh, an untouchable
  host reached through another one, and a container given the whole machine are now
  stopped however their options are written. `sshfs`, `autossh`, `nerdctl` and
  `kubectl debug node/…` have the same rules as `ssh` and `docker`, and mounting the
  Docker socket or one of the machine's disks in a container counts as becoming root.
