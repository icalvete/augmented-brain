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

`augmented-brain doctor` checks both: a line for the door of each space and, once you
have used it, one for the panel from another device.

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
- **The session was archived** in claude.ai, or ended on Claude's side. On Linux it comes
  back by itself, in about ten minutes; on macOS and Windows it takes three steps by
  hand. Both are in [Remote access](remote.md#good-to-know).
- **Your Claude login expired.** Run `claude` once on the computer and sign in again.

`augmented-brain remote status` says how each door stands, and what to do. Other
problems are in [Troubleshooting](troubleshooting.md#remote-access-does-not-work).

## A session for each task

On Linux, the session behind a space's door can open another one for a task ("open me a
session for the website task"). It shows up in your list within a minute, named after
the task, in its folder, already reading it. Open as many as you have tasks. On macOS
and Windows it is not available yet. See [A session per task](task-sessions.md).

## Keep it safe

Both are doors to your computer. Remote access is a permanent one: anyone who gets into
your Claude account can work on the machine through it. Protect that account with a strong
password and two-step verification, keep Claude asking before it acts (never
`bypassPermissions`), and remove the doors you do not use: `augmented-brain remote
uninstall` and `augmented-brain panel stop`. The panel's address works only for your own
devices in your Tailscale network, and only while the panel runs.
