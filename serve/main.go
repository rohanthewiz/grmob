// Command serve hosts the interactive tutorial's browser build locally.
//
// Two modes. Plain (`go run ./serve`, after ./build.sh) is a static file
// server over wasm/ — the same files the site workflow publishes to GitHub
// Pages, so what you see here is what the live site shows. The one thing a
// generic server may get wrong is the wasm MIME type: WebAssembly
// .instantiateStreaming refuses a module served as anything but
// application/wasm, and Go's mime table already knows .wasm, so
// http.FileServer is enough. Python's http.server also works; this exists so
// the route needs no second toolchain.
//
// Dev (`go run ./serve -dev`) is the hot-reload loop: it runs ./build.sh
// itself at startup and again whenever a Go file in the module's build graph
// changes, then tells every open page to swap the new main.wasm in without a
// page load. See dev.go for the mechanism and docs/platforms/wasm.md ("Hot
// reload") for the contract with the page.
package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dir := flag.String("dir", "wasm", "directory to serve")
	dev := flag.Bool("dev", false, "watch the module, rebuild on change, and hot-reload open pages")
	flag.Parse()

	var handler http.Handler = http.FileServer(http.Dir(*dir))
	if *dev {
		d, err := newDevServer(*dir, handler)
		if err != nil {
			log.Fatal(err)
		}
		handler = d
		go d.watch()
		log.Printf("GrMob tutorial on http://localhost%s (serving %s, hot reload on)", *addr, *dir)
	} else {
		log.Printf("GrMob tutorial on http://localhost%s (serving %s)", *addr, *dir)
	}
	log.Fatal(http.ListenAndServe(*addr, handler))
}
