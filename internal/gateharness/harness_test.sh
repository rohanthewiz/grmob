#!/bin/sh
# Every arm of the gate harness, reached without owning a gate that is wrong.
#
# This is the same move one level up. gate.sh exists because a gate's arms could
# only be reached by owning a machine with the fault; harness.sh has the same
# problem one layer further out — its counting arms only run when a gate is
# broken, and a shared helper that silently stopped counting would take both
# harnesses' gate tests green with it. That is strictly worse than the two
# untested copies it replaced, so it is checked here.
#
# Run by both run.sh scripts, before each runs its own gate_test.sh: the file
# they share is checked wherever it is used rather than on a machine somebody
# remembered.
set -e
cd "$(dirname "$0")"

# Not sourced with `.` — the helpers under test manipulate gate_fails and one of
# them exits, so each case runs the harness in a subshell of its own and is
# judged by what that subshell printed and returned.
harness=./harness.sh

bad=0
note() { echo "  $1"; bad=$((bad + 1)); }

# run <script> — the harness plus a fragment, in a subshell, capturing both its
# output and its status.
out=""
status=0
run() {
    out=$(. $harness; eval "$1" 2>&1) && status=0 || status=$?
}

# --- gate_expect ------------------------------------------------------------

run 'gate_expect "a match" "run:" "run:"; echo "fails=$gate_fails"'
case "$out" in
    *"fails=0"*) ;;
    *) note "gate_expect counted a verdict that matches its prefix: $out" ;;
esac

run 'gate_expect "a miss" "run:" "skip:no SDK"; echo "fails=$gate_fails"'
case "$out" in
    *"fails=1"*) ;;
    *) note "gate_expect did not count a verdict that does not match: $out" ;;
esac
case "$out" in
    *"skip:no SDK"*) ;;
    *) note "gate_expect's failure does not print what it got, so a reader is told
  that something is wrong and not what: $out" ;;
esac

# The prefix is a prefix and not an equality: every case in both gate tests
# passes a verdict whose explanation runs on past the action, and a helper that
# compared whole lines would fail on all of them.
run 'gate_expect "a longer verdict" "skip:" "skip:no java; install a JDK"; echo "fails=$gate_fails"'
case "$out" in
    *"fails=0"*) ;;
    *) note "gate_expect matched the action against the whole line: $out" ;;
esac

# And a prefix that is longer than the action still discriminates, which is what
# the two "which skip was it" cases in android/verify's gate test rest on.
run 'gate_expect "the wrong skip" "skip:no java" "skip:no kotlinc"; echo "fails=$gate_fails"'
case "$out" in
    *"fails=1"*) ;;
    *) note "gate_expect accepted the wrong skip when the case asked which one: $out" ;;
esac

# --- gate_distinct ----------------------------------------------------------

run 'gate_distinct "two skips" "skip:a" "skip:b" "why"; echo "fails=$gate_fails"'
case "$out" in
    *"fails=0"*) ;;
    *) note "gate_distinct counted two verdicts that differ: $out" ;;
esac

run 'gate_distinct "two skips" "skip:same" "skip:same" "the reader is sent to the wrong place"; echo "fails=$gate_fails"'
case "$out" in
    *"fails=1"*) ;;
    *) note "gate_distinct did not count two verdicts that are the same sentence: $out" ;;
esac
case "$out" in
    *"the reader is sent to the wrong place"*) ;;
    *) note "gate_distinct's failure does not print the caller's reason: $out" ;;
esac

# --- gate_verdict -----------------------------------------------------------
#
# The footer is the only thing that turns a count into an exit status, so both
# its arms are the ones a run.sh actually depends on.

run 'gate_verdict "somebody'"'"'s gate" "it answers everything"'
if [ "$status" -ne 0 ]; then
    note "gate_verdict exited $status with nothing counted"
fi
case "$out" in
    "OK: it answers everything") ;;
    *) note "a clean gate did not print its own OK sentence: $out" ;;
esac

run 'gate_fails=2; gate_verdict "somebody'"'"'s gate" "it answers everything"'
if [ "$status" -eq 0 ]; then
    note "gate_verdict exited 0 with 2 failures counted, so a broken gate would
  leave the pass green — which is the entire failure mode the gates exist for"
fi
case "$out" in
    *"FAIL: somebody's gate (2)"*) ;;
    *) note "a failing gate did not name its subject and its count: $out" ;;
esac

if [ "$bad" -ne 0 ]; then
    echo "FAIL: the gate harness ($bad)"
    exit 1
fi
echo "OK: the gate harness counts, discriminates and exits"
