#!/usr/bin/env python3
"""Builds the diagrams of the documentation, in English and Spanish.

The .svg files are generated: change this file and run it again, never edit them by
hand. Each diagram comes in two shapes, drawn at the size it is shown so the text stays
readable: "wide" (760 px, for screens) and "narrow" (360 px, for phones, stacked). The
pages pick one with <picture>. Each diagram sits on a white card, so it reads the same
in light and dark themes.

Usage:  python3 docs/img/diagrams.py      (writes the .svg files next to this script)
"""
import os

FONT = "'IBM Plex Sans', Inter, 'Segoe UI', Helvetica, Arial, sans-serif"
C = {
    "l3s": "#5F6B75", "l3f": "#F1F3F5",   # layer 3: you and your assistant
    "l2s": "#3C7A5E", "l2f": "#E3F1E9",   # layer 2: the plugin
    "l1s": "#143024", "l1f": "#D3E6DA",   # layer 1: the program
    "l0s": "#8A6100", "l0f": "#FFF5DA",   # layer 0: your files
    "go": "#2E7D4F", "ask": "#8A6100", "stop": "#B3261E",
    "txt": "#1C2128", "sub": "#4B5A52", "line": "#4B5A52", "white": "#FFFFFF",
}
PAD = 16          # card padding
GAP = 12          # space between boxes
TITLE = 34        # height of a layer title
LINK = 40         # vertical gap between layers, for the arrow


def esc(s):
    return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def text(x, y, s, size=15, fill=C["txt"], anchor="middle", weight="400"):
    return (f'<text x="{x:.1f}" y="{y:.1f}" text-anchor="{anchor}" font-family="{FONT}" '
            f'font-size="{size}" font-weight="{weight}" fill="{fill}">{esc(s)}</text>')


def rect(x, y, w, h, fill, stroke, rx=10, sw=1.5, dash=None):
    d = f' stroke-dasharray="{dash}"' if dash else ""
    return (f'<rect x="{x:.1f}" y="{y:.1f}" width="{w:.1f}" height="{h:.1f}" rx="{rx}" '
            f'fill="{fill}" stroke="{stroke}" stroke-width="{sw}"{d}/>')


def node(x, y, w, h, label, sub, stroke, dash=None):
    s = rect(x, y, w, h, C["white"], stroke, dash=dash)
    if sub:
        s += text(x + w / 2, y + h / 2 - 3, label, weight="600")
        s += text(x + w / 2, y + h / 2 + 16, sub, size=12.5, fill=C["sub"])
    else:
        s += text(x + w / 2, y + h / 2 + 5, label, weight="600")
    return s


def cylinder(x, y, w, h, label, sub, stroke, dash=None):
    e = 7
    d = f' stroke-dasharray="{dash}"' if dash else ""
    s = (f'<path d="M {x:.1f},{y + e:.1f} L {x:.1f},{y + h - e:.1f} A {w / 2:.1f},{e} 0 0 0 '
         f'{x + w:.1f},{y + h - e:.1f} L {x + w:.1f},{y + e:.1f}" fill="{C["white"]}" '
         f'stroke="{stroke}" stroke-width="1.5"{d}/>')
    s += (f'<ellipse cx="{x + w / 2:.1f}" cy="{y + e:.1f}" rx="{w / 2:.1f}" ry="{e}" '
          f'fill="{C["white"]}" stroke="{stroke}" stroke-width="1.5"{d}/>')
    s += text(x + w / 2, y + h / 2 + 3, label, weight="600")
    s += text(x + w / 2, y + h / 2 + 21, sub, size=12.5, fill=C["sub"])
    return s


def arrow(x1, y1, x2, y2, label=None):
    s = (f'<line x1="{x1:.1f}" y1="{y1:.1f}" x2="{x2:.1f}" y2="{y2:.1f}" stroke="{C["line"]}" '
         f'stroke-width="1.8" marker-end="url(#arrow)"/>')
    if label:
        w = 7.2 * len(label) + 18
        my = (y1 + y2) / 2
        s += rect(x1 + 12, my - 11, w, 22, C["white"], C["line"], rx=11, sw=1)
        s += text(x1 + 12 + w / 2, my + 4.5, label, size=12.5, fill=C["sub"])
    return s


def pill(x, y, w, label, color):
    return rect(x, y, w, 32, C["white"], color, rx=16, sw=2) + \
        text(x + w / 2, y + 21, label, size=12.5, fill=color, weight="700")


