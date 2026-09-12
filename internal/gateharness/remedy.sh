# The shape a gate's REMEDY drill is, once, for every gate that names one.
#
# # The hole this closes, which gate.sh names itself
#
# Both verify passes decide in shell whether they can run here, and both were
# extracted into a `*_verdict` function taking its inputs as arguments so that
# every arm could be reached without owning a machine in each state. gate_test.sh
# hands them every combination. What that establishes is that the five sentences
# are DIFFERENT sentences chosen in the right order — and android/verify/gate.sh
# says the rest out loud:
#
#	the table proves the five sentences are DIFFERENT, and nothing proves any
#	of them is true. That is a hole, and the only instrument for it is running
#	the remedy on a machine that has the fault.
#
# It is not hypothetical. Both of that gate's compiler skips once named a gradle
# task that fetches no compiler: a reader who met either skip, ran what it said
# and tried again met the same skip. Nothing anywhere could have caught it,
# because a remedy sentence is the one part of a gate that only runs on a
# machine which cannot run the pass.
#
# # The shape, which is four steps
#
#	strip     make the machine have the fault
#	fault     run the pass; it must skip, with THIS sentence
#	remedy    run what the sentence says to run
#	pass      run the pass again; it must not skip that arm
#
# The first drill of it was the browser pass's ink gate, two sessions before
# this file: strip the precondition, run the remedy, run the pass, expect
# not-a-skip. What is here is that shape with the counting, the matching and the
# footer factored out, the same way harness.sh factored gate_test.sh's.
#
# # Why the pass is run and not the verdict function
#
# gate_test.sh already calls the verdict function; calling it again with
# hand-set arguments would drill nothing this file is about. The subject here is
# the sentence's TRUTH — that a machine in this state really does print it, and
# that doing what it says really does change the answer — and only the pass
# itself, on a machine actually in that state, can say either.
#
# It costs what the pass costs, twice. That is why these are CI jobs rather than
# something `go test ./...` runs: the android drill needs an SDK, a network and
# a cold gradle cache, and the ios one needs an Xcode.
#
# # Sourcing it
#
#	. ../../internal/gateharness/remedy.sh
#
# from a script that has already cd'd to its own directory, as the drills do.
# The names are prefixed `remedy_` for the reason harness.sh's are `gate_`: both
# may be sourced into the same shell.

# The count. Every helper adds to it; remedy_verdict is what reads it.
remedy_fails=0

# The output of the last pass run, so a failure can print what it actually saw.
remedy_output=""

# remedy_run <command...>
#
# Runs a pass and keeps its output, whatever its exit status.
#
# Status ignored on purpose: a pass is expected to SKIP in the faulted step, and
# a skip is an exit 0 in both of these — but a pass that failed for an unrelated
# reason would otherwise abort the drill under `set -e` with no output kept, and
# what a reader needs at that point is the output. The assertions below are
# about what it printed; remedy_ran holds it to having printed anything at all.
remedy_run() {
    remedy_output="$("$@" 2>&1 || true)"
}

# remedy_faulted <what> <sentence>
#
# The stripped machine's answer: the pass must have skipped, and the skip must
# carry this sentence.
#
# Matched as a substring rather than pinned whole, the way gate_expect
# prefix-matches: a drill that pinned the sentence would fail on every rewording
# and would be deleted the second time that happened. What it is holding is that
# the reader of THIS skip is the one being sent to run THIS remedy.
remedy_faulted() {
    case "$remedy_output" in
        *SKIP*"$2"*) ;;
        *)
            echo "  $1: the stripped machine did not skip with that sentence."
            echo "  wanted a SKIP carrying: $2"
            echo "  what the pass printed:"
            echo "$remedy_output" | sed 's/^/    /'
            remedy_fails=$((remedy_fails + 1))
            ;;
    esac
}

# remedy_fixed <what> <sentence>
#
# The remedied machine's answer: the pass must now print this, and must not
# still be skipping the same arm.
#
# Both, because either alone is satisfied by the wrong thing. A pass whose skip
# has gone might have stopped running the arm altogether — which is the silent
# state this whole directory exists to refuse — and a pass that prints the OK
# while also printing the skip is two arms, one of which is still not running.
remedy_fixed() {
    ok=no
    case "$remedy_output" in
        *"$2"*) ok=yes ;;
    esac
    if [ "$ok" != yes ]; then
        echo "  $1: the remedy ran and the pass still does not say it worked."
        echo "  wanted: $2"
        echo "  what the pass printed:"
        echo "$remedy_output" | sed 's/^/    /'
        remedy_fails=$((remedy_fails + 1))
    fi
}

# remedy_gone <what> <sentence>
#
# And the skip itself is gone. Separate from remedy_fixed so that a pass which
# prints both — a second arm skipping for a second reason — is reported as the
# two facts it is.
remedy_gone() {
    case "$remedy_output" in
        *SKIP*"$2"*)
            echo "  $1: the remedy ran and the pass still skips with the same sentence."
            echo "  the sentence: $2"
            echo "  This is the fault the drill exists for: the skip is telling"
            echo "  readers to run something that does not change the answer."
            echo "$remedy_output" | sed 's/^/    /'
            remedy_fails=$((remedy_fails + 1))
            ;;
    esac
}

# remedy_ran <what>
#
# The pass produced output at all. A drill over a pass that printed nothing
# would report every sentence as absent, which is a wrong reading rather than a
# finding — the same reaching arm every walk in this repository carries.
remedy_ran() {
    if [ -z "$remedy_output" ]; then
        echo "  $1: the pass printed nothing. Every sentence below would read as"
        echo "  missing, so the reading is wrong rather than the remedies being."
        remedy_fails=$((remedy_fails + 1))
    fi
}

# remedy_verdict <subject> <ok-sentence>
#
# The footer and the exit status, in harness.sh's shape and for its reasons.
remedy_verdict() {
    if [ "$remedy_fails" -ne 0 ]; then
        echo "FAIL: $1 ($remedy_fails)"
        exit 1
    fi
    echo "OK: $2"
}
