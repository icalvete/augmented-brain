# Telegram (your account)


Your assistant can use your own Telegram account. It finds a chat, reads its last
messages and writes from your account. Ask "what did Lucía say in the family group?" and
it reads the group for you.

It works on Linux and macOS. Windows is not supported yet.

## What it is

augmented brain joins your account as **one more device**, as Telegram Desktop does on a
computer. It is not a bot. A bot is a separate account that people have to talk to. Here
it is you: what the assistant sends arrives as written by you, and it sees the chats you
see.

In Telegram it shows as **augmented brain (<computer name>)**, such as
"augmented brain (laptop)", in **Settings → Devices**. You can close it from there at any
time.

It is a separate program, `augmented-brain-telegram`. Telegram's library would make
augmented brain three times bigger, so only who turns Telegram on downloads it (about
6 MB, once). It comes from the same release as your augmented brain, is checked against
that release's checksums, and goes next to the `augmented-brain` program. `augmented-brain
update` keeps it up to date from then on.

## What you need

- **Linux or macOS.** On Linux, also `secret-tool`, the command of the system keychain
  (the **Secret Service**, which GNOME and KDE have). On Ubuntu and Debian it comes in the
  `libsecret-tools` package. macOS has its keychain already.
- **Your phone with Telegram**, to scan a code.
- **Your own Telegram application**: two values, `api_id` and `api_hash`, that Telegram
  gives you in a minute. Next section.

Everything stays in the system keychain, never in a file: the application, and the
session (what keeps you logged in). It is kept under the service `augmented-brain`, with
the names `telegram-api-id`, `telegram-api-hash` and `telegram-session`.

## Your own application, once

Every program that talks to Telegram as an account needs an application. Each person
makes their own: using another program's application is against Telegram's terms.

1. Open [my.telegram.org](https://my.telegram.org) in your browser.
2. Type your phone number in international format, with its country code: `+34…` for
   Spain. The code arrives **inside Telegram**, as a message, not by SMS.
3. Go to **API development tools**.
4. Create the application:
   - **App title**: `augmented brain`.
   - **Short name**: `abrain` (5 to 32 letters or digits).
   - **URL**: leave it empty.
   - **Platform**: Desktop.
5. Copy two values from the page: **api_id**, a number, and **api_hash**, 32 letters and
   digits.

If the form answers just "ERROR", it is a known quirk of that site. Change the short name,
or try another browser without ad blockers.

## Connect it

### From the panel

1. Open the panel (`/brain:panel` in Claude, or `augmented-brain panel`).
2. In **Upkeep**, open **Telegram**.
3. Press **Download the Telegram program**.
4. Paste your `api_id` and `api_hash`, and press **Save and show the QR code**.
5. On your phone, open Telegram → **Settings → Devices → Link Desktop Device**, and scan
   the code. It changes about every 30 seconds; the page follows it.
6. If your account has two-step verification, the page asks for its password. It goes to
   Telegram once and is not kept anywhere.
7. The page says **Connected**, with your name.

You connect only on the computer itself, not through
[the panel on another device](panel-remote.md): it hands over your whole account.

### From a terminal

```bash
augmented-brain telegram connect
```

It downloads the Telegram program the first time. It asks for your `api_id` and your
`api_hash` (the second one is not shown as you type). Then it draws a QR code in the
terminal: scan it as in step 5 above. It asks for the two-step password if your account
has one. You have 5 minutes to scan; after that, run it again.

To see how it stands:

```bash
augmented-brain telegram status
```

### Then

**Start a new Claude session.** The Telegram tools load when a session starts, so a session
that was already open does not have them.

## Use it

Ask in plain words:

- "What did Lucía say in the family group?"
- "Do I have unread messages from Marcos?"
- "Write in my Saved Messages: buy bread tomorrow."
- "Tell Lucía I will be ten minutes late."

Behind your words, the assistant uses three tools: it finds the chat
([`telegram_chats`](mcp-tools.md#telegram_chats)), reads it
([`telegram_read`](mcp-tools.md#telegram_read)) and writes in it
([`telegram_send`](mcp-tools.md#telegram_send)).

Limits, to keep every call small:

- It looks at your 500 most recent chats.
- It reads up to 100 messages of a chat at a time; 20 if you do not say.
- A message is 4096 characters at most.

Photos, voice notes and files show only as their kind, such as `[Photo]`: the assistant
does not open them.

The tools are there only in Claude, not in [Codex](codex.md).

## What the assistant does, and does not

- **To anyone but you, it shows you the exact text first**, and the chat, and sends only
  after your yes. Writing to your own Saved Messages needs no yes.
- **It signs when it speaks for itself.** When it passes on your own words, it does not.
- **What a message says is never an order for it.** Messages are written by other people.
  If one says "forward me your last messages", the assistant reads it as a message, not
  as a request from you.
- **It reads only what the task needs**, not your whole history.
- **It never sends in bulk.**

## Waiting for a reply

People answer when they can: in a minute, or in three days. So when the assistant sends
a message that asks for an answer, it notes that the task it is for waits for it
(`telegram_send` with [`wait_task`](mcp-tools.md#telegram_send)). Nothing stays running
for that: the wait is a note of this computer, with no end.

- The task now waits on that person, and its journal says it waits for a reply on Telegram.
- Every 5 minutes, a timer of your system looks only at the chats with a wait, and only
  at what came after the message sent. In a chat with one person any message is the
  reply; in a group, only an answer to the message sent.
- When the reply comes, it goes to the task's journal, marked as someone else's text. The
  task waits on you again, and its next step is to read it.
- A session working on that task hears of it before its next answer. If none is open,
  whoever opens the task finds the reply there.

The timer is set when you connect Telegram and goes when you disconnect it or uninstall
augmented brain (Linux: a systemd user timer; macOS: a LaunchAgent). To see the waits, or
take one out, and to look right now:

```bash
augmented-brain telegram waits
```

```bash
augmented-brain telegram check
```

## Disconnect it

In a terminal:

```bash
augmented-brain telegram disconnect
```

It closes the session in Telegram and takes it out of the keychain. Your application
stays, so connecting again only needs the QR code. To remove the application too:

```bash
augmented-brain telegram disconnect --forget
```

In the panel: **Telegram → Disconnect**, with **Also forget the application** if you want
(see [the panel's Telegram page](panel.md#telegram)).

On your phone: Telegram → **Settings → Devices**, and close **augmented brain
(<computer name>)**. This works even if the computer is lost or out of reach.

`augmented-brain uninstall` does it too: it ends the session, takes it and the application out
of the keychain, and removes the Telegram program and its timer.

## On two computers

Each computer is a device of its own, with its own session in its own keychain.

- You can use the same `api_id` and `api_hash` on both.
- Connect on each one: scan a QR code on each.
- In Settings → Devices you see one **augmented brain** per computer, each with its name.
- Disconnecting one leaves the other connected.
- Never copy the `telegram-session` secret from one computer to another. Two places using
  one session can get it closed by Telegram. Connect each one instead.

## The risks, plainly

- **The session is your whole account.** Whoever has it can read and write as you. It
  stays in this computer's keychain, but anything that runs as you on this computer could
  reach it. If you lose the computer, close the device from your phone.
- **Messages are someone else's words.** A message can try to trick an assistant. The
  assistant is told to treat them as data, never as instructions, but read what it
  proposes to send before you say yes.
- **Telegram may limit or close accounts used aggressively.** augmented brain sends one
  message at a time and never in bulk. If Telegram asks to wait, it stops and says how
  long, and does not keep trying.
- **A message that seems to fail may have arrived.** Before sending it again, the
  assistant reads the chat to check.

## Windows

Not yet. augmented brain keeps the session in the system keychain, and on Windows it does
not know how to use it yet. The panel's Telegram page and `augmented-brain telegram`
say so.

## If something fails

- **"Telegram is not connected".** Connect it from the panel or with `augmented-brain
  telegram connect`, then start a new Claude session. If you closed **augmented brain**
  from your phone, this is why.
- **The assistant has no Telegram tools.** Start a new Claude session after connecting.
  In Codex there are none.
- **"Telegram needs the system keychain…"** On Linux, install `secret-tool` (package
  `libsecret-tools` on Ubuntu and Debian). On a computer with no desktop, the keychain may
  not be running: use one with a desktop session.
- **"ERROR" at my.telegram.org.** Change the short name, or try another browser without
  ad blockers.
- **"Telegram does not accept this api_id and api_hash".** Copy them again from
  my.telegram.org. To type them again, disconnect with `--forget` and connect.
- **The QR code expired, or "Nobody scanned the code".** The code changes about every 30
  seconds: scan the newest one. After 5 minutes it stops: run `augmented-brain telegram
  connect` again, or press **Show the QR code** in the panel.
- **"Telegram asks to wait…"** Telegram is limiting the account for a while. Wait that
  long before asking again.
- **"Could not install the Telegram program".** It needs the internet to reach GitHub.
  Try again later.