def svg(w, h, body, label):
    defs = ('<defs><marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" '
            f'markerHeight="7" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" '
            f'fill="{C["line"]}"/></marker></defs>')
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {w} {h:.0f}" width="{w}" '
            f'height="{h:.0f}" role="img" aria-label="{esc(label)}">{defs}'
            f'<rect width="{w}" height="{h:.0f}" rx="16" fill="{C["white"]}"/>{body}</svg>\n')


def grid(x, y, w, rows, item_h):
    """Places rows of items across width w; returns [(x, y, w, item)] and the height."""
    out = []
    for r, row in enumerate(rows):
        iw = (w - GAP * (len(row) - 1)) / len(row)
        for i, item in enumerate(row):
            out.append((x + i * (iw + GAP), y + r * (item_h + GAP), iw, item))
    return out, len(rows) * item_h + (len(rows) - 1) * GAP


def chunks(items, per_row):
    return [items[i:i + per_row] for i in range(0, len(items), per_row)]


# --- layers -------------------------------------------------------------------------

def layers(t, W, per_row):
    """Four stacked layers; per_row says how many boxes fit in a row, for each layer."""
    spec = [
        ("l3", C["l3s"], C["l3f"], [("you", None, None), ("claude", None, None),
                                     ("others", "later", "6 4")], node),
        ("l2", C["l2s"], C["l2f"], [("start", "startsub", None), ("guard", "guardsub", None),
                                     ("mcp", "mcpsub", None)], node),
        ("l1", C["l1s"], C["l1f"], [("p1", "p1s", None), ("p2", "p2s", None),
                                     ("p3", "p3s", None), ("p4", "p4s", None)], node),
        ("l0", C["l0s"], C["l0f"], [("f1", "f1s", None), ("f2", "f2s", None),
                                     ("f3", "f3s", "6 4")], cylinder),
    ]
    links = [t["link32"], t["link21"], t["link10"]]
    b, y, inner = "", PAD, W - 4 * PAD
    for n, (key, stroke, fill, items, draw) in enumerate(spec):
        item_h = 64 if draw is cylinder else 58
        cells, gh = grid(2 * PAD, y + TITLE, inner, chunks(items, per_row[n]), item_h)
        h = TITLE + gh + PAD
        b += rect(PAD, y, W - 2 * PAD, h, fill, stroke, rx=14)
        b += text(2 * PAD, y + 23, t[key], size=15.5, fill=stroke, anchor="start", weight="700")
        for cx, cy, cw, (lbl, sub, dash) in cells:
            b += draw(cx, cy, cw, item_h, t[lbl], t[sub] if sub else None, stroke, dash)
        y += h
        if n < 3:
            b += arrow(W / 2, y + 4, W / 2, y + LINK - 4, links[n])
            y += LINK
    return svg(W, y + PAD, b, t["layers"])


# --- session ------------------------------------------------------------------------

def answers(t):
    return [(t["r3a"], C["go"]), (t["r3b"], C["ask"]), (t["r3c"], C["stop"])]


def session(t, W, narrow):
    steps = [
        (t["m1"], (t["w1"], t["w1s"]), (t["r1"], t["r1s"])),
        (t["m2"], (t["w2"], t["w2s"]), (t["r2"], t["r2s"])),
        (t["m3"], (t["w3"], t["w3s"]), None),
    ]
    b, y = "", PAD
    for m, (w, ws), r in steps:
        if narrow:
            inner = W - 4 * PAD
            h = 36 + 58 + 28 + (58 if r else 32) + PAD
            b += rect(PAD, y, W - 2 * PAD, h, C["l3f"], C["l3s"], rx=14)
            b += text(2 * PAD, y + 24, m, size=15.5, anchor="start", weight="700")
            b += node(2 * PAD, y + 36, inner, 58, w, ws, C["l2s"])
            b += arrow(W / 2, y + 98, W / 2, y + 118)
            ry = y + 122
            if r:
                b += node(2 * PAD, ry, inner, 58, r[0], r[1], C["l1s"])
            else:
                pw = (inner - 2 * 8) / 3
                for j, (lbl, col) in enumerate(answers(t)):
                    b += pill(2 * PAD + j * (pw + 8), ry, pw, lbl, col)
        else:
            h = 86
            b += rect(PAD, y, W - 2 * PAD, h, C["l3f"], C["l3s"], rx=14)
            b += text(2 * PAD, y + h / 2 + 5, m, size=15.5, anchor="start", weight="700")
            b += node(276, y + 14, 204, 58, w, ws, C["l2s"])
            b += arrow(484, y + h / 2, 506, y + h / 2)
            if r:
                b += node(510, y + 14, 218, 58, r[0], r[1], C["l1s"])
            else:
                for j, (lbl, col) in enumerate(answers(t)):
                    b += pill(508 + j * 74, y + 27, 72, lbl, col)
        y += h + GAP
    return svg(W, y - GAP + PAD, b, t["session"])


