#!/usr/bin/env python3
"""Vectorize a flat raster drawing (an Ideogram PNG) into a clean SVG.

Usage: python vectorize.py <input.png> <output.svg> <#bg> <#color> [<#color> ...] [--map #from=#to ...]

Steps: crop to the drawing, map every pixel to the nearest given color (posterize, no
dithering), make the background transparent by flood-filling from the borders (so light
areas enclosed by the drawing, such as letters, stay), trace with vtracer in "cutout"
mode, optionally recolor (--map), and keep only the drawing's paths.
"""
import io
import re
import sys

import vtracer
from PIL import Image, ImageChops, ImageDraw

BG_TOLERANCE = 28  # how far (per channel) a pixel may be from the background color


def hexrgb(h):
    h = h.lstrip("#")
    return tuple(int(h[i:i + 2], 16) for i in (0, 2, 4))


def main():
    args = sys.argv[1:]
    maps = {}
    if "--map" in args:
        i = args.index("--map")
        maps = dict(a.upper().split("=") for a in args[i + 1:])
        args = args[:i]
    src, dst, colors = args[0], args[1], [hexrgb(c) for c in args[2:]]
    bg = colors[0]
    img = Image.open(src).convert("RGB")

    # crop to the drawing, with a small margin
    diff = Image.eval(ImageChops.difference(img, Image.new("RGB", img.size, bg)).convert("L"),
                      lambda v: 255 if v > BG_TOLERANCE else 0)
    box = diff.getbbox()
    m, (w, h) = 12, img.size
    box = (max(box[0] - m, 0), max(box[1] - m, 0), min(box[2] + m, w), min(box[3] + m, h))
    img = img.crop(box)

    # posterize to the given colors, no dithering
    pal = Image.new("P", (1, 1))
    flatpal = [v for c in colors for v in c]
    pal.putpalette(flatpal + flatpal[:3] * (256 - len(colors)))
    q = img.quantize(palette=pal, dither=Image.Dither.NONE).convert("RGB")

    # background to transparent: flood fill from the borders only
    marker = (255, 0, 255)
    W, H = q.size
    for x, y in [(0, 0), (W - 1, 0), (0, H - 1), (W - 1, H - 1)]:
        if q.getpixel((x, y)) == bg:
            ImageDraw.floodfill(q, (x, y), marker, thresh=0)
    flat = q.convert("RGBA")
    fp = flat.load()
    for y in range(H):
        for x in range(W):
            if fp[x, y][:3] == marker:
                fp[x, y] = (0, 0, 0, 0)

    buf = io.BytesIO()
    flat.save(buf, format="PNG")
    svg = vtracer.convert_raw_image_to_svg(
        buf.getvalue(), img_format="png", colormode="color", hierarchical="cutout",
        mode="spline", filter_speckle=8, color_precision=8, layer_difference=8,
        corner_threshold=60, length_threshold=4.0, splice_threshold=45, path_precision=2)

    svg = re.sub(r"<!--.*?-->", "", svg, flags=re.S)
    svg = re.sub(r"<\?xml[^>]*\?>", "", svg)
    # drop transparent leftovers (vtracer paints them black)
    svg = re.sub(r'<path[^>]*fill="#000000"[^>]*/>\s*', "", svg)
    # vtracer averages each region's color: snap every fill back to the nearest given color
    def snap(m):
        c = hexrgb(m.group(1))
        n = min(colors, key=lambda p: sum((a - b) ** 2 for a, b in zip(p, c)))
        return 'fill="#%02X%02X%02X"' % n
    svg = re.sub(r'fill="(#[0-9A-Fa-f]{6})"', snap, svg)
    for a, b in maps.items():
        svg = re.sub(a, b, svg, flags=re.I)
    with open(dst, "w", encoding="utf-8") as f:
        f.write(svg.strip() + "\n")
    used = sorted(set(re.findall(r'fill="(#[0-9A-Fa-f]{6})"', svg)))
    print(f"OK {dst}: {svg.count('<path')} paths, colors {used}, size {img.size}")


if __name__ == "__main__":
    main()
