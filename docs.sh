#!/bin/sh
# Serve the documentation site — the narrative pages under docs/ and the
# generated reference under docs/api/ — with gkdocs.
#
#   ./docs.sh              http://localhost:8000/
#   PORT=9000 ./docs.sh
#
# The server is a nested module (cmd/docs, see its main.go for why), so it has
# to be built from inside its own directory: `go run ./cmd/docs` from here
# would have the root module try to build a package it does not contain.
set -e
cd "$(dirname "$0")/cmd/docs"
exec go run . "$@"
