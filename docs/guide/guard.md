# The guard

[Leer en español](guard.es.md)

Claude is about to run `git push origin main`. Before it does, augmented brain reads the
command and stops it: pushing is for you. Claude tells you the change is ready, and you
push it yourself.

That check is the **guard**. It reads every command Claude runs, and every file it writes,
before it happens. It has three answers:

- **Go ahead**: almost always. `ls`, `go test`, `aws --profile dev s3 ls`.
- **Ask you**: Claude Code shows the command and waits for your yes. `aws --profile prod
  …`, `ssh db01 …`.
- **Stop**: the command does not run. Claude gets the reason, and what to do instead.

The reasons are in your language, the `language` of your
[configuration](configuration.md) (the system's while it cannot be read).

**The guard is a safety net against slips, not a security boundary.** It reads commands
as text. It catches the mistakes an assistant makes, also hidden inside other commands.
A program determined to get around it could. Read the
[disclaimer](../../DISCLAIMER.md).

## What it stops

- **Commits and pushes.** `git commit`, `git push`, and what publishes like a push:
  `git subtree push`, `git send-pack`, `gh release create`, `glab release create`.
  Unless you let Claude in a repository (below). Asking for help passes: `git commit
  --help`, `gh pr merge --help`, `su --version`.
- **Becoming root.** `sudo`, `su`, `doas`, `pkexec`, `run0`, `sudoedit`, `systemd-run`
  without `--user`, `ssh root@…` (also `-o User=root`), a container given the whole
  machine (`docker run --privileged`, `-v /:/host`, `--pid=host`, the Docker socket or one
  of the machine's disks; with `podman` and `nerdctl` too, and `kubectl debug node/…`), and on Windows `gsudo` and `runas`. Claude
  gives you the exact command to run yourself. Running as another user (`sudo -u www-data …`) is fine.
- **Secrets.** A token or key written in a command or in a file Claude writes. Secrets are
  read from the system keychain, an environment variable or a secrets manager, never
  typed in.
- **Wiping your home folder or the whole disk.** `rm -r` of `~` or `/`, also after a
  `cd` (`cd ~ && rm -rf .`), and on Windows however the path is written.
- **PowerShell.** The guard reads the shell, not PowerShell, so it stops every PowerShell
  command, and `cmd` or `powershell` started from Bash too. Claude runs it in Bash (Git
  Bash on Windows) instead. You will see that block on Windows: it is expected.

## What it asks you first

- **Production.** An `aws` command with a profile you marked as production. It checks
  every profile the command may use: `--profile`, `AWS_PROFILE`, `AWS_DEFAULT_PROFILE`,
  or `default` when none is set. The same for tools that call AWS on their own: `cdk`,
  `sam`, `serverless`, `terraform`, `tofu`, `pulumi`, `eksctl`, `copilot`. And any other
  program with a production profile set in the command itself (`AWS_PROFILE=prod bash
  deploy.sh`): whatever it runs may reach AWS with it.
- **Machines not to touch.** `ssh`, `scp`, `rsync`, `sftp`, `mosh`, `sshfs` and `autossh`
  to a host you marked as untouchable, also when they only go through it (`-J`,
  `ProxyJump`) or name it in `-o HostName`. `db01` and `db01.example.com` count as the same host.
- **Starting a release workflow by hand** (`gh workflow run release`, or the same
  through `gh api`). It publishes a version, so it is your call, but you do not need a
  terminal: say yes only if you asked for that release now. If you say no, or nobody
  answers, nothing starts. In `bypassPermissions` mode, where Claude Code may skip that
  dialog, it is stopped instead. Other workflows go on; one named only by its number
  asks too.
- **Pushes that rewrite history**, even where Claude may push: `--force`, `--delete`,
  `--mirror`, `--prune`, a refspec starting with `+` or `:`.
- **What it cannot read.** A program given as a variable (`$TOOL deploy`), commands
  nested too deep (more than four commands, one inside another: `sudo`, `ssh`, `bash -c`,
  `find -exec`, a git alias), a command longer than 256 KB, `rm -rf "$DIR/"*` with a
  variable that may be empty, an `rm -r .` after a `cd` it cannot follow, a script it
  cannot read or larger than 64 KB, a script that the same command writes in a way it
  cannot read and then runs (`curl -o x.sh … && sh x.sh`).
- **The guard's own settings.** Claude never changes augmented brain's configuration or
  Claude Code's settings on its own, since a line there could switch a check off.

## Let Claude commit in a repository

By default, Claude never commits or pushes. For a repository of yours where you want it
to, list it in the configuration, in any space:

```toml
[spaces.work]
git_allowed_repos = ["~/code/blog", "~/code/blog-*"]   # commit and push freely
git_ask_repos = ["~/code/shop"]                        # ask me each time
```

- Claude must name the repository in the command: `git -C ~/code/blog push`.
- A git worktree of a listed repository counts as it.
- `~/code/blog-*` covers every `blog-<name>` folder next to `blog`.
- A repository in both lists asks.
- The lists of every space count together: a repository is yours, not the space's.

The easiest way is the panel's **Guard** page. Adding a repository to the list where
Claude commits and pushes without asking shows a warning, and you must tick a box that
says you understand. Add only repositories where a mistake can be undone.

**Merging a pull request is a push too.** `gh pr merge`, `glab mr merge` and `glab mr
accept` follow the same lists. gh and glab have no `-C`, so the command must say which
repository it is:

- `cd ~/code/blog && gh pr merge 12`: the folder in the `cd` is the repository.
- In the repository's own folder, Claude Code removes that `cd` before the guard sees
  it. There, Claude names the repository: `gh pr merge 12 -R owner/name`, or the pull
  request's address.
- A repository named that way must be one of the folder's remotes, or the merge is
  stopped.
- The folder the session is in does not count on its own.

`gh pr merge --admin`, which skips the branch's protection, always asks. Only the command
itself counts: `gh pr merge` written in a pull request's body or a commit message is not
a merge, and neither is an address or `--admin` in the text given to `--body` or
`--subject`.

## Repositories Claude must never touch

Some repositories are not yours alone, or are one wrong push away from production. List
them as **untouchable** and the guard denies `git commit`, `git push` and merging a pull
request there, whatever the other lists say: it wins over `git_allowed_repos` and
`git_ask_repos`. They live in `guard.toml`, one list for the whole computer:

```toml
untouchable_repos = ["~/work/core", "~/work/cli-*"]
```

- Same form as the other lists: `~`, a `*` in the last part, and a worktree counts as its
  repository.
- The answer says the repository is untouchable and that only you can change it. Claude
  is told not to look for another way.
- Add and remove them in the panel's **Guard** page. Removing one asks you to type the
  folder's name, and warns you: from then on the other lists decide, so if the repository
  is in `git_allowed_repos`, Claude will commit and push there.

## Questions that say what they are about

When a repository is in `git_ask_repos`, the question names the repository, the branch
and where the remote leads (the address, never a password): `Branch: feat/x. Remote:
git.example.com/acme/app`. The same for merging a pull request.

## Only you change the guard's lists

The assistant works in the same account as you, so the guard cannot be a wall. It does
three things so that the lists stay yours:

- **It asks before a write to the guard's files** (`config.toml`, `guard.toml`,
  `secrets.toml`, `remote.toml`, `session.toml`, and Claude Code's settings).
- **It denies reaching the panel.** Reading `panel.json` (the panel's key) with a tool
  or a command, and calling a panel page from a command, are denied, however the
  address is written: the panel's Guard page is how you change the lists, and it works
  with that key. `augmented-brain panel` never prints the key either: it prints a link
  that works once, and run by Claude it prints none.
- **It notices a change it did not see.** The guard keeps a seal of its lists: taken
  when you start a session and when you save in the panel. If the lists differ from the
  seal, something else changed them, and a commit, push or merge that the lists would
  let through asks first, and the Guard page and `doctor` say so, until you accept them
  in the Guard page or you start a new session. A new session takes the lists as they
  are, and Claude Code tells you so itself. When Claude Code compacts a conversation, or
  after `/clear`, the seal stays: that is not you looking. If the seal cannot be read,
  the guard asks too, and `doctor` says why. The Guard page shows what was added and
  removed since then.
- **It tells you what the panel changed.** A change to the lists made in the panel, or
  lists accepted there, reaches you as a line from Claude Code itself, not from Claude,
  after the next answer or message in your open sessions.

What this does not stop: a script that Claude writes in line (`python -c`, a heredoc) can
still edit the files, as it can anything your account can. The guard then notices the
change and asks before a push, but it did not prevent it. It is a safety net, not a
security boundary.

## Production profiles and untouchable machines

You give them in the first-session interview, or any time: "the AWS profile `shop-prod`
is production", "never touch `db01` without asking me". Claude saves them with the
`safety_save` tool. They hold on the whole machine, in every space.

Claude can only add to these lists. To take one out, use the panel's **Guard** page, or
edit `production_aws_profiles` and `untouchable_hosts` yourself
([Configuration](configuration.md)).

These lists are in no backup, because they may name your work's machines. After a
restore on a new machine, the next session offers to ask you for them again.

## Root on one machine, for a while

Sometimes Claude really needs root on a machine, for a migration, say. Add an exception
with an end date to the configuration:

```toml
root_ask = ["db01 until 2026-12-31"]
```

Until that date, root on `db01` asks you each time instead of being stopped. The command
it runs is still checked. After the date, root is stopped again by itself, and
`augmented-brain doctor` says the line can go. The date is required, so no exception is
left behind.

You can also add it in the panel's **Guard** page. It asks you to type the machine's name
again, and reminds you that until that date Claude may run commands as root there, after
you confirm each one. Keep the date close.

## It looks inside

The guard reads what runs inside other commands: `ssh host '…'`, `vagrant ssh -c` (and after `--`),
`docker exec`, `podman exec`, `kubectl exec`, `bash -c`, loops, what a pipe feeds a shell
(`echo 'sudo …' | bash`), short scripts Claude runs (`bash deploy.sh`,
`./deploy.sh`), also when the same command writes them first (`echo '…' > x.sh && bash
x.sh`), `env -S '…'`, and your git aliases (`git ci` when `ci` stands for `commit`). A shell that only checks a script's syntax
(`sh -n install.sh`, `bash -o noexec`) runs nothing, so it is let through. It checks the commands Claude leaves running to watch something, and
those of the helpers Claude starts (subagents and workflows).

## When something is wrong with it

- **The configuration cannot be read.** Then the guard cannot know your lists. It asks
  you before every `aws`, `ssh`, `scp`, `rsync`, `sftp` and `mosh`, and before the tools
  that call AWS. To go back to a version that worked, run `augmented-brain config history`
  and then `augmented-brain config restore <number>`
  ([Configuration versions](config-history.md)).
- **The program is missing or broken.** Claude Code stops every command and file change
  instead of letting them through (Claude Code 2.1.295 or later; `doctor` warns about an
  older one). If the guard itself fails while checking, it refuses that one action and
  Claude tells you: it is a bug in augmented brain, which you can report from the
  panel's Help page.
- **The configuration folder is unknown** (no `HOME`, as in some services and
  containers), **or `config.toml` is gone but the rest is still there.** The guard asks,
  as when the configuration cannot be read.
- **`guard.toml` cannot be read.** The guard asks before every commit and push until it
  is fixed or removed.
- `augmented-brain doctor` tries the guard for real and tells you if it is not working.

## Next

- [Configuration](configuration.md): every list, with an example.
- [Troubleshooting](troubleshooting.md): when the guard stops something it should not.
