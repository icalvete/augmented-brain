#!/usr/bin/env python3
"""Builds the diagrams of the documentation, in English and Spanish.

The .svg files are generated: change this file and run it again, never edit them by
hand. Shared helpers (layers, boxes, cylinders, arrows) keep every diagram in the same
style. Each diagram sits on a white card, so it reads the same in light and dark themes.

Usage:  python3 docs/img/diagrams.py      (writes the .svg files next to this script)
"""
import os

FONT = "'IBM Plex Sans', Inter, 'Segoe UI', Helvetica, Arial, sans-serif"
C = {
    "l3s": "#6E7781", "l3f": "#F2F3F5",   # layer 3: you and your assistant
    "l2s": "#3C7A5E", "l2f": "#E3F1E9",   # layer 2: the plugin
    "l1s": "#143024", "l1f": "#D3E6DA",   # layer 1: the program (the protagonist)
    "l0s": "#9A6B00", "l0f": "#FFF6DD",   # layer 0: your files
    "go": "#2E7D4F", "ask": "#9A6B00", "stop": "#B3261E",
    "txt": "#1C2128", "sub": "#4B5A52", "line": "#4B5A52", "white": "#FFFFFF",
}


def esc(s):
    return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def text(x, y, s, size=14, fill=C["txt"], anchor="middle", weight="400"):
    return (f'<text x="{x}" y="{y}" text-anchor="{anchor}" font-family="{FONT}" '
            f'font-size="{size}" font-weight="{weight}" fill="{fill}">{esc(s)}</text>')


def box(x, y, w, h, fill, stroke, rx=10, sw=1.5, dash=None):
    d = f' stroke-dasharray="{dash}"' if dash else ""
    return (f'<rect x="{x}" y="{y}" width="{w}" height="{h}" rx="{rx}" fill="{fill}" '
            f'stroke="{stroke}" stroke-width="{sw}"{d}/>')


def layer(y, h, title, stroke, fill, note=""):
    s = box(20, y, 960, h, fill, stroke, rx=14)
    s += text(38, y + 26, title, size=15, fill=stroke, anchor="start", weight="700")
    if note:
        s += text(962, y + 26, note, size=12, fill=C["sub"], anchor="end")
    return s


def node(x, y, w, h, label, stroke, sub=None, dash=None, size=14):
    s = box(x, y, w, h, C["white"], stroke, dash=dash)
    if sub:
        s += text(x + w / 2, y + h / 2 - 2, label, size=size, weight="600")
        s += text(x + w / 2, y + h / 2 + 15, sub, size=11, fill=C["sub"])
    else:
        s += text(x + w / 2, y + h / 2 + 5, label, size=size, weight="600")
    return s


def cylinder(x, y, w, h, label, sub, stroke, dash=None):
    e = 8
    d = f' stroke-dasharray="{dash}"' if dash else ""
    body = (f'<path d="M {x},{y + e} L {x},{y + h - e} A {w / 2},{e} 0 0 0 {x + w},{y + h - e} '
            f'L {x + w},{y + e}" fill="{C["white"]}" stroke="{stroke}" stroke-width="1.5"{d}/>')
    top = (f'<ellipse cx="{x + w / 2}" cy="{y + e}" rx="{w / 2}" ry="{e}" fill="{C["white"]}" '
           f'stroke="{stroke}" stroke-width="1.5"{d}/>')
    return (body + top + text(x + w / 2, y + h / 2 + 4, label, size=13, weight="600")
            + text(x + w / 2, y + h / 2 + 21, sub, size=11, fill=C["sub"]))


def arrow(x1, y1, x2, y2, color=C["line"], dash=None, label=None, lx=None):
    d = f' stroke-dasharray="{dash}"' if dash else ""
    s = (f'<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="{color}" '
         f'stroke-width="1.6"{d} marker-end="url(#arrow)"/>')
    if label:
        lx = lx if lx is not None else (x1 + x2) / 2 + 8
        w = 7 * len(label) + 14
        s += box(lx - 2, (y1 + y2) / 2 - 11, w, 20, C["white"], C["line"], rx=10, sw=1)
        s += text(lx - 2 + w / 2, (y1 + y2) / 2 + 3, label, size=11, fill=C["sub"])
    return s


