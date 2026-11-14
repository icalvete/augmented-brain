# Getting started

[Leer en español](getting-started.es.md)

## What it is

augmented brain gives your AI assistant a memory, a method and a guard. Today the
assistant is Claude Code.

- **A memory.** Your tasks, decisions, people and rules are kept as plain text files on
  your computer. Every new session starts by reading them, so you never explain the same
  thing twice.
- **A method.** Every piece of work is a task that goes from idea to closed in six
  phases. You can stop a task today and pick it up next week, in any session.
- **A guard.** Before Claude runs a command, augmented brain reads it. Pushing code,
  becoming root or writing a password is stopped. Touching production asks you first.

You do not learn new commands. You talk to Claude as usual: "new task…", "park it",
"where were we?".

It is free and open source (MIT). It is also beta: a version still being tested, though what you
have keeps working from one version to the next. What it cannot do yet is in
[Known limitations](limitations.md).

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
- **A GitHub account, only for the backup.** Installing needs none. At the end of the
  first session Claude offers to back up your notes to a private repository in your
  GitHub account (a free one is enough). Create it at [github.com](https://github.com/)
  before you start, or say no then and set the backup up later.

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

Paste it into the terminal (`Cmd + V` on a Mac, `Ctrl + Shift + V` on Ubuntu, `Ctrl + V`
or a right click on Windows) and press Enter.

## 3. Answer the installer

It asks a few questions. To accept what it suggests, just press Enter.

1. **The language.** English or Spanish; it suggests your system's. You can change it
   later in the [configuration](configuration.md#settings-for-the-whole-computer).
2. **Whatever is missing**, if anything: git (keeps the history of your notes), Claude
   Code and the Claude desktop app. It tells you what each one is for before installing it.
   If you already had Claude Code but it is behind, it offers to update it. If Claude Code
   is missing, it also asks whether you have a paid plan; a no does not stop it.
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
folder `~/augmented-brain/work` with **Choose…**. A **space** is a separate drawer of
notes: [Spaces](spaces.md) explains them.

**If Claude asks whether you trust the folder, say yes.** Claude Code asks this the first
time it opens in a folder, before loading anything: until you say yes, augmented brain
cannot load your notes.

No desktop app (on Linux without `apt`, or if you said no)? The installer offers to open
Claude in the terminal instead, in your work space. Next times:
`cd ~/augmented-brain/work && claude`, or the `personal` folder for the other space.

## 5. Answer the interview

The first time, Claude introduces itself and asks six questions, one at a time, telling
you what each one is for: who you are, what you look after, how you want answers, what
must not be touched, which tools you use and who you work with. You can skip any of them.

At the end it shows what it saved and offers to set up the backup. Say yes: your notes go
to a private repository of yours, and losing the computer loses nothing. It needs a
GitHub account; without one, say no for now and ask "set up the backup" whenever you
have it. [Backup and restore](backup.md) tells the rest, other git servers included.

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
| "Lucía is my boss, I call her Lu." | Saves her, and next time it knows who she is |
| "Why did we pick supplier B?" | Finds it in your notes and gives you the decision, with date and reason |

More phrases, grouped by what you want: [What to say to Claude](phrases.md).

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
- **Panel:** a page in your browser to see and change everything without asking Claude.

## If something goes wrong

- **Claude does not introduce itself or run the interview.** Start a new session, and if
  it asks whether you trust the folder, say yes. If nothing changes, open the terminal and type `augmented-brain doctor`: it tells you what
  is missing and how to fix it.
- **The terminal cannot find a command** (`command not found`, or "not recognized" on
  Windows). Close the terminal and open another one: the new one finds the programs just
  installed.
- **The installer says it cannot download from GitHub, or you stopped it.** Nothing is
  half done: paste the command again (in a while, if GitHub is having trouble).
- **Claude asks you for a paid plan.** Claude Code needs Pro or above.
- **Something else.** See [Troubleshooting](troubleshooting.md).

## Where to go next

- [Tasks](tasks.md): how a task moves, and how to park and resume it.
- [The guard](guard.md): what it stops, what it asks, and why.
- [The panel](panel.md): everything at a glance, in your browser.
- [Backup and restore](backup.md): so a lost laptop loses nothing.
