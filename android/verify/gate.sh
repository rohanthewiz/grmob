# The preconditions android/verify's JVM harness has, as a function of values.
#
# # Why this is not three inline `if`s
#
# It was, and every arm could only be reached by owning a machine with the
# fault. On a developer's Mac the `kotlinc` path ran and the two SKIP arms never
# did; in a bare container the SKIPs ran and the rest did not. A gate's failure
# mode is a swapped or misordered stance, and that is the mutation that leaves
# the pass green.
#
# Same move as wasm/verify/startup.mjs and aria/verify's localCopyGate, in
# shell: the decision takes its inputs as arguments and prints a verdict, so
# gate_test.sh can hand it every combination.
#
# # The ordering, which was the thing three inline guards could not state
#
# The old script asked `command -v kotlinc` FIRST and took that path, and only
# afterwards checked for a `java`. That order is wrong and the wrongness is
# invisible on every machine anyone has run it on: kotlinc is a JVM application,
# so on a machine with a kotlinc and no java it does not skip — it runs kotlinc,
# which fails, and `set -e` turns a missing optional toolchain into a red pass.
# Exactly the stance inversion this extraction exists to make reachable.
#
# So java is decided first, for both paths, and the order is asserted directly
# rather than being a property of how the arms happen to be written.
#
# # The verdicts
#
# One line, `<action>:<why>`:
#
#	kotlinc:   compile with the kotlinc on PATH
#	cache:     assemble the compiler out of the gradle cache
#	skip:<why> this machine cannot answer the question
#
# There is no `fail`. A missing JDK or Kotlin compiler is a fact about the
# MACHINE, and this script's promise is that it catches what this machine can
# catch — the same stance ios/verify takes toward a missing iPhoneOS SDK.

# jvm_harness_verdict <have-java: yes|no> <have-kotlinc: yes|no> <have-jars: yes|no>
jvm_harness_verdict() {
    if [ "$1" != "yes" ]; then
        echo "skip:no java; install a JDK to check it"
        return
    fi
    if [ "$2" = "yes" ]; then
        echo "kotlinc:"
        return
    fi
    if [ "$3" != "yes" ]; then
        echo "skip:no kotlinc, and the gradle cache has no Kotlin compiler; run android/gradlew -p android compileDebugKotlin once to populate it, or install kotlinc"
        return
    fi
    echo "cache:"
}
