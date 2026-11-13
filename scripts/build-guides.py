#!/usr/bin/env python3
"""Builds the documentation site from its Markdown sources.

The .md files in docs/guide/ (and docs/spec/memory-format.md) are the source: change
them and run this, never edit the .html by hand. It writes:

- docs/guide/<page>.html and <page>.es.html: each page, with a sidebar listing every
  page by group, an "on this page" list for long pages, and links to the page before
  and after;
- docs/guide/index.html and index.es.html: the start of the documentation;
- docs/sitemap.xml;
- docs/llms-full.txt: every English page as Markdown, for AI readers.

A tiny converter covers the Markdown these pages use: headings, paragraphs, lists,
tables, fenced code, <picture> blocks, inline code, bold, links and <a id> anchors.
No dependencies: plain Python 3.

Usage:  python3 scripts/build-guides.py
"""
import html as H
import pathlib
import re

ROOT = pathlib.Path(__file__).resolve().parent.parent
DOCS = ROOT / "docs"
G = DOCS / "guide"
SITE = "https://icalvete.github.io/augmented-brain/"
BASE = SITE + "guide/"
REPO = "https://github.com/icalvete/augmented-brain/blob/main/"
CHANGELOG = REPO + "CHANGELOG.md"
CHANGELOG_ES = REPO + "CHANGELOG.es.md"

