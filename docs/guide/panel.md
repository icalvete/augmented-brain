# The panel

[Leer en español](panel.es.md)

The **panel** is a page in your browser to see and change your spaces without going
through Claude. Open it with `/brain:panel` in Claude, or from a terminal:

```bash
augmented-brain panel
```

It opens your browser on **Today**, with the space of the folder you ran it in. If it
cannot (on a computer with no screen, say), it prints an address to open yourself.

The panel changes your notes the same way Claude does, with the same program, and they
wait for each other: you can have both open at once.

Every page starts with a few words on what it is and what it is for, and a link to its
part of this guide. Once you know your way around, **Hide this help** hides it on that
page, and **Hide it on every page** on all of them. The panel remembers it on this
computer; **What is this page?** brings it back.

## Today

The first page: every space at once. For each one:

- what is due in the next 7 days;
- the tasks waiting on you (the parked ones in one line);
- how many captures wait in the inbox;
- whether the weekly review is due;
- the sessions that were cut off;
- notes with problems.

Below, **Repositories**: those your open tasks name in `repos:` that hold changes not
committed or commits not pushed. That is work that is only on this computer. One whose
tasks are all parked (or left for some day) is marked "forgotten?", here and on the
whole map; a clean one never is. A branch never
pushed counts only if the repository has a remote (a copy on a server,
such as GitHub) to push it to. **See all
repositories** leads to the whole map.

## Tasks

Grouped by area, those waiting on you first, with their status and next step. Filters
show only those waiting on you, under way, parked, untouched, or waiting on one person.
A task nobody has changed for 14 days is marked "untouched N days".

Open one. Its essentials come first, the other sections folded, and the journal newest
first. Write in its journal, change its status or next step, tick its steps, park it or
close it. You can open new tasks too. A closed task opens again only if you choose a new
status for it; its journal says it was reopened from the panel. If a status skips
phases, the journal names them. When a task waits on a person you have a note of,
"waits on" leads to them.

When you save a task, only what you changed on the page is saved. If Claude changed
something else in the meantime, that stays. A step is ticked only if it still says what
the page showed.

## Reminders

The watch dates of your tasks (closed ones too) and the inbox reminders: overdue first,
then the next 30 days. Add one, or mark it done.

## Inbox

Capture an idea, mark one as dealt with, or make it a task.

## Weekly review

The [weekly review](weekly-review.md), on a page. Each capture becomes a task, a
reference or a lesson, or is dropped. Each task untouched for two weeks goes on as it
is, or goes to some day (or you open it to close it). What you decide is noted as you go,
and kept in the review's record when you finish.

## Areas, knowledge, rules and people

Knowledge and rules show the space's own and the shared ones, marked "shared". You can
create areas and change an area's title and summary.

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

## Sessions

Your Claude sessions on this computer, from the last 30 days.

- Those that did not end (closed by force, or cut off when the computer went down) show
  what was said after the last journal entry. Keep it in the notes (it goes to
  `archive/sessions/` with a line in the task's journal) or mark it as reviewed.
- Those running now, and those that ended, with how much disk their transcripts take
  (the files where Claude Code keeps each conversation). You can delete the transcript of
  an ended session; that session can then no longer be resumed.

## Profile

Who you are and how you like to work: the same profile every space uses.

Under **Your assistant** you can give Claude a name and a personality. Every session
starts with them. They change how it talks, never the rules, the guard or its honesty.
With a name, the emails Claude writes in your name end with that name and, under it, a
line the program adds and you cannot remove, such as "Ana's assistant (AI, Claude)": whoever
reads it knows no person wrote it. The page shows that signature. Leave both fields
empty for no name and no personality.

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
- The files Claude cannot change on its own.

Your comments in `config.toml` stay where they are. If the lists are not the ones you last
saw (something changed them outside this page), a notice says so and shows what changed:
what was added and what was removed, list by list, compared with the lists you last
accepted (when those are no longer among the kept versions, it says so). Then an accept
button;
until then (or until a new session starts) a commit, push or merge that the lists allow
asks first. Claude cannot open this page:
the guard denies it the panel's key. See [The guard](guard.md).

## Versions

The last 10 versions of your configuration (spaces, guard, remote access,
session settings, profile and your secrets' names),
kept only on this computer, newest first, with when and who changed it. **What it
changed** shows the lines of each one; **Go back to this one** makes it the
configuration again, keeping the one you had as a version first. From another device
the page only shows. See [Configuration versions](config-history.md).

## Health

The space's notes folder and work folders, whether it is the default space, notes that
are not valid, repositories that notes point at and that are no longer on this computer,
the index, the last backup and the last weekly review.

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
- The address carries a **key**, made new each time the panel starts. Without it the
  panel answers nothing, so another program or web page on your computer cannot use it.
  Treat the address like a password: do not share it. It stops working when the panel
  stops.
- It loads **nothing from the internet**: no fonts, no statistics. It works offline. A
  note cannot put HTML or scripts of its own in the page. The links of the Help page go to
  the website only when you click them.
- Checking whether a note changed does not count as using the panel: a tab left open
  does not keep it running.
