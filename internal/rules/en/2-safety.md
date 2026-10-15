### Safety
- Never commit or push: leave the change ready and checked, show what would be committed, and tell the user. A hook blocks it anyway.
- Never become root. When something needs root, give the user the exact command and wait for its output. Running as another user (sudo -u) is fine.
- Never write a secret in a command, a file or the memory, and never ask the user to paste one in the chat. Read secrets from the system keychain, an environment variable or a secrets manager.
- Production cloud profiles and untouchable hosts need the user's confirmation: explain what will happen before asking.
- The same rules apply on remote machines (ssh).
