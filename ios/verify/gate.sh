# The preconditions ios/verify's app-layer check has, as a function of values.
#
# # Why this is not an inline `if`
#
# It was one, and the arm that skips could only be reached by owning a machine
# with the fault. On a Mac with Xcode the SKIP branch never ran; on a Linux box
# the OK branch never did, and neither machine ever ran both. A gate's failure
# mode is a swapped stance — a skip where a failure belongs, or the reverse —
# and that is precisely the mutation that leaves a pass green.
#
# wasm/verify/startup.mjs made the same move for the browser pass and
# aria/verify's localCopyGate before it. The shape is the same in shell: the
# decision takes its inputs as arguments and prints a verdict, so gate_test.sh
# can hand it every answer without arranging a machine that has the fault.
#
# # The verdicts
#
# One line, `<action>:<why>`. `run` for a machine that can answer the question,
# `skip` for one that cannot. There is no `fail` here and that is the stance:
# an iPhoneOS SDK is a fact about the MACHINE, and this script's promise is that
# it catches what this machine can catch — a pass that fails on a laptop without
# Xcode is a pass people learn to ignore.

# app_layer_verdict <sdk-path> <sdk-path-exists: yes|no>
#
# The path and its existence are separate arguments rather than one test,
# because they are two different states with two different remedies and the
# combined `[ -n "$sdk" ] && [ -d "$sdk" ]` told them apart for nobody:
#
#   no path      xcrun found no iphoneos SDK. Xcode is not installed, or only
#                the command line tools are.
#   a path that  xcrun named an SDK that is not there, which is what a moved or
#   is not there half-removed Xcode leaves behind. Same skip, and a reader who
#                is told which one they have knows whether to install Xcode or
#                to repair it.
app_layer_verdict() {
    if [ -z "$1" ]; then
        echo "skip:no iPhoneOS SDK (xcrun found none; install Xcode to check it)"
        return
    fi
    if [ "$2" != "yes" ]; then
        echo "skip:xcrun names an iPhoneOS SDK at $1 and there is nothing there (a moved or partial Xcode; repair it to check the app layer)"
        return
    fi
    echo "run:"
}
