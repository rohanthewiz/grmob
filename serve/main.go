// Command serve hosts the interactive tutorial's browser build locally.
//
// Two modes. Plain (`go run ./serve`, after ./build.sh) is a static file
// server over wasm/ — the same files the site workflow publishes to GitHub
// Pages, so what you see here is what the live site shows. The one thing a
// generic server may get wrong is the wasm MIME type: WebAssembly
// .instantiateStreaming refuses a module served as anything but
// application/wasm. RWeb's file helper (rweb.FileWithModTime) maps .wasm to
// exactly that, so no Python http.server or other second toolchain is needed
// to put the page on localhost.
//
// Dev (`go run ./serve -dev`) is the hot-reload loop: it runs ./build.sh
// itself at startup and again whenever a Go file in the module's build graph
// changes, then tells every open page to swap the new main.wasm in without a
// page load. See dev.go for the mechanism and docs/platforms/wasm.md ("Hot
// reload") for the contract with the page.
//
// The HTTP side is RWeb (github.com/rohanthewiz/rweb): a small Go server with
// its own router and a channel-driven SSE sender, which is all either mode
// asks of it. Routes, both modes:
//
//	GET       /__dev/events     SSE stream         (dev only, dev.go)
//	GET, HEAD /__dev/client.js  hot-reload client  (dev only, dev.go)
//	GET, HEAD /  /index.html    the host page      (dev: with the client injected)
//	GET, HEAD /*path            any other file under -dir
package main

import (
	"errors"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/rohanthewiz/rweb"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dir := flag.String("dir", "wasm", "directory to serve")
	dev := flag.Bool("dev", false, "watch the module, rebuild on change, and hot-reload open pages")
	flag.Parse()

	files, err := newStaticFiles(*dir)
	if err != nil {
		log.Fatal(err)
	}
	s := rweb.NewServer(rweb.ServerOptions{Address: *addr})

	if *dev {
		d, err := newDevServer(*dir)
		if err != nil {
			log.Fatal(err)
		}
		d.routes(s, files)
		go d.watch()
		// "GrMob app", not "tutorial": apps made by `grmob new` run this same
		// server from their own module (see cmd/grmob and its dev.sh).
		log.Printf("GrMob app on http://localhost%s (serving %s, hot reload on)", *addr, *dir)
	} else {
		getAndHead(s, "/", files.serveIndex)
		getAndHead(s, "/*path", files.serve)
		log.Printf("GrMob app on http://localhost%s (serving %s)", *addr, *dir)
	}
	log.Fatal(s.Run())
}

// getAndHead registers a file route for HEAD as well as GET, as
// http.FileServer answered both (curl -I, link checkers). The handler need
// not know which it got: RWeb sends a HEAD response's headers — Content-Type,
// Content-Length, Last-Modified — and drops its body.
func getAndHead(s *rweb.Server, route string, h rweb.Handler) {
	s.Get(route, h)
	s.Head(route, h)
}

// staticFiles serves the files under one directory, in the way the
// http.FileServer this replaced did for the paths a GrMob page uses: a file
// by its path, a directory by its index.html, a conditional GET answered
// with 304 from the modification time.
//
// rweb.Server.StaticFiles is not used because it cannot own the whole URL
// space: it refuses "/" as its prefix (the page and everything it loads sit
// at the root) and it has no directory-index rule, so "/" would never reach
// index.html. The piece of it worth keeping — the Content-Type table that
// knows application/wasm — is rweb.FileWithModTime, which this calls.
//
// Containment is os.Root's job rather than string checks on the path: every
// open is resolved inside the served directory by the OS-level API, which
// rejects "..", absolute paths and symlinks that lead out of the root, so a
// crafted /../../etc/passwd has nowhere to go. It also means -dir may be
// absolute or relative with no difference in meaning.
type staticFiles struct {
	root *os.Root
}

func newStaticFiles(dir string) (*staticFiles, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &staticFiles{root: root}, nil
}

// serveIndex answers "/" (in dev, dev.go registers its injecting variant
// instead). It has to be a route of its own: RWeb's "/*path" wildcard needs
// at least one character after the slash, so a bare "/" never reaches serve.
func (f *staticFiles) serveIndex(ctx rweb.Context) error {
	return f.serveFile(ctx, "index.html")
}

// serve answers every path the more specific routes did not claim.
func (f *staticFiles) serve(ctx rweb.Context) error {
	// The wildcard parameter arrives percent-encoded ("%20" for a space);
	// the file system wants the decoded name. A malformed escape is a 404,
	// not a 500: it names no file.
	name, err := url.PathUnescape(ctx.Request().Param("path"))
	if err != nil {
		return notFound(ctx)
	}
	return f.serveFile(ctx, name)
}

// serveFile writes one file under the root. name is URL-shaped (forward
// slashes, possibly with a trailing slash or empty for the root) and is
// cleaned to a root-relative path before os.Root sees it; os.Root is still
// the authority on containment, the cleaning is only so "a//b/" and "/" name
// what a browser means by them.
func (f *staticFiles) serveFile(ctx rweb.Context, name string) error {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "" {
		name = "."
	}
	info, err := f.root.Stat(name)
	if err != nil {
		return notFound(ctx)
	}
	// A directory is served by its index.html, the convention every static
	// host (GitHub Pages included) follows, and one level only: an
	// index.html that is itself a directory is a 404.
	if info.IsDir() {
		name = path.Join(name, "index.html")
		if info, err = f.root.Stat(name); err != nil || info.IsDir() {
			return notFound(ctx)
		}
	}

	// Conditional GET. Last-Modified has one-second resolution, so compare at
	// that grain, as http.FileServer does. That shares its blind spot too: a
	// rewrite within the same second as the copy the browser holds reads as
	// unchanged. Harmless for plain mode (one build per ./build.sh run), and
	// dev mode never gets here with the header: its no-store keeps browsers
	// from revalidating at all.
	if ims := ctx.Request().Header("If-Modified-Since"); ims != "" {
		if t, err := http.ParseTime(ims); err == nil && !info.ModTime().Truncate(time.Second).After(t) {
			ctx.SetStatus(http.StatusNotModified)
			return nil
		}
	}

	body, err := f.root.ReadFile(name)
	if err != nil {
		// Stat succeeded a moment ago; a vanished file (build.sh rewriting
		// wasm_exec.js, say) is still just "not there".
		if errors.Is(err, fs.ErrNotExist) {
			return notFound(ctx)
		}
		return err
	}
	return rweb.FileWithModTime(ctx, path.Base(name), body, info.ModTime())
}

func notFound(ctx rweb.Context) error {
	return ctx.SetStatus(http.StatusNotFound).WriteString("404 page not found\n")
}
