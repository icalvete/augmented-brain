# changelog.d

Each pull request that changes something people notice leaves its changelog entry here,
as two files of its own: `<name>.en.md` and `<name>.es.md`. The branch's name is a good
`<name>`. Two pull requests never write the same file, so merging one does not leave the
others in conflict, and none of them has to run CI again for it.

Write the entry exactly as it will read in `CHANGELOG.md` and `CHANGELOG.es.md`: a list
item that starts with `- **What changed, in one sentence.**`, then why it matters, in
plain words, wrapped at 89 columns. No heading: the release writes it.

```markdown
- **The panel opens faster.** It used to read every note before showing the first page;
  now it reads only what that page shows.
```

A pull request does not touch `CHANGELOG.md` or `CHANGELOG.es.md`. The release branch
gathers every entry under the new version and empties this folder:

```bash
python3 -I scripts/changelog.py release 0.23.0-beta
```

`scripts/precheck.sh` checks that each entry has its twin in the other language, that
the branch leaves the changelogs alone, and that a branch changing the program adds an
entry. For a change no one notices (a refactor, a test), put a line `Changelog: none` in
one of its commit messages.
