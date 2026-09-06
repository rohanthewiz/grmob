#!/bin/sh
# Data-layer conformance check: Go generates a bridge transcript from the demo
# app (gen.go), Swift replays it through the real runtime files and compares
# trees (main.swift). Needs only Go and the Xcode Command Line Tools — this is
# the fast feedback loop for TreeStore/parser changes; UI behavior still needs
# a simulator run.
set -e
cd "$(dirname "$0")"

out="${TMPDIR:-/tmp}/grmob-ios-verify"
mkdir -p "$out"

go run . > "$out/transcript.json"

# The runtime targets iOS 17, whose observation/layout APIs correspond to
# macOS 14 — typecheck-equivalent for everything the harness touches.
swiftc -o "$out/harness" -target arm64-apple-macos14.0 \
  main.swift \
  flex.swift \
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

# The app layer — the system-event sinks and the platform services behind them
# — was checked by nothing at all until the compass landed, because its files
# lean on iOS-only frameworks that the macOS target above cannot see:
# CLLocationManager.startUpdatingHeading and AVAudioSession simply do not
# exist there.
#
# So this pass targets iOS proper, which needs the iPhoneOS SDK — Xcode, not
# just the Command Line Tools. It is skipped rather than failed when that is
# missing, which keeps this script's promise (Go and the CLT are enough) while
# still catching the errors on any machine that can catch them.
#
# GomobileBridge.swift, GrMobApp.swift and AppLifecycle.swift are left out:
# they import the generated Mobile.xcframework, which only exists after a
# gomobile bind, and a harness that required a build step would not run here
# at all.
sdk="$(xcrun --sdk iphoneos --show-sdk-path 2>/dev/null || true)"
if [ -n "$sdk" ] && [ -d "$sdk" ]; then
  swiftc -typecheck -target arm64-apple-ios17.0 -sdk "$sdk" \
    ../GrMob/Runtime/*.swift \
    ../GrMob/App/AudioPlayer.swift \
    ../GrMob/App/HeadingSensor.swift \
    ../GrMob/App/SystemEvents.swift
  echo "OK: app layer type-checks against the iOS SDK"
else
  echo "SKIP: app layer (no iPhoneOS SDK; install Xcode to check it)"
fi
