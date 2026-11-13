- **A backup can no longer choose where its files land, and works with your git
  settings.** `restore` puts the notes where they were only inside your home folder and
  outside its hidden and system folders (`~/.config`, `~/Library`, `~/AppData`, any
  `.git`); otherwise they go to `~/augmented-brain/<space>` and it tells you, so a
  backup someone changed cannot drop files where your computer runs them. It refuses a
  space name that is not valid, and reads the backup from `main` even when the
  repository's default branch is another. The backup ignores signing, global hooks and
  password windows, which made it fail for anyone with `commit.gpgsign` or
  `core.hooksPath` set. The next session and `doctor` say when a backup began and never
  ended, and when the last good one is over 3 days old with notes changed since.
