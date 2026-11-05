# Remote access

[Leer en español](remote.es.md)

Claude Code can be reached from another computer or from your phone: from
[claude.ai/code](https://claude.ai/code) or Claude Desktop you pick a session that runs on
your machine and work in it as if you were sitting there. For that, a small program,
`claude remote-control`, has to be running on the machine.

`augmented-brain remote` keeps it running for you. It starts by itself when the computer
starts, and starts again if it stops. You set it up once.

This is for Linux for now. macOS is still to do.

## One door per space

Each space (work, personal…) gets its own door, with its own name:
`<computer name>-<space>`, such as `my-laptop-personal`. A session opened through that door
starts in a folder that belongs to that space, so Claude loads that space's notes and
rules, and nothing crosses between spaces. By default only your default space has a door.
To open one for work too, see [Configuration](configuration.md).

## Install it

```bash
augmented-brain remote install
```

It creates the folder, writes the service, and starts it. It then tells you what is
missing. Two things always are, because only you can do them.

### 1. Let Claude trust the folder

Claude refuses to run in a folder it has not been told to trust, and there is no one at
the keyboard to answer when it starts by itself. So once, for each folder `install` names,
open Claude there and accept the question:

```bash
cd ~/.config/augmented-brain/remote/personal && claude
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

## Connect from another computer

1. Open [claude.ai/code](https://claude.ai/code) or Claude Desktop, signed in to the same
   Claude account as on the machine.
2. Choose the environment named `<computer name>-<space>`, the name `status` shows.
3. Start a session. Start as many as you like: each one begins in that space's folder.

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

`remote.toml`, in the configuration folder: `enabled`, `spaces`, `name_prefix` and
`permission_mode`. See [Configuration](configuration.md). The commands are in
[Commands](commands.md).
