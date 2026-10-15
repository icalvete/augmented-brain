# Brand assets

Logo, icons and palette for augmented brain. Everything in `out/` is generated
from the sources below, so edit those and rebuild instead of touching the outputs.
To see every asset in place (README, browser tab, phone, social card), open
[`docs/brand.html`](../../docs/brand.html).

## Sources

- `src/logo.svg`: the full logo (brain and wordmark), vectorized from the original drawing.
- `src/brain.svg`: the brain alone, for the symbol, the icons and the headers.
- `fonts/Quicksand.ttf`: Quicksand, by Andrew Paglinawan, under the SIL Open
  Font License (`fonts/OFL.txt`). The wordmark is converted to paths, so the
  SVGs do not need the font installed.

## Rebuild

```bash
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
.venv/bin/python build.py          # writes out/
```

Both SVGs in `src/` come from flat PNG drawings. To vectorize a new one:

```bash
.venv/bin/python vectorize.py drawing.png src/brain.svg '#E1EBE2' '#143024' '#9CC9AF' '#6B9A82'
```

The first colour is the background (removed); the others are the drawing's palette.

## Outputs

| File | Use |
|------|-----|
| `logo.svg`, `logo.png` | Main logo: brain with the wordmark across it |
| `symbol.svg`, `symbol.png` | Brain only |
| `logo-dark.*`, `symbol-dark.*` | Same, for dark backgrounds: a light border keeps the dark outline visible |
| `header.svg`, `header.png` | Header for the README and the site: the main logo on a white rounded card, readable on light and dark pages |
| `og-image.*` | Social preview, 1280×640 (GitHub: Settings → Social preview) |
| `favicon.svg`, `favicon.ico`, `favicon-16.png`, `favicon-32.png` | Browser icons, with the light border so they work on dark tabs (the 16 px one is a simplified silhouette) |
| `apple-touch-icon.png` | 180×180 icon with a solid background |
| `palette.svg`, `palette.png`, `palette.css` | Colours, also as CSS variables (`--ab-ink`, `--ab-mint`, `--ab-sage`, `--ab-soft`) |

In a README:

```html
<img alt="augmented brain" src="assets/brand/out/header.svg" width="320">
```

There is a single header on purpose: GitHub may pick the light image of a `<picture>` on
a dark page (it follows the system theme, not GitHub's), so the header itself must read
on both backgrounds. It sits on a white card with rounded corners, so the logo keeps its
original colours on any page.

## Palette

| Name | Hex | Use |
|------|-----|-----|
| ink | `#143024` | Outlines and text |
| mint | `#9CC9AF` | Fill |
| sage | `#6B9A82` | Shade |
| soft | `#E1EBE2` | Light background |

The PNGs carry no metadata: `build.py` rewrites them without any extra chunks.
