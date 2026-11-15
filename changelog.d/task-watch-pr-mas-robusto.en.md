- **`watch pr` no longer reports a result of before, nor gives up when it need not.**
  Right after a push or a rerun it waits 90 seconds before it believes a green or a red,
  since GitHub still shows the old one for a while. A pull request whose changes start no
  required check now ends saying so: a success when GitHub lets it merge, blocked when it
  does not. And outside a git repository it takes the repository from the task's folders,
  or asks for `--repo` at once.
