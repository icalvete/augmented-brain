<p align="center">
  <img alt="augmented brain" src="assets/brand/out/header.svg" width="320">
</p>

<p align="center"><b>A second brain for working with AI coding assistants.</b><br>
Tasks, areas and rules that your assistant remembers from one session to the next.</p>

<p align="center"><a href="README.es.md">Leer en español</a></p>

---

> **Status: 0.0.1, design stage.** Nothing to install yet. This repository holds the
> project's goal, its design and its brand. Follow along or open an issue with ideas.

## The problem

AI coding assistants are great inside one session and forget everything when it ends.
You explain the same context again, half-finished work gets lost between sessions, and
nothing stops the assistant from running a risky command on the wrong machine.

## What augmented brain does

It gives your assistant a memory and a way of working, kept as plain Markdown files
that you own.

- **Tasks with a lifecycle.** Every piece of work goes through the same steps: define,
  analyse, plan, execute (with a journal), validate, close. You can park a task and pick
  it up days later, in another session, exactly where you left it.
- **Areas.** Ongoing responsibilities (your platform, your side project) group their
  tasks and what you learned about them, so you start from the right map.
- **Rules that stick.** How you like to work is loaded at the start of every session.
  The important ones are enforced, not just asked for: no commits or pushes unless you
  say so, no root, no secrets written into commands or files.
- **Safe on servers and in the cloud.** The same protections apply over `ssh`. AWS
  production profiles ask for confirmation first.
- **Work and personal kept apart.** Each *space* has its own memory, rules and backup.
  The folder you open the assistant in picks the space.
- **Backup and restore.** Lose your laptop and get everything back on a new one.
- **Your tools.** Guidance to connect Google Calendar, Gmail, Drive, Jira and other
  services through MCP.

## How it will work

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
augmented-brain init
```

`init` asks three questions (language, where to keep your memory, which assistant you
use) and sets everything up. Then you open your assistant and say *"new task: …"*.

- **One binary**, written in Go. No Python, Node or anything else to install.
- **Linux and macOS** first; Windows later.
- **Claude Code** first. The method lives in an MCP server, so other assistants can use
  it too.
- **English and Spanish.**

## Roadmap

| Version | What |
|---|---|
| 0.0.1 | Goal, design and brand (this release) |
| 0.1 | Memory format, `init` and `doctor`, Claude Code plugin with the safety hooks |
| 0.2 | MCP server with the task lifecycle, weekly review and backup / restore |
| 0.3 | Documentation site, installer and first public binaries |

## Inspiration

*Building a Second Brain* by Tiago Forte (the PARA method: projects, areas, resources,
archive) and *Getting Things Done* by David Allen (the inbox and the weekly review),
adapted to working with an AI assistant.

## Development

Needs Go 1.25 or later (an older `go` downloads it by itself).

```bash
go test ./...
go build ./cmd/augmented-brain
```

## Repository

- [`docs/spec/memory-format.md`](docs/spec/memory-format.md): how the memory is stored (draft).
- [`docs/brand.html`](docs/brand.html): logo, icons and palette, shown in place.
- [`assets/brand/`](assets/brand/): brand sources and the script that builds every asset.
- [`scripts/privacy-check.sh`](scripts/privacy-check.sh): blocks commits that would publish personal data or secrets.

## License

MIT. See [LICENSE](LICENSE).
