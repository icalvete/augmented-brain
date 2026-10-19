# Getting started

[Leer en español](getting-started.es.md)

This guide is for starting from scratch, even if you have never opened a terminal. It
takes about ten minutes.

## What you need

- **A computer with Linux, macOS or Windows.** On a Mac, macOS 13 (Ventura) or later: see
  the Apple menu → About This Mac. On Windows, Windows 10 or 11. What changes on each
  system is in [Installing](install.md).
- **A paid Claude account**: Pro or above. augmented brain is free, but it works with
  Claude Code, and Claude Code is not in the free plan. The plans are at
  [claude.com/pricing](https://claude.com/pricing).
- **Your computer's password**, in case something missing has to be installed.

## 1. Open the terminal

The terminal is a window where you type commands instead of clicking buttons. You only
need it to install.

- **On a Mac:** press `Cmd + Space`, type `Terminal` and press Enter.
- **On Ubuntu:** press `Ctrl + Alt + T`.
- **On Windows:** press the Windows key, type `PowerShell` and press Enter.

## 2. Paste the install command

Copy this line, on a Mac or Linux:

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
```

Or this one, on Windows:

```powershell
irm https://github.com/icalvete/augmented-brain/releases/latest/download/install.ps1 | iex
```

Paste it into the terminal (`Cmd + V` on a Mac, `Ctrl + Shift + V` on Ubuntu, a right
click on Windows) and press Enter.

## 3. Answer the installer

It asks a few questions. To accept what it suggests, just press Enter.

1. **The language.** English or Spanish.
2. **Whatever is missing**, if anything: git (keeps the history of your notes), Claude
   Code and the Claude desktop app. It tells you what each one is for before installing it.
   If you already had Claude Code but it is behind, it offers to update it.
3. **Open the Claude app now?** Say yes.

If it asks for your password, type it even though you see nothing: the terminal does not
show it, not even as asterisks. Press Enter when done.

On a Mac, if git is missing, a window opens to install Apple's developer tools. Click
**Install** and accept. It takes a few minutes; the installer waits and carries on by itself.

On Windows there is no password to type: when Git is installed, Windows shows a window
asking for permission. Click **Yes**.

## 4. Sign in to Claude

In the Claude app:

1. Sign in with your account.
2. Open the **Code** tab.
3. Click **+ New** and type "Hi".

Without choosing a folder, you work in your personal space. For the work one, pick the
folder `~/augmented-brain/work` with **Choose…**.

## 5. Answer the interview

The first time, Claude introduces itself and asks six questions, one at a time, telling
you what each one is for: who you are, what you look after, how you want answers, what
must not be touched, which tools you use and who you work with. You can skip any of them.

At the end it shows what it saved and offers to set up the backup. Say yes: your notes go
to a private repository of yours, and losing the computer loses nothing.

## Your first week

Talk to it as usual. These phrases pay off from day one:

| You say | What it does |
|---|---|
| "New task: prepare Thursday's meeting." | Opens a task and asks what it needs to define it |
| "Park it, we carry on tomorrow." | Writes down where you stopped and what comes next |
| "Where were we?" | Lists your live tasks with their next step |
| "Let's carry on with Thursday's meeting." | Resumes the task where you left it, in any session |
| "Note this: call the insurance." | Puts it in the inbox, without opening a task |
| "We go with supplier B, it is cheaper." | Writes it in the task's journal, with the date |
| "Save that as a lesson." | What you learned gets a note of its own |
| "Close the task." | Closes it; it is never deleted |
| "Marta is my boss, I call her Martie." | Saves her, and next time it knows who she is |
| "Why did we pick supplier B?" | Finds it in your notes and gives you the decision, with date and reason |

## Words you will hear

- **Space:** a separate drawer, with its own notes and rules. There are two, `work` and
  `personal`. The folder you open Claude in decides which one you are in.
- **Task:** something with a start and an end. It goes through six phases, from idea to
  closed.
- **Area:** something you always look after (the website, the house, a team). It groups
  its tasks.
- **Inbox:** where loose ideas wait until you sort them.
- **Guard:** the check that stops a risky command, or asks you first.
- **Backup:** your notes, pushed to a private git repository of yours after every session.
- **Plugin:** the piece that connects augmented brain to Claude Code. The installer puts
  it in.

## If something goes wrong

- **Claude does not introduce itself or run the interview.** Start a new session. If
  nothing changes, open the terminal and type `augmented-brain doctor`: it tells you what
  is missing and how to fix it.
- **The terminal cannot find a command** (`command not found`, or "not recognized" on
  Windows). Close the terminal and open another one: the new one finds the programs just
  installed.
- **Claude asks you for a paid plan.** Claude Code needs Pro or above.
- **"This system is not supported yet".** It works on Linux, macOS and Windows; see
  [Installing](install.md) for what each one needs.
- **"Claude Code needs macOS 13".** Update the Mac in System Settings → General → Software
  Update, then paste the command again.
- **Something else.** Tell us in
  [the project's issues](https://github.com/icalvete/augmented-brain/issues), with what
  showed up on screen.

To see what happens inside, go on to [How it works](how-it-works.md).
