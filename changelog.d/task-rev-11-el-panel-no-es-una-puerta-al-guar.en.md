- **The panel is no longer a way around the guard.** `augmented-brain panel` printed the
  panel's address with its key, and `/brain:panel` has Claude run it: with that address
  and two commands the guard let through (`127.1` instead of `127.0.0.1`), Claude could
  add a repository to the ones it may push to, and the panel sealed the change so the
  guard noticed nothing. Now the command prints a link that opens the panel once, within
  2 minutes (10 for `panel remote`), and run by Claude with the browser open it prints
  none; if something used your link first, the browser says so. A change to the guard's
  lists made in the panel, or lists accepted there, reaches you as a line from Claude
  Code itself, not from Claude. The guard denies a panel address however the host is
  written. Compacting a conversation, `/clear` or resuming no longer takes changed lists
  as seen (only a session you start does), and a seal that cannot be read asks instead
  of letting a push through. A plain `*` in a command that only names the configuration
  folder is no longer denied.
