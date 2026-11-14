- **No call to Claude Code, gh or git can hang a command any more.** `doctor`, `update`
  and the panel's Help page used to wait for good on a Claude Code stuck on a first-run
  screen or a slow network; now every call has a limit that holds, and `doctor` says what
  it could not check.
- **Waiting for a session to end its answer takes the time it says.** Closing or
  restarting a session waits 30 minutes at most by the clock, looks far less often, and
  writes in its log every 5 minutes that it is still waiting.
- **`watch pr` always watches.** After a reboot it could join a watch that no longer
  existed and say nothing for hours; now it always starts one when none runs, and two
  asked at once share one.
- **The remote access watch gives up instead of restarting for ever.** When restarting
  does not bring the session back, it waits longer each time and, after 5 tries, leaves
  the service alone; `remote status` and `doctor` say so.
- **`clean` keeps the scratch folders of sessions of another Claude folder** on Linux,
  and the service's log on macOS no longer grows without end.
