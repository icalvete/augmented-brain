#!/usr/bin/env python3
"""changelog.py: changelog entries without conflicts between pull requests.

A pull request does not touch CHANGELOG.md or CHANGELOG.es.md. It adds its entry as two
files of its own in changelog.d/: <name>.en.md and <name>.es.md (the branch's name is a
good <name>), each with the entry's lines exactly as they will read in the changelog.
Two pull requests never write the same file, so merging one leaves the others as they
were. The release branch gathers them:

  python3 -I scripts/changelog.py release 0.23.0-beta     # date: today
  python3 -I scripts/changelog.py release 0.23.0-beta --date 2026-11-20

writes a section `## <version>` under `## Unreleased` in both changelogs, with every
entry of changelog.d/ (the newest first, by when git added it; an entry still written
under `## Unreleased`, the old way, goes after them) and removes the files.

  python3 -I scripts/changelog.py check                   # the files of changelog.d/
  python3 -I scripts/changelog.py check --base origin/main

checks that every entry has its twin in the other language and starts with "- ". With
--base, also what the branch changed since it left that commit: it must not touch the
changelogs unless it changes VERSION (a release), and if it changes the program
(cmd/, internal/, plugin/, the installers; tests aside) it must add an entry, unless a
commit of the branch says `Changelog: none` on a line of its own (a change no one
notices). scripts/precheck.sh runs it with --base; CI runs it without.
"""
import argparse
import datetime
import os
import re
import subprocess
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DIR = os.path.join(ROOT, "changelog.d")
LANGS = {
    "en": ("CHANGELOG.md", "## Unreleased", "Released {}."),
    "es": ("CHANGELOG.es.md", "## Sin publicar", "Publicada el {}."),
}
NAME = re.compile(r"^[a-z0-9][a-z0-9._-]*\.(en|es)\.md$")
PROGRAM = re.compile(r"^(cmd/|internal/|plugin/|install\.sh$|install\.ps1$)")


def git(*args):
    return subprocess.run(["git", "-C", ROOT, *args], check=True, capture_output=True,
                          text=True).stdout


def entries():
    """The names of changelog.d/ (without .<lang>.md), and the problems found."""
    problems, names = [], {"en": set(), "es": set()}
    if not os.path.isdir(DIR):
        return [], problems
    for f in sorted(os.listdir(DIR)):
        if f == "README.md":
            continue
        m = NAME.match(f)
        if not m:
            problems.append(f"changelog.d/{f}: name it <name>.en.md and <name>.es.md")
            continue
        names[m.group(1)].add(f[: -len(".xx.md")])
        text = read(os.path.join(DIR, f)).strip()
        if not text.startswith("- "):
            problems.append(f"changelog.d/{f}: an entry starts with \"- \" (a list item)")
        if re.search(r"^#", text, re.M):
            problems.append(f"changelog.d/{f}: no headings; the release writes them")
    for a, b in (("en", "es"), ("es", "en")):
        for n in sorted(names[a] - names[b]):
            problems.append(f"changelog.d/{n}.{a}.md has no twin changelog.d/{n}.{b}.md")
    return sorted(names["en"] & names["es"]), problems


def read(path):
    with open(path, encoding="utf-8") as fh:
        return fh.read()


def added_at(name):
    """When git added the entry (seconds), or the file's time if it is not committed."""
    path = os.path.join(DIR, name + ".en.md")
    try:
        out = git("log", "--diff-filter=A", "--format=%ct", "-1", "--", path).strip()
        if out:
            return int(out)
    except subprocess.CalledProcessError:
        pass
    return int(os.path.getmtime(path))


def check(base):
    names, problems = entries()
    if base:
        since = git("merge-base", base, "HEAD").strip()
        changed = git("diff", "--name-only", since).split()
        changed += git("ls-files", "--others", "--exclude-standard").split()
        release = "VERSION" in changed
        for f in ("CHANGELOG.md", "CHANGELOG.es.md"):
            if f in changed and not release:
                problems.append(f"{f} changed outside a release: put the entry in "
                                "changelog.d/<name>.en.md and .es.md instead")
        program = [f for f in changed if PROGRAM.match(f) and not f.endswith("_test.go")
                   and "/testdata/" not in f]
        added = [f for f in changed if f.startswith("changelog.d/") and NAME.match(f[12:])]
        log = git("log", "--format=%B", f"{since}..HEAD")
        none = re.search(r"^Changelog: none\s*$", log, re.M)
        if program and not added and not release and not none:
            problems.append(
                "this branch changes the program but adds no entry to changelog.d/ "
                f"({program[0]}{' and more' if len(program) > 1 else ''}). Add "
                "changelog.d/<name>.en.md and .es.md, or, for a change no one notices, "
                "a commit line `Changelog: none`.")
    for p in problems:
        print(p, file=sys.stderr)
    if not problems:
        print(f"changelog: {len(names)} entr{'y' if len(names) == 1 else 'ies'} waiting "
              "in changelog.d/")
    return 1 if problems else 0


def release(version, date):
    names, problems = entries()
    if problems:
        for p in problems:
            print(p, file=sys.stderr)
        return 1
    names.sort(key=lambda n: (-added_at(n), n))
    new = {}
    for lang, (file, unreleased, released) in LANGS.items():
        path = os.path.join(ROOT, file)
        lines = read(path).split("\n")
        if f"## {version}" in lines:
            print(f"{file} already has a section ## {version}", file=sys.stderr)
            return 1
        try:
            at = lines.index(unreleased)
        except ValueError:
            print(f"{file} has no line {unreleased}", file=sys.stderr)
            return 1
        end = next((i for i in range(at + 1, len(lines)) if lines[i].startswith("## ")),
                   len(lines))
        old = "\n".join(lines[at + 1:end]).strip()
        parts = [read(os.path.join(DIR, f"{n}.{lang}.md")).strip() for n in names]
        if old:
            parts.append(old)
        if not parts:
            print("nothing to release: changelog.d/ is empty and so is "
                  f"{unreleased} in {file}", file=sys.stderr)
            return 1
        section = [unreleased, "", f"## {version}", "", released.format(date), "",
                   "\n".join(parts), ""]
        new[path] = "\n".join(lines[:at] + section + lines[end:])
    for path, text in new.items():
        with open(path + ".tmp", "w", encoding="utf-8") as fh:
            fh.write(text)
        os.replace(path + ".tmp", path)
    for n in names:
        for lang in LANGS:
            os.remove(os.path.join(DIR, f"{n}.{lang}.md"))
    print(f"changelog: {len(names)} entr{'y' if len(names) == 1 else 'ies'} gathered "
          f"under ## {version}; changelog.d/ emptied")
    return 0


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    c = sub.add_parser("check", help="check changelog.d/ (and the branch, with --base)")
    c.add_argument("--base", help="the commit the branch left from, e.g. origin/main")
    r = sub.add_parser("release", help="gather changelog.d/ under a new version")
    r.add_argument("version")
    r.add_argument("--date", default=datetime.date.today().isoformat())
    a = ap.parse_args()
    if a.cmd == "check":
        return check(a.base)
    if not re.fullmatch(r"\d{4}-\d{2}-\d{2}", a.date):
        ap.error("--date is YYYY-MM-DD")
    return release(a.version, a.date)


if __name__ == "__main__":
    sys.exit(main())
