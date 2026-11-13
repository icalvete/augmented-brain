# The panel

[Leer en español](panel.es.md)

The **panel** is a page in your browser to see and change your spaces without going
through Claude. Open it with `/brain:panel` in Claude, or from a terminal:

```bash
augmented-brain panel
```

It opens your browser on **Today**, with the space of the folder you ran it in. From a
terminal it also prints an address, in case the browser did not open (on a computer with
no screen, say). That address opens the panel **once**, within 2 minutes; run the command
again for another. When Claude runs it for you, it only opens the browser: Claude never
sees an address that opens the panel.

The panel changes your notes the same way Claude does, with the same program, and they
wait for each other: you can have both open at once.

Every page starts with a few words on what it is and what it is for, and a link to its
part of this guide. Once you know your way around, **Hide this help** hides it on that
page, and **Hide it on every page** on all of them. The panel remembers it on this
computer; **What is this page?** brings it back.

## Spaces and pages

At the side, under **Profile**, the **Space** list picks the space you are looking at.
Some pages show only that space; others show all of them, or this computer as a whole.
The menu has three blocks: **Day to day**, **Notes** and **Upkeep**. When the window is
too short for it, the side scrolls on its own, so **Stop the panel** is always within
reach. On a phone, the menu folds under the name of the page you are on; open, it shows
each block in two columns, and a shadow at its foot says there is more below.

