#!/bin/sh
# Builds the browser build of the interactive tutorial into wasm/:
#   wasm/main.wasm    (the tutorial app + the GrMob runtime, js/wasm)
#   wasm/wasm_exec.js (Go's JS support shim, refreshed from GOROOT so it
#                      always matches the toolchain that produced main.wasm)
#   wasm/*.gz, *.br   (precompressed copies of the page's files, for serve)
#
# The site workflow (.github/workflows/site.yml) runs this script and stages
# main.wasm and wasm_exec.js with the tracked host files, so a local build
# and the deployed one are byte-for-byte the same recipe. It does not stage
# the .gz/.br copies: GitHub Pages ignores them and gzips on its own.
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

# Precompressed siblings: name.gz always, name.br when the brotli CLI is
# installed. `go run ./serve` sends one when the browser's Accept-Encoding
# allows it (serve/encoding.go); nothing else reads them. Measured on the
# 13.1 MB module: gzip -9 gives 3.0 MB in ~1 s, brotli -q 9 2.4 MB in
# ~0.9 s, run side by side, so a cached build goes from 0.2 s to 1.3 s.
# Brotli's top quality, 11, gives 2.1 MB but takes ~20 s, so 9 is the trade
# for a script run on every build.
#
# The old siblings are deleted first, whether or not new ones are made, so
# none can outlive the file it was made from. GRMOB_PRECOMPRESS=0 skips
# making them: serve -dev sets it (rebuilds stay fast, and localhost with
# no-store gains nothing), and so does the site workflow.
#
# Each copy is written to a .tmp and renamed, so serve never reads half a
# file. gzip -n leaves the name and time out of the header, so the same
# input gives the same bytes. The two codecs run side by side; `wait PID`
# returns that job's status, which set -e turns into a failed build.
site_files="main.wasm wasm_exec.js grmob-runtime.js index.html"
for f in $site_files; do
	rm -f "wasm/$f.gz" "wasm/$f.br" "wasm/$f.gz.tmp" "wasm/$f.br.tmp"
done
precompress() { # precompress EXT COMMAND... — COMMAND FILE writes to stdout
	ext=$1
	shift
	for f in $site_files; do
		"$@" "wasm/$f" >"wasm/$f.$ext.tmp" || return 1
		mv "wasm/$f.$ext.tmp" "wasm/$f.$ext" || return 1
	done
}
if [ "${GRMOB_PRECOMPRESS:-1}" != 0 ]; then
	precompress gz gzip -9 -n -c &
	gz_pid=$!
	if command -v brotli >/dev/null 2>&1; then
		precompress br brotli -q 9 -c &
		wait $!
	fi
	wait $gz_pid
fi

ls -lh wasm/main.wasm*
