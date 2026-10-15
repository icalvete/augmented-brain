#!/usr/bin/env python3
"""Builds the augmented brain logo and its assets from src/ and the font.

Usage: python build.py [output_dir]   (default: out/)
No absolute paths or keys: everything is relative to this file.
"""
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)

from textpath import text_path  # noqa: E402
FONT = os.path.join(HERE, "fonts", "Quicksand.ttf")
BRAIN = os.path.join(HERE, "src", "brain.svg")   # the brain alone (symbol, favicons, header)
LOGO = os.path.join(HERE, "src", "logo.svg")     # the full logo, vectorized from the original

# Palette, measured on the original logo
INK = "#143024"     # dark green: outline and text
MINT = "#9CC9AF"    # sage green: fill
SAGE = "#6B9A82"    # darker sage: shade


def brain_group():
    """Returns (content, width, height) of the brain."""
    with open(BRAIN, encoding="utf-8") as f:
        s = f.read()
    w = float(re.search(r'width="([\d.]+)"', s).group(1))
    h = float(re.search(r'height="([\d.]+)"', s).group(1))
    inner = re.search(r"<svg[^>]*>(.*)</svg>", s, re.S).group(1).strip()
    return inner, w, h


def text_width(line, size):
    return text_path(line, FONT, size, 0, 0, wght=700)[1]


def wordmark(x, y, size, lines, leading=0.92, fill=INK, outline=None, outline_w=0.0):
    """Multi-line text as paths. The first line starts at x; the others are
    right-aligned with it. With outline, a stroke is drawn behind the fill."""
    parts = []
    first_w = text_width(lines[0], size)
    for i, line in enumerate(lines):
        d, w = text_path(line, FONT, size, 0, 0, wght=700)
        lx = x if i == 0 else x + first_w - w
        ly = y + i * size * leading
        tr = f'transform="translate({lx:.1f} {ly:.1f})"'
        if outline:
            parts.append(f'<path {tr} d="{d}" fill="none" stroke="{outline}" '
                         f'stroke-width="{outline_w:.1f}" stroke-linejoin="round"/>')
        parts.append(f'<path {tr} d="{d}" fill="{fill}"/>')
    return "\n".join(parts)


def svg(w, h, body, bg=None):
    rect = f'<rect width="100%" height="100%" fill="{bg}"/>' if bg else ""
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {w:.0f} {h:.0f}" '
            f'width="{w:.0f}" height="{h:.0f}">{rect}{body}</svg>\n')


HALO_W = 14  # width of the light border for dark backgrounds, in source units


def halo(inner):
    """On dark backgrounds the dark outline disappears: draw the same paths behind,
    all in the light background colour and stroked, like a sticker border."""
    flat = re.sub(r'fill="#[0-9A-Fa-f]{6}"', f'fill="{BG_SOFT}"', inner)
    return (f'<g stroke="{BG_SOFT}" stroke-width="{HALO_W}" stroke-linejoin="round">'
            f'{flat}</g>{inner}')


def logo(dark=False):
    """Main logo: the original drawing, vectorized (src/logo.svg), with a small margin.
    dark=True adds the light border for dark backgrounds."""
    with open(LOGO, encoding="utf-8") as f:
        src = f.read()
    w = float(re.search(r'width="([\d.]+)"', src).group(1))
    h = float(re.search(r'height="([\d.]+)"', src).group(1))
    inner = re.search(r"<svg[^>]*>(.*)</svg>", src, re.S).group(1).strip()
    if dark:
        inner = halo(inner)
    pad = 16
    return svg(w + 2 * pad, h + 2 * pad, f'<g transform="translate({pad} {pad})">{inner}</g>')


def symbol(square=False, bg=None, pad_ratio=0.03, dark=False):
    """The brain alone. square=True centres it in a square (favicons).
    dark=True adds the light border for dark backgrounds."""
    inner, bw, bh = brain_group()
    if dark:
        inner = halo(inner)
    pad = max(bw, bh) * pad_ratio
    if not square:
        return svg(bw + 2 * pad, bh + 2 * pad,
                   f'<g transform="translate({pad:.1f} {pad:.1f})">{inner}</g>', bg)
    side = max(bw, bh) + 2 * pad
    ox, oy = (side - bw) / 2, (side - bh) / 2
    return svg(side, side, f'<g transform="translate({ox:.1f} {oy:.1f})">{inner}</g>', bg)


CARD_LINE = "#D7E2DB"  # hairline around the header card, so it also shows on white pages


def header():
    """Header for the README and the site: the main logo (name inside the brain) on a
    white rounded card. One version for every background: GitHub may show a light image
    on a dark page, and on its own card the logo keeps its original colours."""
    src = logo()
    lw = float(re.search(r'width="([\d.]+)"', src).group(1))
    lh = float(re.search(r'height="([\d.]+)"', src).group(1))
    inner = re.search(r"<svg[^>]*>(.*)</svg>", src, re.S).group(1)
    pad = lh * 0.10
    W, H = lw + 2 * pad, lh + 2 * pad
    line = 3
    card = (f'<rect x="{line / 2}" y="{line / 2}" width="{W - line:.1f}" height="{H - line:.1f}" '
            f'rx="{H * 0.12:.0f}" fill="#FFFFFF" stroke="{CARD_LINE}" stroke-width="{line}"/>')
    return svg(W, H, card + f'<g transform="translate({pad:.1f} {pad:.1f})">{inner}</g>')


