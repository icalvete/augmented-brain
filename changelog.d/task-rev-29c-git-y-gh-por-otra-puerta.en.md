- **git and gh rules hold however the command is put.** A push from a repository where
  Claude may push now asks when it goes somewhere that is not one of its remotes, and is
  denied when that place is an untouchable repository. git settings that run a command
  or bring an alias are checked however they are given, and so are the commands git
  runs from its options. Merging, committing a file or moving a branch through `gh api`
  or `glab api` is stopped like `gh pr merge`; running a workflow again, a
  `repository_dispatch` and saving a gh alias ask first. In an untouchable repository,
  `merge`, `cherry-pick`, `revert`, `am` and `pull` are denied too.
