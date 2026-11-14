- **Large spaces answer faster.** With 2,000 tasks, each tool call of the assistant
  takes about 100 ms instead of 145. The list of tasks is no longer built in a way that
  slowed down with the square of their number (8,000 tasks: 10 ms instead of 3 s). The
  record of which tasks a session has seen is no longer rewritten on every message when
  nothing changed. An open note page in the panel checks for changes without reading
  the whole space, and a search for a common word no longer prepares every result it
  will not show.
