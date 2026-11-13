- **A production AWS profile set in a command asks, whatever the program.**
  `AWS_PROFILE=prod bash deploy.sh` used to go through while `./deploy.sh` asked: a few
  programs with a rule of their own skipped the general one. Now every program gets it.
  Asking for help no longer stops: `git commit --help`, `gh pr merge --help`,
  `su --version` and `git -c core.pager=cat log` go through. And a few reasons say what
  really happened: `cmd` started from Bash, a script downloaded and run in the same
  command, a tool call that could not be read.
