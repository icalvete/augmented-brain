#!/usr/bin/env python3
"""privacy-hashed.py: the personal denylist, checked where the denylist itself is not.

privacy-check.sh looks for the maintainer's personal data with a denylist that lives
outside the repo, because it contains the very data it protects. So CI, which does not
have it, could not catch a slip. This script closes that gap: the repo keeps keyed
fingerprints of the denylist (scripts/privacy-hashes), never the entries, and CI looks
for them with the key, which is a GitHub Actions secret.

Why keyed (HMAC-SHA256) and not plain SHA-256: the entries are short and guessable
(first names, host names). A plain hash of a first name is found in seconds by hashing a
list of names, so publishing plain hashes would publish the data. Without the key, a
keyed fingerprint tells nothing. For the same reason the key must be long (MIN_KEY
characters): a short one can be guessed offline, and with it every entry.

How text is matched: both the entries and the text are put in one canonical form first:
compatibility characters folded (NFKC), case folded, accents dropped, and split into
words (runs of letters and digits; anything else only separates them). An entry is the
fingerprint of its words joined by a space, and also of its words written together. In
the text, every run of 1 to MAX_WORDS consecutive words, across line ends, is joined by
spaces and looked up. So the entry "jane doe" catches "Jane Doe", "jane.doe@example.com",
"jane_doe", "JaneDoe", "janedoe" and "Jane" at the end of one line with "Doe" at the
start of the next; "josé garcía" catches "Jose Garcia" in any Unicode form; "600 123
456" catches "600123456" and "600-123-456". "janedoes" is another word, and is not
caught. Files in UTF-16 are read as text. Literal exceptions in scripts/privacy-allow
are removed first, as in privacy-check.sh.

Besides the tracked files and their names, scan reads the author, the committer and the
message of each commit in AB_PRIVACY_COMMITS (a git range, such as base..head; without
it, the last commit), and the text in AB_PRIVACY_TEXT (CI puts there the branch name and
the pull request's title).

    AB_PRIVACY_KEY=... scripts/privacy-hashed.py build [denylist]   # rewrite scripts/privacy-hashes
    AB_PRIVACY_KEY=... scripts/privacy-hashed.py scan [file...]     # default: every tracked file
    scripts/privacy-hashed.py selftest                              # the matching, on made-up data

Exit code: 0 clean, 1 something found, 2 usage or setup error. Prints file:line, never
what matched.
"""
import codecs
import hashlib
import hmac
import os
import re
import subprocess
import sys
import tempfile
import unicodedata

HERE = os.path.dirname(os.path.abspath(__file__))
HASHES = os.path.join(HERE, "privacy-hashes")
ALLOW = os.path.join(HERE, "privacy-allow")
MAX_WORDS = 8
MIN_KEY = 32
WORD = re.compile(r"[^\W_]+")  # letters and digits, any alphabet


def key_problem(k):
    """Why k cannot be the key, or None."""
    if len(k) < MIN_KEY:
        return (f"AB_PRIVACY_KEY has {len(k)} characters, fewer than {MIN_KEY}: a short key can be "
                "guessed, and with it the denylist. Make a longer one, for example with "
                "python3 -c 'import secrets; print(secrets.token_urlsafe(32))', put it in the "
                "AB_PRIVACY_KEY secret and run build again.")
    return None


def key():
    k = os.environ.get("AB_PRIVACY_KEY", "")
    if not k and os.environ.get("AB_PRIVACY_REQUIRED") == "true":
        print("privacy-hashed: AB_PRIVACY_KEY is not set, and this repository needs it (the AB_PRIVACY_KEY secret).", file=sys.stderr)
        sys.exit(2)
    if not k:
        print("privacy-hashed: AB_PRIVACY_KEY is not set; skipping the fingerprint check.", file=sys.stderr)
        sys.exit(0 if sys.argv[1:2] == ["scan"] else 2)
    problem = key_problem(k)
    if problem:
        print("privacy-hashed: " + problem, file=sys.stderr)
        sys.exit(2)
    return k.encode()


def canon(text):
    """Text in the form both sides compare: NFKC, case folded, without accents."""
    t = unicodedata.normalize("NFKD", text)
    t = "".join(c for c in t if not unicodedata.combining(c))
    return unicodedata.normalize("NFKC", t).casefold()


def literals(path):
    """Non-blank, non-comment lines of a file."""
    out = []
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line and not line.startswith("#"):
                out.append(line)
    return out


