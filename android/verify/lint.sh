#!/bin/sh
# Android lint over the app module, as a pass/fail: FAIL on any lint *error*,
# warnings reported by count and left alone.
#
# # Why errors only
#
# Lint's warnings here are mostly about the world moving (NewerVersionAvailable,
# GradleDependency, OldTargetApi) or style (UseKtx). None of them is a defect in
# what the app does, and a pass that fails whenever a library ships a release
# would be switched off within a week. Lint's errors are the other kind: an API
# called below its level (NewApi), a manifest that hides the app from devices
# (PermissionImpliesUnsupportedChromeOsHardware). Those are what this gates on,
# and it is gradle's own `abortOnError` default that draws the line, so there
# is no severity table to keep in step here.
#
# # Why it was not here before
#
# Until 2026-09-19 lintDebug failed on four errors nobody had fixed (three
# RestrictedApi on MainActivity.dispatchKeyEvent, one camera uses-feature), and
# 19 Haptics false positives before that. A gate that is red on arrival
# teaches everyone to ignore it. It is clean now, which is the precondition.
#
# # Skip, not fail, when gradle cannot run
#
# The same stance as sources.sh: the build runs `--offline` against whatever
# the local cache holds, and a machine without the SDK, a JDK or the Go-built
# grmob.aar cannot lint the app at all. Those skip, and say why. What FAILs is
# only lint itself reporting an error, which gradle announces with a fixed
# sentence this script looks for; any other gradle failure is a skip that
# prints gradle's complaint.
set -e
cd "$(dirname "$0")"

. ./gate.sh

have() { command -v "$1" >/dev/null && echo yes || echo no; }

out="${TMPDIR:-/tmp}/grmob-android-lint"
mkdir -p "$out"

if [ -z "$ANDROID_HOME" ] && [ -f ../local.properties ]; then
  ANDROID_HOME=$(sed -n 's/^sdk\.dir=//p' ../local.properties | tail -1)
fi
[ -n "$ANDROID_HOME" ] || ANDROID_HOME="$HOME/Library/Android/sdk"
export ANDROID_HOME

if [ "$(have java)" != yes ]; then
  skipped "Android lint (no java on PATH)"
  exit 0
fi
if [ ! -d "$ANDROID_HOME/platforms" ]; then
  skipped "Android lint (no Android SDK at $ANDROID_HOME)"
  exit 0
fi
# The app module depends on the gomobile-built aar; without it the module does
# not resolve and lint never starts. ../build.sh produces it.
if [ ! -f ../app/libs/grmob.aar ]; then
  skipped "Android lint (no app/libs/grmob.aar — run android/build.sh once)"
  exit 0
fi

if ../gradlew --offline -q -p .. :app:lintDebug > "$out/lint.log" 2>&1; then
  report=../app/build/reports/lint-results-debug.txt
  warnings=""
  if [ -f "$report" ]; then
    # The text report's last line is "N errors, M warnings" (or "No issues
    # found."); quoted as-is so the count is lint's, not a re-derivation.
    warnings=" ($(tail -1 "$report"))"
  fi
  echo "OK: Android lint found no errors$warnings"
  exit 0
fi

if grep -q "Lint found errors" "$out/lint.log"; then
  # The errors themselves, which gradle's log does not list.
  report=../app/build/reports/lint-results-debug.txt
  [ -f "$report" ] && grep -B1 -A3 ': Error:' "$report" | head -60
  echo "FAIL: Android lint reported errors (full report: app/build/reports/lint-results-debug.html)"
  exit 1
fi

sed 's/^/      /' "$out/lint.log" | head -8
skipped "Android lint (gradle could not run it offline; see above)"
exit 0
