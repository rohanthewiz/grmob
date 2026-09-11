"""Has the screen arrived yet? One line per captured frame.

    python3 android/device/arrival.py settled.bin frame3.bin 940

Prints the elapsed milliseconds the caller passed and the share of the
reference screen's INK that the frame reproduces — a reference frame of the
same screen, settled, is captured first. It is called in a loop by
`android/device/launch.sh --frames`.

# What it is for, and what it is not for

It is not a measurement. `am start -W` measures the launch, in milliseconds the
system itself recorded, with nothing contending for the CPU while it does so.
This is the check on the *assumption* under that number: TotalTime ends at the
first frame of the app's window, which is the launch only if that first frame
already carries the screen.

     10%  10%  10%  99%  99%      one step: the first frame of the window is
                                  the screen, and TotalTime is the whole launch
     10%  10%  55%  70%  99%      stages: something drew before the content,
                                  and TotalTime ends at the first of them

So the shape of the column is the whole output. The frames are captured by an
adb pull each, which costs about a second and steals CPU from the emulator
being timed — a launch measured at 4.6s alone was reported at 9.5s while these
were running — which is why this runs on request and never alongside the
numbers it vouches for.

# Why the reference's ink, and not either simpler thing

Two simpler metrics were tried first and both are defeated by the same fact:
a page of text on white is mostly white.

    ink per frame               A blank window and the settled contents screen
                                differ by 4.7 percentage points, because 4.7%
                                is all the ink there is. The launcher wallpaper
                                scored 35% and everything else scored 4.7% — a
                                reading that separates the launcher from the app
                                and nothing from anything.

    whole-frame match           A blank white window reproduces every white
                                pixel of the settled screen, which is 95% of
                                them. It scored 89% against 100%, on a column
                                whose whole job is to tell those two apart.

So the denominator is the reference's ink: the sampled pixels that differ from
the reference's own commonest colour, which for this app is the page. A blank
window reproduces none of them and the settled screen reproduces all of them,
and the reading between the two is a page that is partly drawn — which is the
distinction the column exists to draw. It is also what the iOS instrument does,
one platform over: it takes the first frame whose title BAND matches the
settled one, a band being that instrument's way of pointing at the ink.

# What is excluded, and why

The status bar. It carries a clock, which changes between the reference frame
and the run, and signal and battery glyphs the system draws before the app
exists — so leaving it in would put a floor under every frame including the
blank ones, which is exactly the distinction being drawn. The bottom navigation
strip is excluded on the same grounds: it is the system's, it is ink, and it is
there before the app is.
"""

import struct
import sys

# Sample every Nth pixel on both axes. A 1080x2400 frame is 2.6M pixels and the
# question is a proportion, which a 1-in-64 sample answers to well under a
# percent — and the loop runs a dozen times per launch.
STRIDE = 8

# How far a channel may differ and still count as the same pixel. The tolerance
# paint.py uses, for the same reason: it absorbs antialiasing along a rounded
# card edge and a shadow's gradient, both of which can land a step differently
# between two renders of the same tree.
TOLERANCE = 12


def frame(path: str):
    """(width, height, pixels) from screencap's raw format, or None."""
    data = open(path, "rb").read()
    if len(data) < 16:
        return None
    width, height, _fmt, _cs = struct.unpack_from("<IIII", data, 0)
    if len(data) - 16 != width * height * 4:
        return None
    return width, height, memoryview(data)[16:]


def main() -> int:
    if len(sys.argv) != 4:
        print(__doc__)
        return 2
    ref_path, path, elapsed_ms = sys.argv[1], sys.argv[2], sys.argv[3]

    ref = frame(ref_path)
    if ref is None:
        print(f"{ref_path} is not a screencap framebuffer; capture the settled "
              f"screen first")
        return 1
    got = frame(path)
    if got is None:
        # A capture that raced the launch and came back empty or short.
        # Reported rather than skipped: a gap in the column is itself
        # information about how fast the frames are arriving.
        print(f"{elapsed_ms:>6} ms   (no frame)")
        return 0
    if got[:2] != ref[:2]:
        print(f"{elapsed_ms:>6} ms   (a {got[0]}x{got[1]} frame against a "
              f"{ref[0]}x{ref[1]} reference — did the device rotate?)")
        return 0

    width, height, px = got
    rpx = ref[2]
    # The system's own furniture, top and bottom; see the module docstring.
    top = height // 20
    bottom = height - height // 20

    # The reference's ground is its commonest colour, measured rather than
    # assumed to be white: the page's background comes from the Go theme and a
    # dark mode changes it. Same argument as paint.py, one file over.
    counts: dict = {}
    for y in range(top, bottom, STRIDE):
        row = (y * width) * 4
        for x in range(0, width, STRIDE):
            i = row + x * 4
            c = (rpx[i], rpx[i + 1], rpx[i + 2])
            counts[c] = counts.get(c, 0) + 1
    ground = max(counts, key=counts.get)

    same = 0
    total = 0
    for y in range(top, bottom, STRIDE):
        row = (y * width) * 4
        for x in range(0, width, STRIDE):
            i = row + x * 4
            # Only the reference's ink is asked about. A pixel the settled
            # screen leaves at the page colour says nothing about whether the
            # page is there — it is the same colour in a blank window.
            if all(abs(rpx[i + k] - ground[k]) <= TOLERANCE for k in range(3)):
                continue
            total += 1
            if (abs(px[i] - rpx[i]) <= TOLERANCE
                    and abs(px[i + 1] - rpx[i + 1]) <= TOLERANCE
                    and abs(px[i + 2] - rpx[i + 2]) <= TOLERANCE):
                same += 1

    if total == 0:
        print(f"{elapsed_ms:>6} ms   (the reference frame carries no ink)")
        return 1

    share = same / total
    bar = "#" * round(share * 50)
    print(f"{elapsed_ms:>6} ms   {share*100:5.1f}%  {bar}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
