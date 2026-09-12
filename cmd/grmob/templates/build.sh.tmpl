#!/bin/sh
# Builds the browser target into wasm/:
#
#   wasm/main.wasm         the app + the GrMob engine, compiled for js/wasm
#   wasm/wasm_exec.js      Go's JS shim, from the toolchain that built main.wasm
#   wasm/grmob-runtime.js  the DOM renderer, from the grmob module in go.mod
#   wasm/camera.js         the CameraView bridge, from the same module
#
# The three .js files are copied on every build rather than kept in this
# repository, because each has to match something else exactly: the shim
# matches the compiler, the runtime matches the Go side's patch format. A
# copy that outlives a Go or grmob upgrade fails quietly (styles dropped,
# events ignored) rather than loudly, so they are never edited or committed —
# fix bugs upstream in grmob and bump go.mod.
#
# ./dev.sh runs this script on every save; run it by hand for a one-off build.
set -eu
cd "$(dirname "$0")"

# -trimpath and -s -w: no build-machine paths or symbol tables in a module
# the browser downloads, which is roughly 40% smaller.
GOOS=js GOARCH=wasm go build -trimpath -ldflags='-s -w' -o wasm/main.wasm ./wasm

# sync copies a file only when its content differs. A plain cp would give the
# runtime JS a new mtime on every build, and the dev server treats a changed
# .js in wasm/ as a page edit — so every hot swap would be followed by a full
# page reload that throws away the app's state.
sync() {
	if ! cmp -s "$1" "$2"; then
		rm -f "$2" # a previous copy keeps the read-only mode of its source
		cp "$1" "$2"
		chmod u+w "$2"
	fi
}

goroot=$(go env GOROOT)
if [ -f "$goroot/lib/wasm/wasm_exec.js" ]; then # Go >= 1.24
	sync "$goroot/lib/wasm/wasm_exec.js" wasm/wasm_exec.js
else
	sync "$goroot/misc/wasm/wasm_exec.js" wasm/wasm_exec.js
fi

grmob=$(go list -m -f '{{.Dir}}' github.com/rohanthewiz/grmob)
sync "$grmob/wasm/grmob-runtime.js" wasm/grmob-runtime.js
sync "$grmob/wasm/camera.js" wasm/camera.js

ls -lh wasm/main.wasm