# The sidebar, in order. Each page: slug, title (en, es), description (en, es) for
# search engines, a short line (en, es) for sharing and for the index. A page with no
# Spanish twin says es=False; its source may live elsewhere (src).
GROUPS = [
    (("Get started", "Empezar"), [
        dict(slug="getting-started", title=("Getting started", "Primeros pasos"),
             desc=("What augmented brain is, and how to install it from scratch, step by step, even if you have never opened a terminal: what you need, the installer, the first session and what to say in your first week.",
                   "Qué es augmented brain y cómo instalarlo desde cero, paso a paso, aunque nunca hayas abierto una terminal: qué necesitas, el instalador, la primera sesión y qué decir la primera semana."),
             line=("What it is, and from nothing to your first session, without knowing the terminal.",
                   "Qué es, y de cero a tu primera sesión, sin saber de terminales.")),
        dict(slug="install", title=("Installing", "Instalación"),
             desc=("How to install augmented brain on Linux, macOS and Windows: what the one command does on each system, where it puts things, what is different on each one, updating and uninstalling.",
                   "Cómo instalar augmented brain en Linux, macOS y Windows: qué hace el comando en cada sistema, dónde deja cada cosa, qué cambia en cada uno, actualizar y desinstalar."),
             line=("One command on each system: what it does, where things go, updating and removing it.",
                   "Un comando en cada sistema: qué hace, dónde deja cada cosa, actualizarlo y quitarlo.")),
    ]),
    (("Concepts", "Conceptos"), [
        dict(slug="spaces", title=("Spaces", "Espacios"),
             desc=("Spaces in augmented brain: work and personal kept apart, how the folder you open Claude in picks the space, the default space, adding a space and the shared notes.",
                   "Los espacios de augmented brain: trabajo y personal por separado, cómo la carpeta en la que abres Claude elige el espacio, el espacio por defecto, añadir un espacio y las notas compartidas."),
             line=("Work and personal apart: the folder you open Claude in picks the space.",
                   "Trabajo y personal por separado: la carpeta en la que abres Claude elige el espacio.")),
        dict(slug="tasks", title=("Tasks", "Tareas"),
             desc=("Tasks in augmented brain: what a task file looks like, the six phases, parking and resuming, the journal, steps and subtasks, watch dates and closing.",
                   "Las tareas de augmented brain: cómo es el fichero de una tarea, las seis fases, aparcar y retomar, el diario, pasos y subtareas, fechas para revisar y cerrar."),
             line=("Six phases, park and resume, steps and subtasks, dates to check again.",
                   "Seis fases, aparcar y retomar, pasos y subtareas, fechas para revisar.")),
        dict(slug="notes", title=("Areas, knowledge and the rest", "Áreas, conocimiento y lo demás"),
             desc=("The other notes of augmented brain: areas, references and lessons, people, the inbox, rules, your profile, repositories and the index.",
                   "Las otras notas de augmented brain: áreas, referencias y lecciones, personas, el buzón, reglas, tu perfil, repositorios y el índice."),
             line=("Areas, knowledge, people, the inbox, rules and your profile.",
                   "Áreas, conocimiento, personas, el buzón, reglas y tu perfil.")),
        dict(slug="editing-notes", title=("Editing notes by hand", "Editar las notas a mano"),
             desc=("How to edit augmented brain's notes by hand without breaking them: before and after, the header's rules, the journal line with its long dash, the inbox, and what not to touch.",
                   "Cómo editar a mano las notas de augmented brain sin romperlas: antes y después, las reglas de la cabecera, la línea del diario con su raya larga, el buzón y lo que no se toca."),
             line=("Change a note with any editor and keep it readable.",
                   "Cambia una nota con cualquier editor sin que deje de leerse.")),
        dict(slug="guard", title=("The guard", "El guardia"),
             desc=("The guard of augmented brain: what it stops (pushes, root, secrets), what it asks you first (production, untouchable machines), the repository lists, root exceptions, and why it is a safety net, not a security boundary.",
                   "El guardia de augmented brain: lo que para (push, root, secretos), lo que te pregunta antes (producción, máquinas intocables), las listas de repositorios, las excepciones de root, y por qué es una red, no un muro de seguridad."),
             line=("What it stops, what it asks you first, and what it is not.",
                   "Lo que para, lo que te pregunta antes, y lo que no es.")),
        dict(slug="how-it-works", title=("How it works inside", "Cómo funciona por dentro"),
             desc=("How augmented brain works inside, in four layers: you and your assistant, the plugin, the program and your files. A session step by step and where every file lives.",
                   "Cómo funciona augmented brain por dentro, en cuatro capas: tú y tu asistente, el plugin, el programa y tus ficheros. Una sesión paso a paso y dónde vive cada fichero."),
             line=("Four layers, a session step by step, and where every file lives.",
                   "Cuatro capas, una sesión paso a paso y dónde vive cada fichero.")),
        dict(slug="what-claude-may-do", title=("What Claude may do without asking you", "Qué puede hacer Claude sin preguntarte"),
             desc=("What Claude may do on its own and what asks you first: the guard and Claude Code's permissions side by side, the repository lists, the seal of the lists and accepting changes in the Guard page.",
                   "Qué puede hacer Claude por su cuenta y qué te pregunta antes: el guardia y los permisos de Claude Code uno al lado del otro, las listas de repositorios, el sello de las listas y aceptar cambios en la página Guardia."),
             line=("Never, always asks, without asking: the guard and Claude Code together.",
                   "Nunca, pregunta siempre, sin preguntar: el guardia y Claude Code juntos.")),
        dict(slug="where-things-live", title=("Where everything lives", "Dónde vive cada cosa"),
             desc=("Where augmented brain keeps your notes, configuration, state and secrets' names, how each is protected, and what the backup takes and leaves.",
                   "Dónde guarda augmented brain tus notas, la configuración, el estado y los nombres de tus secretos, cómo se protege cada cosa, y qué lleva la copia de seguridad y qué deja fuera."),
             line=("Notes, configuration, state and secrets: where, and how they are kept safe.",
                   "Notas, configuración, estado y secretos: dónde, y cómo se protegen.")),
    ]),
    (("Daily use", "Día a día"), [
        dict(slug="phrases", title=("What to say to Claude", "Qué decirle a Claude"),
             desc=("The phrases that work with augmented brain: tasks, the inbox and reminders, knowledge, rules, people, safety, the backup and the panel.",
                   "Las frases que funcionan con augmented brain: tareas, el buzón y los recordatorios, conocimiento, reglas, personas, seguridad, la copia de seguridad y el panel."),
             line=("Plain words for tasks, notes, reminders and safety.",
                   "Palabras normales para tareas, notas, recordatorios y seguridad.")),
        dict(slug="panel", title=("The panel", "El panel"),
             desc=("The panel of augmented brain: a page in your browser to see and change your spaces. Today, tasks, reminders, the inbox, the weekly review, repositories, secrets, sessions, the guard, versions, health and help.",
                   "El panel de augmented brain: una página en tu navegador para ver y cambiar tus espacios. Hoy, tareas, recordatorios, buzón, revisión semanal, repositorios, secretos, sesiones, guardia, versiones, salud y ayuda."),
             line=("Everything at a glance, in your browser, and every page in it.",
                   "Todo de un vistazo, en tu navegador, y cada página que tiene.")),
        dict(slug="panel-remote", title=("The panel from another device", "El panel desde otro dispositivo"),
             desc=("How to open the panel of augmented brain from your phone or another computer with augmented-brain panel remote, through your own Tailscale network: what you need once, who can open it, and why it is always temporary.",
                   "Cómo abrir el panel de augmented brain desde el móvil u otro ordenador con augmented-brain panel remote, a través de tu propia red de Tailscale: lo que hace falta una vez, quién puede abrirlo y por qué siempre es temporal."),
             line=("Your panel on the phone, through your own private network.",
                   "Tu panel en el móvil, por tu propia red privada.")),
        dict(slug="config-history", title=("Configuration versions", "Versiones de la configuración"),
             desc=("augmented brain keeps the last 10 versions of your configuration on this computer: see what changed and go back, from the panel's Versions page or with augmented-brain config history and config restore.",
                   "augmented brain guarda las 10 últimas versiones de tu configuración en este ordenador: ver qué cambió y volver atrás, desde la página Versiones del panel o con augmented-brain config history y config restore."),
             line=("See what changed in your configuration, and go back.",
                   "Ver qué cambió en tu configuración, y volver atrás.")),
        dict(slug="weekly-review", title=("The weekly review", "La revisión semanal"),
             desc=("The weekly review of augmented brain: emptying the inbox and looking at stalled tasks, with Claude or in the panel, in about ten minutes.",
                   "La revisión semanal de augmented brain: vaciar el buzón y mirar las tareas paradas, con Claude o en el panel, en unos diez minutos."),
             line=("Ten minutes a week: the inbox empty, nothing stalled.",
                   "Diez minutos a la semana: el buzón vacío y nada parado.")),
        dict(slug="backup", title=("Backup and restore", "Copia de seguridad y restauración"),
             desc=("The backup of augmented brain: a private git repository per space, run by itself after every session, and how to restore everything on a new machine.",
                   "La copia de seguridad de augmented brain: un repositorio git privado por espacio, que se hace sola después de cada sesión, y cómo recuperarlo todo en una máquina nueva."),
             line=("A private repository per space, and a new machine in minutes.",
                   "Un repositorio privado por espacio, y una máquina nueva en minutos.")),
        dict(slug="connect-services", title=("Connect your services", "Conecta tus servicios"),
             desc=("How to give Claude Code your calendar, mail, documents and Jira through MCP connectors, alongside augmented brain.",
                   "Cómo dar a Claude Code tu calendario, tu correo, tus documentos y Jira con conectores MCP, junto a augmented brain."),
             line=("Calendar, mail, documents and Jira, through MCP.",
                   "Calendario, correo, documentos y Jira, por MCP.")),
        dict(slug="remote", title=("Remote access", "Acceso remoto"),
             desc=("How to keep Claude reachable from another computer or your phone, also after a reboot, with augmented-brain remote: install, the trust step, linger, connecting from claude.ai or Claude Desktop, a session per task, and the security note.",
                   "Cómo mantener Claude accesible desde otro ordenador o el móvil, también tras reiniciar, con augmented-brain remote: instalar, el paso de confianza, el linger, conectarse desde claude.ai o Claude Desktop, una sesión por tarea y la nota de seguridad."),
             line=("Reach Claude on this computer from anywhere, even after a reboot, with a session per task.",
                   "Llega al Claude de este ordenador desde cualquier sitio, incluso tras reiniciar, con una sesión por tarea.")),
        dict(slug="import", title=("Bring in a memory you already have", "Trae una memoria que ya tienes"),
             desc=("How to bring a memory Claude Code already wrote into augmented brain's spaces with augmented-brain import: look first, then write.",
                   "Cómo traer a los espacios de augmented brain una memoria que Claude Code ya escribió, con augmented-brain import: primero mirar, luego escribir."),
             line=("Claude Code's own memory, brought into your spaces.",
                   "La memoria de Claude Code, traída a tus espacios.")),
        dict(slug="anywhere", title=("From your phone or another computer", "Desde el móvil u otro ordenador"),
             desc=("Working with augmented brain away from the computer that holds your notes: remote access and the panel from another device together, which session to use, and what to do when it stops answering.",
                   "Trabajar con augmented brain lejos del ordenador que guarda tus notas: el acceso remoto y el panel desde otro dispositivo juntos, qué sesión usar y qué hacer cuando deja de responder."),
             line=("Remote access and the panel together, and what to do when it drops.",
                   "Acceso remoto y panel juntos, y qué hacer cuando se cae.")),
        dict(slug="update-and-uninstall", title=("Updating and uninstalling", "Actualizar y desinstalar"),
             desc=("Updating augmented brain and uninstalling it without losing anything: what update does step by step, the three levels of uninstall, and moving to a new computer.",
                   "Actualizar augmented brain y desinstalarlo sin perder nada: qué hace update paso a paso, los tres niveles de uninstall y pasarse a un ordenador nuevo."),
             line=("Update, uninstall and move to a new computer, losing nothing.",
                   "Actualizar, desinstalar y cambiar de ordenador sin perder nada.")),
    ]),
    (("Reference", "Referencia"), [
        dict(slug="commands", title=("Commands", "Comandos"),
             desc=("Every command of augmented brain, with every option: init, space, check, doctor, import, backup, restore, panel, remote, session, config, update, uninstall, mcp, hook, version and help.",
                   "Todos los comandos de augmented brain, con todas sus opciones: init, space, check, doctor, import, backup, restore, panel, remote, session, config, update, uninstall, mcp, hook, version y help."),
             line=("Every command and option of the program, and its hooks.",
                   "Cada comando y opción del programa, y sus hooks.")),
        dict(slug="mcp-tools", title=("MCP tools", "Herramientas MCP"),
             desc=("The MCP tools augmented brain gives your assistant, grouped by purpose, with every parameter: tasks, the inbox, the weekly review, knowledge, people, safety and backup.",
                   "Las herramientas MCP que augmented brain da a tu asistente, agrupadas por uso, con cada parámetro: tareas, buzón, revisión semanal, conocimiento, personas, seguridad y copia de seguridad."),
             line=("The tools Claude uses behind your words, with every parameter.",
                   "Las herramientas que usa Claude detrás de tus palabras, con cada parámetro.")),
        dict(slug="configuration", title=("Configuration", "Configuración"),
             desc=("Every setting of augmented brain's config.toml, for the whole computer and for each space, with an example; secrets.toml, remote.toml, the other files and the environment variables.",
                   "Cada ajuste del config.toml de augmented brain, de todo el ordenador y de cada espacio, con un ejemplo; secrets.toml, remote.toml, los demás ficheros y las variables de entorno."),
             line=("Every setting of config.toml, with an example.",
                   "Cada ajuste de config.toml, con un ejemplo.")),
        dict(slug="memory-format", title=("Memory format", "Formato de la memoria"), es=False,
             src=DOCS / "spec" / "memory-format.md",
             desc=("The memory format of augmented brain: folders, note headers, the task lifecycle, configuration, backup and format versions.",
                   ""),
             line=("Folders, note headers and what the program reads. The contract.",
                   "Carpetas, cabeceras y lo que lee el programa. El contrato (en inglés).")),
        dict(slug="compatibility", title=("What stays stable", "Lo que no cambia"),
             desc=("What augmented brain promises not to break inside 1.x: your notes, your configuration, the commands, the assistant's tools, backups and installing; what it does not promise, and how it is checked.",
                   "Lo que augmented brain promete no romper dentro de las 1.x: tus notas, tu configuración, los comandos, las herramientas del asistente, las copias e instalar; lo que no promete, y cómo se comprueba."),
             line=("What every 1.x keeps working, and what may change.",
                   "Lo que toda 1.x sigue haciendo, y lo que puede cambiar.")),
    ]),
    (("Help", "Ayuda"), [
        dict(slug="troubleshooting", title=("Troubleshooting", "Si algo falla"),
             desc=("Problems with augmented brain and how to fix them: doctor, the panel's Help page, the guard, the backup, notes, the panel and sessions; and common questions.",
                   "Problemas con augmented brain y cómo arreglarlos: doctor, la página de Ayuda del panel, el guardia, la copia de seguridad, las notas, el panel y las sesiones; y preguntas frecuentes."),
             line=("Problems and fixes, doctor, and common questions.",
                   "Problemas y arreglos, doctor, y preguntas frecuentes.")),
        dict(slug="limitations", title=("Known limitations", "Limitaciones conocidas"),
             desc=("What augmented brain cannot do yet, or does only in part: only Claude Code, PowerShell on Windows, remote access on a Mac and Windows, what is checked by hand on a Mac, and what the guard cannot see.",
                   "Lo que augmented brain todavía no sabe hacer, o hace solo en parte: solo Claude Code, PowerShell en Windows, acceso remoto en Mac y Windows, lo que se comprueba a mano en un Mac y lo que el guardia no ve."),
             line=("What it cannot do yet, and what to do instead.",
                   "Lo que aún no sabe hacer, y qué hacer en su lugar.")),
        dict(slug="changelog", title=("Changelog", "Registro de cambios"), url=(CHANGELOG, CHANGELOG_ES),
             line=("What changed in each version.", "Qué cambió en cada versión.")),
    ]),
]

