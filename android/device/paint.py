"""Does a View paint outside its own layout box?

    android/device/ui.sh paint             # the whole thing, in one command

or by hand, against a box from somewhere else:

    android/device/ui.sh raw /tmp/f.bin
    android/device/ui.sh viewbox                 # -> l t r b
    python3 android/device/paint.py /tmp/f.bin 121 1544 959 2227

Prints, for rows and columns just inside and just outside each edge, how many
sampled pixels differ from the page's own background colour. A View that stays
inside its box is page-coloured everywhere outside and painted right up to the
boundary inside.

# Why this exists

A Compose modifier chain can size a View correctly and put no bound at all on
where it paints: Modifier.width/height set the layout size and nothing else,
and Compose does not clip a child's drawing unless a modifier says to. osmdroid
over-draws deliberately, so its tiles are ready before a scroll reaches them,
and the result was core.MapView painting 389dp of tiles through a 260dp slot
and over the captions laid out beneath it.

Nothing in `go test ./...`, `mobile/verify` or `assembleDebug` can see that. It
is not a fact about the tree, the styles or the calls; it is a fact about
pixels, and the only instrument that reads pixels is a screenshot. This is that
instrument, and it is what turned "the map looks right now" into "0 of 838
sampled pixels outside the box, on all four edges".

# Why the framebuffer and not a PNG

`adb shell screencap` without -p writes RGBA_8888 behind a 16-byte header
(width, height, format, colourspace), so this reads pixels with no image
library at all. The PNG path needs Pillow, which is not installable on a
managed Python without a virtualenv — which is a dependency this does not need
to have.

# Why the ground colour is measured and not assumed

The page's background comes from the Go theme, which an app can change and a
dark mode does change. The strip well clear of the View is sampled and its
commonest colour taken as the ground, so the comparison is against whatever
this app actually draws rather than against white.
"""

import struct
import sys
from collections import Counter

# How far a channel may differ from the ground and still count as the ground.
# Generous enough to absorb a shadow's gradient and the antialiasing along a
# card's rounded edge; far below the contrast of a map tile.
GROUND_TOLERANCE = 12

# Fraction of a sampled line that has to be off-ground before the line counts
# as painted. Not 1.0, because a map's own content includes light land and a
# card's corner is round.
PAINTED_FRACTION = 0.5


def main() -> int:
    # The four numbers may arrive as four arguments or as one whitespace
    # separated string, because `ui.sh viewbox` prints them on one line and the
    # two common shells disagree about what that becomes: bash splits an
    # unquoted "$box" into four words and zsh passes it as one. Accepting both
    # is cheaper than a README that works in one shell.
    args = " ".join(sys.argv[1:]).split()
    if len(args) != 5:
        print(__doc__)
        print(f"want a framebuffer path and four bounds; got {len(args)} value(s)")
        return 2
    path = args[0]
    try:
        left, top, right, bottom = (int(v) for v in args[1:5])
    except ValueError:
        print(__doc__)
        print(f"the four bounds must be integers; got {args[1:5]}")
        return 2

    data = open(path, "rb").read()
    width, height, _fmt, _cs = struct.unpack_from("<IIII", data, 0)
    if len(data) - 16 != width * height * 4:
        print(f"not a 16-byte-header RGBA framebuffer: {len(data)} bytes "
              f"for {width}x{height}")
        return 1
    px = memoryview(data)[16:]

    def rgb(x: int, y: int):
        i = (y * width + x) * 4
        return px[i], px[i + 1], px[i + 2]

    def is_ground(x: int, y: int, ground) -> bool:
        return max(abs(a - b) for a, b in zip(rgb(x, y), ground)) <= GROUND_TOLERANCE

    # The page's own colour, taken well above the View. 60px because a shadow
    # or a card edge can reach a dozen; every 7th pixel because the mode of a
    # strip does not need every sample.
    probe_y = max(top - 60, 0)
    ground = Counter(rgb(x, probe_y) for x in range(left, right, 7)).most_common(1)[0][0]

    span = right - left
    print(f"View box  x {left}..{right}  y {top}..{bottom}   "
          f"({bottom - top}px)")
    print(f"page ground rgb{ground}, sampling {span}px of width, "
          f"ground probed at y={probe_y}\n")

    rows = [
        ("12px above top", top - 12), ("6px above top", top - 6),
        ("just inside top", top + 3), ("just inside bottom", bottom - 4),
        ("6px below bottom", bottom + 6), ("12px below bottom", bottom + 12),
        ("24px below bottom", bottom + 24),
    ]
    outside_paint = 0
    for label, y in rows:
        if not (0 <= y < height):
            continue
        n = sum(1 for x in range(left, right) if not is_ground(x, y, ground))
        painted = n > span * PAINTED_FRACTION
        if painted and ("above" in label or "below" in label):
            outside_paint += 1
        print(f"  y={y:5d}  {label:<20s} {n:5d}/{span} off-ground  "
              f"{'PAINTED' if painted else 'page'}")

    print()
    # The vertical edges, sampled clear of the horizontal ones so a rounded
    # corner is not mistaken for a leak.
    tall = range(top + 40, bottom - 40)
    total = len(tall)
    cols = [
        ("12px left of edge", left - 12), ("just inside left", left + 3),
        ("just inside right", right - 4), ("12px right of edge", right + 12),
    ]
    for label, x in cols:
        if not (0 <= x < width):
            continue
        n = sum(1 for y in tall if not is_ground(x, y, ground))
        painted = n > total * PAINTED_FRACTION
        if painted and "of edge" in label:
            outside_paint += 1
        print(f"  x={x:5d}  {label:<20s} {n:5d}/{total} off-ground  "
              f"{'PAINTED' if painted else 'page'}")

    print()
    if outside_paint:
        print(f"LEAK: {outside_paint} line(s) outside the box are painted. The View "
              f"is drawing past its own layout bounds — see GrMobMapView.kt's "
              f"mapClip for the shape of the fix.")
        return 1
    print("CLEAN: every line outside the box is page-coloured.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
