# Carry on in Codex


When Claude's usage runs out in the middle of a task, you can carry on with the same task
in [Codex](https://developers.openai.com/codex), OpenAI's coding assistant. Codex starts
with your space, as Claude does: your profile, your rules and your tasks. It can read the
task, write in its journal and move it on. The notes are the same files, so when Claude
is back, it finds the task as Codex left it.

This is experimental. It was tried with Codex CLI 0.170.0.

## Set it up once

You need Codex installed and working. Then, in a terminal:

```bash
augmented-brain codex install
```

It adds augmented brain to Codex as a plugin, with three things:

- **The start:** every new Codex session gets your space, as a Claude session does.
- **The guard:** Codex's commands and file changes go through the
  [guard](guard.md#in-codex-experimental).
- **The task tools:** where_am_i, pending and the task tools (task_get, task_journal,
  task_update…). The other tools stay in Claude.

Codex runs a hook only once you trust it. Running `codex install` is that trust: it
trusts augmented brain's hooks for you, and says so. It also turns off a Codex setting,
`unified_exec_tty`, which lets the assistant type into a terminal where the guard cannot
see. It changes nothing else of yours in Codex.

You can run it again at any time. `augmented-brain update` runs it for you when it
updates the program.

## Carry on with a task

1. In Claude, park the task, or ask Claude to write where it is in its journal. If the
   usage ran out first, nothing is lost: what Claude did is in the task.
2. Open Codex in the task's folder, or in a folder of its space.
3. Tell Codex which task: "carry on with the task of the blog".

Codex reads the task, sees its next step and goes on from there. Keep the task up to date
as you would with Claude: Codex is told to.

When Claude is back, open a new Claude session. It reads the task as Codex left it.

## What is different in Codex

- **The guard never asks.** Codex cannot show you a question from the guard. So what the
  guard would ask you about in Claude, it refuses in Codex. The assistant then tells you
  the command, for you to run yourself or from Claude.
- **Fewer tools.** No inbox, knowledge, rules, closing a task or other spaces. Codex tells
  you to do those from Claude or the [panel](panel.md).
- **One space per session.** Codex works in the space of the folder you open it in. In a
  folder of no space, augmented brain does nothing there.
- **No messages between sessions** and no notifications: Codex tells you in its chat.
- **Each assistant has its own conversation.** Only the notes are shared. Write what
  matters in the task.

## Check it

```bash
augmented-brain codex status
```

It says whether augmented brain is installed in Codex and, if something is wrong, what.
`augmented-brain doctor` says the same, when Codex is on the computer.

The usual problem: **a new version changed the hooks** and Codex stopped running them,
because it trusts a hook only as it was. `augmented-brain update` fixes it by itself; if
you updated some other way, run `augmented-brain codex install` again.

## Remove it

```bash
augmented-brain codex uninstall
```

It removes the plugin and the trust in its hooks. `unified_exec_tty` stays off: it only
protects. Your notes do not change. `augmented-brain uninstall` removes it too.

See also: [The guard in Codex](guard.md#in-codex-experimental) and
[Commands](commands.md#augmented-brain-codex).
