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
  ../GrMob/Runtime/TreeStore.swift

"$out/harness" "$out/transcript.json"

# The view layer cannot run here — SwiftUI needs a host app — but it can be
# type-checked, and that is worth having on its own: Renderer.swift carries a
# hand-written Layout (GrMobFlexStack) and a deep opaque-type modifier chain,
# both of which fail at compile time in ways no data-layer test would notice.
# Typecheck-only, so nothing is linked and no simulator is involved.
swiftc -typecheck -target arm64-apple-macos14.0 ../GrMob/Runtime/*.swift

echo "OK: view layer type-checks"

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
# import it as, and this settles it. Nothing is linked: the symbols are declared
# and never defined, which is what a question about declarations wants.
# Typecheck-only against the macOS target, so this needs no more than the
# Command Line Tools the passes above already need.
swiftc -typecheck -target arm64-apple-macos14.0 \
  -import-objc-header importer.h importer.swift

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
