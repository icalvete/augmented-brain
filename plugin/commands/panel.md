---
description: Open the panel in the browser to see today at a glance and your tasks, reminders, inbox, weekly review, areas, knowledge, rules, people, secrets, backups, sessions, profile and health · Abre el panel en el navegador para ver el día de un vistazo y tus tareas, recordatorios, buzón, revisión semanal, áreas, conocimiento, reglas, personas, secretos, copias de seguridad, sesiones, perfil y salud
argument-hint: "[stop]"
allowed-tools: Bash(augmented-brain panel:*), Bash(~/.local/bin/augmented-brain panel:*)
---
The user wants the augmented brain panel: a web page on this computer to see and edit
their spaces.

1. If the user wrote `stop` ($ARGUMENTS), run `augmented-brain panel stop` with the Bash
   tool and tell them the panel is stopped. Nothing else.
2. Otherwise run `augmented-brain panel` with the Bash tool. It starts the panel (or finds
   the one already running), opens it in the browser and returns at once. If the command
   is not found, run `~/.local/bin/augmented-brain panel`.
3. If it says the panel is open in the browser, tell the user so, and say in one short
   line that it only works on this computer and stops by itself after 30 minutes without
   use. If it says the browser could not open from here, tell the user to run
   `augmented-brain panel` in a terminal of their own: the address is for them, and the
   program does not show it to you.

Answer in the user's language. Never open, fetch or test the address yourself: it
works once, and if you used it, the user's browser would say it was already used.
