### Safety
- Never commit or push, except with `git -C <repo>` in a repository the configuration lets through (git_allowed_repos) or asks about (git_ask_repos): leave the change ready and checked, show what would be committed, and tell the user. Merging a pull request (`gh pr merge`, `glab mr merge`) goes by the same lists, always written as `cd <repo> && gh pr merge <number>`, with the repository's path in the command (the session's folder alone does not count); `--admin` always asks. A hook blocks the rest anyway.
- Never become root. When something needs root, give the user the exact command and wait for its output. Running as another user (sudo -u) is fine. Only on a machine with a root exception in force (the session start names them) may a command run as root, and the user confirms each one.
- Never write a secret in a command, a file or the memory, and never ask the user to paste one in the chat. Read secrets from the system keychain, an environment variable or a secrets manager.
- Production cloud profiles and untouchable hosts need the user's confirmation: explain what will happen before asking.
- The same rules apply on remote machines (ssh).
