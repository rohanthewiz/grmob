#!/bin/sh
# Conformance check for the JavaScript runtime: Go generates patch
# transcripts from real example apps (gen.go), Node replays them through the
# actual wasm/grmob-runtime.js against a minimal DOM (dom.mjs) and compares
# the result with Go's final render, then runs unit tests over the event and
# focus paths.
#
# Needs only Go and Node — no npm, no lockfile, no node_modules, no network.
# This is the fast feedback loop for runtime changes; anything that depends on
# real rendering (layout, whether enterkeyhint relabels a soft keyboard,
# whether focus() opens one) still needs a browser.
#
# The .mjs extension is deliberate: it makes these files ES modules on every
# Node from 12 onward, where a bare .js would depend on the module-detection
# behavior of the version in use.
set -e
cd "$(dirname "$0")"

out="${TMPDIR:-/tmp}/grmob-wasm-verify"
mkdir -p "$out"

go run . > "$out/transcript.json"

# Globbed rather than `node --test .`, whose directory-discovery rules have
# shifted between Node releases; the glob means the same thing everywhere.
# The dot reporter keeps the output as short as ios/verify's; pass
# --test-reporter=spec for the test names when one of them fails.
GRMOB_TRANSCRIPT="$out/transcript.json" \
  node --test --test-reporter=dot ./*_test.mjs

# Only reached when node exits 0, because of set -e above.
echo "OK: grmob-runtime.js replays Go's transcripts and passes its unit tests"