PAGES = [p for _, pages in GROUPS for p in pages if "url" not in p]

UI = {
    "en": dict(docs="Docs", menu="Documentation menu", onpage="On this page", prev="Previous", next="Next",
               skip="Skip to content", main="Main", copy="Copy", copied="Copied", english=" (in English)",
               index_title="Documentation", index_desc="The documentation of augmented brain: getting started, concepts, daily use, reference and help.",
               index_line="Everything about augmented brain, from the first session to every setting.",
               index_lead="Start with the first group if you are new. The reference is for when you need the exact name of a command, a tool or a setting.",
               footer="augmented brain · MIT license", logo="augmented brain logo: a green brain with the name across it",
               locale="en_US", other_locale="es_ES"),
    "es": dict(docs="Documentación", menu="Menú de la documentación", onpage="En esta página", prev="Anterior", next="Siguiente",
               skip="Saltar al contenido", main="Principal", copy="Copiar", copied="Copiado", english=" (en inglés)",
               index_title="Documentación", index_desc="La documentación de augmented brain: primeros pasos, conceptos, día a día, referencia y ayuda.",
               index_line="Todo sobre augmented brain, de la primera sesión a cada ajuste.",
               index_lead="Si eres nuevo, empieza por el primer grupo. La referencia es para cuando necesitas el nombre exacto de un comando, una herramienta o un ajuste.",
               footer="augmented brain · licencia MIT", logo="Logo de augmented brain: un cerebro verde con el nombre encima",
               locale="es_ES", other_locale="en_US"),
}


