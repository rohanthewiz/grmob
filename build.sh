#!/bin/sh
# Builds the browser build of the interactive tutorial into wasm/:
#   wasm/main.wasm    (the tutorial app + the GrMob runtime, js/wasm)
#   wasm/wasm_exec.js (Go's JS support shim, refreshed from GOROOT so it
#                      always matches the toolchain that produced main.wasm)
#
# The same three files the site workflow (.github/workflows/site.yml) stages
# for GitHub Pages; this script is what it runs, so a local build and the
# deployed one are byte-for-byte the same recipe.
#
# Serve the result with: go run ./serve
set -eu
cd "$(dirname "$0")"

# -trimpath and -s -w for the same reason as any shipped binary: no build
# machine paths in the module, no symbol/DWARF tables. It is a ~40% saving on
# a wasm module and the site is the one place its size is a user-visible cost.
GOOS=js GOARCH=wasm go build -trimpath -ldflags='-s -w' -o wasm/main.wasm ./wasm

goroot=$(go env GOROOT)
rm -f wasm/wasm_exec.js # a prior copy keeps GOROOT's read-only mode; cp can't overwrite it
if [ -f "$goroot/lib/wasm/wasm_exec.js" ]; then # Go >= 1.24
	cp "$goroot/lib/wasm/wasm_exec.js" wasm/
else # Go <= 1.23
	cp "$goroot/misc/wasm/wasm_exec.js" wasm/
fi
chmod u+w wasm/wasm_exec.js

ls -lh wasm/main.wasm
