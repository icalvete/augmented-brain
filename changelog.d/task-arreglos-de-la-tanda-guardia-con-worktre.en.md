- **The guard says how to write a commit or a push in a repository it already lets
  through.** The lists only count a repository named with `git -C` and its full path. A
  `cd` into an allowed repository, or a path in a variable, was refused with advice to add
  the repository to the lists, which it already was in, and Claude stopped there. Now the
  refusal gives the exact `git -C` command to use; what the guard allows has not changed.
