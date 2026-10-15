<p align="center">
  <img alt="augmented brain" src="assets/brand/out/header.svg" width="320">
</p>

<p align="center"><b>A second brain for working with AI coding assistants.</b><br>
Tasks, areas and rules that your assistant remembers from one session to the next.</p>

<p align="center"><a href="README.es.md">Leer en español</a></p>

---

AI coding assistants are great inside one session and forget everything when it ends.
You explain the same context again, half-finished work gets lost, and nothing stops a
risky command on the wrong machine.

**augmented brain gives your assistant a memory and a way of working**, kept as plain
Markdown files that you own, and guards it while it works.

## Start in three steps

Linux or macOS, with [Claude Code](https://docs.claude.com/en/docs/claude-code) installed.

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
augmented-brain init
augmented-brain doctor
```

`init` asks your language, a name for your first space and the folder you work in, and
installs the Claude Code plugin. `doctor` tells you if anything is missing, and how to
fix it. Then open Claude Code in your work folder.

If `augmented-brain` is not found right after installing, open a new terminal; the installer
prints the line to add to your shell profile when its folder is not in your `PATH`.

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
- **Backup and restore** to a private git repository of yours, one per space.
- **Your services** (calendar, mail, Jira…) through MCP: see
  [Connect your services](docs/guide/connect-services.md).

## Commands

| Command | What it does |
|---|---|
| `augmented-brain init` | Set up augmented brain and your first space |
| `augmented-brain space add` | Add another space (personal, a client…) |
| `augmented-brain doctor` | Say what is missing, and how to fix it |
| `augmented-brain check` | Check the notes of this folder's space and rebuild its index |
| `augmented-brain backup setup <url>` | Back up this space to a private git repository |
| `augmented-brain backup` | Save the latest changes (`--all` for every space) |
| `augmented-brain restore <url>` | Bring a space back on a new machine |

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

Inspired by *Building a Second Brain* by Tiago Forte and *Getting Things Done* by David
Allen, adapted to working with an AI assistant.

## Development

Needs Go 1.25 or later (an older `go` downloads it by itself).

```bash
go test ./...
go build ./cmd/augmented-brain
```

`scripts/e2e-claude.sh` runs the whole story (open, park, resume, close a task, and the
guard) against a real, isolated Claude Code. It spends usage: run it before a release.

The brand (logo, icons, palette) is in [`assets/brand/`](assets/brand/), shown in place in
[`docs/brand.html`](docs/brand.html). [`scripts/privacy-check.sh`](scripts/privacy-check.sh)
blocks commits that would publish personal data or secrets.

## License

MIT. See [LICENSE](LICENSE).
