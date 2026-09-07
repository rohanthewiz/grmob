#!/bin/sh
# Downloads the W3C ARIA specification, which aria/gen reads to generate
# aria/verify/testdata/aria.json and aria/verify's conformance test checks that
# fixture against.
#
# This is the one thing in this repository that needs the network, and it is
# deliberately not on any verification path. run.sh, `go test ./...`, and the
# three platform harnesses all work from the generated fixture and never come
# here — see aria/spec/spec.go, "Nothing here runs during verification".
#
# The download is not committed. It is 1.4MB of HTML whose every fact this
# repository cares about is already in the 5KB fixture it produces, and a copy
# in the tree would be a second statement of the same thing that could drift
# from the first.
#
#   sh aria/fetch.sh          fetch the spec
#   go run ./aria/gen         regenerate the fixture from it
#   go test ./aria/...        check the fixture against it
set -e
cd "$(dirname "$0")"

url="https://www.w3.org/TR/wai-aria-1.2/"
out="spec/testdata/wai-aria-1.2.html"

mkdir -p "$(dirname "$out")"

# --fail so an error page is an error rather than a 500-byte "specification"
# that parses to no roles. Parse has its own floor for that case; failing here
# names the cause.
curl -fsSL --max-time 120 -o "$out.tmp" "$url"
mv "$out.tmp" "$out"

echo "OK: $url -> aria/$out ($(wc -c < "$out" | tr -d ' ') bytes)"
