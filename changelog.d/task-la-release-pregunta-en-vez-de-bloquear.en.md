- **Starting a release workflow asks you instead of being stopped.** When an assistant
  runs `gh workflow run release`, the guard now shows you the permission dialog. Say yes
  and the release starts, with no terminal; say no, or do not answer, and nothing starts.
  In `bypassPermissions` mode, where that dialog may not show, it is still stopped.
