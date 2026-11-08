# Remote access

[Leer en español](remote.es.md)

Claude Code can be reached from another computer or from your phone: from
[claude.ai/code](https://claude.ai/code) or Claude Desktop you pick a session that runs on
your machine and work in it as if you were sitting there. For that, a small program,
`claude remote-control`, has to be running on the machine.

`augmented-brain remote` keeps it running for you. It starts by itself when the computer
starts, and starts again if it stops. You set it up once.

This is for Linux with systemd (the part of Linux that starts and watches services) for
now: not a container, nor WSL without systemd. macOS and Windows are still to do.

## One door per space

Each space (work, personal…) gets its own door, with its own name:
`<computer name>-<space>`, such as `my-laptop-personal`. A session opened through that door
starts in a folder that belongs to that space, so Claude loads that space's notes and
rules, and nothing crosses between spaces. By default only your default space has a door.
To open one for work too, see [Configuration](configuration.md).

That folder is the space's first work folder, the one you named when you made the space:
where you already work, so a session from your phone opens where you would open one at
the computer. To start somewhere else inside the space, set `folders` in
[`remote.toml`](configuration.md). Only a space with no work folder gets a folder of its
own, `~/.config/augmented-brain/remote/<space>`.

## Install it

```bash
augmented-brain remote install
```

It checks that systemd can keep the service, writes it and starts it. If systemd cannot,
it says why and changes nothing. It then tells you what is missing, if anything:
two things only you can do.

### 1. Let Claude trust the folder

Claude refuses to run in a folder it has not been told to trust, and there is no one at
the keyboard to answer when it starts by itself. If you already use Claude in your work
folder, it trusts it and there is nothing to do. Otherwise, once for each folder `install`
names, open Claude there and accept the question:

```bash
cd ~/projects && claude
```

Accept "trust this folder", then close Claude (`/exit`). augmented brain never edits
Claude's own settings for you. The service tries again every 30 seconds, so within half a
minute it is running.

### 2. Turn on "linger"

A user service normally runs only while you are logged in. To have it after a reboot with
nobody logged in, the system must be told to keep your services: that is called
**linger**, and it needs an administrator. `install` prints the exact command and does
not run it:

```bash
sudo loginctl enable-linger your-user-name
```

## Check it

```bash
augmented-brain remote status
```

For each space it says whether the service is running, whether Claude trusts its folder,
and whether linger is on, with what to do about each problem. `augmented-brain doctor`
shows the same, one line per space.

When a service fails, it says why in the system journal:

```bash
journalctl --user -u augmented-brain-remote-personal -n 20
```

A service installed by an earlier version sent its errors nowhere; `status` and `doctor`
tell you to run `augmented-brain remote install` again, which brings it up to date.

## Connect from another computer

1. Open [claude.ai/code](https://claude.ai/code) or Claude Desktop, signed in to the same
   Claude account as on the machine.
2. Choose the environment named `<computer name>-<space>`, the name `status` shows.
3. Start a session. Start as many as you like: each one begins in that space's folder.

## A session per task

The door's own sessions all start in the space's folder. To work on a task in its own
session, in its own folder, ask the session you are in: "open me a session for the website
task". It runs

```bash
augmented-brain session start task-new-website
```

and within a minute a new session, named after the task, shows up in claude.ai and Claude
Desktop. It starts by reading its task and telling you where it stands. The folder is the
first of the task's repositories found in the space's work folders, or the one `--folder`
gives.

- **Claude must trust that folder itself.** Trusting the space's folder is not enough. The
  first time, the command says what to do: on the machine (or through `ssh`), go to the
  folder, run `claude`, accept the question and close it with `/exit`. Then ask again.
- **One session per folder.** Claude serves each folder once, so a task cannot use the
  door's own folder, or a folder another task's session is using. Give it another one with
  `--folder` (a git worktree of the repository, for example).
- **It ends with its session**, with `augmented-brain session stop <task>`, or when the
  machine restarts. After a reboot only the doors come back: ask for the task sessions
  again. `augmented-brain session list` shows the open ones.

## Keep it safe

**This is a permanent door to your computer.** Anyone who gets into your Claude account
can work on the machine through it.

- Claude asks before it acts, as always (permission mode `default`), and the guard stays
  in force: it stops pushes, root and secrets, and asks before production.
- **Do not set `permission_mode` to `bypassPermissions`.** It lets whoever connects run
  anything with no question, and leaves the machine open.
- Protect your Claude account: a strong password and two-step verification.
- Remove the door whenever you do not need it: `augmented-brain remote uninstall`.

## Good to know

- **A laptop that sleeps drops the connection.** The session is back when it wakes up. The
  program does not change your sleep settings.
- If your Claude login expires, the service keeps running but nobody can connect. `status`
  and `doctor` say so; run `claude` once and sign in again.
- Sessions you start from the phone belong to the space of their folder: the work door
  opens work notes. To reach another space, use that space's door.

## Take it away

```bash
augmented-brain remote uninstall
```

It stops and removes the services. The folders, and their place in the space's work
folders, stay, and nothing else is deleted. Remove them by hand if you want.

## The settings

`remote.toml`, in the configuration folder: `enabled`, `spaces`, `name_prefix`,
`permission_mode` and `folders`. See [Configuration](configuration.md). The commands are in
[Commands](commands.md).
