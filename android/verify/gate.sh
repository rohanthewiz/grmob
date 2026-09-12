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

# The preconditions android/verify's Kotlin SOURCE pass has, as a function of
# values. See sources.sh for what the pass does; this decides whether it can run.
#
# # Why this gate is separate from the one above
#
# They answer about different machines. jvm_harness_verdict is about executing
# two import-free Kotlin files on a JVM, and its whole toolchain is a compiler.
# This one is about type-checking Kotlin that imports Compose, the Android SDK,
# osmdroid and coil, which needs three more things — an Android SDK, a resolved
# compile classpath, and the Compose compiler plugin — each of which can be
# absent on a machine where the other pass runs perfectly.
#
# # The order, and why it is this order
#
# Each precondition is asked before the one whose answer its absence would make
# a guess:
#
#	java         gradle and the Kotlin compiler are both JVM applications. With
#	             no JDK neither can be run, so "the classpath did not resolve"
#	             and "no compiler in the cache" would both be true and neither
#	             would be the reader's problem.
#	sdk          the gradle task prints the platform jars from AGP's own
#	             bootClasspath, and AGP fails the whole task when it cannot find
#	             an SDK. Asked first, a missing SDK is reported as a missing SDK;
#	             asked after, it arrives as a gradle failure whose message is
#	             four lines of stack.
#	classpath    whether that gradle call actually produced one.
#	compiler     and only then the Kotlin side, which is the one precondition
#	             that is about the machine's Kotlin cache rather than its
#	             Android setup.
#
# # Why a missing Compose plugin is a skip and not a weaker run
#
# Without the plugin `@Composable` is an ordinary annotation with no meaning to
# the compiler, and this file compiles clean:
#
#	fun Bad() { Text("nope") }
#
# With it, two errors — a @Composable invoked outside a @Composable context is
# the single largest class of Compose-specific mistake, and the plugin is the
# only thing that knows the rule. A compile without it is a different, weaker
# check wearing the same OK, which is exactly the shape this repository's guards
# exist to refuse. So its absence is named rather than absorbed.
#
# The plugin also has to be the compiler's own version, which is why sources.sh
# asks the cache for it by version rather than taking the newest: they are
# published in lockstep and a mismatched pair fails inside the compiler.

# kotlin_source_verdict <have-java> <have-sdk> <have-classpath> <have-compiler> <have-plugin>
kotlin_source_verdict() {
    if [ "$1" != "yes" ]; then
        echo "skip:no java; install a JDK to check it"
        return
    fi
    if [ "$2" != "yes" ]; then
        echo "skip:no Android SDK; export ANDROID_HOME, or write sdk.dir into android/local.properties"
        return
    fi
    if [ "$3" != "yes" ]; then
        echo "skip:gradle did not resolve a compile classpath; run android/gradlew -p android :app:dependencies to see why"
        return
    fi
    if [ "$4" != "yes" ]; then
        echo "skip:the gradle cache has no Kotlin compiler; run android/gradlew -p android :app:printVerifyClasspath once to populate it"
        return
    fi
    if [ "$5" != "yes" ]; then
        echo "skip:the gradle cache has no Compose compiler plugin at the Kotlin compiler's version, and without it a @Composable called from a plain function compiles clean"
        return
    fi
    echo "run:"
}
