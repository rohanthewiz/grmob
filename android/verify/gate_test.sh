#!/bin/sh
# Every answer jvm_harness_verdict can give, including the order.
#
# The order is the case that could not be stated while these were three inline
# guards, and it is the one that was wrong: kotlinc is a JVM application, so a
# machine with a kotlinc and no java used to run kotlinc and fail under `set -e`
# rather than skip. A missing toolchain turning a pass red is the same class of
# mistake as a missing one turning it silently green, and neither shows up on a
# machine that has everything.
set -e
cd "$(dirname "$0")"
. ./gate.sh

fails=0
expect() { # expect <what> <want-prefix> <got>
    case "$3" in
        "$2"*) ;;
        *) echo "  $1: got '$3', want a '$2' verdict"; fails=$((fails + 1)) ;;
    esac
}

# java, kotlinc, jars
expect "everything present prefers kotlinc" "kotlinc:" "$(jvm_harness_verdict yes yes yes)"
expect "no kotlinc falls back to the cache" "cache:" "$(jvm_harness_verdict yes no yes)"
expect "no compiler at all skips" "skip:" "$(jvm_harness_verdict yes no no)"
expect "no java skips" "skip:" "$(jvm_harness_verdict no no no)"

# The order. A kotlinc without a java is the machine the old arrangement got
# wrong: it would have run kotlinc, which needs a JVM, and `set -e` would have
# turned an absent optional toolchain into a failing pass.
expect "a kotlinc with no java still skips" "skip:no java" \
    "$(jvm_harness_verdict no yes yes)"
expect "a full cache with no java still skips" "skip:no java" \
    "$(jvm_harness_verdict no no yes)"

# And the two skips are different sentences: one machine needs a JDK and the
# other needs a compiler, and a gate that said the same thing for both would
# send half its readers to install the wrong thing.
nojava="$(jvm_harness_verdict no no no)"
nokotlin="$(jvm_harness_verdict yes no no)"
if [ "$nojava" = "$nokotlin" ]; then
    echo "  the two skips are the same sentence, so a missing JDK and a missing"
    echo "  Kotlin compiler send the reader to the same place"
    fails=$((fails + 1))
fi

if [ "$fails" -ne 0 ]; then
    echo "FAIL: android/verify's JVM harness gate ($fails)"
    exit 1
fi
echo "OK: the JVM harness gate answers every precondition, in order"
