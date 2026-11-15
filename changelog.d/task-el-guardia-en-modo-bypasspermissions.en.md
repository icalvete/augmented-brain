- **In bypassPermissions mode the guard stops what it would ask about.** Anthropic's
  documentation does not promise that Claude Code shows the guard's questions in that
  mode, so a production command, a push to an "ask" repository or anything else the
  guard asks about could run without your yes. Now it is refused there, and Claude tells
  you what it wanted to do. Release and memory_trust already worked this way.