class Matcher:
    """Looks up text against a set of fingerprints made with one key."""

    def __init__(self, k, prints, allowed=()):
        self.key = k
        self.prints = prints
        self.allowed = [canon(a) for a in allowed if canon(a)]
        self.seen = {}  # text -> fingerprint: code repeats the same words a lot

    def fingerprint(self, text):
        fp = self.seen.get(text)
        if fp is None:
            fp = hmac.digest(self.key, text.encode("utf-8"), hashlib.sha256).hex()
            if len(self.seen) < 1_000_000:
                self.seen[text] = fp
        return fp

    def entry_prints(self, entry):
        """The fingerprints of an entry, and its number of words."""
        words = WORD.findall(canon(entry))
        if not words:
            return set(), 0
        return {self.fingerprint(" ".join(words)), self.fingerprint("".join(words))}, len(words)

    def words(self, lines):
        """(word, line number) of every word in lines, allowed literals taken out."""
        out = []
        for n, line in enumerate(lines, 1):
            low = canon(line)
            for a in self.allowed:
                low = low.replace(a, " ")
            out.extend((w, n) for w in WORD.findall(low))
        return out

    def lines_hit(self, lines):
        """Line numbers where a match starts, in order, each once."""
        words = self.words(lines)
        found = []
        for i in range(len(words)):
            run = [w for w, _ in words[i:i + MAX_WORDS]]
            for j in range(1, len(run) + 1):
                if self.fingerprint(" ".join(run[:j])) in self.prints:
                    if words[i][1] not in found:
                        found.append(words[i][1])
                    break
        return found

    def hit(self, text):
        return bool(self.lines_hit(text.splitlines()))


def decode(data):
    """The text in data, or None for a binary file. UTF-16 is read as text."""
    if data.startswith(codecs.BOM_UTF16_LE) or data.startswith(codecs.BOM_UTF16_BE):
        return data.decode("utf-16", "replace")
    head = data[:8192]
    if b"\0" in head:
        even, odd = head[0::2], head[1::2]
        if odd and odd.count(0) >= 0.9 * len(odd) and even.count(0) <= 0.1 * len(even):
            return data.decode("utf-16-le", "replace")
        if even and even.count(0) >= 0.9 * len(even) and odd.count(0) <= 0.1 * len(odd):
            return data.decode("utf-16-be", "replace")
        return None  # binary: short words show up there by chance, as in privacy-check.sh
    return data.decode("utf-8-sig", "replace")


def build(args):
    m = Matcher(key(), set())
    src = args[0] if args else os.environ.get(
        "AB_DENYLIST", os.path.expanduser("~/.config/augmented-brain-dev/denylist"))
    prints = set()
    for entry in literals(src):
        ps, n = m.entry_prints(entry)
        if n > MAX_WORDS:
            print(f"privacy-hashed: an entry has {n} words, more than {MAX_WORDS}; shorten it.", file=sys.stderr)
            sys.exit(2)
        prints |= ps
    with open(HASHES, "w", encoding="utf-8") as f:
        f.write("# Keyed fingerprints (HMAC-SHA256) of the maintainer's personal denylist.\n")
        f.write("# Generated by scripts/privacy-hashed.py build; do not edit. Sorted, so order tells nothing.\n")
        for p in sorted(prints):
            f.write(p + "\n")
    print(f"privacy-hashed: {len(prints)} fingerprints written to {os.path.relpath(HASHES)}.")


def commits(rng):
    """(short id, identity, message) of each commit in the git range rng."""
    args = ["git", "log", "--format=%h%x00%an <%ae>%n%cn <%ce>%x00%B%x1e"]
    args += [rng] if rng else ["-1"]
    r = subprocess.run(args, capture_output=True, text=True, encoding="utf-8", errors="replace")
    if r.returncode != 0:
        print(f"privacy-hashed: cannot read the commits of {rng or 'HEAD'}: {r.stderr.strip()}", file=sys.stderr)
        sys.exit(2)
    out = []
    for rec in r.stdout.split("\x1e"):
        parts = rec.strip("\n").split("\0")
        if len(parts) == 3:
            out.append(tuple(parts))
    return out


