# The shape a gate's test is, once, for every harness that has a gate.
#
# # What this is for
#
# ios/verify and android/verify each decide, in shell, whether their pass can
# run on this machine: an iPhoneOS SDK for one, a JDK and a Kotlin compiler for
# the other. Both decisions were extracted into a `*_verdict` function taking
# its inputs as arguments (see either gate.sh) precisely so their arms could be
# reached without owning a machine in each state — a gate that skips where it
# should run is a check that silently stops happening, and it is the mutation
# that leaves a pass green.
#
# Both then grew a gate_test.sh to hand those functions every answer, and the
# two scripts came out the same shape with no code in common: a failure counter,
# an `expect` that prefix-matches a verdict, a "these two skips must not be the
# same sentence" comparison, and a FAIL/OK footer. A third harness with a gate
# would have written a third copy, which is the point at which a shape becomes a
# thing worth naming.
#
# # The verdict format these assume
#
# One line, `<action>:<why>` — `run:`/`skip:` for ios, `kotlinc:`/`cache:`/
# `skip:` for android. The action is the decision and the rest is what to tell
# the reader, which is why gate_expect matches a PREFIX: a test that pinned the
# whole line would fail on every reworded explanation and would be deleted the
# second time that happened.
#
# # Sourcing it
#
#	. ../../internal/gateharness/harness.sh
#
# from a script that has already cd'd to its own directory, which both do. The
# names are prefixed `gate_` because this is sourced into a script that has also
# sourced its own gate.sh, and the two must not collide.
#
# The helpers below are themselves only exercised by a gate with a fault in it,
# which is the situation they exist to catch — the same regress the gates
# themselves were in. harness_test.sh beside this file is what reaches them, and
# each run.sh runs it before its own gate_test.sh.

# The count. Every helper adds to it; gate_verdict is what reads it.
gate_fails=0

# gate_expect <what> <want-prefix> <got>
#
# The verdict's action, and nothing else. `want-prefix` is matched against the
# start of `got`, so "skip:" asks only whether the gate skipped and
# "skip:no java" asks which skip it was — both spellings are used, and which one
# a case wants is the case's own statement of how much it is about.
gate_expect() {
    case "$3" in
        "$2"*) ;;
        *) echo "  $1: got '$3', want a '$2' verdict"; gate_fails=$((gate_fails + 1)) ;;
    esac
}

# gate_distinct <what> <a> <b> <why>
#
# Two verdicts that must not be the same sentence.
#
# Both gates have a pair of skips that mean different things — Xcode missing
# against Xcode broken, a missing JDK against a missing Kotlin compiler — and a
# gate that printed one sentence for both would send half its readers to install
# the wrong thing. That is not something gate_expect can ask: each skip passes
# its own prefix test whatever the text says, so the claim is about the PAIR.
#
# `why` is printed on failure and is the case's own words for what the reader
# would be told to do wrongly.
gate_distinct() {
    if [ "$2" = "$3" ]; then
        echo "  $1: both answers are '$2'"
        echo "  $4"
        gate_fails=$((gate_fails + 1))
    fi
}

# gate_verdict <subject> <ok-sentence>
#
# The footer, and the exit status. `subject` names whose gate this was and is
# printed with the count on failure; `ok-sentence` is what a passing gate says
# it established, which is not the same sentence — "android/verify's JVM harness
# gate" is who, and "answers every precondition, in order" is what.
gate_verdict() {
    if [ "$gate_fails" -ne 0 ]; then
        echo "FAIL: $1 ($gate_fails)"
        exit 1
    fi
    echo "OK: $2"
}
