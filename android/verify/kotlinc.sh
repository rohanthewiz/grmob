# Finding a Kotlin compiler in the gradle cache, for the two passes that need one.
#
# run.sh compiles the import-free runtime files and runs them on a JVM;
# sources.sh compiles everything that imports Compose or the Android SDK. Both
# have to assemble a compiler out of the jars the app build has already
# downloaded, and before this file existed both would have carried their own
# copy of the same find-and-sort. The rule this repository applies to fixtures
# applies to toolchain discovery too: a thing written out twice proves only that
# one person made the same choice twice.
#
# Nothing here runs a compiler or decides whether one should be run. That is
# gate.sh's job; this file only answers "where is it".

# newest_jar <group/artifact> [version]
#
# The path of one artifact's jar in the local gradle cache, or "" if the cache
# has none.
#
# Jars carrying a classifier (-sources, -javadoc) are skipped: they hold text,
# not classes, and sort next to the real one. With no version the highest wins,
# on the reasoning that the cache holds whatever versions the app's own
# resolution pulled and a Kotlin compiler is backward-compatible with older
# standard libraries.
#
# # When to pass a version, and why it is not the default
#
# Newest-wins is right for the compiler and wrong for anything the compiled code
# is checked AGAINST — see android/app/build.gradle's printVerifyClasspath, where
# a cache holding two Compose releases makes newest-wins fail the pass on the
# library instead of the source. The one caller here that passes a version is
# sources.sh asking for the Compose compiler plugin, which has to be the
# compiler's own version and not merely a recent one.
newest_jar() {
    cache="${GRADLE_USER_HOME:-$HOME/.gradle}/caches/modules-2/files-2.1"
    dir="$cache/$1"
    [ -n "$2" ] && dir="$dir/$2"
    find "$dir" -name '*.jar' 2>/dev/null \
        | grep -v -e '-sources\.jar$' -e '-javadoc\.jar$' \
        | sort -V | tail -1
}

# kotlin_cache_compiler
#
# Sets KOTLIN_COMPILER_CP (what to put on `java -cp` to run K2JVMCompiler),
# KOTLIN_STDLIB (the standard library, which the compiler needs told about
# separately) and KOTLIN_VERSION. Returns 1 and leaves them empty if the cache
# is missing any piece.
#
# Six jars, because kotlin-compiler-embeddable is deliberately *not* a fat jar:
# it expects the standard library, reflection, the daemon client, coroutines and
# JetBrains' own annotations to be supplied alongside it. (The last is needed
# only by the code generator, which stamps @NotNull onto every non-nullable
# parameter it emits — so a hello-world compiles without it and anything with a
# function signature does not.)
kotlin_cache_compiler() {
    KOTLIN_COMPILER_CP=""
    KOTLIN_STDLIB=""
    KOTLIN_VERSION=""

    kc=$(newest_jar org.jetbrains.kotlin/kotlin-compiler-embeddable)
    sl=$(newest_jar org.jetbrains.kotlin/kotlin-stdlib)
    rf=$(newest_jar org.jetbrains.kotlin/kotlin-reflect)
    dm=$(newest_jar org.jetbrains.kotlin/kotlin-daemon-embeddable)
    co=$(newest_jar org.jetbrains.kotlinx/kotlinx-coroutines-core-jvm)
    an=$(newest_jar org.jetbrains/annotations)

    for jar in "$kc" "$sl" "$rf" "$dm" "$co" "$an"; do
        [ -z "$jar" ] && return 1
    done

    KOTLIN_COMPILER_CP="$kc:$sl:$rf:$dm:$co:$an"
    KOTLIN_STDLIB="$sl"
    # The version, read off the compiler jar's own path rather than asked of it:
    # the cache lays artifacts out as .../<artifact>/<version>/<hash>/<file>, so
    # the version is two directories up, and running the compiler to ask it
    # would cost a JVM start for a string that is already on disk.
    KOTLIN_VERSION=$(basename "$(dirname "$(dirname "$kc")")")
    return 0
}
