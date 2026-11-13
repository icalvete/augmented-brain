- **The guard never hangs, and never fails silently.** A few rare commands (a `find`
  inside a `find` many times, thousands of `$(`, a git alias that calls itself) kept it
  busy past the hook's 10 seconds; now every command nested in another one counts, and
  past four levels, or past 256 KB, it asks you. If the guard fails inside, it refuses
  that one action in plain words instead of showing Go's trace. A tool call in an
  unexpected shape, a configuration folder it cannot find, `config.toml` gone, a shell
  script it cannot read, or an answer it could not write no longer let a command through
  unchecked. `doctor` warns when Claude Code is older than 2.1.295, which lets
  everything through when the guard fails.
