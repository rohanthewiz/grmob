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
# question was asking. The 2516ms is the more interesting half: a Go screen
# reaches Compose as JSON, and this one is 423,472 bytes of it (12,382 without
# the cards; both from a test in examples/tutorial). Go builds and serialises
# all of it in under a millisecond on a desktop. Everything that happens to
# those bytes afterwards — the JNI crossing, the parse, the node tree Kotlin
# builds from it — is paid for all 49 rows whether or not Compose composes them,
# because the whole tree crosses either way.
#
# That is the lever Android has and iOS did not need: send fewer nodes, not
# compose fewer views. Nothing here does that yet.
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
frames="${2:-}"

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
  adb shell am start -W -n "$pkg/$activity" 2>&1
}

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
