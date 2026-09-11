#!/bin/sh
# Data-layer conformance check: Go generates a bridge transcript from the demo
# app (gen.go), Swift replays it through the real runtime files and compares
# trees (main.swift). Needs only Go and the Xcode Command Line Tools — this is
# the fast feedback loop for TreeStore/parser changes; UI behavior still needs
# a simulator run.
set -e
cd "$(dirname "$0")"

# The app-layer gate, and its own tests first.
#
# The gate is a function of values (see gate.sh) precisely so its arms can be
# reached without owning a machine that has the fault, and running the tests
# here is what makes that true on every machine this pass runs on rather than on
# one somebody remembered.
#
# The harness gate_test.sh counts with is shared with android/verify's, so it
# gets the same treatment one layer out: its arms only run when a gate is
# WRONG, and a helper that had stopped counting would take both gate tests
# green with it.
sh ../../internal/gateharness/harness_test.sh
. ./gate.sh
sh ./gate_test.sh

out="${TMPDIR:-/tmp}/grmob-ios-verify"
mkdir -p "$out"

go run . > "$out/transcript.json"

# The runtime targets iOS 17, whose observation/layout APIs correspond to
# macOS 14 — typecheck-equivalent for everything the harness touches.
swiftc -o "$out/harness" -target arm64-apple-macos14.0 \
  main.swift \
  flex.swift \
  mincontent.swift \
  stack.swift \
  band.swift \
  pin.swift \
  selectmenu.swift \
  ../GrMob/Runtime/GrMobSelectMenu.swift \
  ../GrMob/Runtime/GrMobStack.swift \
  ../GrMob/Runtime/GrMobStackBridge.swift \
  ../GrMob/Runtime/GrMobNode.swift \
  ../GrMob/Runtime/GrMobStyle.swift \
  ../GrMob/Runtime/GrMobFlex.swift \
  ../GrMob/Runtime/GrMobMinContent.swift \
  ../GrMob/Runtime/TreeStore.swift

"$out/harness" "$out/transcript.json"

