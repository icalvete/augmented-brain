<!-- Detail for AI coding agents and contributors. English only: see AGENTS.md. -->

# Rules of the house, in detail

- **Nothing personal, no secrets.** The repo is public. `scripts/privacy-check.sh` runs as
  the pre-commit hook (`git config core.hooksPath .githooks`) and in CI. Never bypass it.
  Its denylist of the maintainer's personal data lives outside the repo; CI checks it by
  keyed fingerprints (`scripts/privacy-hashed.py`, `scripts/privacy-hashes`, the key in the
  `AB_PRIVACY_KEY` secret). After changing the denylist, run `scripts/privacy-hashed.py build`
  and commit the fingerprints.
- **Two languages, always in step.** Everything a person reads exists in English and
  Spanish: a new message goes in `internal/i18n`, in the file of its subject, with both
  languages (`TestCatalog` checks they have the same placeholders); no package keeps a
  language table of its own. Outside Go: `say` in `install.sh`, the templates,
  `README.md` and `README.es.md`, each entry of `changelog.d/` (`.en.md` and `.es.md`),
  `docs/index.html` and `docs/es/index.html`, each guide and its `.es` twin. Change one,
  change the other. The exceptions are for developers only: `AGENTS.md`, `dev/`, code,
  comments and commit messages are in English alone. `dev/` lives outside `docs/` for
  that reason: Pages publishes `docs/`, and its tests want a Spanish twin for every guide.
- **A change people notice brings its changelog entry, in a file of its own.** Never in
  `CHANGELOG.md` or `CHANGELOG.es.md`: two pull requests that both add a line there
  conflict as soon as one merges. The entry goes in `changelog.d/<name>.en.md` and
  `.es.md` (the branch's name; how to write it in `changelog.d/README.md`), and the
  release gathers them. `scripts/changelog.py check --base origin/main` (in precheck)
  wants one when the branch changes the program, and accepts a commit line
  `Changelog: none` for a change no one notices.
- **The docs are checked as people read them.** `cmd/augmented-brain/docsite_test.go`: every
  relative link and #anchor leads somewhere, a Spanish page links Spanish guides, every
  guide has its `.es` twin with the same headings and examples, sits in `GROUPS` and is
  linked from both READMEs, and every `augmented-brain …` example is a real command with
  flags it reads. CI also checks the generated pages match the guides.
- **The docs follow the code.** A new command or flag goes in `commands` (`cmd/augmented-brain/commands.go`) and in the usage text; its help is its section of the reference (`TestEveryCommandHasHelp`); a new command, flag, MCP tool, tool parameter or `config.toml` setting also goes in its reference page, in both languages, or `go test` fails (`TestCommandsDocumented`, `TestToolsDocumented`, `TestSettingsDocumented`). A change people notice also updates the guide that tells it.
- **Generated files are not edited by hand.** Change the source and regenerate:
  `docs/img/*.svg` from `docs/img/diagrams.py`; every `docs/guide/*.html`,
  `docs/sitemap.xml`, `docs/llms-full.txt` and the help skill's `plugin/skills/help/guides/` from the `.md` sources with
  `scripts/build-guides.py` (its `GROUPS` list is the sidebar: a new page goes there), which
  also writes `internal/i18n/cmdhelp.go`, what `augmented-brain help <command>` prints, from
  each command's section of `docs/guide/commands.md` and its twin (first paragraph,
  `Syntax:`, the items of its arguments and options, first example); a space's `MEMORY.md` is written
  by the program.
- **The memory format is a contract.** `docs/spec/memory-format.md` describes it; a change
  to folders, header fields or note types updates the spec, the index and their tests.
  A change that older notes do not already follow adds a step to
  `format.NotesMigrations` (or `ConfigMigrations` for the configuration folder), which
  raises the format version by one: the step must be safe to repeat, and gets a test
  that runs it on notes written the old way.
  Anything an earlier program would refuse is such a change, even when the step has
  nothing to move: a new header field, note type or task status (headers are read
  strictly, so an earlier program would leave those notes out instead of asking to
  update).
- **What people have keeps working.** This is a promise in the README and the
  DISCLAIMER: every change to how notes or settings are kept ships with its migration
  (above), and a command, flag, setting or MCP tool that is renamed or removed keeps
  working under its old name, saying what replaced it. `compat-check` ([checks.md](checks.md)) is the
  other half: what this version writes, the last two releases still read.
- **Every guard rule says whether it holds outside the spaces.** With `reach = "spaces"`
  a session outside them keeps only the rules that protect the guard's own files and what
  the lists name (`internal/guard/outside.go`); every other rule is general and returns
  early when `p.outside` is set. A new general rule adds that line and, if its reason is
  new, its key to `generalOnly` or `generalAny` in `outside_test.go`; `checkPinned` runs
  every pinned row in both modes.
- **A test package that runs git calls `testenv.CleanGit()` first in its `TestMain`.**
  Git hands its hooks `GIT_DIR` and friends: without this, tests run from the pre-push
  hook make their throwaway repositories inside the real one.
- **A new or updated dependency regenerates `THIRD_PARTY_NOTICES.md`** with
  `scripts/third-party-notices.py` (precheck fails otherwise): its license travels in the
  release archives next to `LICENSE`.
- **Replace files with `paths.WriteAtomic`** (a crash or a power cut never leaves half a file; a note goes through `WriteAtomicNoLink`, state written at every message through `WriteAtomicQuick`), and run
  git through the backup package's runner, which never waits for a password.
- **A new MCP tool** goes in `internal/mcpserver/server.go`, does its work in
  `internal/brain`, and is added to the tool list in `server_test.go`.
- **The assistant reads what the hooks print.** Keep the session text under the budget,
  write it as instructions to the assistant, and tell it to answer in the user's language.
- **SessionEnd has 1.5 seconds in all**, and a plugin's `timeout` does not raise it. Work
  there must take well under a second (a small state file), or run in a detached process,
  as the automatic backup does. Stop runs after every answer: keep it to milliseconds.
- **The installer must work without a terminal** (CI, `AB_YES=1`) and with one; its own
  messages go through `say`, and other tools' noise through `quiet`.
- **Plain words for people.** Short sentences, one idea each; a technical word is
  explained the first time. The user of this project is a newcomer.
