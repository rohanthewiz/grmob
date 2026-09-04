// Command serve hosts the interactive tutorial's browser build locally.
// Run ./build.sh first, then: go run ./serve
//
// It is a plain static file server over wasm/ — the same files the site
// workflow publishes to GitHub Pages — so what you see here is what the live
// site shows. The one thing a generic server may get wrong is the wasm MIME
// type: WebAssembly.instantiateStreaming refuses a module served as anything
// but application/wasm, and Go's mime table already knows .wasm, so
// http.FileServer is enough. Python's http.server also works; this exists so
// the route needs no second toolchain.
package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dir := flag.String("dir", "wasm", "directory to serve")
	flag.Parse()
	log.Printf("GrMob tutorial on http://localhost%s (serving %s)", *addr, *dir)
	log.Fatal(http.ListenAndServe(*addr, http.FileServer(http.Dir(*dir))))
}