def L(lang):
    return 1 if lang == "es" else 0


def has_es(p):
    return p.get("es", True)


def file_name(slug, lang):
    """The html file of a page in lang; the English one when it has no Spanish twin."""
    p = next((x for x in PAGES if x["slug"] == slug), None)
    if lang == "es" and (p is None or has_es(p)):
        return slug + ".es.html"
    return slug + ".html"


def href(p, lang):
    return p["url"][L(lang)] if "url" in p else file_name(p["slug"], lang)


def label(p, lang):
    t = p["title"][L(lang)]
    if lang == "es" and "url" not in p and not has_es(p):
        t += UI["es"]["english"]
    return t


def slugify(text, seen):
    """The anchor of a heading, as GitHub makes it: lowercase, no punctuation, spaces as
    dashes, accented letters kept. A repeated one gets -1, -2…"""
    s = re.sub(r"<[^>]+>", "", text).strip().lower()
    s = re.sub(r"[^\w\- ]", "", s).replace(" ", "-")
    base, n = s, 0
    while s in seen:
        n += 1
        s = f"{base}-{n}"
    seen.add(s)
    return s


def link(url):
    if url.startswith(("http://", "https://", "mailto:", "#")):
        return url
    if url.startswith("../spec/memory-format.md"):
        return "memory-format.html" + url[len("../spec/memory-format.md"):]
    if url.startswith("../../"):
        return REPO + url[6:]
    if url.startswith("../spec/"):
        return REPO + "docs/spec/" + url[8:]
    path, _, frag = url.partition("#")
    if path.endswith(".md"):
        path = path[:-3] + ".html"
    return path + ("#" + frag if frag else "")


