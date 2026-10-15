# Connect your services

augmented brain keeps your tasks and notes. To read your calendar, your mail, your
documents or your tickets, your assistant also needs those services. Claude Code talks to
them through **MCP** (Model Context Protocol): small connectors that give it tools such as
"list today's meetings" or "read this Jira issue".

## The easy way: claude.ai connectors

If you sign in to Claude Code with your claude.ai account, the connectors you turn on in
claude.ai also work in Claude Code.

1. Open **claude.ai → Settings → Connectors**.
2. Connect the services you want: Google Calendar, Gmail, Google Drive, Atlassian (Jira
   and Confluence), and others. Each one asks you to sign in to that service once.
3. **Start a new Claude Code session.** Connectors load when a session starts: a session
   that was already open will not see a connector you just added.
4. In Claude Code, type `/mcp` to see them. A connector marked as needing authentication
   is fixed from claude.ai, not from Claude Code.

That is all. Your assistant uses them when a task needs them ("what meetings do I have
tomorrow?", "summarise the open issues of this epic").

## Without a claude.ai account

If you use Claude Code with an API key, claude.ai connectors are not available. You can
add MCP servers yourself:

```bash
claude mcp add --scope user <name> -- <command that starts the server>
```

Many services publish an official MCP server; use those when they exist. Read what a
server can do before adding it: it acts with your account.

## Good habits

- **Connect only what you use.** Every connector adds tools your assistant has to choose
  from.
- **Your safety rules still apply.** augmented brain's guard watches commands and file
  writes. Actions inside a connector (sending an email, editing a ticket) are confirmed
  through Claude Code's own permission prompts: read them before you accept.
- **Never paste a password or a token in the chat** to connect a service. Every connector
  signs in through the service's own page.
