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

# --- kotlin_source_verdict --------------------------------------------------
#
# The source pass has five preconditions where the harness has three, and the
# two extra ones are the interesting cases: an Android SDK and a Compose
# compiler plugin are each absent on machines where everything else is present,
# so both arms are ordinary rather than exotic — and neither can be reached by
# running this on a working developer machine, which is the whole reason the
# gate is a function of values.

# java, sdk, classpath, compiler, plugin
gate_expect "everything present runs" "run:" \
    "$(kotlin_source_verdict yes yes yes yes yes)"
gate_expect "no java skips" "skip:no java" \
    "$(kotlin_source_verdict no yes yes yes yes)"
gate_expect "no SDK skips" "skip:no Android SDK" \
    "$(kotlin_source_verdict yes no yes yes yes)"
gate_expect "no classpath skips" "skip:gradle" \
    "$(kotlin_source_verdict yes yes no yes yes)"
gate_expect "no compiler skips" "skip:the gradle cache has no Kotlin compiler" \
    "$(kotlin_source_verdict yes yes yes no yes)"
gate_expect "no Compose plugin skips" "skip:the gradle cache has no Compose" \
    "$(kotlin_source_verdict yes yes yes yes no)"

# The order. Each precondition has to be asked before the one its absence would
# make a guess, and every one of these is a machine on which the LATER answer is
# also "no" — which is exactly when a misordered gate sends the reader to fix
# the wrong thing.
gate_expect "no java outranks no SDK" "skip:no java" \
    "$(kotlin_source_verdict no no no no no)"
gate_expect "no SDK outranks a failed resolution" "skip:no Android SDK" \
    "$(kotlin_source_verdict yes no no no no)"
gate_expect "a failed resolution outranks a missing compiler" "skip:gradle" \
    "$(kotlin_source_verdict yes yes no no no)"
gate_expect "a missing compiler outranks a missing plugin" \
    "skip:the gradle cache has no Kotlin compiler" \
    "$(kotlin_source_verdict yes yes yes no no)"

# The plugin arm is the one that must not quietly become a weaker run. Without
# the Compose compiler plugin a @Composable called from a plain function
# compiles clean, so a gate that answered "run:" here would keep the pass green
# over the single largest class of Compose mistake.
gate_distinct "a missing plugin and a working machine" \
    "$(kotlin_source_verdict yes yes yes yes no)" \
    "$(kotlin_source_verdict yes yes yes yes yes)" \
    "so a machine that cannot check @Composable contexts reports the same
  verdict as one that can"

# And the five skips are five different sentences: each names a different thing
# to install or run, and a gate that reused one would send four readers in five
# to the wrong remedy. Every pair, because the reuse that matters is whichever
# two somebody happens to collapse.
no_java="$(kotlin_source_verdict no yes yes yes yes)"
no_sdk="$(kotlin_source_verdict yes no yes yes yes)"
no_cp="$(kotlin_source_verdict yes yes no yes yes)"
no_kotlin="$(kotlin_source_verdict yes yes yes no yes)"
no_plugin="$(kotlin_source_verdict yes yes yes yes no)"

why="so two machines needing different things are told to fix the same one"
gate_distinct "no java / no SDK"       "$no_java"   "$no_sdk"    "$why"
gate_distinct "no java / no classpath" "$no_java"   "$no_cp"     "$why"
gate_distinct "no java / no compiler"  "$no_java"   "$no_kotlin" "$why"
gate_distinct "no java / no plugin"    "$no_java"   "$no_plugin" "$why"
gate_distinct "no SDK / no classpath"  "$no_sdk"    "$no_cp"     "$why"
gate_distinct "no SDK / no compiler"   "$no_sdk"    "$no_kotlin" "$why"
gate_distinct "no SDK / no plugin"     "$no_sdk"    "$no_plugin" "$why"
gate_distinct "no classpath / no compiler" "$no_cp" "$no_kotlin" "$why"
gate_distinct "no classpath / no plugin"   "$no_cp" "$no_plugin" "$why"
gate_distinct "no compiler / no plugin" "$no_kotlin" "$no_plugin" "$why"

gate_verdict "android/verify's gates" \
    "both gates answer every precondition, in order"
