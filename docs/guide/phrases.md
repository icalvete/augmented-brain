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

## Inbox and reminders

| You say | What happens |
|---|---|
| "Note this: look at the new hosting prices." | One dated line in the inbox, no task opened |
| "Remind me on Thursday to call the insurance." | A capture with a date: it comes up at the start of the session |
| "Let's do the weekly review." | Goes through the inbox and the stalled tasks with you, one at a time |

## Knowledge, rules and people

| You say | What happens |
|---|---|
| "Save that as a lesson." | A lesson, with a title that states the conclusion |
| "Write down how the deploy works." | A reference, in the task's area |
| "The newsletter is a new area." | Creates the area |
| "From now on, answer in short sentences." | A rule, loaded in every session |
| "This holds everywhere." | Saves the rule or lesson as shared, for every space |
| "Diego runs the hosting company; I call him Dieguito." | Saves Diego as a person, with his nickname |
| "Why did we pick the cheaper host?" | Finds the decision in your notes, with date and reason |
| "Change my profile: I prefer the recommendation first." | Rewrites your profile |

## Safety

| You say | What happens |
|---|---|
| "The AWS profile shop-prod is production." | The guard asks you before commands that use it |
| "Never touch db01 without asking me." | The guard asks you before ssh, scp or rsync to it |
| "I stored a new secret, telegram/alerts-bot, for alerts to my phone." | Writes down its name and what it is for, never its value |

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
- the guard's lists, if something changed them outside the panel.

## Next

- [MCP tools](mcp-tools.md): the tool behind each phrase, and what it takes.
- [The panel](panel.md): the same things, by clicking.