def scan(args):
    allowed = literals(ALLOW) if os.path.exists(ALLOW) else []
    m = Matcher(key(), set(literals(HASHES)), allowed)
    files = args or subprocess.run(["git", "ls-files"], capture_output=True, text=True, check=True).stdout.splitlines()
    found = False
    for name in files:
        if m.hit(name):
            print(f"  PERSONAL  {name} (file name)")
            found = True
        try:
            with open(name, "rb") as f:
                text = decode(f.read())
        except (IsADirectoryError, FileNotFoundError):
            continue
        if text is None:
            continue
        for n in m.lines_hit(text.splitlines()):
            print(f"  PERSONAL  {name}:{n}")
            found = True
    # Who made the commits and what they say: a personal name in them is just as public.
    rng = os.environ.get("AB_PRIVACY_COMMITS", "").strip()
    listed = commits(rng)
    for short, ident, message in listed:
        if m.hit(ident):
            print(f"  PERSONAL  commit {short}: author or committer")
            found = True
        if m.hit(message):
            print(f"  PERSONAL  commit {short}: message")
            found = True
    extra = os.environ.get("AB_PRIVACY_TEXT", "")
    if extra.strip() and m.hit(extra):
        print("  PERSONAL  branch name or pull request title (AB_PRIVACY_TEXT)")
        found = True
    if found:
        print("privacy-hashed: blocked. Remove the data above and try again.", file=sys.stderr)
        sys.exit(1)
    print(f"privacy-hashed: clean ({len(files)} files, {len(listed)} commits, {len(m.prints)} fingerprints).")


# Made-up entries and the ways a slip writes them: every one must be caught.
SELFTEST_ENTRIES = ["jane doe", "josé garcía", "600 123 456", "acme-corp"]
SELFTEST_CAUGHT = [
    "contact: Jane Doe",
    "jane.doe@example.com",
    "jane_doe",
    "janedoe",
    "JaneDoe",
    "Jane  Doe",
    "Jane\nDoe",
    "600123456",
    "600-123-456",
    unicodedata.normalize("NFC", "José García"),
    unicodedata.normalize("NFD", "josé garcía"),
    "jose garcia",
    "ACME-Corp",
    "acme corp",
    "acmecorp",
]
SELFTEST_CLEAN = ["janedoes", "jane smith", "john doe", "acme", "600 123", "garcíalorca", "joséfina"]


def selftest(_args):
    k = b"a made-up key, only for the selftest, long enough"
    m = Matcher(k, set())
    for e in SELFTEST_ENTRIES:
        m.prints |= m.entry_prints(e)[0]
    bad = []
    for text in SELFTEST_CAUGHT:
        if not m.hit(text):
            bad.append(f"missed {text!r}")
    for text in SELFTEST_CLEAN:
        if m.hit(text):
            bad.append(f"caught {text!r}, which is not an entry")
    for enc in ("utf-16", "utf-16-le", "utf-16-be"):
        with tempfile.NamedTemporaryFile(delete=False) as f:
            f.write("notes\ncontact: Jane Doe\n".encode(enc))
        with open(f.name, "rb") as g:
            text = decode(g.read())
        os.unlink(f.name)
        if text is None or m.lines_hit(text.splitlines()) != [2]:
            bad.append(f"missed the entry in a {enc} file")
    if decode(b"\x89PNG\r\n\x1a\n\0\0\0\rIHDR") is not None:
        bad.append("read a binary file as text")
    allowing = Matcher(k, m.prints, ["jane doe & co"])
    if allowing.hit("Jane Doe & Co") or not allowing.hit("Jane Doe & Co, and Jane Doe"):
        bad.append("the allow list is not applied as a literal")
    if key_problem("x" * (MIN_KEY - 1)) is None or key_problem("x" * MIN_KEY) is not None:
        bad.append(f"a key shorter than {MIN_KEY} characters is not refused")
    if bad:
        print("privacy-hashed selftest:", *bad, sep="\n  ", file=sys.stderr)
        sys.exit(1)
    print(f"privacy-hashed selftest: ok ({len(SELFTEST_CAUGHT) + 1} variants caught, {len(SELFTEST_CLEAN)} left alone).")


if __name__ == "__main__":
    cmd, rest = (sys.argv[1], sys.argv[2:]) if len(sys.argv) > 1 else ("", [])
    commands = {"build": build, "scan": scan, "selftest": selftest}
    if cmd not in commands:
        print(__doc__.split("\n\n")[-2], file=sys.stderr)
        sys.exit(2)
    commands[cmd](rest)
