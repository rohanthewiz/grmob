#!/bin/sh
# Data-layer conformance check for the Kotlin runtime: Go generates the case
# tables (gen.go), Kotlin runs the real runtime files against them on a plain
# JVM and compares (Harness.kt).
#
# Two decisions are checked, and both are here for the same reason — a Kotlin
# file that imports nothing can be executed off a device:
#
#   GrMobSelectMenu.kt  how a flat option list becomes a picker menu
#   GrMobProgress.kt    what a core.ValueRange's three numbers amount to
#
# This is the Android analog of ios/verify's selectmenu pass, and it closes the
# gap that file's own doc used to record: GrMobSelectMenu.kt imports nothing
# precisely so a JVM harness could execute it, and until now no such harness
# existed. The Android build's only check was `compileDebugKotlin`, which
# proves the file parses.
#
# # Why not a gradle test source set
#
# The obvious shape — `android/app/src/test`, JUnit, `./gradlew test` — needs
# JUnit resolved through gradle, and this repository's Android build runs
# `--offline` against whatever the local cache happens to hold. A check that
# only runs on a machine that has already downloaded the right test
# dependencies is a check nobody runs. It would also drag the whole AGP
# pipeline in to execute a function that touches no Android API at all.
#
# So this compiles the two files directly, with the Kotlin compiler and the
# standard library. Nothing here is Android, and nothing here is gradle beyond
# the jars its cache already holds for the app build.
#
# # Finding a compiler
#
# In order of preference: a `kotlinc` on PATH, then the compiler jars in the
# gradle cache (kotlinc.sh). If neither is there the pass is SKIPPED rather than
# failed — the same stance ios/verify takes toward the iPhoneOS SDK, and for the
# same reason: the script's promise is that it catches what the machine can
# catch.
#
# # The other half of the pass
#
# This script also runs sources.sh, which asks the prior question about the same
# tree: does the Kotlin that imports Compose and the Android SDK compile? Those
# files cannot be executed off a device and were held to their contracts
# textually; a type-check against the classpath gradle resolves is the strongest
# thing available without one. See that script for the two stages and the
# reasoning, and gate.sh for both gates.
set -e
cd "$(dirname "$0")"

# The JVM harness gate, and its own tests first.
#
# The gate is a function of values (see gate.sh) precisely so its arms can be
# reached without owning a machine that has the fault, and running the tests
# here is what makes that true on every machine this pass runs on rather than on
# one somebody remembered.
#
# The harness gate_test.sh counts with is shared with ios/verify's, so it gets
# the same treatment one layer out: its arms only run when a gate is WRONG, and
# a helper that had stopped counting would take both gate tests green with it.
sh ../../internal/gateharness/harness_test.sh
. ./gate.sh
. ./kotlinc.sh
sh ./gate_test.sh

# The Compose census's source half, reported here rather than skipped in
# silence.
#
# mobile/verify reads two claims about foundation-layout's own arithmetic —
# Modifier.width sets a maximum, and a Row measures an unweighted child against
# what is left — out of the sources jar for the version the BOM resolves. That
# jar is not fetched by any build: `./gradlew :app:fetchComposeLayoutSources`
# puts it in the cache once, and until somebody runs it the check skips.
#
# `go test` prints a skip only under -v, so the check most likely to catch an
# androidx release change was also the one least likely to be noticed missing.
# This runs it by name and says which of the two happened, next to the other
# SKIPs in this pass. It is here rather than after the harness because the
# harness's arms each exit, and because this needs no compiler of any kind.
#
# Not a fetch. Every other thing this pass touches is already in a cache the app
# build filled, and a verify script that reaches the network would stop being
# runnable in the `--offline` position the rest of it is written for. Naming the
# gap is what was missing; closing it is one command, and the message carries it.
if command -v go >/dev/null; then
  census_test=TestTheComposeCensusClaimsAreWhatTheSourceSays
  if census=$( (cd ../.. && go test ./mobile/verify/ -run "^$census_test\$" -v) 2>&1 ); then
    case "$census" in
      *"--- SKIP"*)
        echo "SKIP: the Compose census's source half — foundation-layout's sources are"
        echo "      not cached, so androidx's own arithmetic is unread on this machine."
        echo "      ./gradlew :app:fetchComposeLayoutSources  (once; it is a network call)"
        ;;
      *)
        echo "OK: the Compose census's claims were read out of foundation-layout's sources"
        ;;
    esac
  else
    echo "$census"
    echo "FAIL: the Compose census no longer matches foundation-layout's source"
    exit 1
  fi
