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