def svg(w, h, body, label):
    defs = ('<defs><marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" '
            f'markerHeight="7" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" fill="{C["line"]}"/>'
            '</marker></defs>')
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {w} {h}" width="{w}" height="{h}" '
            f'role="img" aria-label="{esc(label)}">{defs}'
            # a white card behind everything, so dark themes (GitHub's) keep the contrast
            f'<rect width="{w}" height="{h}" rx="16" fill="{C["white"]}"/>{body}</svg>\n')


T = {
    "en": {
        "layers": "augmented brain in four layers",
        "l3": "Layer 3 · You and your assistant", "you": "You", "claude": "Claude Code",
        "others": "Other MCP assistants", "later": "later",
        "l2": "Layer 2 · The plugin: the doors", "l2n": "installed by Claude Code",
        "start": "Session start", "startsub": "loads your space",
        "guard": "Before each command", "guardsub": "the guard",
        "mcp": "MCP server", "mcpsub": "the tools: tasks, notes",
        "l1": "Layer 1 · The program: augmented-brain", "l1n": "one file, nothing runs in the background",
        "p1": "Guard", "p1s": "go ahead, ask, stop",
        "p2": "Method", "p2s": "tasks, areas, inbox",
        "p3": "Index", "p3s": "MEMORY.md, rebuilt",
        "p4": "Backup", "p4s": "git, per space",
        "l0": "Layer 0 · Your files", "l0n": "plain text, yours",
        "f1": "Configuration", "f1s": "~/.config/augmented-brain",
        "f2": "Notes of a space", "f2s": "~/augmented-brain/<space>",
        "f3": "Backup", "f3s": "your private git repository",
        "hooks": "hooks", "tools": "tools", "reads": "reads, writes",
        # session
        "session": "A session, step by step",
        "m1": "1 · You open Claude", "m2": "2 · You say “park it”", "m3": "3 · Claude runs a command",
        "w1": "Session start hook", "w1s": "finds the space of the folder",
        "w2": "task_park tool", "w2s": "writes the task file",
        "w3": "Guard hook", "w3s": "reads the command first",
        "r1": "Claude knows", "r1s": "profile, rules, live tasks",
        "r2": "Next session", "r2s": "resumes it, with the next step",
        "r3a": "go ahead", "r3b": "ask you", "r3c": "stop",
    },
    "es": {
        "layers": "augmented brain en cuatro capas",
        "l3": "Capa 3 · Tú y tu asistente", "you": "Tú", "claude": "Claude Code",
        "others": "Otros asistentes MCP", "later": "más adelante",
        "l2": "Capa 2 · El plugin: las puertas", "l2n": "lo instala Claude Code",
        "start": "Inicio de sesión", "startsub": "carga tu espacio",
        "guard": "Antes de cada comando", "guardsub": "el guardia",
        "mcp": "Servidor MCP", "mcpsub": "las herramientas: tareas, notas",
        "l1": "Capa 1 · El programa: augmented-brain", "l1n": "un solo fichero, nada funciona de fondo",
        "p1": "Guardia", "p1s": "adelante, pregunta, para",
        "p2": "Método", "p2s": "tareas, áreas, buzón",
        "p3": "Índice", "p3s": "MEMORY.md, se rehace",
        "p4": "Copia", "p4s": "git, por espacio",
        "l0": "Capa 0 · Tus ficheros", "l0n": "texto normal, tuyo",
        "f1": "Configuración", "f1s": "~/.config/augmented-brain",
        "f2": "Notas de un espacio", "f2s": "~/augmented-brain/<espacio>",
        "f3": "Copia de seguridad", "f3s": "tu repositorio git privado",
        "hooks": "hooks", "tools": "herramientas", "reads": "lee, escribe",
        "session": "Una sesión, paso a paso",
        "m1": "1 · Abres Claude", "m2": "2 · Dices “apárcala”", "m3": "3 · Claude ejecuta un comando",
        "w1": "Hook de inicio", "w1s": "busca el espacio de la carpeta",
        "w2": "Herramienta task_park", "w2s": "escribe el fichero de la tarea",
        "w3": "Hook del guardia", "w3s": "lee antes el comando",
        "r1": "Claude ya sabe", "r1s": "perfil, reglas, tareas vivas",
        "r2": "La próxima sesión", "r2s": "la retoma, con su siguiente paso",
        "r3a": "adelante", "r3b": "te pregunta", "r3c": "lo para",
    },
}