else
  echo "SKIP: the Compose census's source half (no go on PATH to run it with)"
fi

# Does the Kotlin that imports Compose and the Android SDK compile at all.
#
# Ahead of the harness below because it is the prior question: the harness asks
# whether two functions compute the right answers, and that is only worth asking
# of a package that resolves. See sources.sh, which carries its own gate and
# skips rather than fails on a machine missing the Android half of the toolchain.
sh ./sources.sh

out="${TMPDIR:-/tmp}/grmob-android-verify"
mkdir -p "$out"

# The generated case table, written to the scratch directory and never into
# the repo — the same arrangement ios/verify's transcript has.
go run . > "$out/Cases.kt"

SRC="../app/src/main/java/com/grmob/runtime/GrMobSelectMenu.kt ../app/src/main/java/com/grmob/runtime/GrMobProgress.kt Harness.kt $out/Cases.kt"

# The decision, once, before either path. See gate.sh: java is asked about
# first because kotlinc is itself a JVM application, so the old order — kotlinc
# first, java afterwards — turned a machine with a kotlinc and no JDK into a
# failing pass rather than a skipped one.
have() { command -v "$1" >/dev/null && echo yes || echo no; }

# Whether the cache can supply a compiler is not known until the jars have been
# looked for, which happens below; the gate is therefore consulted twice, and
# the first call passes "no" for the jars deliberately. That is safe because the
# jars only matter on the path this call cannot take — with a kotlinc on PATH
# the verdict is "kotlinc" whatever they say, and without one the verdict here
# would be a skip that the second call re-decides with the real answer.
verdict="$(jvm_harness_verdict "$(have java)" "$(have kotlinc)" no)"
case "${verdict%%:*}" in
  skip)
    if [ "$(have kotlinc)" = yes ] || [ "$(have java)" != yes ]; then
      echo "SKIP: JVM harness (${verdict#*:})"
      exit 0
    fi
    ;;
  kotlinc)
    # shellcheck disable=SC2086
    kotlinc $SRC -include-runtime -d "$out/harness.jar" -nowarn
    java -jar "$out/harness.jar"
    exit 0
    ;;
esac

# No kotlinc, and a java. Assemble the compiler out of the gradle cache the
# Android build already populates — kotlinc.sh does the finding, because
# sources.sh needs the same six jars and a toolchain lookup written out twice
# proves only that one person made the same choice twice.
# An `if` rather than `kotlin_cache_compiler && jars=yes`: whether `set -e`
# fires on the failing half of an AND-OR list that is itself the last command
# differs between shells, and this script is run by whatever /bin/sh is.
jars=no
if kotlin_cache_compiler; then
  jars=yes
fi

# Now the jars are known, so the gate is asked again with the real answer. The
# java arm has already been settled above and cannot change; what this decides
# is the cache-versus-skip half.
verdict="$(jvm_harness_verdict "$(have java)" no "$jars")"
if [ "${verdict%%:*}" = skip ]; then
  echo "SKIP: JVM harness (${verdict#*:})"
  exit 0
fi

# -no-stdlib because the standard library is supplied explicitly; the compiler
# would otherwise look for one beside its own jar, where an embeddable build has
# none.
# shellcheck disable=SC2086
java -cp "$KOTLIN_COMPILER_CP" org.jetbrains.kotlin.cli.jvm.K2JVMCompiler \
  $SRC -d "$out/classes" -no-stdlib -classpath "$KOTLIN_STDLIB" -nowarn

java -cp "$out/classes:$KOTLIN_STDLIB" com.grmob.runtime.HarnessKt
