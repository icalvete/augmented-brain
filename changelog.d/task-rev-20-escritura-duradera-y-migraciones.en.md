- **A power cut no longer leaves a note empty.** Each note is now on the disk before the
  program goes on (it used to stay a while in memory only), and the temporary files a
  write cut short leaves are tidied at the next session. A note that is a symbolic link
  is never written through: augmented brain says so instead of writing somewhere else.
  A migration that stops half way keeps the one copy made before it started, with links
  copied as links, instead of making another, and changes to `config.toml` are checked
  before they reach the file, not after.
