#!/usr/bin/env python3
"""Builds docs/guide/how-it-works(.es).html from the Markdown guides.

The .md files are the source: change them and run this, never edit the .html by hand.
The page frame (header, footer, styles) is taken from the connect-services pages, so all
guides look the same. A tiny converter covers the Markdown these pages use: headings,
paragraphs, lists, tables, fenced code, <picture> blocks, inline code, bold and links.

Usage:  python3 scripts/build-guides.py
"""
import html as H
import pathlib
import re

G = pathlib.Path(__file__).resolve().parent.parent / "docs" / "guide"
BASE = "https://icalvete.github.io/augmented-brain/guide/"
REPO = "https://github.com/icalvete/augmented-brain/blob/main/"


def link(url):
    if url.startswith("../../"):
        return REPO + url[6:]
    if url.startswith("../spec/"):
        return REPO + "docs/spec/" + url[8:]
    return url.replace(".md", ".html")


def inline(s):
    parts = re.split(r"(`[^`]+`)", s)
    out = []
    for p in parts:
        if p.startswith("`") and p.endswith("`") and len(p) > 1:
            out.append("<code>" + H.escape(p[1:-1], quote=False) + "</code>")
            continue
        p = H.escape(p, quote=False)
        p = re.sub(r'"([^"<>]+)"', r"“\1”", p)
        p = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", p)
        p = re.sub(r"\[([^\]]+)\]\(([^)]+)\)", lambda m: f'<a href="{link(m.group(2))}">{m.group(1)}</a>', p)
        out.append(p)
    return "".join(out)


def item(lines):
    """The inside of a list item: its text, and any fenced code in it."""
    out, text, code = [], [], None
    for ln in lines:
        if ln.startswith("```"):
            if code is None:
                if text:
                    out.append(inline(" ".join(t.strip() for t in text if t.strip())))
                    text = []
                code = []
            else:
                out.append("<pre><code>" + H.escape("\n".join(code), quote=False) + "</code></pre>")
                code = None
        elif code is not None:
            code.append(ln)
        else:
            text.append(ln)
    if text and any(t.strip() for t in text):
        out.append(inline(" ".join(t.strip() for t in text if t.strip())))
    return "".join(out)


def convert(md):
    lines = md.splitlines()
    out, i = [], 0
    while i < len(lines):
        ln = lines[i]
        if not ln.strip():
            i += 1
        elif ln.startswith("```"):
            j = i + 1
            while not lines[j].startswith("```"):
                j += 1
            code = H.escape("\n".join(lines[i + 1:j]), quote=False)
            out.append(f"  <pre><code>{code}</code></pre>")
            i = j + 1
        elif ln.startswith("#"):
            n = len(ln) - len(ln.lstrip("#"))
            out.append(f"  <h{n}>{inline(ln[n:].strip())}</h{n}>")
            i += 1
        elif ln.startswith("<picture>"):
            j = i
            while not lines[j].startswith("</picture>"):
                j += 1
            out.append('  <figure class="diagram">' + "".join(l.strip() for l in lines[i:j + 1]) + "</figure>")
            i = j + 1
        elif ln.startswith("!["):
            m = re.match(r"!\[([^\]]*)\]\(([^)]+)\)", ln)
            out.append(f'  <p class="diagram"><img src="{m.group(2)}" alt="{H.escape(m.group(1))}"></p>')
            i += 1
        elif ln.startswith("|"):
            rows = []
            while i < len(lines) and lines[i].startswith("|"):
                rows.append([c.strip() for c in lines[i].strip("|").split("|")])
                i += 1
            head, body = rows[0], rows[2:]
            t = "<thead><tr>" + "".join(f"<th>{inline(c)}</th>" for c in head) + "</tr></thead><tbody>"
            t += "".join("<tr>" + "".join(f"<td>{inline(c)}</td>" for c in r) + "</tr>" for r in body)
            out.append(f'  <div class="table-wrap"><table>{t}</tbody></table></div>')
        elif re.match(r"\d+\. ", ln):
            # A numbered list: each item may go on in lines indented by three spaces,
            # with fenced code inside.
            items = []
            while i < len(lines) and (re.match(r"\d+\. ", lines[i]) or lines[i].startswith("   ") or
                                      (not lines[i].strip() and i + 1 < len(lines) and lines[i + 1].startswith("   "))):
                if re.match(r"\d+\. ", lines[i]):
                    items.append([re.sub(r"^\d+\. ", "", lines[i])])
                else:
                    items[-1].append(lines[i][3:])
                i += 1
            out.append("  <ol>" + "".join(f"<li>{item(x)}</li>" for x in items) + "</ol>")
        elif ln.startswith("- "):
            items = []
            while i < len(lines) and (lines[i].startswith("- ") or lines[i].startswith("  ")):
                if lines[i].startswith("- "):
                    items.append(lines[i][2:])
                else:
                    items[-1] += " " + lines[i].strip()
                i += 1
            out.append("  <ul>" + "".join(f"<li>{inline(x)}</li>" for x in items) + "</ul>")
        else:
            para = []
            while i < len(lines) and lines[i].strip() and not re.match(r"(#|```|\||- |!\[|<picture>|\d+\. )", lines[i]):
                para.append(lines[i].strip())
                i += 1
            out.append(f"  <p>{inline(' '.join(para))}</p>")
    return "\n".join(out)


def frame(lang):
    src = (G / ("connect-services.es.html" if lang == "es" else "connect-services.html")).read_text()
    head, rest = src.split("<main", 1)
    return head, rest.split("</main>", 1)[1]


def page(lang, title, desc, ogdesc, back):
    head, tail = frame(lang)
    md = (G / ("how-it-works.es.md" if lang == "es" else "how-it-works.md")).read_text()
    md = "\n".join(l for l in md.splitlines() if not re.match(r"\[(Read in English|Leer en español)\]", l))
    name = "how-it-works.es.html" if lang == "es" else "how-it-works.html"
    other = "how-it-works.html" if lang == "es" else "how-it-works.es.html"
    old_other = "connect-services.html" if lang == "es" else "connect-services.es.html"
    old_title = "Conecta tus servicios" if lang == "es" else "Connect your services"
    h = head.replace(old_title, title)
    h = re.sub(r'(<meta name="description" content=")[^"]*', r"\g<1>" + desc, h)
    h = re.sub(r'(<meta property="og:description" content=")[^"]*', r"\g<1>" + ogdesc, h)
    h = re.sub(r'(<meta name="twitter:description" content=")[^"]*', r"\g<1>" + ogdesc, h)
    h = h.replace(BASE + "connect-services.es.html", BASE + "how-it-works.es.html")
    h = h.replace(BASE + "connect-services.html", BASE + "how-it-works.html")
    h = h.replace(f'href="{old_other}"', f'href="{other}"')
    assert "connect-services" not in h, "a link to the other guide is left"
    body = convert(md) + "\n" + back
    (G / name).write_text(h + '<main class="wrap prose">\n' + body + "\n</main>" + tail)
    print("ok", name)


page("en", "How it works",
     "How augmented brain works, in four layers: you and your assistant, the plugin, the program and your files. What happens in a session and where every file lives.",
     "Four layers, a session step by step, and where every file lives.",
     '  <a class="back" href="../">← Back to augmented brain</a>')
page("es", "Cómo funciona",
     "Cómo funciona augmented brain, en cuatro capas: tú y tu asistente, el plugin, el programa y tus ficheros. Qué pasa en una sesión y dónde vive cada fichero.",
     "Cuatro capas, una sesión paso a paso y dónde vive cada fichero.",
     '  <a class="back" href="../es/">← Volver a augmented brain</a>')
