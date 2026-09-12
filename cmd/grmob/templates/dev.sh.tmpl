#!/bin/sh
# The edit loop: http://localhost:8080 (pass -addr :9000 for another port).
#
# grmob's dev server runs ./build.sh at startup and on every save of a Go file
# the app compiles, then swaps the new main.wasm into open pages without a page
# load. A compile error shows as an overlay until the next good build. Edits to
# wasm/index.html reload the page. App state does not survive a swap (a fresh
# module has a fresh Go heap), so the counter restarts at 0.
#
# It is `go run` of the grmob module in go.mod, so the server and the page it
# talks to are always the same version.
set -eu
cd "$(dirname "$0")"
exec go run github.com/rohanthewiz/grmob/serve -dev "$@"
