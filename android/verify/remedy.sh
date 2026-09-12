#!/bin/sh
# Drills android/verify's two gradle-cache remedy sentences: start from a cache
# that has never been filled, meet each skip in turn, run the command it names,
# and require the next answer to have moved on.
#
# See internal/gateharness/remedy.sh for the shape and for the hole it closes.
# This is the pass those sentences were WRONG in — gate.sh records it: both
# compiler skips once named a gradle task that fetches no compiler, so a reader
# who met either skip, ran what it said and tried again met the same skip. The
# table in gate_test.sh could not have caught it, and cannot now; only running
# the remedy on a machine that has the fault can.
#
# # Why a scratch GRADLE_USER_HOME
#
# It is the fault, arranged. Both sentences are about a cache this build has
# never filled, and the only honest way to be in that state on a machine that
# has filled it is a cache of one's own. Nothing else here is stripped: the
# SDK, the JDK and the network are the machine's, because those are the things
# the sentences assume a reader HAS.
#
# It is also what makes this a CI job rather than something run.sh does. A cold
# gradle home downloads a Gradle distribution and a compile classpath before it
# can answer anything, which is minutes and a network — the two costs
# android/verify's `--offline` stance exists to keep out of the pass itself.
#
# # The order, which is the gate's own
#
# kotlin_source_verdict asks java, then the SDK, then the classpath, then the
# compiler, then the Compose plugin. A cold cache fails the classpath question
# first, so the drill meets that sentence, runs its command, and is then failed
# by the NEXT question — which is the compiler, with the second sentence. Two
# remedies, in the order a reader on a cold machine actually meets them, with
# the second only reachable because the first one worked.
#
# # What this does not drill
#
# The JVM harness gate's compiler skip, which names the same
# `:app:fetchKotlinCompiler`. Its command is therefore drilled here and its
# PASS is not: run.sh's harness arm is a separate script with a separate
# toolchain question in front of it (a kotlinc on PATH takes a different
# branch entirely), and a drill that ran it would be arranging two machines at
# once. What is established here is that the command those sentences name does
# what they say it does.
set -eu
cd "$(dirname "$0")"

. ../../internal/gateharness/remedy.sh

# The drill's own preconditions, and they are the things the sentences assume
# the reader has rather than the things they tell the reader to get. Skips
# rather than failures, for the reason the gate being drilled skips: these are
# facts about the machine.
if ! command -v java > /dev/null 2>&1; then
    echo "SKIP: the android remedy drill (no java; the remedies are gradle tasks)"
    exit 0
fi
sdk="${ANDROID_HOME:-}"
if [ -z "$sdk" ] && [ -f ../local.properties ]; then
    sdk=$(sed -n 's/^sdk\.dir=//p' ../local.properties | tail -1)
fi
if [ -z "$sdk" ] || [ ! -d "$sdk/platforms" ]; then
    echo "SKIP: the android remedy drill (no Android SDK; export ANDROID_HOME)"
    exit 0
fi

cold="$(mktemp -d)"
trap 'rm -rf "$cold"' EXIT INT TERM
export GRADLE_USER_HOME="$cold"

classpath_skip="gradle did not resolve a compile classpath"
compiler_skip="the gradle cache has no Kotlin compiler"
sources_ok="compile against the classpath gradle resolves, with the Compose compiler plugin"

echo "the Kotlin source pass, against a gradle cache that has never been filled:"
remedy_run sh ./sources.sh
remedy_ran "the faulted pass"
remedy_faulted "the unresolved-classpath skip" "$classpath_skip"

echo "running what that sentence says — :app:printVerifyClasspath:"
../gradlew --no-daemon -q -p .. :app:printVerifyClasspath > /dev/null

echo "the same pass again:"
remedy_run sh ./sources.sh
remedy_ran "the half-remedied pass"
remedy_gone "the unresolved-classpath skip" "$classpath_skip"
# The next question in the gate's own order, which is only reachable because
# the first remedy worked. A cold cache has no compiler either, so meeting
# this sentence here is the drill's evidence that the classpath one cleared.
remedy_faulted "the missing-compiler skip" "$compiler_skip"

echo "running what THAT sentence says — :app:fetchKotlinCompiler:"
../gradlew --no-daemon -q -p .. :app:fetchKotlinCompiler

echo "and the pass once more:"
remedy_run sh ./sources.sh
remedy_ran "the remedied pass"
remedy_gone "the missing-compiler skip" "$compiler_skip"
remedy_fixed "the Kotlin source compile" "$sources_ok"

remedy_verdict "android/verify's remedy drill" \
  "both of android/verify's gradle-cache skips name a command that clears them, run on a cache that has never been filled"
