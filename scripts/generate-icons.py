#!/usr/bin/env python3
"""
Generate platform-specific icons from assets/icon.png.

Requires: pip install pillow

Output (all in assets/):
  icon.ico                    Windows icon (multi-size)
  icon-NxN.png                Linux hicolor theme icons
  bore-desktop.iconset/       macOS iconset (run iconutil -c icns on macOS)
"""

import os
import sys
from PIL import Image

ASSETS_DIR = os.path.join(os.path.dirname(__file__), "..", "assets")
SOURCE = os.path.join(ASSETS_DIR, "icon.png")

if not os.path.exists(SOURCE):
    print(f"Error: {SOURCE} not found.")
    sys.exit(1)

src = Image.open(SOURCE).convert("RGBA")
print(f"Source: {SOURCE} ({src.size[0]}x{src.size[1]})")

# --- Windows .ico ---
ico_sizes = [(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]
ico_path = os.path.join(ASSETS_DIR, "icon.ico")
img_256 = src.resize((256, 256), Image.LANCZOS)
img_256.save(ico_path, format="ICO", sizes=ico_sizes)
print(f"  {ico_path} ({[s[0] for s in ico_sizes]})")

# --- Linux PNGs ---
linux_sizes = [16, 24, 32, 48, 64, 128, 256, 512]
for s in linux_sizes:
    out = os.path.join(ASSETS_DIR, f"icon-{s}x{s}.png")
    src.resize((s, s), Image.LANCZOS).save(out, format="PNG")
    print(f"  {out}")

# --- macOS .iconset ---
iconset_dir = os.path.join(ASSETS_DIR, "bore-desktop.iconset")
os.makedirs(iconset_dir, exist_ok=True)

icns_specs = {
    "icon_16x16.png": 16,
    "icon_16x16@2x.png": 32,
    "icon_32x32.png": 32,
    "icon_32x32@2x.png": 64,
    "icon_128x128.png": 128,
    "icon_128x128@2x.png": 256,
    "icon_256x256.png": 256,
    "icon_256x256@2x.png": 512,
    "icon_512x512.png": 512,
    "icon_512x512@2x.png": 1024,
}

for name, size in icns_specs.items():
    out = os.path.join(iconset_dir, name)
    src.resize((size, size), Image.LANCZOS).save(out, format="PNG")
print(f"  {iconset_dir}/ (use 'iconutil -c icns' on macOS)")

print("\nDone!")
