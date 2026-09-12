#!/bin/bash
# Time cold launches of a grmob Android app, over adb.
#
# Usage (from the repository root, with one device or emulator attached):
#   android/device/launch.sh                    3 launches of com.grmob.app
#   android/device/launch.sh 5                  5 launches
#   android/device/launch.sh 3 --frames         also capture the arrival, frame
#                                               by frame, for the one check
#                                               below that `am start -W` cannot
#                                               make of itself
#   android/device/launch.sh 5 --stages         also print, per launch, what the
#                                               app spent inside the bridge, the
#                                               JSON parse and the node build —
#                                               the three stages that scale with
#                                               the screen's tree. See Startup.kt
#
# # What it answers
#
# Whether a change to a screen's tree made the app start faster. The question
# arrived from the other host: the tutorial's contents screen went from 7.4s to
# 1.7s on iOS when it stopped being a Column inside a Scroll and became a
# core.List, because SwiftUI builds a view per node for a scroll's whole content
# and materialises only the visible rows of a LazyVStack. Compose's LazyColumn
# is the same idea, and "should be the same" is an expectation, not a reading.
#
# # The reading it was built for, and why the answer is not iOS's
#
# Four arms, means of five cold launches each, one emulator (sdk_gphone64_arm64,
# 1080x2400), the tutorial app unless stated:
#
#     home = title + progress card only, same binary     2529 ms
#     home = the whole contents as a core.List           5045 ms
#     home = the whole contents as a scrolled Column     6062 ms
#     examples/mobileapp, for scale                      3458 ms
#
# The first arm is the control that makes the rest mean anything: same APK, same
# Go binary, same 49 lessons of data, with only the chapter cards taken off the
# screen. Everything above it is the screen.
#
#     List vs Column          1017 ms   the composition of the rows that are
#                                       not on screen, which is the only thing
#                                       that differs between the two arms
#     List vs near-empty      2516 ms   everything else that scales with the
#                                       tree, which laziness does not touch
#
# So the lazy container wins on Android, in the direction and for the reason
# predicted — and it wins 29% of the screen's cost where on iOS it won all of
# it. **The finding was about SwiftUI, not about tree size**, which is what the
# question was asking.
#
# # What the 2516ms turned out to be
#
# The obvious next move was windowing: teach core.List to send a slice of its
# rows and ask for more as the host scrolls, which is a protocol change. It was
# worth measuring before building, and the measurement said not to build it.
#
# GrMobRuntime times the two calls that scale with the screen and Startup.kt
# prints them (`adb shell setprop log.tag.GrMobStartup DEBUG`). Five cold
# launches of the List arm, this emulator:
#
#     bridge  Go's render + marshal + the gomobile crossing        17 ms
#     parse   org.json turning 423,472 bytes into JSONObjects    1666 ms
#     build   GrMobNode.parse walking those into the node tree    427 ms
#
# Neither Go nor the FFI is in it. It is the parse, and a probe run in the same
# process settled what kind of cost that is: org.json takes ~1.1s on a second
# and third pass over the same string, so it is the parser and not a cold JIT,
# and android.util.JsonReader consuming every token of it takes 0.9-1.4s, so
# swapping to a streaming parser buys nothing. The payload was the lever.
#
# And the payload was mostly nothing. 92.4% of those bytes were core.Style,
# written out field by field for 336 nodes carrying 1,168 non-zero style fields
# between them — about three and a half each, out of fifty-eight. The fix is
# `json:",omitzero"` on core.Style and core.Node; see the note above core.Style.
#
#     home = the whole contents as a core.List, before      4850 ms
#     home = the whole contents as a core.List, after       3530 ms
#
#     bytes on the wire      423,472 -> 53,408      7.9x
#     parse                   1666ms -> 249ms
#     build                    427ms -> 185ms
#     TotalTime               4850ms -> 3530ms      -27% of the whole launch
#
# A later pass took the payload to 51,242: the tags had stopped one level too
# high, so a present Padding still wrote all six of core.EdgeInsets' untagged
# ints and the axis pair was zero in all 77 insets on that screen. Five cold
# launches each way, same emulator, an hour after the arms above:
#
#     bytes on the wire       53,408 -> 51,242      4.0%
#     parse                   201.3ms -> 197.4ms    +-17 / +-31
#     build                   178.4ms -> 180.5ms    +-52 / +-9
#     TotalTime               3464ms -> 3233ms      and not because of this
#
# Nothing in that is a reading. Parse and build together move 1.9ms where the
# spread is 17-52ms, and a 4% cut predicts about 8ms if the parse is linear in
# length — under the floor. The 231ms in the last row is the machine: the
# untagged arm's five runs fall 4000, 3764, 3394, 3106, 3058, which is warming,
# not the tags.
#
# That is the other use of this script, and it is worth saying because it is
# the less obvious one: it sets the size below which a payload change cannot be
# reported as a win here. It is about 8ms, or 4% of this screen.
#
# (4850 rather than 5045 for the before because both arms of THIS A/B were
# measured with the stage clocks compiled in, an hour apart from the four arms
# above and on a busier machine. Compare within a table, not across them.)
#
# So the screen's own cost is now about 1000ms over the near-empty control
# rather than 2516ms, and roughly 450ms of that is still the parse and build.
# Windowing would attack what is left, and what is left is no longer the
# largest thing in the launch — which is exactly what sizing it first was for.
#
# # Why iOS never needed any of this
#
# The same payload, on the simulator: 6ms to parse and build the node tree
# (LiveMapUITests carries that reading). Android's org.json spent 1666ms on the
# identical bytes. Two hundred times is not a device-speed difference, it is a
# parser, and it is why one host's lever was the view layer and the other's was
# the wire.
#
# One emulator, one device shape, and Debug builds of both halves. The A/B is
# sound — every arm ran on the same machine within the same hour — but the
# absolute numbers are an emulator's.
#
# # Why `am start -W` and not a screenshot cadence
#
# The iOS instrument is `simctl io screenshot` on a fixed cadence, taking the
# first frame whose content matches the settled one, because iOS offers nothing
# better. Android does: ActivityTaskManager times the launch itself and reports
#
#     TotalTime: the app's own time — from the intent to the first frame of the
#                app's window, excluding whatever the previous app spent leaving
#     WaitTime:  TotalTime plus the system's own work around it
#
# in milliseconds, with no screenshots to steal CPU from the emulator while the
# thing being measured is running. TotalTime is the number reported below; it is
# the one that excludes the launcher, which is the iOS instrument's first false
# start written into the platform.
#
# # The assumption that makes it the right number, and the check for it
#
# TotalTime ends at the activity's first frame, which is the number we want
# only if that frame carries the screen. There is a real way for it not to, and
# this app has the shape that invites it: Android 12's SplashScreen puts the
# app icon up as a starting window the moment the intent lands, so SOMETHING of
# this app's is on screen within a few hundred milliseconds of a launch that
# takes five seconds. If TotalTime ended there it would be a constant, and the
# A/B below would read zero difference no matter what the tree did.
#
# `--frames` is the check, and it settles the question by comparing against the
# system's own number for the SAME launch rather than by argument: it captures a
# reference frame of the settled screen, then captures the framebuffer as fast
# as adb will go after the intent, printing how much of the reference's ink each
# frame carries — and finally prints what ActivityTaskManager logged for that
# launch. Read it as one question: does the Displayed line land where the ink
# does?
#
#     splash splash splash CONTENT     and Displayed at the arrow
#                          ^ here      TotalTime is the content, which is what
#                                      the A/B needs it to be
#
#     splash splash splash CONTENT     and Displayed back at the splash
#     ^ there                          TotalTime is the starting window, and
#                                      the numbers above mean nothing
#
# It is not on by default because it is a check on the instrument rather than a
# measurement: the captures contend with the emulator for the CPU being timed,
# and by a lot. The same launch that `am start -W` reports at 4.6s alone is
# reported at 9.5s while the frames are being pulled — which is fine for this
# question, since both the ink and the Displayed line are slowed together.
#
# # What runs this
#
# A person, deliberately, like everything else in this directory. See README.md.
set -e