def og_image():
    """Social preview (GitHub Open Graph), 1280×640, logo centred."""
    W, H = 1280, 640
    logo_svg = logo()
    lw = float(re.search(r'width="([\d.]+)"', logo_svg).group(1))
    lh = float(re.search(r'height="([\d.]+)"', logo_svg).group(1))
    inner = re.search(r"<svg[^>]*>(.*)</svg>", logo_svg, re.S).group(1)
    s = (H * 0.80) / lh
    x, y = (W - lw * s) / 2, (H - lh * s) / 2
    body = f'<g transform="translate({x:.1f} {y:.1f}) scale({s:.4f})">{inner}</g>'
    return svg(W, H, body, bg=BG_SOFT)


BG_SOFT = "#E1EBE2"  # light sage background (from the original), for OG and the Apple icon

PALETTE = [("ink", INK, "outline and text"), ("mint", MINT, "fill"),
           ("sage", SAGE, "shade"), ("soft", BG_SOFT, "light background")]


def palette_svg():
    sw, W = 220, 220 * len(PALETTE)
    body = ""
    for i, (name, hexv, use) in enumerate(PALETTE):
        x = i * sw
        body += f'<rect x="{x}" y="0" width="{sw}" height="160" fill="{hexv}"/>'
        for j, (t, sz) in enumerate([(name, 26), (hexv, 22), (use, 18)]):
            d, _ = text_path(t, FONT, sz, x + 16, 200 + j * 30, wght=700 if j == 0 else 500)
            body += f'<path d="{d}" fill="{INK}"/>'
    return svg(W, 290, body, bg="#FFFFFF")


def palette_css():
    lines = [":root {"] + [f"  --ab-{n}: {h};  /* {u} */" for n, h, u in PALETTE] + ["}"]
    return "\n".join(lines) + "\n"


def png(svg_text, path, width, bg=None):
    import cairosvg
    cairosvg.svg2png(bytestring=svg_text.encode(), write_to=path,
                     output_width=width, background_color=bg)


def favicon_16(fav_svg, path):
    """At 16 px the folds blur into a blob: just the silhouette in the fill colour
    with a 1 px outline."""
    import io
    import cairosvg
    from PIL import Image, ImageFilter
    big = Image.open(io.BytesIO(cairosvg.svg2png(bytestring=fav_svg.encode(), output_width=256)))
    mask = big.getchannel("A").resize((16, 16), Image.LANCZOS).point(lambda a: 255 if a > 110 else 0)
    edge = Image.eval(mask.filter(ImageFilter.MinFilter(3)), lambda a: 255 - a)
    icon = Image.new("RGBA", (16, 16), (0, 0, 0, 0))
    icon.paste(Image.new("RGBA", (16, 16), MINT), (0, 0), mask)
    border = Image.composite(mask, Image.new("L", (16, 16), 0), edge)
    icon.paste(Image.new("RGBA", (16, 16), INK), (0, 0), border)
    icon.save(path)


def main():
    out = sys.argv[1] if len(sys.argv) > 1 else os.path.join(HERE, "out")
    os.makedirs(out, exist_ok=True)
    # The browser tab can be dark or light: the favicon always carries the light border
    fav = symbol(square=True, dark=True)
    svgs = {
        "logo.svg": logo(),
        "logo-dark.svg": logo(dark=True),
        "symbol.svg": symbol(),
        "symbol-dark.svg": symbol(dark=True),
        "favicon.svg": fav,
        "header.svg": header(),
        "og-image.svg": og_image(),
        "palette.svg": palette_svg(),
    }
    for name, content in svgs.items():
        with open(os.path.join(out, name), "w", encoding="utf-8") as f:
            f.write(content)
    with open(os.path.join(out, "palette.css"), "w", encoding="utf-8") as f:
        f.write(palette_css())

    p = lambda n: os.path.join(out, n)  # noqa: E731
    png(svgs["logo.svg"], p("logo.png"), 1024)
    png(svgs["logo-dark.svg"], p("logo-dark.png"), 1024)
    png(svgs["symbol.svg"], p("symbol.png"), 512)
    png(svgs["symbol-dark.svg"], p("symbol-dark.png"), 512)
    png(svgs["header.svg"], p("header.png"), 640)
    png(svgs["og-image.svg"], p("og-image.png"), 1280)
    png(svgs["palette.svg"], p("palette.png"), 880)
    for sz in (16, 32, 48):
        png(fav, p(f"favicon-{sz}.png"), sz)
    # iOS fills transparency with black: the Apple icon gets a background
    png(symbol(square=True, bg=BG_SOFT, pad_ratio=0.12), p("apple-touch-icon.png"), 180)

    from PIL import Image
    favicon_16(fav, p("favicon-16.png"))
    imgs = [Image.open(p(f"favicon-{s}.png")) for s in (16, 32, 48)]
    imgs[2].save(p("favicon.ico"), sizes=[(16, 16), (32, 32), (48, 48)])
    os.remove(p("favicon-48.png"))
    # Public repo: PNGs are rewritten without any metadata chunks
    for n in os.listdir(out):
        if n.endswith(".png"):
            with Image.open(p(n)) as im:
                im.load()
                clean = Image.frombytes(im.mode, im.size, im.tobytes())
            clean.save(p(n), optimize=True)
    for n in sorted(os.listdir(out)):
        print(f"OK {n}")


if __name__ == "__main__":
    main()
