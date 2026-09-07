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
#
# The counting, the prefix match and the FAIL/OK footer are
# internal/gateharness/harness.sh, which android/verify's gate test sources too:
# this file and that one had grown the same four things independently, and a
# third harness with a gate would have written them a third time. What stays
# here is the table, which is the only part that is about iOS.
set -e
cd "$(dirname "$0")"
. ./gate.sh
. ../../internal/gateharness/harness.sh

gate_expect "an SDK that is there" "run:" "$(app_layer_verdict /some/sdk yes)"
gate_expect "no SDK at all" "skip:" "$(app_layer_verdict "" no)"
gate_expect "a path that is not there" "skip:" "$(app_layer_verdict /some/sdk no)"

# The two skips say different things, which is the whole reason the path and its
# existence are separate arguments: one machine needs Xcode installed and the
# other needs it repaired, and a single combined test told them apart for
# nobody.
gate_distinct "the two skips" \
    "$(app_layer_verdict "" no)" "$(app_layer_verdict /some/sdk no)" \
    "so the reader cannot tell a missing Xcode from a broken one"

# And the "no SDK" answer must not depend on the existence flag: a caller that
# passed an empty path with a stale yes would otherwise be told to repair an
# Xcode it does not have.
gate_expect "no SDK, existence claimed" "skip:no iPhoneOS SDK" "$(app_layer_verdict "" yes)"

gate_verdict "ios/verify's app-layer gate" "the app-layer gate answers every precondition"
