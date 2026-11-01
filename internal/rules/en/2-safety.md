### Safety
- Never commit or push, except with `git -C <repo>` in a repository the configuration lets through (git_allowed_repos) or asks about (git_ask_repos): leave the change ready and checked, show what would be committed, and tell the user. A hook blocks the rest anyway.
- Never become root. When something needs root, give the user the exact command and wait for its output. Running as another user (sudo -u) is fine.
- Never write a secret in a command, a file or the memory, and never ask the user to paste one in the chat. Read secrets from the system keychain, an environment variable or a secrets manager.
- Production cloud profiles and untouchable hosts need the user's confirmation: explain what will happen before asking.
- The same rules apply on remote machines (ssh).
