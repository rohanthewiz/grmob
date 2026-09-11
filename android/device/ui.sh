#!/bin/bash
# Drive a running grmob Android app by the text on screen, over adb.
#
# Usage (from the repository root, with one device or emulator attached):
#   android/device/ui.sh texts                 every text node, top to bottom
#   android/device/ui.sh bounds "Show Belém"   "l t r b" of the first match
#   android/device/ui.sh tap    "Show Belém"   tap the centre of that node
#   android/device/ui.sh drag   540 1900 540 600    a slow, non-fling drag
#   android/device/ui.sh shot   /tmp/x.png     screenshot
#   android/device/ui.sh raw    /tmp/x.bin     raw framebuffer, for paint.py
#
# # Why by text and not by pixels
#
# Compose publishes every text node's semantics, so `uiautomator dump` carries
# both the string and the node's bounds. Addressing by substring is what makes
# a run repeatable across densities, screen sizes and layout changes — the same
# reason ios/GrMobUITests addresses by accessibility label rather than by
# coordinate. A script that taps (540, 1200) works once, on one emulator.
#
# # The one detail that is not obvious
#
# `adb shell input swipe` with a short duration is a FLING, not a drag. Three
# fast swipes travelled from lesson 4.1 to 7.3 and the run was unrepeatable
# until the duration went up: 900ms stays below Compose's fling threshold and
# moves a predictable amount. That number is in `drag` below and is the reason
# this file exists rather than a line of adb in a session log.
#
# # What runs this
#
# A person, deliberately. Nothing in `go test ./...`, `android/verify/run.sh`
# or any CI touches it — it needs a device, and a harness wired into a suite is
# a third thing to keep true. See README.md in this directory.
set -e

here="$(cd "$(dirname "$0")" && pwd)"
tmp="${TMPDIR:-/tmp}/grmob-device"
mkdir -p "$tmp"

# Pulls the accessibility tree. Written to a temp file rather than piped,
# because both `bounds` and `texts` read it and a dump costs about a second.
dump() {
  adb shell uiautomator dump /sdcard/grmob-ui.xml >/dev/null 2>&1
  adb shell cat /sdcard/grmob-ui.xml > "$tmp/ui.xml" 2>/dev/null
}

case "${1:-}" in
  texts)
    dump
    python3 - "$tmp/ui.xml" <<'PY'
import re, sys
xml = open(sys.argv[1], encoding="utf-8", errors="replace").read()
# In document order, which for a Compose tree is top-to-bottom on screen.
for m in re.finditer(r'text="([^"]+)"', xml):
    t = m.group(1)
    if t.strip():
        print(t)
PY
    ;;

  bounds)
    dump
    python3 - "$tmp/ui.xml" "$2" <<'PY'
import re, sys
xml = open(sys.argv[1], encoding="utf-8", errors="replace").read()
want = sys.argv[2]
# The bounds attribute follows text on the same node, so one pattern spanning
# both is enough and needs no XML parser.
for m in re.finditer(r'text="([^"]*)"[^>]*bounds="\[(\d+),(\d+)\]\[(\d+),(\d+)\]"', xml):
    if want in m.group(1):
        print(m.group(2), m.group(3), m.group(4), m.group(5))
        break
PY
    ;;

  # The layout box of the View an AndroidView wraps — a MapView, a WebView.
  # Compose reports it as a ViewFactoryHolder, and its bounds are the answer
  # paint.py measures the painted pixels against.
  viewbox)
    dump
    python3 - "$tmp/ui.xml" <<'PY'
import re, sys
xml = open(sys.argv[1], encoding="utf-8", errors="replace").read()
for m in re.finditer(r'ViewFactoryHolder"[^>]*bounds="\[(\d+),(\d+)\]\[(\d+),(\d+)\]"', xml):
    print(m.group(1), m.group(2), m.group(3), m.group(4))
PY
    ;;

  tap)
    read -r l t r b <<<"$("$here/ui.sh" bounds "$2")"
    if [ -z "${l:-}" ]; then echo "not found: $2" >&2; exit 1; fi
    adb shell input tap $(( (l + r) / 2 )) $(( (t + b) / 2 ))
    ;;

  # A drag and not a fling: see the header. Four coordinates, x1 y1 x2 y2.
  drag)
    adb shell input swipe "$2" "$3" "$4" "$5" 900
    ;;

  # The whole paint check in one command: find the View's box, grab the
  # framebuffer, measure one against the other. One command rather than three
  # because the three have to pass four numbers between them, and an unquoted
  # "$box" is four arguments in bash and one in zsh — which is a footgun in a
  # README, not a thing to explain in one.
  paint)
    box="$("$here/ui.sh" viewbox | head -1)"
    if [ -z "$box" ]; then
      echo "no AndroidView on screen. Scroll the View into view first — a" >&2
      echo "ViewFactoryHolder that is not laid out is not in the dump." >&2
      exit 1
    fi
    raw="$("$here/ui.sh" raw "$tmp/paint.bin")"
    python3 "$here/paint.py" "$raw $box"
    ;;

  shot)
    adb shell screencap -p /sdcard/grmob.png
    adb pull /sdcard/grmob.png "${2:-$tmp/shot.png}" >/dev/null 2>&1
    echo "${2:-$tmp/shot.png}"
    ;;

  # The framebuffer as RGBA_8888 behind a 16-byte header, which is what
  # paint.py reads. A PNG would need a decoder; this needs none.
  raw)
    adb shell screencap /sdcard/grmob.bin
    adb pull /sdcard/grmob.bin "${2:-$tmp/raw.bin}" >/dev/null 2>&1
    echo "${2:-$tmp/raw.bin}"
    ;;

  *)
    sed -n '2,20p' "$0"
    exit 1
    ;;
esac
