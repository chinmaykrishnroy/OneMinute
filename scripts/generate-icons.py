"""Generate OneMinute icon assets from the same geometry as the base SVG."""

from pathlib import Path
from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parents[1]
APP = ROOT / "apps" / "web" / "app"
ICONS = ROOT / "apps" / "web" / "public" / "icons"
INK, GREEN, PAPER, PURPLE = "#252822", "#d7edb7", "#fffefb", "#7048b6"

SVG = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" role="img" aria-label="OneMinute">
  <rect x="16" y="16" width="480" height="480" rx="136" fill="{INK}"/>
  <rect x="38" y="38" width="436" height="436" rx="116" fill="{GREEN}"/>
  <circle cx="256" cy="256" r="164" fill="{INK}"/>
  <circle cx="256" cy="256" r="136" fill="{PAPER}"/>
  <path d="M256 256V132" fill="none" stroke="{INK}" stroke-width="30" stroke-linecap="round"/>
  <path d="M256 256L326 186" fill="none" stroke="{INK}" stroke-width="30" stroke-linecap="round"/>
  <circle cx="256" cy="256" r="43" fill="{INK}"/>
  <circle cx="256" cy="256" r="23" fill="{PURPLE}"/>
</svg>
'''


def raster(size: int) -> Image.Image:
    scale = 4
    image = Image.new("RGBA", (size * scale, size * scale), (0, 0, 0, 0))
    draw = ImageDraw.Draw(image)
    def box(values): return tuple(round(value * size * scale / 512) for value in values)
    draw.rounded_rectangle(box((16, 16, 496, 496)), radius=round(136 * size * scale / 512), fill=INK)
    draw.rounded_rectangle(box((38, 38, 474, 474)), radius=round(116 * size * scale / 512), fill=GREEN)
    draw.ellipse(box((92, 92, 420, 420)), fill=INK)
    draw.ellipse(box((120, 120, 392, 392)), fill=PAPER)
    width = max(1, round(30 * size * scale / 512))
    draw.line((box((256, 256))[0], box((256, 256))[1], box((256, 132))[0], box((256, 132))[1]), fill=INK, width=width)
    radius = width // 2
    x, y = box((256, 132))
    draw.ellipse((x-radius, y-radius, x+radius, y+radius), fill=INK)
    draw.line((box((256, 256))[0], box((256, 256))[1], box((326, 186))[0], box((326, 186))[1]), fill=INK, width=width)
    x, y = box((326, 186))
    draw.ellipse((x-radius, y-radius, x+radius, y+radius), fill=INK)
    draw.ellipse(box((213, 213, 299, 299)), fill=INK)
    draw.ellipse(box((233, 233, 279, 279)), fill=PURPLE)
    return image.resize((size, size), Image.Resampling.LANCZOS)


def main() -> None:
    APP.mkdir(parents=True, exist_ok=True)
    ICONS.mkdir(parents=True, exist_ok=True)
    (APP / "icon.svg").write_text(SVG, encoding="utf-8")
    (ICONS / "icon.svg").write_text(SVG, encoding="utf-8")
    for size in (32, 48, 180, 192, 512):
        raster(size).save(ICONS / f"icon-{size}.png", optimize=True)
    raster(180).save(APP / "apple-icon.png", optimize=True)
    raster(32).save(APP / "icon.png", optimize=True)
    raster(256).save(ICONS / "favicon.ico", sizes=[(16, 16), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)])


if __name__ == "__main__":
    main()
