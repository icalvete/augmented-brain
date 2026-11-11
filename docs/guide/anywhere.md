# Working from your phone or another computer

[Leer en español](anywhere.es.md)

Your notes live on one computer. You can still work with them from your phone, a tablet
or another computer. There are two ways, and they do different things:

- **Talk to Claude** on that computer, from anywhere: [remote access](remote.md). You pick a
  session in [claude.ai/code](https://claude.ai/code) or Claude Desktop, and work as if you
  were sitting there.
- **Look at your notes** in the panel, from anywhere: [the panel from another
  device](panel-remote.md). Today, your tasks, the inbox, everything the panel shows.

You can use one, the other, or both. Remote access works on Linux, macOS and Windows; the
panel from another device works wherever [Tailscale](https://tailscale.com) does.

## Set it up once

On the computer that holds your notes:

1. **Remote access.** Run `augmented-brain remote install`, then do what it asks: let
   Claude trust the folder and, on Linux, turn on "linger" so it survives a reboot (on
   macOS and Windows it starts when you log in). Details in [Remote access](remote.md).
2. **The panel.** Install Tailscale on the computer and on your phone, signed in to the
   same account, then run `augmented-brain panel remote`. It prints an address and a QR
   code. Details in [The panel from another device](panel-remote.md).

`augmented-brain doctor` checks both, one line each.

## Which session to use

Remote access opens **one door per space**, named `<computer name>-<space>`, such as
`my-laptop-personal`. A session started through it begins in that space's first work
folder, so Claude loads that space's notes and rules.

- In claude.ai or Claude Desktop, pick the environment with that name and start a session.
  You can start as many as you like.
- To work on a task, say so: "resume task-bake-bread". Claude reads the task and carries
  on where it was left.
- To reach another space, use that space's door. Notes never cross between spaces.

## When it stops answering

- **The computer is asleep.** A laptop that sleeps drops the connection. It comes back when
  it wakes up.
- **The session was archived.** Archiving a session in claude.ai, or leaving it unused
  for long, can end it on Claude's side while the service on your computer keeps running
  with nothing behind it. Start the service again, on the computer:

  ```bash
  systemctl --user restart augmented-brain-remote-personal
  ```

  The new session can take up to a few minutes to appear: Claude's side first lets the
  old one expire. The watch does this by itself within a few minutes; if you restart
  by hand, run `augmented-brain remote watch` first, so the sessions of your tasks
  started from there keep running.
- **Your Claude login expired.** `augmented-brain remote status` says so. Run `claude`
  once on the computer and sign in again.

`augmented-brain remote status` says how each door stands, and what to do.

## An old session with the same name

When the door of a space moves to another folder (a new version starts it in your work
folder, say), Claude's side keeps the old session too, with the same name, in another
group of the list. Nothing answers in the old one: what you write there is lost. Use the
one in the group named after the new folder, and archive the old one in claude.ai or
Claude Desktop.

## Starting new sessions from the phone

Ask the session behind the space's door to open one for a task ("open me a session for
the website task"). It runs `augmented-brain session start`, and a session of its own for
that task shows up in your list within a minute, named after the task, in its folder,
already reading it. Open as many as you have tasks. The first time in a folder it tells
you how to make Claude trust it. See [A session per task](remote.md#a-session-per-task).

## Keep it safe

Both are doors to your computer. Remote access is a permanent one: anyone who gets into
your Claude account can work on the machine through it. Protect that account with a strong
password and two-step verification, keep Claude asking before it acts (never
`bypassPermissions`), and remove the doors you do not use: `augmented-brain remote
uninstall` and `augmented-brain panel stop`. The panel's address works only for your own
devices in your Tailscale network, and only while the panel runs.