ANCHOR = r'<a id="[\w\-]+"></a>'


def inline(s):
    parts = re.split(r"(`[^`]+`|" + ANCHOR + ")", s)
    out = []
    for p in parts:
        if p.startswith("`") and p.endswith("`") and len(p) > 1:
            out.append("<code>" + H.escape(p[1:-1], quote=False) + "</code>")
            continue
        if re.fullmatch(ANCHOR, p):
            out.append(p)
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
    """The page's html, and its level-2 headings as (anchor, html) for the table of
    contents. TOC marks where the table goes: before the first level-2 heading."""
    lines = md.splitlines()
    out, toc, seen, i = [], [], set(), 0
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
        elif re.match(r"#{1,6} ", ln):
            n = len(ln) - len(ln.lstrip("#"))
            text = ln[n:].strip()
            if n == 1:
                out.append(f"  <h1>{inline(text)}</h1>")
            else:
                a = slugify(text.replace("`", ""), seen)
                if n == 2:
                    if not toc:
                        out.append("TOC")
                    toc.append((a, inline(text)))
                out.append(f'  <h{n} id="{a}">{inline(text)}</h{n}>')
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
                rows.append([c.strip() for c in lines[i].strip().strip("|").split("|")])
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
            while i < len(lines) and lines[i].strip() and not re.match(r"(#{1,6} |```|\||- |!\[|<picture>|\d+\. )", lines[i]):
                para.append(lines[i].strip())
                i += 1
            out.append(f"  <p>{inline(' '.join(para))}</p>")
    return out, toc