T = {
    "en": {
        "layers": "augmented brain in four layers",
        "l3": "3 · You and your assistant", "you": "You", "claude": "Claude Code",
        "others": "Other assistants", "later": "later, through MCP",
        "l2": "2 · The plugin: the doors",
        "start": "Session start", "startsub": "loads your space",
        "guard": "Before each command", "guardsub": "the guard",
        "mcp": "MCP server", "mcpsub": "the tools",
        "l1": "1 · The program: augmented-brain",
        "p1": "Guard", "p1s": "go, ask, stop",
        "p2": "Method", "p2s": "tasks and areas",
        "p3": "Index", "p3s": "MEMORY.md",
        "p4": "Backup", "p4s": "git, per space",
        "l0": "0 · Your files",
        "f1": "Configuration", "f1s": "~/.config/augmented-brain",
        "f2": "Notes of a space", "f2s": "~/augmented-brain/<space>/notes",
        "f3": "Backup", "f3s": "your private git repo",
        "link32": "hooks and tools", "link21": "start the program", "link10": "reads and writes",
        "session": "A session, step by step",
        "m1": "1 · You open Claude", "m2": "2 · You say “park it”", "m3": "3 · Claude runs a command",
        "w1": "Start hook", "w1s": "finds your space",
        "w2": "task_park tool", "w2s": "writes the task file",
        "w3": "Guard hook", "w3s": "reads it first",
        "r1": "Claude knows", "r1s": "profile, rules, tasks",
        "r2": "Next session", "r2s": "picks it up there",
        "r3a": "go", "r3b": "ask", "r3c": "stop",
    },
    "es": {
        "layers": "augmented brain en cuatro capas",
        "l3": "3 · Tú y tu asistente", "you": "Tú", "claude": "Claude Code",
        "others": "Otros asistentes", "later": "más adelante, por MCP",
        "l2": "2 · El plugin: las puertas",
        "start": "Inicio de sesión", "startsub": "carga tu espacio",
        "guard": "Antes de cada comando", "guardsub": "el guardia",
        "mcp": "Servidor MCP", "mcpsub": "las herramientas",
        "l1": "1 · El programa: augmented-brain",
        "p1": "Guardia", "p1s": "sigue, pregunta, para",
        "p2": "Método", "p2s": "tareas y áreas",
        "p3": "Índice", "p3s": "MEMORY.md",
        "p4": "Copia", "p4s": "git, por espacio",
        "l0": "0 · Tus ficheros",
        "f1": "Configuración", "f1s": "~/.config/augmented-brain",
        "f2": "Notas de un espacio", "f2s": "~/augmented-brain/<espacio>/notas",
        "f3": "Copia de seguridad", "f3s": "tu repo git privado",
        "link32": "hooks y herramientas", "link21": "arrancan el programa", "link10": "lee y escribe",
        "session": "Una sesión, paso a paso",
        "m1": "1 · Abres Claude", "m2": "2 · Dices “apárcala”", "m3": "3 · Claude lanza un comando",
        "w1": "Hook de inicio", "w1s": "busca tu espacio",
        "w2": "Herramienta task_park", "w2s": "escribe la tarea",
        "w3": "Hook del guardia", "w3s": "lo lee antes",
        "r1": "Claude ya sabe", "r1s": "perfil, reglas, tareas",
        "r2": "La próxima sesión", "r2s": "sigue desde ahí",
        "r3a": "sigue", "r3b": "pregunta", "r3c": "para",
    },
}


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    for lang, t in T.items():
        out = {
            "how-layers": layers(t, 760, [3, 3, 4, 3]),
            "how-layers-narrow": layers(t, 360, [2, 1, 2, 1]),
            "how-session": session(t, 760, narrow=False),
            "how-session-narrow": session(t, 360, narrow=True),
        }
        for name, content in out.items():
            path = os.path.join(here, f"{name}.{lang}.svg")
            with open(path, "w", encoding="utf-8") as f:
                f.write(content)
            print("wrote", os.path.relpath(path))


if __name__ == "__main__":
    main()
