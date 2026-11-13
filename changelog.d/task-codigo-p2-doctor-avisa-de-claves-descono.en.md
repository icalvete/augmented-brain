- **`doctor` and the session start name settings of `guard.toml` and `session.toml` the
  program does not know.** They were ignored without a word, so a misspelled
  `untouchable_repo` protected no repository. The message gives the right name when it
  looks like a typo. The files are still read, so a setting of a newer version does not
  break an earlier one.
