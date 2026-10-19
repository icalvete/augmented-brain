<p align="center">
  <img alt="augmented brain" src="assets/brand/out/header.svg" width="320">
</p>

<p align="center"><b>Your brain, augmented with AI.</b><br>
Your assistant remembers where you left off, works with a method, and asks before touching anything risky.<br>
All in plain text files of yours, backed up to your own git.</p>

<p align="center"><a href="https://icalvete.github.io/augmented-brain/">Website</a> ·
<a href="docs/guide/how-it-works.md">How it works</a> · <a href="README.es.md">Leer en español</a></p>

<p align="center"><a href="https://github.com/sponsors/icalvete"><img alt="Sponsor" src="https://img.shields.io/badge/Sponsor-EA4AAA?style=for-the-badge&logo=githubsponsors&logoColor=white"></a></p>

> [!WARNING]
> **Alpha.** This is an early version: it may have bugs, its format and commands may
> change. Tested by a person on Ubuntu, macOS and Windows 11. Read the [disclaimer](DISCLAIMER.md)
> before you use it, and back up anything important.

---

AI coding assistants are great inside one session and forget everything when it ends.
You explain the same context again, half-finished work gets lost, and nothing stops a
risky command on the wrong machine.

**augmented brain gives your assistant a memory and a way of working**, kept as plain
Markdown files that you own, and guards it while it works.

## What it does, day to day

You talk to it as usual. This is what changes:

| You | What happens |
|---|---|
| "Where were we?" | It lists your live tasks, each with its next step and who holds it up |
| "Park it, we carry on tomorrow." | It writes down where you stopped and what comes next; tomorrow, in a new session, it starts there |
| "Note: renew the certificate in November." | It goes to the inbox, no task opened; you sort it out in the weekly review |
| "Remind me on Thursday to call the insurance." | On Thursday, as soon as you open Claude, it reminds you |
| "Why did we pick Postgres?" | It looks it up in your notes and gives you the decision, with date and reason |
| "Save that as a lesson." | What you learned gets its own note, for next time |
| "Delete the old production bucket." | Before running it, it asks you |
| You open Claude in `~/augmented-brain/personal` | It only sees your personal notes; not the work ones |
| "Tell Marcos we moved it to Monday." | It knows Marcos is your manager and drafts the message |
| Your laptop dies | `augmented-brain restore` on the new one and it is all back |

## Start in two steps

You need Linux, macOS (13 or later) or Windows (10 or 11), and a paid Claude account (Pro
or above), because Claude Code is not in the free plan. augmented brain needs no root or
administrator. Never opened a terminal? Follow
[getting started](docs/guide/getting-started.md). What the installer does on each system,
where it puts things and how to update or uninstall: [Installing](docs/guide/install.md).