def layers(t):
    b = ""
    # layer 3
    b += layer(20, 100, t["l3"], C["l3s"], C["l3f"])
    b += node(44, 52, 200, 52, t["you"], C["l3s"])
    b += node(300, 52, 300, 52, t["claude"], C["l3s"])
    b += node(656, 52, 300, 52, t["others"], C["l3s"], sub=t["later"], dash="6 4")
    b += f'<line x1="244" y1="78" x2="300" y2="78" stroke="{C["line"]}" stroke-width="1.6"/>'
    # layer 2
    b += layer(150, 112, t["l2"], C["l2s"], C["l2f"], t["l2n"])
    b += node(44, 188, 280, 58, t["start"], C["l2s"], sub=t["startsub"])
    b += node(360, 188, 280, 58, t["guard"], C["l2s"], sub=t["guardsub"])
    b += node(676, 188, 280, 58, t["mcp"], C["l2s"], sub=t["mcpsub"])
    b += arrow(380, 104, 184, 186, label=t["hooks"], lx=200)
    b += arrow(450, 104, 500, 186)
    b += arrow(560, 104, 816, 186, label=t["tools"], lx=690)
    # layer 1
    b += layer(292, 112, t["l1"], C["l1s"], C["l1f"], t["l1n"])
    for i, (k, sk) in enumerate([("p1", "p1s"), ("p2", "p2s"), ("p3", "p3s"), ("p4", "p4s")]):
        b += node(44 + i * 230, 330, 206, 58, t[k], C["l1s"], sub=t[sk])
    for x in (184, 500, 816):
        b += arrow(x, 246, x, 328)
    # layer 0
    b += layer(434, 130, t["l0"], C["l0s"], C["l0f"], t["l0n"])
    b += cylinder(80, 472, 240, 70, t["f1"], t["f1s"], C["l0s"])
    b += cylinder(380, 472, 240, 70, t["f2"], t["f2s"], C["l0s"])
    b += cylinder(680, 472, 240, 70, t["f3"], t["f3s"], C["l0s"], dash="6 4")
    b += arrow(380, 388, 480, 470, label=t["reads"], lx=440)
    b += arrow(838, 388, 800, 470, dash="6 4")
    return svg(1000, 584, b, t["layers"])


def session(t):
    b = ""
    rows = [
        (t["m1"], t["w1"], t["w1s"], t["r1"], t["r1s"], C["l3s"]),
        (t["m2"], t["w2"], t["w2s"], t["r2"], t["r2s"], C["l3s"]),
        (t["m3"], t["w3"], t["w3s"], None, None, C["l3s"]),
    ]
    for i, (m, w, ws, r, rs, _) in enumerate(rows):
        y = 20 + i * 100
        b += box(20, y, 960, 84, C["l3f"], C["l3s"], rx=14)
        b += text(40, y + 47, m, size=15, fill=C["txt"], anchor="start", weight="700")
        b += node(340, y + 14, 280, 56, w, C["l2s"], sub=ws)
        b += arrow(300, y + 42, 338, y + 42)
        b += arrow(620, y + 42, 658, y + 42)
        if r:
            b += node(660, y + 14, 300, 56, r, C["l1s"], sub=rs)
        else:
            for j, (lbl, col) in enumerate([(t["r3a"], C["go"]), (t["r3b"], C["ask"]), (t["r3c"], C["stop"])]):
                x = 660 + j * 102
                b += box(x, y + 26, 94, 32, C["white"], col, rx=16, sw=2)
                b += text(x + 47, y + 47, lbl, size=13, fill=col, weight="700")
    return svg(1000, 320, b, t["session"])


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    for lang, t in T.items():
        for name, fn in [("how-layers", layers), ("how-session", session)]:
            path = os.path.join(here, f"{name}.{lang}.svg")
            with open(path, "w", encoding="utf-8") as f:
                f.write(fn(t))
            print("wrote", os.path.relpath(path))


if __name__ == "__main__":
    main()
