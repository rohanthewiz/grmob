#!/bin/sh
# Drills ios/verify's one remedy sentence: strip the SDK, meet the skip, put
# the SDK back, meet the pass.
#
# See internal/gateharness/remedy.sh for the shape and for the hole it closes.
# In short: gate_test.sh proves this gate's two skips are different sentences
# chosen in the right order, and nothing proved either of them was TRUE — that a
# machine in that state really prints it, and that the machine it names really
# clears it.
#
# # What "strip" means here, and why it is a shim
#
# The remedy this gate names is "install Xcode", which is not a command. The
# executable form of the same claim is the other direction: take the iPhoneOS
# SDK away from the pass, and it must skip with the sentence that tells a reader
# to install Xcode; give it back, and the app layer must type-check.
#
# Taken away by putting an `xcrun` in front of the real one on PATH, which
# answers `--sdk iphoneos --show-sdk-path` with nothing and delegates everything
# else. That is exactly what a machine with only the Command Line Tools does,
# and it is the one way to reach that state on a machine that has Xcode without
# moving anybody's Xcode.
#
# The shim delegates rather than exiting for every call because run.sh is not
# the only caller in the tree and swiftc itself reaches xcrun: a shim that
# refused everything would strip far more than the precondition, and the pass
# would then skip for a reason the drill did not arrange.
#
# # What this costs
#
# Two full ios/verify passes, which is why it is a CI job of its own rather than
# something run.sh does. The second of the two is the same work the ordinary
# iOS job does; it is repeated here because a drill's two halves have to be the
# same machine in the same run, or the second half is a fact about a different
# one.
set -eu
cd "$(dirname "$0")"

. ../../internal/gateharness/remedy.sh

shim="$(mktemp -d)"
trap 'rm -rf "$shim"' EXIT INT TERM

real_xcrun="$(command -v xcrun || true)"
if [ -z "$real_xcrun" ]; then
    # The drill's own precondition, and it is the one state this cannot
    # arrange: with no xcrun at all there is nothing to shim and no SDK to put
    # back, so the second half would be asserting that a machine without Xcode
    # type-checks against the iOS SDK. A skip rather than a failure, for the
    # reason the gate it drills skips — this is a fact about the machine.
    echo "SKIP: the ios remedy drill (no xcrun on PATH; it needs a Mac with Xcode)"
    exit 0
fi

cat > "$shim/xcrun" <<SHIM
#!/bin/sh
# Answers "where is the iPhoneOS SDK" with nothing, and passes everything else
# through to the real xcrun. See ios/verify/remedy.sh.
for arg in "\$@"; do
    if [ "\$arg" = "--show-sdk-path" ]; then
        case " \$* " in
            *" iphoneos "*) exit 1 ;;
        esac
    fi
done
exec "$real_xcrun" "\$@"
SHIM
chmod +x "$shim/xcrun"

sdk_skip="no iPhoneOS SDK"
app_ok="OK: app layer type-checks against the iOS SDK"

echo "the ios app-layer gate, with the iPhoneOS SDK hidden:"
remedy_run env PATH="$shim:$PATH" sh ./run.sh
remedy_ran "the faulted pass"
remedy_faulted "the missing-SDK skip" "$sdk_skip"

echo "and with it back:"
remedy_run sh ./run.sh
remedy_ran "the remedied pass"
remedy_gone "the missing-SDK skip" "$sdk_skip"
remedy_fixed "the app layer" "$app_ok"

remedy_verdict "ios/verify's remedy drill" \
  "ios/verify skips with the sentence that names Xcode on a machine with no iPhoneOS SDK, and type-checks the app layer on one that has it"
