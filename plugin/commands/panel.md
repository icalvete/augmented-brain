---
description: Open the panel in the browser to see today at a glance and your tasks, reminders, inbox, weekly review, areas, knowledge, rules, people, secrets, sessions, profile and health · Abre el panel en el navegador para ver el día de un vistazo y tus tareas, recordatorios, buzón, revisión semanal, áreas, conocimiento, reglas, personas, secretos, sesiones, perfil y salud
argument-hint: "[stop]"
allowed-tools: Bash(augmented-brain panel:*), Bash(~/.local/bin/augmented-brain panel:*)
---
The user wants the augmented brain panel: a web page on this computer to see and edit
their spaces.

1. If the user wrote `stop` ($ARGUMENTS), run `augmented-brain panel stop` with the Bash
   tool and tell them the panel is stopped. Nothing else.
2. Otherwise run `augmented-brain panel` with the Bash tool. It starts the panel (or finds
   the one already running), tries to open the browser, prints the address and returns
   at once. If the command is not found, run `~/.local/bin/augmented-brain panel`.
3. Give the user the address it printed, whole, as a link they can click: it carries the
   key that lets their browser in, so do not shorten it. Say in one or two
   short lines that it only works on this computer and stops by itself after 30
   minutes without use.

Answer in the user's language. Do not open, fetch or test the address yourself.
