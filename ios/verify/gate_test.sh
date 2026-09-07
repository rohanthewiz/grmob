#!/bin/sh
# Every answer app_layer_verdict can give, handed over directly.
#
# The whole reason gate.sh exists: these arms used to be reachable only by
# owning a machine with the fault, so nothing had ever run the SKIP branch on a
# Mac with Xcode or the OK branch on a machine without it. A gate that skips
# where it should run is a check that silently stops happening, and it is the
# mutation that leaves the pass green.
#
# Run by run.sh before the pass itself, so the gate is exercised on every
# machine the pass runs on rather than on a machine somebody remembered to test.
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

expect "an SDK that is there" "run:" "$(app_layer_verdict /some/sdk yes)"
expect "no SDK at all" "skip:" "$(app_layer_verdict "" no)"
expect "a path that is not there" "skip:" "$(app_layer_verdict /some/sdk no)"

# The two skips say different things, which is the whole reason the path and its
# existence are separate arguments: one machine needs Xcode installed and the
# other needs it repaired, and a single combined test told them apart for
# nobody.
missing="$(app_layer_verdict "" no)"
stale="$(app_layer_verdict /some/sdk no)"
if [ "$missing" = "$stale" ]; then
    echo "  the two skips are the same sentence, so the reader cannot tell a"
    echo "  missing Xcode from a broken one"
    fails=$((fails + 1))
fi

# And the "no SDK" answer must not depend on the existence flag: a caller that
# passed an empty path with a stale yes would otherwise be told to repair an
# Xcode it does not have.
expect "no SDK, existence claimed" "skip:no iPhoneOS SDK" "$(app_layer_verdict "" yes)"

if [ "$fails" -ne 0 ]; then
    echo "FAIL: ios/verify's app-layer gate ($fails)"
    exit 1
fi
echo "OK: the app-layer gate answers every precondition"