def esc(s):
    return H.escape(s, quote=True)


def head(lang, name, title, desc, ogdesc, alternates):
    """The <head> and the top of the page. alternates: {lang: file} of every language
    this page exists in."""
    u = UI[lang]
    url = BASE + name
    links = "".join(f'<link rel="alternate" hreflang="{l}" href="{BASE}{f}">\n' for l, f in alternates.items())
    links += f'<link rel="alternate" hreflang="x-default" href="{BASE}{alternates["en"]}">\n'
    locale = f'<meta property="og:locale" content="{u["locale"]}">\n'
    if len(alternates) > 1:
        locale += f'<meta property="og:locale:alternate" content="{u["other_locale"]}">\n'
    full = f"{title} · augmented brain"
    return f"""<!doctype html>
<html lang="{lang}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{esc(full)}</title>
<meta name="description" content="{esc(desc)}">
<link rel="canonical" href="{url}">
{links}<meta name="theme-color" content="#143024">
<meta name="color-scheme" content="light">
<meta property="og:type" content="website">
<meta property="og:site_name" content="augmented brain">
<meta property="og:title" content="{esc(full)}">
<meta property="og:description" content="{esc(ogdesc)}">
<meta property="og:url" content="{url}">
{locale}<meta property="og:image" content="{SITE}img/og-image.png">
<meta property="og:image:width" content="1280">
<meta property="og:image:height" content="640">
<meta property="og:image:alt" content="{esc(u["logo"])}">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="{esc(full)}">
<meta name="twitter:description" content="{esc(ogdesc)}">
<meta name="twitter:image" content="{SITE}img/og-image.png">
<link rel="icon" href="../img/favicon.svg" type="image/svg+xml">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Quicksand:wght@600;700&family=IBM+Plex+Sans:wght@400;500;600&family=IBM+Plex+Mono:wght@400;500&display=swap">
<link rel="stylesheet" href="../site.css">
</head>
<body>
<a class="skip" href="#main">{u["skip"]}</a>
"""


def header(lang, other):
    """The bar at the top: the brand, the documentation, the other language, GitHub."""
    u = UI[lang]
    home = "../es/" if lang == "es" else "../"
    docs = "index.es.html" if lang == "es" else "index.html"
    switch = ""
    if other:
        o = "en" if lang == "es" else "es"
        switch = f'\n    <a href="{other}" hreflang="{o}" data-lang="{o}">{"English" if o == "en" else "Español"}</a>'
    return f"""<header class="wrap top">
  <a class="brand" href="{home}"><img src="../img/favicon.svg" alt="">augmented brain</a>
  <nav aria-label="{u["main"]}">
    <a href="{docs}">{u["docs"]}</a>{switch}
    <a href="https://github.com/icalvete/augmented-brain">GitHub</a>
  </nav>
</header>
"""


