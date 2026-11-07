#!/usr/bin/env python3
"""Write THIRD_PARTY_NOTICES.md: the license of every module compiled into the program.

Their licenses (MIT, BSD, Apache 2.0) ask that their notice travels with the program, so
the release archives carry this file next to LICENSE. Run it after a dependency changes;
`--check` fails when the file is not up to date (precheck runs it).
"""
import pathlib
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
OUT = ROOT / "THIRD_PARTY_NOTICES.md"
NAMES = ("LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING", "LICENCE")


def build() -> str:
    listing = subprocess.run(
        ["go", "list", "-deps", "-f",
         "{{with .Module}}{{if not .Main}}{{.Path}} {{.Version}} {{.Dir}}{{end}}{{end}}",
         "./cmd/augmented-brain"],
        cwd=ROOT, check=True, capture_output=True, text=True).stdout
    mods = sorted({tuple(l.split(" ", 2)) for l in listing.splitlines() if l.strip()})
    parts = ["# Third-party notices", "",
             "augmented brain is MIT-licensed (see LICENSE). It is built with these modules,",
             "whose licenses follow, each as its authors wrote it.", ""]
    for path, version, folder in mods:
        lic = next((pathlib.Path(folder) / n for n in NAMES if (pathlib.Path(folder) / n).is_file()), None)
        if lic is None:
            sys.exit(f"no license file found for {path} {version}")
        parts += [f"## {path} {version}", "", "```", lic.read_text(encoding="utf-8").strip(), "```", ""]
    return "\n".join(parts)


def main() -> None:
    text = build()
    if "--check" in sys.argv[1:]:
        if not OUT.is_file() or OUT.read_text(encoding="utf-8") != text:
            sys.exit("THIRD_PARTY_NOTICES.md is not up to date: run scripts/third-party-notices.py")
        return
    OUT.write_text(text, encoding="utf-8")


if __name__ == "__main__":
    main()