pkg="${PKG:-com.grmob.app}"
activity="${ACTIVITY:-.MainActivity}"
runs="${1:-3}"
# The two optional switches, either order, so neither has to be remembered as
# "the second argument". --frames is the check on the instrument; --stages is
# the attribution inside the app.
frames=""
stages=""
for arg in "$@"; do
  case "$arg" in
    --frames) frames="--frames" ;;
    --stages) stages="--stages" ;;
  esac
done

here="$(cd "$(dirname "$0")" && pwd)"
tmp="${TMPDIR:-/tmp}/grmob-device/launch"
mkdir -p "$tmp"

# A force-stop is what makes the next start COLD: the process is gone and Zygote
# forks a fresh one. `am start -W` prints the LaunchState it actually got, and
# the loop below refuses to average a WARM run in with the cold ones — a warm
# launch skips process creation and class loading entirely and is a different
# measurement wearing the same units.
cold_launch() {
  adb shell am force-stop "$pkg" >/dev/null
  # The system needs a moment to finish tearing the process down; without it the
  # next start occasionally reports LaunchState: WARM and a number half the size.
  sleep 2
  # Only when the stage line is wanted: a cleared buffer is what makes "the
  # first matching line" mean "this launch's". Skipped otherwise so the default
  # run leaves the reader's logcat alone.
  [ -n "$stages" ] && adb logcat -c
  adb shell am start -W -n "$pkg/$activity" 2>&1
}