def sidebar(lang, current):
    u = UI[lang]
    out = [f'<nav class="docnav" aria-label="{u["docs"]}">\n<details open id="docnav"><summary>{u["menu"]}</summary>']
    for g, pages in GROUPS:
        out.append(f'<p class="docnav-group">{g[L(lang)]}</p>\n<ul>')
        for p in pages:
            cur = ' aria-current="page"' if p.get("slug") == current else ""
            out.append(f'  <li><a href="{href(p, lang)}"{cur}>{esc(label(p, lang))}</a></li>')
        out.append("</ul>")
    out.append("</details>\n</nav>")
    # On a narrow screen the menu starts folded above the page, so the page comes first.
    out.append('<script>if (window.matchMedia("(max-width: 820px)").matches) '
               'document.getElementById("docnav").removeAttribute("open");</script>')
    return "\n".join(out) + "\n"


def pager(lang, slug):
    u = UI[lang]
    i = next(n for n, p in enumerate(PAGES) if p["slug"] == slug)
    out = ['  <nav class="pager" aria-label="' + u["prev"] + " / " + u["next"] + '">']
    if i > 0:
        p = PAGES[i - 1]
        out.append(f'    <a class="prev" href="{href(p, lang)}"><span>← {u["prev"]}</span>{esc(label(p, lang))}</a>')
    if i + 1 < len(PAGES):
        p = PAGES[i + 1]
        out.append(f'    <a class="next" href="{href(p, lang)}"><span>{u["next"]} →</span>{esc(label(p, lang))}</a>')
    out.append("  </nav>")
    return "\n".join(out)


# A Copy button on every code block, so nobody has to select a long command by hand.
COPY = """<script>
document.querySelectorAll(".prose pre").forEach(function (pre) {
  var b = document.createElement("button");
  b.className = "copy"; b.type = "button"; b.textContent = "{copy}";
  b.addEventListener("click", function () {
    var text = pre.querySelector("code").textContent;
    var done = function () { b.textContent = "{copied}"; setTimeout(function () { b.textContent = "{copy}"; }, 1500); };
    var select = function () {
      var r = document.createRange(); r.selectNodeContents(pre.querySelector("code"));
      var s = window.getSelection(); s.removeAllRanges(); s.addRange(r);
    };
    if (navigator.clipboard) { navigator.clipboard.writeText(text).then(done, select); } else { select(); }
  });
  var box = document.createElement("div"); box.className = "codebox";
  pre.parentNode.insertBefore(box, pre); box.appendChild(pre); box.appendChild(b);
});
document.querySelectorAll("[data-lang]").forEach(function (a) {
  a.addEventListener("click", function () { try { localStorage.setItem("ab-lang", a.dataset.lang); } catch (e) {} });
});
</script>"""


def footer(lang):
    u = UI[lang]
    return (f'<footer>\n  <div class="wrap"><span>{u["footer"]} · <a href="{CHANGELOG if lang == "en" else CHANGELOG_ES}">'
            f'{"Changelog" if lang == "en" else "Registro de cambios"}</a></span></div>\n</footer>\n'
            + COPY.replace("{copy}", u["copy"]).replace("{copied}", u["copied"]) + "\n</body>\n</html>\n")


def source(p, lang):
    path = p.get("src", G / (p["slug"] + (".es.md" if lang == "es" else ".md")))
    md = pathlib.Path(path).read_text()
    # The line to the other language is for GitHub readers: the site has its own switch.
    return "\n".join(l for l in md.splitlines() if not re.match(r"\[(Read in English|Leer en español)\]", l))


# A page gets an "on this page" list when it has this many sections or more. MCP tools
# has its own list, by purpose.
TOC_MIN = 5
NO_TOC = {"mcp-tools"}


