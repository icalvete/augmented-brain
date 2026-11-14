# Remote access


Claude Code can be reached from another computer or from your phone: from
[claude.ai/code](https://claude.ai/code) or Claude Desktop you pick a session that runs on
your machine and work in it as if you were sitting there. For that, a small program,
`claude remote-control`, has to be running on the machine.

`augmented-brain remote` keeps it running for you. It starts by itself, and starts again
if it stops. You set it up once. It works on the three systems, each with its own way of
starting programs by themselves:

- **Linux**: a user service of systemd (the part of Linux that starts and watches
  services). It can start when the computer starts, before anyone logs in. Not in a
  container, nor in WSL without systemd.
- **macOS**: a LaunchAgent, the way macOS starts a program when you log in.
- **Windows**: a task of the Task Scheduler that starts when you log in, with no window.

On macOS and Windows it starts **when you log in**, not before: after a reboot, log in
once and it is back. Neither system lets an ordinary user start something with nobody
logged in, and augmented brain never asks for an administrator.

## One door per space

Each space (work, personal…) gets its own door, with its own name:
`<computer name>-<space>`, such as `my-laptop-personal`. A session opened through that door
starts in a folder that belongs to that space, so Claude loads that space's notes and
rules, and nothing crosses between spaces. By default only your default space has a door.
To open one for work too, list both spaces in `remote.toml`, in the configuration folder,
and run `augmented-brain remote install` again:

```toml
spaces = ["personal", "work"]
```

That folder is the space's first work folder, the one you named when you made the space:
where you already work, so a session from your phone opens where you would open one at
the computer. To start somewhere else inside the space, name the folder in `remote.toml`:

```toml
folders = { work = "~/code/clients" }
```

Only a space with no work folder gets a folder of its own, `remote/<space>` inside the
configuration folder (see [Where everything lives](where-things-live.md)). Every setting
of `remote.toml` is in [Configuration](configuration.md#remotetoml).

## Install it

```bash
augmented-brain remote install
```

It checks that the system can keep the service, writes it and starts it. If it cannot,
it says why and changes nothing. It then tells you what is missing, if anything:
one thing only you can do, two on Linux.

### 1. Let Claude trust the folder

Claude refuses to run in a folder it has not been told to trust, and there is no one at
the keyboard to answer when it starts by itself. If you already use Claude in your work
folder, it trusts it and there is nothing to do. Otherwise, once for each folder `install`
names, open Claude there and accept the question:

```bash
cd ~/projects && claude
```

On Windows, in PowerShell, `install` writes it as `cd "C:\Users\you\projects"; claude`.
Accept "trust this folder", then close Claude (`/exit`). augmented brain never edits
Claude's own settings for you. The service tries again by itself (every 30 seconds, every
minute on Windows), so within a minute it is running.

### 2. On Linux, turn on "linger"

A user service normally runs only while you are logged in. To have it after a reboot with
nobody logged in, the system must be told to keep your services: that is called
**linger**, and it needs an administrator. `install` prints the exact command and does
not run it:

```bash
sudo loginctl enable-linger your-user-name
```

macOS and Windows have no linger: there the service starts when you log in.

## Keep it safe

**This is a permanent door to your computer.** Anyone who gets into your Claude account
can work on the machine through it.

- Claude asks before it acts, as always (permission mode `default`), and the guard stays
  in force: it stops pushes, root and secrets, and asks before production.
- **Do not set `permission_mode` to `bypassPermissions`.** It lets whoever connects run
  anything with no question, and leaves the machine open.
- Protect your Claude account: a strong password and two-step verification.
- Remove the door whenever you do not need it: `augmented-brain remote uninstall`.

## Check it

```bash
augmented-brain remote status
```

For each space it says whether the service is running, whether Claude trusts its folder,
and on Linux whether linger is on, with what to do about each problem.

On Linux, `install` also sets up a **watch**: a small timer that looks at the services
every two minutes and brings back one whose session the server archived (see "Good to
know" below). `status` says how it stands: whether it is running and how long ago it
last looked. `augmented-brain doctor` shows the same, one line per space, and one more
for the watch only when it fails.

When a service fails, it says why. On Linux, in the system journal:

```bash
journalctl --user -u augmented-brain-remote-personal -n 20
```

On macOS, in `~/Library/Logs/augmented-brain/augmented-brain-remote-<space>.log`; on
Windows, in `augmented-brain-remote-<space>.log` inside
`%LOCALAPPDATA%\augmented-brain\remote`. `status` names the file.

A service installed by an earlier version sent its errors nowhere; `status` and `doctor`
tell you to run `augmented-brain remote install` again, which brings it up to date.

## Connect from another computer

1. Open [claude.ai/code](https://claude.ai/code) or Claude Desktop, signed in to the same
   Claude account as on the machine.
2. Choose the environment named `<computer name>-<space>`, the name `status` shows.
3. Start a session. Start as many as you like: each one begins in that space's folder.

The door's sessions all start in the space's folder. On Linux, a task can also get a
session of its own, in its own folder: ask the session you are in, "open me a session
for the website task". See [A session per task](task-sessions.md).

## Good to know

- **A laptop that sleeps drops the connection.** The session is back when it wakes up. The
  program does not change your sleep settings.
- **On macOS and Windows, a reboot leaves it off until you log in.** If the computer
  restarts by itself (an update, a power cut), nobody can connect until someone logs in.
- If your Claude login expires, the service keeps running but nobody can connect. `status`
  and `doctor` say so; run `claude` once and sign in again.
- **If the server archives the session** (you archived it in claude.ai, say), the service
  keeps running with nobody behind it. On Linux the watch brings it back by itself, with
  a new, empty session of the same name, in about ten minutes (see
  [`remote watch`](commands.md#augmented-brain-remote-watch)).
- Sessions you start from the phone belong to the space of their folder: the work door
  opens work notes. To reach another space, use that space's door.

### An archived session on macOS and Windows

There is no watch there, and installing again is not enough: Claude keeps a note of the
archived session, `bridge-pointer.json`, and a new service picks it up again and loses
it at once. Set the note aside by hand:

1. Stop the doors:

   ```bash
   augmented-brain remote uninstall
   ```

2. In `.claude/projects` inside your home folder, open the folder whose name ends in the
   name of the door's folder (Claude names it after the whole path, with `-` in place of
   each separator). Rename `bridge-pointer.json` there to `bridge-pointer.json.archived`.
   Do not delete it.
3. Start the doors again:

   ```bash
   augmented-brain remote install
   ```

A new, empty session of the same name shows up within a few minutes: Claude's side first
lets the old one go. The conversation of the archived session does not come back.

## Take it away

```bash
augmented-brain remote uninstall
```

It stops the services and removes them, and on Linux the watch too. On Windows it also
ends the programs those services had started. The folders, and their place in the
space's work folders, stay, and nothing else is deleted. Remove them by hand if you want.

## The settings

`remote.toml`, in the configuration folder: `enabled`, `spaces`, `name_prefix`,
`permission_mode` and `folders`. See [Configuration](configuration.md#remotetoml). The
commands are in [Commands](commands.md).