# GrMobRuntime measures its own mount and Startup.kt prints it, gated on the
# platform's per-tag switch so a library does not narrate every launch of every
# app built on it. Setting the property here rather than telling the reader to
# is the difference between an instrument and a note about one; it survives
# until reboot, which is harmless and is also what lets a later manual run print
# the same line.
if [ -n "$stages" ]; then
  adb shell setprop log.tag.GrMobStartup DEBUG
fi

echo "$runs cold launches of $pkg"

total=0
n=0
for i in $(seq 1 "$runs"); do
  out="$(cold_launch)"
  state="$(printf '%s\n' "$out" | sed -n 's/^LaunchState: //p')"
  ms="$(printf '%s\n' "$out" | sed -n 's/^TotalTime: //p')"
  wait_ms="$(printf '%s\n' "$out" | sed -n 's/^WaitTime: //p')"
  if [ "$state" != "COLD" ]; then
    echo "  run $i: LaunchState $state — not counted (see cold_launch)"
    continue
  fi
  printf '  run %d: %5s ms   (WaitTime %s)\n' "$i" "$ms" "$wait_ms"
  if [ -n "$stages" ]; then
    # The mount finishes before the first frame, so by the time `am start -W`
    # has returned the line is already in the buffer — no sleep needed. A miss
    # means the app was built without the tag enabled or the property did not
    # take, and saying so beats printing nothing.
    line="$(adb logcat -d -s GrMobStartup:D 2>/dev/null | sed -n 's/.*GrMobStartup: //p' | head -1)"
    printf '           %s\n' "${line:-no GrMobStartup line — is this a build with Startup.kt?}"
  fi
  total=$(( total + ms ))
  n=$(( n + 1 ))
done

if [ "$n" -eq 0 ]; then
  echo "no cold launches completed" >&2
  exit 1
fi
printf 'mean of %d cold launches: %d ms\n' "$n" "$(( total / n ))"

[ "$frames" = "--frames" ] || exit 0

# --- the instrument's own check ------------------------------------------
#
# The reference first: the screen as it settles, from the launch that has just
# finished. `screencap` with no -p is the raw framebuffer behind a 16-byte
# header (the same format paint.py reads), which needs no PNG decoder on either
# end.
echo
echo "arrival, frame by frame (--frames):"
ref="$tmp/settled.bin"
adb exec-out screencap > "$ref"

adb shell am force-stop "$pkg" >/dev/null
sleep 2
adb logcat -c
start_ns=$(python3 -c 'import time; print(time.monotonic_ns())')
adb shell am start -n "$pkg/$activity" >/dev/null
# A timestamp per frame, read AFTER the capture returns, so each line is an
# upper bound on when that frame was on screen rather than a claim about when
# it was taken. A pull of ten megabytes over adb is most of a second, which is
# the resolution of this column and the reason it is a shape and not a number.
#
# Enough frames to outlast a contended launch: the pulls roughly double it, so a
# five-second launch has to be watched for fifteen or the column ends before the
# content arrives and says nothing at all.
for i in $(seq 1 "${FRAMES:-20}"); do
  f="$tmp/frame$i.bin"
  adb exec-out screencap > "$f" 2>/dev/null || true
  now_ns=$(python3 -c 'import time; print(time.monotonic_ns())')
  python3 "$here/arrival.py" "$ref" "$f" "$(( (now_ns - start_ns) / 1000000 ))"
done

# The system's own verdict on the launch the column just watched. Both clocks
# start at the intent, and the column's resolution is most of a second, so what
# is being read off the two is which STEP the Displayed line falls in — not an
# agreement to the millisecond, which neither side can offer.
echo
echo "what ActivityTaskManager logged for that same launch:"
adb logcat -d -s ActivityTaskManager:I 2>/dev/null \
  | sed -n "s/.*Displayed $pkg[^:]*: */  Displayed after /p" | tail -1
