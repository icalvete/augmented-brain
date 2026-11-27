- **A reply on Telegram comes to its task by itself, however late.** When your assistant
  sends a message that asks for an answer, the task it is for now waits on that person,
  and the reply goes to the task's journal on its own, marked as someone else's text,
  with reading it as the next step. Nothing stays running: a timer of your system
  (systemd on Linux, a LaunchAgent on macOS) looks every 5 minutes, only at the chats
  with a wait. `augmented-brain telegram waits` lists the waits, and `telegram check`
  looks right away. Only one connection to Telegram runs at a time now, so the panel and
  two sessions no longer write over each other's session. See
  [Waiting for a reply](https://icalvete.github.io/augmented-brain/guide/telegram.html#waiting-for-a-reply).
