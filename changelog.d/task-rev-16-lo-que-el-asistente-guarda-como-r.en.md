- **You hear of every rule, profile change or guard host Claude saves.** Claude saves
  them without asking, and every later session loads them, so a text it read could talk
  it into one. Now Claude Code itself tells you, once, at the end of the answer, what was
  saved: the rule's title and where it applies, the parts of the profile that changed,
  the hosts added. If you did not ask for it, tell Claude to undo it.
- **The list of secrets refuses more values.** Besides the tokens of known services, a
  secret's name or description is now refused when it holds a long run of letters and
  digits, a password said with its label, an address with a password, a JWT or a PIN with
  its number. The description has 160 characters at most.