def page(p, lang):
    u = UI[lang]
    slug = p["slug"]
    name = file_name(slug, lang)
    body, toc = convert(source(p, lang))
    alternates = {"en": slug + ".html"}
    if has_es(p):
        alternates["es"] = slug + ".es.html"
    other = alternates.get("es" if lang == "en" else "en")
    if lang == "en" and not has_es(p):
        other = "index.es.html"
    contents = ""
    if len(toc) >= TOC_MIN and slug not in NO_TOC:
        contents = (f'  <nav class="toc" aria-label="{u["onpage"]}"><p>{u["onpage"]}</p><ul>'
                    + "".join(f'<li><a href="#{a}">{t}</a></li>' for a, t in toc) + "</ul></nav>")
    html_body = "\n".join(contents if b == "TOC" else b for b in body if b != "TOC" or contents)
    out = (head(lang, name, p["title"][L(lang)], p["desc"][L(lang)], p["line"][L(lang)], alternates)
           + header(lang, other)
           + '\n<div class="wrap docs">\n' + sidebar(lang, slug)
           + '<main class="prose" id="main">\n' + html_body + "\n" + pager(lang, slug) + "\n</main>\n</div>\n\n"
           + footer(lang))
    (G / name).write_text(out)
    print("ok", name)


def index(lang):
    u = UI[lang]
    name = "index.es.html" if lang == "es" else "index.html"
    body = [f'  <h1>{u["index_title"]}</h1>', f'  <p>{u["index_lead"]}</p>']
    seen = set()
    for g, pages in GROUPS:
        body.append(f'  <h2 id="{slugify(g[L(lang)], seen)}">{g[L(lang)]}</h2>')
        body.append('  <div class="guides">' + "".join(
            f'<a class="guide" href="{href(p, lang)}"><h3>{esc(label(p, lang))}</h3><span>{esc(p["line"][L(lang)])}</span></a>'
            for p in pages) + "</div>")
    alternates = {"en": "index.html", "es": "index.es.html"}
    other = alternates["es" if lang == "en" else "en"]
    out = (head(lang, name, u["index_title"], u["index_desc"], u["index_line"], alternates)
           + header(lang, other)
           + '\n<div class="wrap docs">\n' + sidebar(lang, "")
           + '<main class="prose docs-index" id="main">\n' + "\n".join(body) + "\n</main>\n</div>\n\n"
           + footer(lang))
    (G / name).write_text(out)
    print("ok", name)


def sitemap():
    def url(loc, en, es=None):
        s = f"  <url>\n    <loc>{loc}</loc>\n"
        s += f'    <xhtml:link rel="alternate" hreflang="en" href="{en}"/>\n'
        if es:
            s += f'    <xhtml:link rel="alternate" hreflang="es" href="{es}"/>\n'
        s += f'    <xhtml:link rel="alternate" hreflang="x-default" href="{en}"/>\n  </url>\n'
        return s
    out = '<?xml version="1.0" encoding="UTF-8"?>\n'
    out += '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">\n'
    out += url(SITE, SITE, SITE + "es/") + url(SITE + "es/", SITE, SITE + "es/")
    for slug in ["index"] + [p["slug"] for p in PAGES]:
        p = next((x for x in PAGES if x["slug"] == slug), {})
        en = BASE + slug + ".html"
        es = BASE + slug + ".es.html" if has_es(p) else None
        out += url(en, en, es)
        if es:
            out += url(es, en, es)
    out += "</urlset>\n"
    (DOCS / "sitemap.xml").write_text(out)
    print("ok sitemap.xml")


def absolute(url, slug):
    """A link of a page as a full address, so it still works out of the page."""
    if url.startswith("#"):
        return BASE + slug + ".html" + url
    url = link(url)
    if url.startswith(("http://", "https://", "mailto:")):
        return url
    if url.startswith("../"):
        return SITE + url[3:]
    return BASE + url


def llms_full():
    """Every English page, as Markdown, in the order of the sidebar: one file an AI reader
    can take whole. Generated, so it never drifts from the pages."""
    out = ["# augmented brain: the full documentation", "",
           "> Your brain, augmented with AI. augmented brain gives an AI assistant (Claude Code today) "
           "memory, a method and a guard. This file is every page of the documentation at "
           f"{BASE}, in English, generated from the Markdown sources by scripts/build-guides.py.", ""]
    for p in PAGES:
        md = re.sub(r"\]\(([^)\s]+)\)", lambda m: "](" + absolute(m.group(1), p["slug"]) + ")", source(p, "en").strip())
        out += ["", "---", "", f"Source: {BASE}{p['slug']}.html", "", md, ""]
    (DOCS / "llms-full.txt").write_text("\n".join(out))
    print("ok llms-full.txt")


for p in PAGES:
    page(p, "en")
    if has_es(p):
        page(p, "es")
index("en")
index("es")
sitemap()
llms_full()
