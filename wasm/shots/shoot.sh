#!/bin/sh
# Re-takes the screenshots in docs/images.
#
#   ./shoot.sh                 every shot, in an order that leaves the
#                              composite until after its parts
#   ./shoot.sh todo signup     just those
#   ./shoot.sh --probe todo    drive the same actions and print the action
#                              script's return value instead of writing a
#                              PNG. This is how the DOM is read at all —
#                              the browser-automation tooling refuses this
#                              page's content — and it is how the strings
#                              in internal/shotclaims were checked against
#                              a real render before they were written down.
#
# Everything is built into a temporary directory and served from there.
# Nothing under wasm/ is written to, which is the difference from the first
# version of this harness: that one rewrote wasm/main.go's dot-import with
# sed before each build, and therefore had an opinion about what happens if
# it died between the edit and the restore. This has none, because the app
# is a query parameter — see wasm/shots/host.
#
# Requires a Chrome (or GRMOB_CHROME) and a Node with a global WebSocket,
# which is Node 22 and later. Both are what wasm/verify already needs.
set -eu
cd "$(dirname "$0")"

out=${GRMOB_SHOTS_OUT:-../../docs/images}

# The order shots are taken in when none are named. The composite is last
# because it is a picture of the three before it; see scripts/hero.js.
all="counter todo signup tabs-list tutorial-contents tutorial-lesson hero"

probe=""
if [ "${1:-}" = "--probe" ]; then
	probe=1
	shift
fi

if [ "$#" -eq 0 ]; then
	set -- $all
fi

www=$(mktemp -d)
trap 'rm -rf "$www"' EXIT INT TERM

echo "building the shots host (js/wasm)…"
GOOS=js GOARCH=wasm go build -trimpath -o "$www/main.wasm" ./host

cp index.html hero.html "$www/"
cp ../grmob-runtime.js "$www/"

# wasm_exec.js from the toolchain that produced the module, for the reason
# build.sh copies it rather than tracking one: a shim from a different Go
# release and a module from this one do not agree about the import object.
goroot=$(go env GOROOT)
if [ -f "$goroot/lib/wasm/wasm_exec.js" ]; then # Go >= 1.24
	cp "$goroot/lib/wasm/wasm_exec.js" "$www/"
else # Go <= 1.23
	cp "$goroot/misc/wasm/wasm_exec.js" "$www/"
fi
chmod u+w "$www/wasm_exec.js"

# The finished PNGs, for the composite page to arrange. Copied rather than
# served from docs/images directly so that the server has one root and the
# page has one kind of URL.
mkdir -p "$www/images"
cp "$out"/*.png "$www/images/" 2>/dev/null || true

for name in "$@"; do
	script="scripts/$name.js"
	if [ ! -f "$script" ]; then
		echo "no such shot: $name (scripts/ holds: $(ls scripts | sed 's/\.js$//' | tr '\n' ' '))" >&2
		exit 1
	fi
	if [ -n "$probe" ]; then
		node shot.mjs --dir "$www" --out - "$script"
		continue
	fi
	node shot.mjs --dir "$www" --out "$out/$name.png" "$script"
	# Back into the served copy, so a composite taken later in this same
	# run arranges what was just taken rather than what was there before.
	cp "$out/$name.png" "$www/images/"
done
