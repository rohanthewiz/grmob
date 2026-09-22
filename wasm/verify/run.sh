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
# checks seven keyboard facts, five about paint, seven about layout, one about
# ARIA's value rules and one about the CSSOM — that every bundled palette's
# ControlBorder reaches the
# screen as the hex the contrast census did its arithmetic about, that a real
# comps.Chip still draws that tone, that a sticky band pins, that
# comps.GroupHeader's two inset arrangements lay out identically at every
# offer (overflow included, which is where ios/verify's flex solver says they
# do not), that a real band's tap target spans the band and its control is
# taller than its badge once there are glyphs in both, that a fixed-size
# container squeezes its child along its main axis and lets it spill across,
# that a Row honours core.FlexShrink(0) wherever the pinned child sits — which
# is what puts a browser behind the CSS column of the pin census instead of one
# solver — that a browser resolves a value range the way core.Progress says
# it does, and that it reads a CSS shorthand back the way the table
# cssstyle.mjs was written against says it does. The fifth keyboard fact builds
# the tutorial for js/wasm and pages 4.9's calendar with PageUp and PageDown
# through a live Go render. The sixth, in the same build, presses lesson 2.2's
# button with Control+Alt+K and F6 with nothing focused, and checks that a bare
# K presses nothing. The sixth layout fact boots the same build through the
# site's own index.html at 1280px and checks that the first tree it renders is
# already the two-pane split. The seventh keyboard fact types into lesson 2.3's
# UPPERCASE field mid-text and checks the caret stays with the typing. The
# seventh layout fact scrolls lesson 4.33's message thread: open at the end,
# one older page at the top with the reader's row kept, sends followed only
# from the end. The fourth paint fact switches the site page's colour scheme
# and checks the panes' CSS and the app's palette move together, with no
# flash for a remembered pick. The fifth measures a core.CanvasMirrorsRTL
# canvas reflecting under dir="rtl" and a plain one staying put.
# It skips when there is no Chrome to launch, which keeps the promise above
# intact — see that file for the claims and why no amount of widening dom.mjs
# would settle them.
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
# the palette check that a table of hexes in a .mjs file cannot reach — and so
# are the bands it measures (bandRenders), which are the widget itself rather
# than a model of it.
GRMOB_TRANSCRIPT="$out/transcript.json" node ./browser.mjs

# The screenshot harness's clipping check (wasm/shots/clipped.mjs), against
# pages built to fail. Here rather than in shoot.sh because shoot.sh is run to
# take pictures, rarely, and a check that is only exercised when it is needed
# is the check nobody has seen fire. Skips without a Chrome, like the pass
# above.
node --test --test-reporter=dot ../shots/*_test.mjs
