- **Saving a production profile or a machine no longer makes the guard ask about every
  commit.** Claude adding one to lists you had accepted used to leave them "not
  accepted" until the next session, so each commit, push, `aws` or `ssh` asked first.
  Adding only makes the guard ask more, so the lists stay accepted; a change you had not
  accepted yet still waits for you.
