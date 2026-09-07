#!/bin/sh
# Conformance check for the JavaScript runtime: Go generates patch
# transcripts from real example apps (gen.go), Node replays them through the
# actual wasm/grmob-runtime.js against a minimal DOM (dom.mjs) and compares
# the result with Go's final render, then runs unit tests over the event and
# focus paths.
#
# Needs only Go and Node — no npm, no lockfile, no node_modules, no network.
# This is the fast feedback loop for runtime changes; most of what depends on
# real rendering (layout, whether enterkeyhint relabels a soft keyboard,
# whether focus() opens one) still needs a browser and stays out of scope.
#
# Some of what was in that bucket is not any more: browser.mjs drives a
# headless Chrome over the DevTools protocol at the end of this script and
# checks four keyboard facts, two about paint, two about layout and one about
# ARIA's value rules — that every bundled palette's ControlBorder reaches the
# screen as the hex the contrast census did its arithmetic about, that a real
# components.Chip still draws that tone, that a sticky band pins, that
# components.GroupHeader's two inset arrangements lay out identically at every
# offer (overflow included, which is where ios/verify's flex solver says they
# do not), and that a browser resolves a value range the way core.Progress says
# it does. It skips when there is no Chrome to launch, which keeps the promise
# above intact — see that file for the claims and why no amount of widening
# dom.mjs would settle them.
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

# The browser pass. Last, because it is the slow one (a Chrome launch) and
# because everything above has to hold before a question about the tab order
# is worth asking. It prints its own OK or SKIP.
#
# It gets the transcript too, and needs it: the widget swatches it paints are
# real components rendered by Go (gen.go's widgetCases), which is the half of
# the palette check that a table of hexes in a .mjs file cannot reach.
GRMOB_TRANSCRIPT="$out/transcript.json" node ./browser.mjs
