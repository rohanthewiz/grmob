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
# gradle cache. If neither is there the pass is SKIPPED rather than failed —
# the same stance ios/verify takes toward the iPhoneOS SDK, and for the same
# reason: the script's promise is that it catches what the machine can catch.
set -e
cd "$(dirname "$0")"

# The JVM harness gate, and its own tests first.
#
# The gate is a function of values (see gate.sh) precisely so its arms can be
# reached without owning a machine that has the fault, and running the tests
# here is what makes that true on every machine this pass runs on rather than on
# one somebody remembered.
. ./gate.sh
sh ./gate_test.sh

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

# No kotlinc, and a java. Assemble the compiler out of the gradle cache the Android build
# already populates. Six jars, because kotlin-compiler-embeddable is
# deliberately *not* a fat jar: it expects the standard library, reflection,
# the daemon client, coroutines and JetBrains' own annotations to be supplied
# alongside it. (The last is needed only by the code generator, which stamps
# @NotNull onto every non-nullable parameter it emits — so a hello-world
# compiles without it and anything with a function signature does not.) The
# newest of each is taken, on the reasoning that the cache holds whatever
# versions the app's own resolution pulled and the compiler is
# backward-compatible with older stdlibs.
cache="$HOME/.gradle/caches/modules-2/files-2.1"
newest_jar() {
  # $1 is a group/artifact directory under the cache; jars with a classifier
  # (-sources, -javadoc) are skipped, and the highest version wins.
  find "$cache/$1" -name '*.jar' 2>/dev/null \
    | grep -v -e '-sources\.jar$' -e '-javadoc\.jar$' \
    | sort -V | tail -1
}

KOTLINC_JAR=$(newest_jar org.jetbrains.kotlin/kotlin-compiler-embeddable)
STDLIB=$(newest_jar org.jetbrains.kotlin/kotlin-stdlib)
REFLECT=$(newest_jar org.jetbrains.kotlin/kotlin-reflect)
DAEMON=$(newest_jar org.jetbrains.kotlin/kotlin-daemon-embeddable)
COROUTINES=$(newest_jar org.jetbrains.kotlinx/kotlinx-coroutines-core-jvm)
ANNOTATIONS=$(newest_jar org.jetbrains/annotations)

# Now the jars are known, so the gate is asked again with the real answer. The
# java arm has already been settled above and cannot change; what this decides
# is the cache-versus-skip half.
jars=yes
for jar in "$KOTLINC_JAR" "$STDLIB" "$REFLECT" "$DAEMON" "$COROUTINES" "$ANNOTATIONS"; do
  [ -z "$jar" ] && jars=no
done
verdict="$(jvm_harness_verdict "$(have java)" no "$jars")"
if [ "${verdict%%:*}" = skip ]; then
  echo "SKIP: JVM harness (${verdict#*:})"
  exit 0
fi

CP="$KOTLINC_JAR:$STDLIB:$REFLECT:$DAEMON:$COROUTINES:$ANNOTATIONS"

# -no-stdlib because the standard library is supplied explicitly above; the
# compiler would otherwise look for one beside its own jar, where an
# embeddable build has none.
# shellcheck disable=SC2086
java -cp "$CP" org.jetbrains.kotlin.cli.jvm.K2JVMCompiler \
  $SRC -d "$out/classes" -no-stdlib -classpath "$STDLIB" -nowarn

java -cp "$out/classes:$STDLIB" com.grmob.runtime.HarnessKt
