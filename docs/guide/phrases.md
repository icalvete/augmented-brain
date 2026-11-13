# What to say to Claude

[Leer en español](phrases.es.md)

You do not learn commands. You talk to Claude in your own words, in English or Spanish,
and it uses augmented brain's tools behind the scenes. These are phrases that work. Any
wording with the same meaning works too.

## Tasks

| You say | What happens |
|---|---|
| "New task: move the blog to a new host." | Opens a task in definition, waiting on you, and asks what "done" means |
| "Where were we?" | Lists your live tasks, those waiting on you first, each with its next step |
| "Let's carry on with the blog." | Reads the task, checks the real state, and picks it up where you left it |
| "Go on to the plan." | Moves the task to its next phase; you approve each change. If you skip a phase, the journal says which |
| "We go with the cheaper host." | Writes the decision in the task's journal, with the date |
| "Add a step: switch the domain." | Adds a checkbox to the task's plan |
| "The database is copied." | Ticks that step |
| "That part waits on Diego: make it a subtask." | Opens a subtask with its own status and next step |
| "On the 16th, check the certificate is renewed." | Adds a watch date to the task |
| "Park it, we carry on tomorrow." | Writes the exact next step and the essentials, and pauses the task |
| "Leave it for some day." | Parks it as someday: out of the daily list, kept in its area |
| "Close the task." | Closes it with its result; it is never deleted. A closed task stays closed: closing it again is refused |
| "Reopen the blog task: the certificate failed again." | Opens it again; the reason goes to its journal |
| "Open me a session for the blog task." | Starts a Claude session of its own for that task, in its folder; it shows up in Claude Desktop and claude.ai within a minute. Linux only for now ([how](remote.md#a-session-per-task)) |

## Inbox and reminders

| You say | What happens |
|---|---|
| "Note this: look at the new hosting prices." | One dated line in the inbox, no task opened |
| "Remind me on Thursday to call the insurance." | A capture with a date: it comes up at the start of the session |
| "Note this in work: renew the parking card." | The capture goes to the inbox of your space "work", marked with the space it came from. It comes up there at every session start until it is done |
| "I called the insurance, that one is done." | Takes that capture out of the inbox without waiting for the weekly review |
| "Let's do the weekly review." | Goes through the inbox and the stalled tasks with you, one at a time |

## Knowledge, rules and people

| You say | What happens |
|---|---|
| "Save that as a lesson." | A lesson, with a title that states the conclusion |
| "Write down how the deploy works." | A reference, in the task's area |
| "The newsletter is a new area." | Creates the area |
| "From now on, answer in short sentences." | A rule, loaded in every session |
| "This holds everywhere." | Saves the rule or lesson as shared, for every space |
| "That rule no longer holds." | Retires it: it moves to `archive/`, with when and why, and stops reaching your sessions. Nothing is deleted. Claude asks you first |
| "Diego runs the hosting company; I call him Dieguito." | Saves Diego as a person, with his nickname |
| "Why did we pick the cheaper host?" | Finds the decision in your notes, with date and reason |
| "Change my profile: I prefer the recommendation first." | Rewrites your profile |
| "Which space am I in?" | Says the space of this session, its notes folder, its areas and how many tasks are live |

## Safety

| You say | What happens |
|---|---|
| "The AWS profile shop-prod is production." | The guard asks you before commands that use it |
| "Never touch db01 without asking me." | The guard asks you before ssh, scp or rsync to it |
| "I stored a new secret, telegram/alerts-bot, for alerts to my phone." | Writes down its name and what it is for, never its value |
| "Ask me before you save a rule." | This is not set by talking: you choose "Ask me first" on the panel's Guard page, and Claude tells you where. From then on, Claude Code asks you before a rule, your profile or the guard's lists are saved ([more](guard.md#asking-before-claude-saves-a-rule)) |

## Backup, sessions and the panel

| You say | What happens |
|---|---|
| "Set up the backup." | Creates a private repository in your GitHub and makes the first copy |
| "Recover the session that was cut off." | Reads what was said after the last journal entry, and writes down what is missing |
| "~/code/blog is the blog, built with Hugo." | Writes that line for the repository, shown in the panel |
| `/brain:panel` | Opens the panel in your browser |

## When Claude talks first

At the start of a session, Claude may bring things up without being asked:

- a watch date or reminder due in the next 7 days, or already past;
- the weekly review, when the last one is a week old and there is something to go
  through;
- a session that was cut off, to recover;
- a backup that failed, or one that left files out;
- a new version, to update;
- notes with problems;
- what another space sent to this one, until it is done;
- the guard's lists, if something changed them outside the panel;
- open sessions still running the program from before an update, which need restarting.

And while you work:

- tasks that changed outside this session (another session, the panel or an edit by
  hand), before it answers your next message;
- another model answering than the one you chose, once per session, if you set the
  model of your sessions on the Guard page;
- the window using a lot of memory: at 2 GB, then 4 and 8 GB, once each (Linux and
  macOS; [what to do](troubleshooting.md#the-window-uses-a-lot-of-memory)).

## Next

- [MCP tools](mcp-tools.md): the tool behind each phrase, and what it takes.
- [The panel](panel.md): the same things, by clicking.