# The view layer cannot run here — SwiftUI needs a host app — but it can be
# type-checked, and that is worth having on its own: Renderer.swift carries a
# hand-written Layout (GrMobFlexStack) and a deep opaque-type modifier chain,
# both of which fail at compile time in ways no data-layer test would notice.
# Typecheck-only, so nothing is linked and no simulator is involved.
swiftc -typecheck -target arm64-apple-macos14.0 ../GrMob/Runtime/*.swift

echo "OK: view layer type-checks"

# The same files again, this time OPTIMISED and WHOLE-MODULE — which is a
# different question from whether they type-check, and one that had never been
# asked here.
#
# A Debug build compiles file by file and leaves an opaque `some View` declared
# in another file abstract. A Release build substitutes it with its underlying
# type, and the view-modifier chains in this runtime nest deeply enough that
# doing so used to abort the compiler outright:
#
#     Abort: function substOpaqueTypesWithUnderlyingTypes at ...:651
#     Possible non-terminating type substitution detected
#
# So the app could be run, tested and demonstrated for months while being
# impossible to ship — every pass in this script was green, every simulator run
# worked, and `xcodebuild -configuration Release` had simply never been run.
# GrMobBoxModifier is the fix; this line is what keeps it fixed.
#
# It costs about seven seconds and needs nothing xcodebuild needs: -O -wmo on
# the Command Line Tools' own compiler reproduces the abort exactly (verified
# by removing the fix: SIGABRT, 134). The object file is thrown away — it is
# for a different platform than the app's and is of no use; the exit code is
# the whole result.
if ! wmo=$(swiftc -c -O -wmo -target arm64-apple-macos14.0 \
        ../GrMob/Runtime/*.swift -o "$out/wmo.o" 2>&1); then
  echo "$wmo"
  echo "FAIL: the view layer does not survive whole-module optimisation, so the"
  echo "      app cannot be built for Release. See GrMobBoxModifier."
  exit 1
fi

echo "OK: view layer survives whole-module optimisation (the Release build)"

# What the Swift importer makes of gobind's C spellings.
#
# mobile/verify maps every bound Go type onto the Swift type the shell will see,
# and its standing rule is that each reading comes off gobind's source or its
# golden output. That rule could not reach the last step: gobind emits an
# Objective-C header and the shell writes Swift, so the importer is between the
# two — and the types whose Swift name nobody had read were refused with an
# instruction to run `gomobile bind` on a Mac and write the row from what it
# produced.
#
# importer.h declares the C, importer.swift states what Swift is expected to
# import it as, and a typecheck settles it. Nothing is linked: the symbols are
# declared and never defined, which is what a question about declarations wants.
#
# Run through `go test` rather than by calling swiftc here, so there is ONE
# spelling of that command and it is in the package whose tables it settles.
# The reason is the gap it closes: mobile/verify holds two type tables to what
# importer.swift SAYS, and until that test existed the only thing that ever held
# importer.swift to a compiler was this line — so on any machine that is not a
# Mac the pairing was verified, the reading behind it was not, and nothing said
# which. It says now, and a Mac with a Go toolchain settles it in
# `go test ./...` without running this script at all.
#
# GRMOB_IMPORTER=required, because this pass only runs where the toolchain is:
# a skip here would mean swiftc had gone missing between the typechecks above
# and this line, which is a broken machine rather than a machine without Xcode.
# This is the same arrangement android/verify has with the Compose census, one
# notch stricter because that check's subject is a download and this one's is
# the compiler this script has already used four times.
importer_test=TestTheImporterReadingIsSettledByACompiler
if ! importer=$( (cd ../.. && GRMOB_IMPORTER=required \
    go test ./mobile/verify/ -run "^$importer_test\$" -count=1) 2>&1 ); then
  echo "$importer"
  echo "FAIL: the Swift importer disagrees with mobile/verify's type tables"
  exit 1
fi

echo "OK: the Swift importer spells gobind's C the way mobile/verify says"

# The app layer — the system-event sinks, the platform services behind them,
# and the shell that wires the whole thing together — needs two things the
# passes above deliberately do without.
#
# The first is the iPhoneOS SDK: these files lean on iOS-only frameworks that
# the macOS target cannot see (CLLocationManager.startUpdatingHeading,
# AVAudioSession). That needs Xcode rather than just the Command Line Tools,
# so the pass is skipped rather than failed when it is missing — which keeps
# this script's promise that Go and the CLT are enough, while still catching
# the errors on any machine that can catch them.
#
# The second is the `GrMob` module, which GomobileBridge.swift imports and
# which only exists after a `gomobile bind` (ios/build.sh). Requiring a bind
# would cost the harness its whole audience, so instead gomobile_stub.swift is
# compiled as that module: it declares the bound surface and nothing else, so
# `import GrMob` resolves and the three files that were checked by nothing —
# GomobileBridge, GrMobApp and AppLifecycle, including the @main entry point —
# type-check like any others. See that file for what holds it to the Go
# source it stands for.
sdk="$(xcrun --sdk iphoneos --show-sdk-path 2>/dev/null || true)"
sdk_there=no
[ -n "$sdk" ] && [ -d "$sdk" ] && sdk_there=yes
verdict="$(app_layer_verdict "$sdk" "$sdk_there")"
if [ "${verdict%%:*}" = "run" ]; then
  # -emit-module only: nothing is linked, and the .swiftmodule is written to
  # the scratch directory rather than beside the sources so a stale one can
  # never shadow the real framework in an Xcode build.
  swiftc -emit-module -module-name GrMob \
    -emit-module-path "$out/GrMob.swiftmodule" \
    -target arm64-apple-ios17.0 -sdk "$sdk" \
    gomobile_stub.swift

  swiftc -typecheck -target arm64-apple-ios17.0 -sdk "$sdk" -I "$out" \
    ../GrMob/Runtime/*.swift \
    ../GrMob/App/*.swift
  echo "OK: app layer type-checks against the iOS SDK"
else
  echo "SKIP: app layer (${verdict#*:})"
fi