**1. Install it** with one command. On Linux or a Mac, in the terminal:

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
```

On Windows, in PowerShell:

```powershell
irm https://github.com/icalvete/augmented-brain/releases/latest/download/install.ps1 | iex
```

It asks the language first. If git or Claude Code is missing, it offers to install them,
and the Claude desktop app too, the easiest way to work (on a Mac, Windows, Ubuntu and
Debian). Then it sets everything up (two spaces, `work` and `personal`, in
`~/augmented-brain`, and the Claude Code plugin) and asks whether to open Claude right away.

**2. Answer Claude.** The first time, it asks six quick questions, saying what each one is
for: who you are, what you look after, how you want answers, what must not be touched,
which tools you use and who you work with. With that it creates your areas, sets up the
guard and learns who is who.

From then on, open the Claude app, click **+ New** and write. Without choosing a folder you
are in your personal space; for work, pick the folder `~/augmented-brain/work` with
**Choose…**. From a terminal:

```bash
cd ~/augmented-brain/work && claude
```

(On Windows, `cd ~\augmented-brain\work; claude`.)

If something is off, `augmented-brain doctor` tells you what is missing and how to fix it.

## What it looks like

> **You:** New task: add a contact form to the website.
>
> **Claude:** Opened *task-contact-form*, in definition. What fields should it have?
>
> *…you agree on name, email and message, and stop for the day…*
>
> **You:** Park it, we continue tomorrow with the plan.
>
> **Claude:** Parked. Next: write the plan.

The next day, in a **new session**:

> **You:** Let's continue with the contact form.
>
> **Claude:** It is parked with the definition agreed: name, email and message, sent by
> email. Next is the plan. The project folder is still empty: where is the website code?

No context to repeat. The assistant read the task and checked the real state before
trusting it.

## What you get

- **Tasks with a lifecycle**: define, analyse, plan, execute with a journal, validate,
  close. Park a task and resume it days later, in any session.
- **Areas** that group the tasks and the knowledge of each responsibility.
- **Rules loaded in every session**, and the important ones enforced by a guard: no
  commit or push unless you allow it, never root, no secrets in commands or files.
  Production AWS profiles and hosts you mark as untouchable ask you first, also over
  `ssh`.
- **Work and personal apart**: each *space* has its own memory, rules and backup. The
  folder you open the assistant in picks the space.
- **Backup and restore** to a private git repository of yours, one per space. More
  below.
- **Your services** (calendar, mail, Jira…) through MCP: see
  [Connect your services](docs/guide/connect-services.md).

## Lose the laptop, keep the brain

The notes folder of each space is a git repository, and the backup pushes it to a
private repository of yours. **To set it up, tell Claude:** "set up the backup". It
creates the private repository in your GitHub, connects it and makes the first copy. Or,
with another repository:

```bash
augmented-brain backup setup git@gitlab.com:you/my-brain.git   # once
augmented-brain backup                                         # by hand, if you like
```

- **A new machine in minutes.** Install augmented brain and run
  `augmented-brain restore <url>`: it replaces the empty space with yours, with notes,
  tasks, rules, your profile and its settings. Step by step, in
  [How it works](docs/guide/how-it-works.md#on-a-new-machine).
- **Several machines, one brain.** Before pushing, it fetches what your other machine did
  and merges it.
- **Full history.** Every backup is a commit, with the machine and the time: see what
  changed, go back to any day.
- **Your place.** GitHub, GitLab or your own server. augmented brain has no cloud.
- **One repository per space.** Work can go to your company's git.
- **No secrets go up.** If a note seems to hold a password or a token, it stops.
- **Nothing gets overwritten.** If two machines changed the same note, it stops and tells
  you what to do.

Once set up, **it runs by itself every time a Claude session ends**; if it fails, Claude
tells you at the start of the next one.

## Commands

| Command | What it does |
|---|---|
| `augmented-brain init` | Sets everything up: the `work` and `personal` spaces and the plugin. The installer runs it; `--custom` to choose yourself |
| `augmented-brain space add` | Add another space (personal, a client…) |
| `augmented-brain doctor` | Say what is missing, and how to fix it |
| `augmented-brain check` | Check the notes of this folder's space and rebuild its index |
| `augmented-brain backup setup <url>` | Back up this space to a private git repository |
| `augmented-brain backup` | Save the latest changes (`--all` for every space) |
| `augmented-brain restore <url>` | Bring a space back on a new machine |
| `augmented-brain update` | Update augmented brain and its Claude Code plugin (Claude tells you when there is a new version) |
| `augmented-brain uninstall` | Remove the program and the plugin; your notes stay (`--all` to start from scratch) |

Everyday work happens inside Claude Code, in plain words: "new task…", "what is
pending?", "park it", "note this in the inbox".

## How it works

- **Your memory is Markdown**, one folder per space: `tasks/`, `areas/`, `knowledge/`,
  `rules/`. Open it and edit it with any editor. The format is in
  [docs/spec/memory-format.md](docs/spec/memory-format.md).
- **One program**, `augmented-brain`, written in Go. Nothing else to install.
- **An MCP server** gives your assistant the tools of the method, so other assistants
  can use it too. Claude Code is the first one supported.
- **A Claude Code plugin** loads your space when a session starts and runs the guard
  before every command and file write. The guard is a safety net against mistakes, not
  a security boundary.

The four layers, a session step by step and where every file lives:
[How it works](docs/guide/how-it-works.md).

Inspired by *Building a Second Brain* by Tiago Forte and *Getting Things Done* by David
Allen, adapted to working with an AI assistant.

## Development

Needs Go 1.25 or later (an older `go` downloads it by itself). Working with an AI coding
agent? Its handbook is [AGENTS.md](AGENTS.md): the map of the code, the checks and the
rules of the house.

```bash
go test ./...
go build ./cmd/augmented-brain
```

`scripts/e2e-claude.sh` runs the whole story (open, park, resume, close a task, and the
guard) against a real, isolated Claude Code. It spends usage: run it before a release.

The brand (logo, icons, palette) is in [`assets/brand/`](assets/brand/), shown in place in
[`docs/brand.html`](docs/brand.html). [`scripts/privacy-check.sh`](scripts/privacy-check.sh)
blocks commits that would publish personal data or secrets.

## Support the project

augmented brain is free and MIT-licensed. If it saves you time, you can
[sponsor its development](https://github.com/sponsors/icalvete): it pays for the time to answer issues and add features. There is also paid support
(answers by email within two working days) and a one-hour setup session by video call.

## License

MIT. See [LICENSE](LICENSE) and the [disclaimer](DISCLAIMER.md). Not affiliated with
Anthropic; "Claude" and "Claude Code" are trademarks of Anthropic, PBC.
