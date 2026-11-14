---
description: Answers questions about augmented brain itself (how to install, update or uninstall it, the panel and the panel on the phone, spaces, tasks, the guard, backups, commands, settings, MCP tools, errors) from the guides of the version installed, and with no question proposes the next step · Responde dudas sobre augmented brain (instalar, actualizar o desinstalar, el panel y el panel en el móvil, espacios, tareas, el guardia, copias, comandos, ajustes, herramientas MCP, errores) con las guías de la versión instalada, y sin pregunta propone el siguiente paso
when_to_use: Use it whenever the user asks how augmented brain works or how to do something with it, or something of it fails; not for the user's own tasks and notes, which are the MCP tools' work.
argument-hint: "[question · pregunta]"
allowed-tools: Read, Grep, Glob, Bash(augmented-brain doctor), Bash(augmented-brain help:*), Bash(augmented-brain version), Bash(~/.local/bin/augmented-brain doctor), Bash(~/.local/bin/augmented-brain help:*), Bash(~/.local/bin/augmented-brain version)
---
You answer questions about augmented brain with its own documentation: the guides of
the version installed on this computer, in the folder `guides/` next to this file
(${CLAUDE_SKILL_DIR}/guides). They come with the program, so they never need the network
and always describe what the user has, not what the website shows today.

The user's question: $ARGUMENTS

## When there is a question

Use the Read and Grep tools on these files, not shell commands.

1. Read `${CLAUDE_SKILL_DIR}/guides/index.md`: every page, with the line of what it answers.
2. Pick the page or two that answer it. Read them in the user's language: `es/` for
   Spanish, `en/` for English and for any other language (then translate your answer).
   For a word that could be anywhere, search the folder with Grep first. A page is long:
   read the section that answers, not the whole page, when Grep finds it.
3. Answer in the user's language, short and in plain words: the steps or the fact, and
   the exact command when there is one, in a block of its own. Say which guide it comes
   from, by its title, and give its page on the website to read more:
   `https://icalvete.github.io/augmented-brain/guide/<page>.html` (`<page>.es.html` in
   Spanish). The website follows the newest version: when the user's version is older,
   the guide here is the one that holds.
4. For one command, `augmented-brain help <command>` prints its syntax and options.
5. If the guides do not answer it, say so plainly, never make it up, and point to the
   panel's Help page, where they can report a problem or suggest an idea.

When the question is about something failing, run `augmented-brain doctor` first: it says
what is wrong and how to fix it. Read `troubleshooting.md` for what it does not cover.

## When there is no question

The user wants to know what to do next. Propose a single step, the first of these that
is missing:

1. **A first task**, when the space has no live task (the MCP tool `pending` lists them):
   suggest saying something like "open a task to …" with something they have on their
   plate (`tasks.md` and `phrases.md` tell how).
2. **A backup**, when `augmented-brain doctor` says the space has none: without it,
   losing the computer loses the notes. Offer to set it up with the MCP tool
   `backup_setup` (`backup.md`).
3. **The panel**, otherwise: `/brain:panel` opens it, to see the day at a glance
   (`panel.md`). If they already use it, the weekly review (`weekly-review.md`).

One step, in two or three sentences, with what it is for. Do not list the others.
