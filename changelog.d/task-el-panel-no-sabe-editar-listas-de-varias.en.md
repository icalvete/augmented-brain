- **The Guard page changes a list that takes many lines.** Adding a repository to a list
  kept by hand over several lines, with comments, failed with "the change would break
  config.toml". Now the value goes in before the closing `]` and taking one out takes
  only its line; your comments stay. Adding a repository also takes an optional "Why",
  written next to it with the day.