| Page | What it is for | Shows |
|---|---|---|
| [Today](#today) | What is due and what waits on you | every space |
| [Tasks](#tasks) | See, change, open and close tasks | one space |
| [Sessions](#sessions) | Your Claude sessions, and the ones cut off | this computer |
| [Reminders](#reminders) | Watch dates and inbox reminders | one space |
| [Inbox](#inbox) | Loose ideas, to deal with later | one space |
| [Weekly review](#weekly-review) | Empty the inbox, look at stalled tasks | one space |
| [Areas, knowledge, rules](#areas-knowledge-rules-and-people) | Read them; create areas | one space, plus the shared notes |
| [People](#areas-knowledge-rules-and-people) | Who is who | every space |
| [Repositories](#repositories) | Work only on this computer | every space |
| [Secrets](#secrets) | The names of your secrets | the same list in every space |
| [Backup](#backup) | Copies of your notes | every space |
| [Health](#health) | Whether the space is in order | one space |
| [Guard](#guard) | What Claude may do | this computer |
| [Versions](#versions) | Earlier configurations | this computer |
| [Help](#help) | Answers, and reporting a problem | — |
| [Profile](#profile) | Who you are, your assistant | every space |

## Today

The first page: every space at once. For each one:

- what is due in the next 7 days;
- the tasks waiting on you (the parked ones in one line);
- how many captures wait in the inbox;
- whether the weekly review is due;
- the sessions that were cut off, each with **Keep it in the notes** and **Mark as
  reviewed** (see [Sessions](#sessions));
- notes with problems.

Below, **Repositories**: those your open tasks name in `repos:` that hold changes not
committed or commits not pushed. That is work that is only on this computer. One whose
tasks are all parked (or left for some day) is marked "forgotten?", here and on the
whole map; a clean one never is. A branch never pushed counts only if the repository has
a remote (a copy on a server, such as GitHub) to push it to. **See all repositories**
leads to the whole map.

## Tasks

Grouped by area, those waiting on you first, with their status and next step. Filters
show only those waiting on you, under way, parked, untouched, or waiting on one person.
A task nobody has changed for 14 days is marked "untouched N days". At the end, folded,
the tasks left for **Someday** and the **Closed** ones.

Open one. Its essentials come first, the other sections folded, and the journal newest
first. Write in its journal, change its status or next step, tick its steps, park it or
close it. You can open new tasks too. If a status skips phases, the journal names them.
When a task waits on a person you have a note of, "waits on" leads to them.

To open a closed task again, find it under **Closed** and choose a new status for it; its
journal says it was reopened from the panel.

When you save a task, only what you changed on the page is saved. If Claude changed
something else in the meantime, that stays. A step is ticked only if it still says what
the page showed.

## Sessions

Your Claude sessions on this computer, from the last 30 days. It is in the menu, under
**Day to day**; **Health** and a cut-off session on **Today** also lead to it.

- Those that did not end (closed by force, or cut off when the computer went down) show
  what was said after the last journal entry. Keep it in the notes (it goes to
  `archive/sessions/` with a line in the task's journal) or mark it as reviewed.
- Those running now, and those that ended, with how much disk their transcripts take
  (the files where Claude Code keeps each conversation). You can delete the transcript of
  an ended session; that session can then no longer be resumed.

## Reminders

The watch dates of your tasks (closed ones too) and the inbox reminders: overdue first,
then the next 30 days. Mark one done, or add a watch date to a task. A reminder that
belongs to no task is a capture with a date: make it in the [Inbox](#inbox).

## Inbox

Capture an idea, with a date to be reminded if you want one. Mark one as dealt with, or
make it a task.

## Weekly review

The [weekly review](weekly-review.md), on a page. Each capture becomes a task, a
reference or a lesson, or is dropped. Each task untouched for two weeks goes on as it
is, or goes to some day (or you open it to close it). What you decide is noted as you go,
and kept in the review's record when you finish.

## Areas, knowledge, rules and people

Knowledge and rules show the space's own and the shared ones, marked "shared". Here you
read them; you do not write them. A reference or a lesson comes from Claude, or from a
capture in the [weekly review](#weekly-review); a rule, from Claude. You can create areas
and change an area's title and summary.

People are listed once, with a badge for each space they are in. A new person starts in
the space you are in. A person's page is one form: the name stays (a new name would be a
new person), then a role and other names for each space, then a list of details. Each
detail has a box per space; tick where it should show. Empty a detail's text to remove it.
A detail with no box ticked is not saved, and the page says so next to it. There is always
an empty row at the end to add one, and "Another detail" reloads the page with one more
(save first: it does not keep what you typed). The page also shows the tasks that wait on
them and those that link to them.

## Repositories

Your map of repositories: every git repository in the work folders of each space, up to
three folders down. It skips folders inside another repository, hidden folders,
`node_modules` and `vendor`. They are grouped by space, those with work pending first.

For each one: its branch, where it pushes (as `host/owner/name`, never with a password),
files changed, commits not pushed, the open tasks that name it, and one line of yours on
what it is. Write that line right there. It is kept in the space's notes, in the
reference "Repositories" (`knowledge/ref-repositories.md`), so the backup keeps it.
Claude can write it too.

The panel only reads your repositories. Git runs without taking locks, so it never gets
in your way, and it never asks for a password. It only runs inside the work folders, and
links to folders outside them are not followed.

## Secrets

The names of the secrets you keep in your system keychain (where your computer keeps
passwords), never their values. Each one as `service/name`, what it is for, and which
tasks name it. So you know which there are, and a new machine knows which to ask you for.

**Check the keychain** says whether each one is on this computer. It asks `secret-tool` on
Linux and `security` on macOS; elsewhere it cannot tell. It asks only when you click,
because a locked keychain asks you for its password. When the keychain lacks one, the
page shows the command that stores it (that command asks you for the value). When the
keychain cannot be asked (on Linux with no desktop session, or a locked keychain on
macOS), the secret shows "cannot tell here", not missing: check it there before storing
it again.

You can add a name, or take one off the list (the secret stays in the keychain). Anything
that looks like a value is refused. The list is the same in every space, and lives in
`secrets.toml` (see [Configuration](configuration.md#secretstoml)).

## Backup

Whether the notes of each space have a copy elsewhere, in a private repository, so they
survive if this computer breaks. A space with no backup says so in red.

- For a space with a backup: how the last copy went, where it goes, the last copies
  with the date and the computer that made them, and **Back up now**. The copy runs in
  the background and takes a few seconds; reload the page to see how it went. It also
  runs by itself every time a session ends.
- Each copy shows **What it changed**, note by note, with the lines. **Go back to this
  copy** shows first what would change in your notes now, and asks for a box and the
  space's name. What you had is kept as a copy first, so it can be undone. See [Go back
  to a copy](backup.md#go-back-to-a-copy).
- For a space with none: **Set it up on my GitHub** makes a private repository called
  `augmented-brain-<space>` in your GitHub account and the first copy. It needs the
  GitHub command line, `gh`, installed and logged in; the page says how if it is not.
  **I already have a private repository** takes its address instead.
- **Bring the notes back on another computer** shows the command to run there. Restoring
  is never done from the panel: it is for a computer that does not have the notes yet.

Moving a backup that exists somewhere else is done in a terminal, with `augmented-brain
backup setup`. From another device the page only shows. See [Backup](backup.md).

## Health

The space's notes folder and work folders, whether it is the default space, notes that
are not valid, repositories that notes point at and that are no longer on this computer,
the index, the last backup, the last weekly review, and a link to the space's
[Sessions](#sessions).

## Guard

Every list the guard answers to, in one page, to read and change:

- **Untouchable repositories**, where Claude never commits, pushes or merges. Removing one
  warns that the other lists decide again for it (if one lets Claude commit and push
  there, it will), and asks you to type the folder's name.
- **Repositories** each space lets Claude commit in, or ask you about, with a form to add
  or remove. The list where Claude commits and pushes without asking carries a warning,
  and adding to it asks you to tick that you understand.
- **Root exceptions**, with the date each ends (the ones that ended are marked). Adding
  one warns what it allows and asks you to type the machine's name again.
- **Production profiles** and **untouchable machines**.
- **The model of your sessions** (`preferred_model` in `session.toml`): when a session
  answers with another one, Claude tells you. Below it, what Claude Code said about its
  model at the last session start.
- **What the assistant saves on its own** (`memory_trust` in `session.toml`): whether
  Claude saves a rule, your profile and additions to the guard's lists and you are told,
  or asks you first. [The guard](guard.md#asking-before-claude-saves-a-rule) tells it.
- The files Claude cannot change on its own.

How it behaves:

- Your comments in `config.toml` stay where they are.
- If the lists are not the ones you last saw (something changed them outside this page),
  a notice says what was added and what was removed, list by list, compared with the
  lists you last accepted. When those are no longer among the kept versions, it says so.
- Then a button, **I have seen them, accept**. Until you press it, or start a new session, a commit, push
  or merge that the lists allow asks you first.
- Whatever you change or accept here, Claude Code tells you in your open sessions, after
  the next answer or message, in a line that starts with "augmented brain · the guard's
  lists". If you did not do it, look here.
- Claude cannot open this page: it never sees the panel's address, and the guard denies
  it the panel's key.
- From another device the page only shows. The lists decide what Claude may do on this
  computer, so they change only from this computer's panel.

How each list works is in [The guard](guard.md).

## Versions

The last 10 versions of your configuration (spaces, guard, remote access,
session settings, profile and your secrets' names),
kept only on this computer, newest first, with when and who changed it. **What it
changed** shows the lines of each one; **Go back to this one** makes it the
configuration again, keeping the one you had as a version first. It asks you to tick a
box, and warns first when the guard's lists would change, with what. They are never in
the backup: your notes are, with their own history. From another device
the page only shows. See [Configuration versions](config-history.md).

## Help

Links to the answers on this website, in your language. And a way to tell us about a
problem or propose an idea on GitHub (the website where the project lives; you need a
free account there).

Click **Report a problem on GitHub**. The panel runs the health check (what
`augmented-brain doctor` says) at that moment. Then it opens GitHub's form in a new tab,
with the version, your system and that check filled in.

Before it opens, the panel takes out what is personal: your home folder (shown as `~`),
your notes and work folders, e-mail addresses, the names of your spaces, untouchable
machines, production AWS profiles, and any line that looks like a password or a key.
You write what happened, and read the rest: change or delete what you prefer not to
share. Nothing is sent until you press GitHub's submit button. To be honest about it:
the text travels to GitHub inside the page's address as soon as it opens, so GitHub can
see it then, even if you never submit.

## Profile

At the top of the side, above the space list (on a phone, last in the menu). Who you are
and how you like to work: the same profile every space uses.

Under **Your assistant** you can give Claude a name and a personality. Every session
starts with them. They change how it talks, never the rules, the guard or its honesty.
With a name, every session asks Claude to sign the emails it writes in your name with
that name and, under it, a line such as "Ana's assistant (AI, Claude)", so whoever reads
it knows no person wrote it. The program writes that line, not you, so you cannot leave
it out; Claude is the one who adds it to the email. The page shows the signature. Leave
both fields empty for no name and no personality.

## Search and reading notes

The **search** box at the side looks through the notes of the space and the shared ones:
titles, summaries and text. A note shows up when it holds every word, in any case, with
accents or without.

Notes read as text, not as raw Markdown (the plain-text format notes are
written in): headings, lists, tables, code and links. A
`[[link]]` to another note takes you to it. If a note changes on disk while its page is
open (Claude wrote in it, say), the page tells you, with a link to see it as it is now.

## Made to stay yours

- It runs **only while you use it**. After 30 minutes without use it stops by itself. To
  stop it now: `augmented-brain panel stop`, or the link at the bottom of the page.
- It works **only on this computer**. Other machines on your network cannot reach it.
  To open it from your phone or another computer of yours, use
  `augmented-brain panel remote`: see [The panel from another device](panel-remote.md).
- The address you get opens it **once**. The panel then takes your browser to its
  pages, whose address carries a **key**, made new each time the panel starts. Without
  it the panel answers nothing, so another program or web page on your computer cannot
  use it. Treat the address of the pages like a password: do not share it. It stops
  working when the panel stops. If something else used your link first, your browser
  says "This link was already used" instead of opening: you know.
- It loads **nothing from the internet**: no fonts, no statistics. It works offline. A
  note cannot put HTML or scripts of its own in the page. The links of the Help page go to
  the website only when you click them.
- Checking whether a note changed does not count as using the panel: a tab left open
  does not keep it running.
