#!/bin/sh
# Source-layer conformance check for the Kotlin runtime: does it compile?
#
# # The gap this closes
#
# run.sh executes the two Kotlin files that import nothing. Every other file in
# android/app/src — the renderer, the style solver, the map, both editors, the
# sensors — imports Compose or the Android SDK, and until this script existed
# nothing in this repository compiled any of them. They were held to their
# contracts TEXTUALLY, by mobile/verify searching their source for the calls
# they are supposed to make. Those tests are real checks of the rules and no
# check at all of whether the file parses, resolves, or type-checks.
#
# That is not a theoretical gap. The first run of this script found that
# GrMobCodeEditor.kt called `GrMobNode.isDisabled()`, which Renderer.kt declared
# `private` — file-private, in Kotlin, for a top-level declaration. The Android
# app did not compile, had not compiled since the editor landed, and nothing
# here could have said so.
#
# # What is compiled, in two stages
#
#	com.grmob.runtime   always. The runtime package imports Compose, the Android
#	                    SDK, osmdroid and coil, and imports NOTHING from the
#	                    gomobile-produced .aar — deliberately, which is what
#	                    makes this stage runnable on a checkout that has never
#	                    seen gomobile or the NDK.
#	com.grmob.app       only when android/app/libs/grmob.aar is present. The app
#	                    layer talks to the Go bridge, so its classpath includes
#	                    a binding ../build.sh produces; a checkout without one
#	                    gets the first stage and is told the second was skipped.
#
# The split is the whole reason this is worth having as a separate script rather
# than a gradle task: `:app:compileDebugKotlin` is the second stage and cannot
# be the first.
#
# # Where the classpath comes from
#
# `android/gradlew :app:printVerifyClasspath`, which prints the resolution the
# app is actually built against plus AGP's own platform jars. See that task in
# android/app/build.gradle for why it is not a glob over the gradle cache: this
# machine's cache holds two Compose releases and the wrong one fails the pass on
# the library rather than on the source.
#
# # Where the compiler comes from
#
# The gradle cache, always — never a kotlinc on PATH, which is the opposite of
# what run.sh prefers. The reason is the Compose compiler plugin: it must be the
# compiler's own version, the cache holds a matched pair because the app build
# resolved both, and a kotlinc on PATH is an unrelated release. See gate.sh for
# why the plugin is required rather than optional.
set -e
cd "$(dirname "$0")"

. ./gate.sh
. ./kotlinc.sh

have() { command -v "$1" >/dev/null && echo yes || echo no; }

out="${TMPDIR:-/tmp}/grmob-android-sources"
rm -rf "$out"
mkdir -p "$out"

# The SDK, found the way ../build.sh finds it: an exported ANDROID_HOME wins,
# then the sdk.dir a local.properties may carry (which is what AGP itself
# prefers, and what an Android Studio checkout has instead of the variable),
# then the standard macOS install location. Exported rather than passed, because
# it is AGP below that reads it.
if [ -z "$ANDROID_HOME" ] && [ -f ../local.properties ]; then
  ANDROID_HOME=$(sed -n 's/^sdk\.dir=//p' ../local.properties | tail -1)
fi
[ -n "$ANDROID_HOME" ] || ANDROID_HOME="$HOME/Library/Android/sdk"
export ANDROID_HOME
have_sdk=no
[ -d "$ANDROID_HOME/platforms" ] && have_sdk=yes

# The classpath, from gradle. --offline for the reason the rest of this
# directory is written for: a verify script that reaches the network stops being
# runnable in the position the Android build already occupies.
#
# Failure is captured rather than propagated: `set -e` would turn "this machine
# has no gradle cache" into a red pass, and which of the two it is belongs to
# the gate below.
have_cp=no
if [ "$have_sdk" = yes ] && [ "$(have java)" = yes ]; then
  if ../gradlew --offline -q -p .. :app:printVerifyClasspath > "$out/cp.raw" 2>"$out/cp.err"; then
    grep -q '^CP	' "$out/cp.raw" && have_cp=yes
  fi
fi

# The compiler, and the Compose plugin at its exact version.
have_compiler=no
have_plugin=no
if kotlin_cache_compiler; then
  have_compiler=yes
  COMPOSE_PLUGIN=$(newest_jar \
    org.jetbrains.kotlin/kotlin-compose-compiler-plugin-embeddable "$KOTLIN_VERSION")
  [ -n "$COMPOSE_PLUGIN" ] && have_plugin=yes
fi

verdict="$(kotlin_source_verdict "$(have java)" "$have_sdk" "$have_cp" \
  "$have_compiler" "$have_plugin")"
if [ "${verdict%%:*}" = skip ]; then
  echo "SKIP: the Kotlin source compile (${verdict#*:})"
  [ -s "$out/cp.err" ] && sed 's/^/      /' "$out/cp.err" | head -5
  exit 0
fi

# Turn the resolution into something a compiler can read.
#
# An .aar is a zip with the module's classes in classes.jar; AGP normally
# unpacks them with an artifact transform, and running that transform would mean
# pulling in the rest of the variant for a file that unzip can get in a
# millisecond. Each one goes in a numbered directory because every classes.jar
# has the same name.
mkdir -p "$out/jars"
CP=""
n=0
while IFS='	' read -r kind path; do
  case "$kind:$path" in
    BOOT:*|CP:*.jar) CP="$CP:$path" ;;
    CP:*.aar)
      n=$((n + 1))
      d="$out/jars/$n"
      mkdir -p "$d"
      # An .aar with no classes.jar is legal — a resource-only module — so a
      # failed extraction is skipped rather than fataled.
      if (cd "$d" && unzip -o -q "$path" classes.jar 2>/dev/null); then
        CP="$CP:$d/classes.jar"
      fi
      ;;
  esac
done < "$out/cp.raw"
CP="${CP#:}"

src=../app/src/main/java/com/grmob
stage="com.grmob.runtime"
FILES="$src/runtime/*.kt"
if [ -f ../app/libs/grmob.aar ]; then
  stage="com.grmob.runtime and com.grmob.app"
  FILES="$FILES $src/app/*.kt"
fi

# -jvm-target 17 matches the app's own compileOptions; -no-stdlib because the
# standard library is supplied explicitly, where an embeddable compiler build
# has none beside it. Warnings are off for the same reason run.sh turns them
# off: androidx deprecations are not this repository's to fix, and a pass that
# prints forty of them is a pass nobody reads.
# shellcheck disable=SC2086
java -cp "$KOTLIN_COMPILER_CP" org.jetbrains.kotlin.cli.jvm.K2JVMCompiler \
  $FILES \
  -d "$out/classes" -no-stdlib -jvm-target 17 -nowarn \
  -classpath "$KOTLIN_STDLIB:$CP" \
  -Xplugin="$COMPOSE_PLUGIN"

echo "OK: $stage compile against the classpath gradle resolves, with the Compose compiler plugin"
if [ ! -f ../app/libs/grmob.aar ]; then
  echo "SKIP: com.grmob.app (no android/app/libs/grmob.aar; run android/build.sh to make one)"
fi
