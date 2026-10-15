"""Turns text into an SVG path with fontTools (no installed fonts needed)."""
from fontTools.ttLib import TTFont
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.varLib.instancer import instantiateVariableFont

_cache = {}


def load(path, wght=700):
    key = (path, wght)
    if key not in _cache:
        f = TTFont(path)
        if "fvar" in f:
            axes = {a.axisTag: a.defaultValue for a in f["fvar"].axes}
            if "wght" in axes:
                lo = min(a.minValue for a in f["fvar"].axes if a.axisTag == "wght")
                hi = max(a.maxValue for a in f["fvar"].axes if a.axisTag == "wght")
                axes["wght"] = max(lo, min(hi, wght))
            f = instantiateVariableFont(f, axes)
        _cache[key] = f
    return _cache[key]


def text_path(text, font_path, size, x=0.0, y=0.0, wght=700, tracking=0.0):
    """Returns (d, width) of the text with its baseline at y. Tracking in em."""
    f = load(font_path, wght)
    upm = f["head"].unitsPerEm
    scale = size / upm
    cmap = f.getBestCmap()
    gs = f.getGlyphSet()
    hmtx = f["hmtx"]
    pen = SVGPathPen(gs)
    cx = 0.0
    for ch in text:
        name = cmap.get(ord(ch))
        if name is None:
            continue
        # y grows downwards in SVG: flip the axis
        tp = TransformPen(pen, (scale, 0, 0, -scale, x + cx * scale, y))
        gs[name].draw(tp)
        cx += hmtx[name][0] + tracking * upm
    return pen.getCommands(), cx * scale
