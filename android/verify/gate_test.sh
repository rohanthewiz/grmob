#!/bin/sh
# Every answer jvm_harness_verdict can give, including the order.
#
# The order is the case that could not be stated while these were three inline
# guards, and it is the one that was wrong: kotlinc is a JVM application, so a
# machine with a kotlinc and no java used to run kotlinc and fail under `set -e`
# rather than skip. A missing toolchain turning a pass red is the same class of
# mistake as a missing one turning it silently green, and neither shows up on a
# machine that has everything.
#
# The counting, the prefix match and the FAIL/OK footer are
# internal/gateharness/harness.sh, which ios/verify's gate test sources too:
# this file and that one had grown the same four things independently. What
# stays here is the table, which is the only part that is about the JVM.
set -e
cd "$(dirname "$0")"
. ./gate.sh
. ../../internal/gateharness/harness.sh

# java, kotlinc, jars
gate_expect "everything present prefers kotlinc" "kotlinc:" "$(jvm_harness_verdict yes yes yes)"
gate_expect "no kotlinc falls back to the cache" "cache:" "$(jvm_harness_verdict yes no yes)"
gate_expect "no compiler at all skips" "skip:" "$(jvm_harness_verdict yes no no)"
gate_expect "no java skips" "skip:" "$(jvm_harness_verdict no no no)"

# The order. A kotlinc without a java is the machine the old arrangement got
# wrong: it would have run kotlinc, which needs a JVM, and `set -e` would have
# turned an absent optional toolchain into a failing pass.
gate_expect "a kotlinc with no java still skips" "skip:no java" \
    "$(jvm_harness_verdict no yes yes)"
gate_expect "a full cache with no java still skips" "skip:no java" \
    "$(jvm_harness_verdict no no yes)"

# And the two skips are different sentences: one machine needs a JDK and the
# other needs a compiler, and a gate that said the same thing for both would
# send half its readers to install the wrong thing.
gate_distinct "the two skips" \
    "$(jvm_harness_verdict no no no)" "$(jvm_harness_verdict yes no no)" \
    "so a missing JDK and a missing Kotlin compiler send the reader to the same place"

gate_verdict "android/verify's JVM harness gate" \
    "the JVM harness gate answers every precondition, in order"
