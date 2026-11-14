package main

import (
	"io/fs"
	"strconv"
	"strings"
)

// Precompressed siblings.
//
// The tutorial's main.wasm is ~13 MB; gzip -9 makes it ~3.0 MB and brotli
// ~2.1–2.4 MB (quality 11 vs 9). This server never compresses on the fly:
// gzip -9 of the module costs about a second of CPU per request, and brotli
// at a useful quality far more. Instead build.sh writes the compressed copies
// once, next to each file:
//
//	wasm/main.wasm      the file itself (always served when nothing better fits)
//	wasm/main.wasm.br   brotli, when the brotli CLI was on PATH at build time
//	wasm/main.wasm.gz   gzip, always
//
// and serveFile picks the best sibling the request's Accept-Encoding allows.
// The response keeps the original's identity — its Content-Type (so
// main.wasm.br still goes out as application/wasm, which
// WebAssembly.instantiateStreaming insists on) and its Last-Modified — and
// adds Content-Encoding, so the browser decodes it before the page sees a
// byte.
//
// GitHub Pages, the deployed host, does not look for siblings: it gzips
// application/wasm itself (checked 2026-11-13 — main.wasm went out with
// Content-Encoding: gzip at 3.13 MB, and br was never offered), so the site
// workflow neither builds nor stages these files. They are for this server,
// which is mostly a localhost one, where they matter when a phone on the LAN
// opens the page.
//
// Order is preference: brotli first because it is smaller for the same
// file. Browsers advertise br only to secure contexts. Checked 2026-11-13
// with headless Chrome 157 booting the tutorial from this server: to
// localhost and 127.0.0.1 it sent "gzip, deflate, br, zstd" and got the .br;
// to the machine's LAN address over plain http it sent "gzip, deflate" and
// got the .gz. A phone on the LAN is that second case, which is why gzip is
// the one build.sh always writes.
var precompressed = []struct {
	coding string // the Accept-Encoding / Content-Encoding token
	ext    string // the sibling's suffix on disk
}{
	{"br", ".br"},
	{"gzip", ".gz"},
}

// pickSibling returns the precompressed sibling of name that the request
// may be sent, and its content coding; ("", "") means send name itself.
//
// orig is name's FileInfo. A sibling older than its original is skipped
// rather than trusted: index.html and grmob-runtime.js are edited by hand,
// and their .gz from the last build.sh run is stale until the next one.
// build.sh deletes and rewrites every sibling after writing the originals,
// so a fresh sibling is never older. Equal times pass (a sibling written in
// the same second on a one-second filesystem); that leaves the same blind
// spot Last-Modified has, one rewrite within the second, and is the price of
// not reading both files to compare them.
func (f *staticFiles) pickSibling(name string, orig fs.FileInfo, acceptEncoding string) (sibling, coding string) {
	// No header at all: RFC 9110 allows any coding then, but every client
	// that can decode one sends the header, and the ones that send none
	// (curl without --compressed, a bare http.Get with its transport's
	// gzip turned off) expect the plain bytes.
	if acceptEncoding == "" {
		return "", ""
	}
	for _, p := range precompressed {
		if !acceptsEncoding(acceptEncoding, p.coding) {
			continue
		}
		info, err := f.root.Stat(name + p.ext)
		if err != nil || !info.Mode().IsRegular() || info.ModTime().Before(orig.ModTime()) {
			continue
		}
		return name + p.ext, p.coding
	}
	return "", ""
}

// acceptsEncoding reports whether an Accept-Encoding header value allows the
// content coding token (RFC 9110 §12.5.3). The rules that matter:
//
//   - a listed coding is acceptable unless its weight is q=0 ("br;q=0" is a
//     refusal, not a low preference);
//   - "*" stands for every coding not listed by name, with its own weight;
//   - names are case-insensitive, and "x-gzip" is gzip.
//
// Weights other than zero are not ranked against each other. The server's
// own order (brotli, then gzip) decides between two acceptable codings, as
// most servers do: browsers send every coding at q=1 anyway, and a client
// that truly prefers gzip to br can refuse br outright.
//
// A malformed weight makes its element unacceptable rather than the whole
// header: sending a coding the client did not clearly accept is the one
// mistake that breaks the page, so doubt resolves toward plain bytes.
func acceptsEncoding(header, coding string) bool {
	named, star := false, false
	for elem := range strings.SplitSeq(header, ",") {
		token, params, _ := strings.Cut(elem, ";")
		token = strings.ToLower(strings.TrimSpace(token))
		if token == "x-gzip" {
			token = "gzip"
		}
		ok := weightPositive(params)
		switch token {
		case coding:
			// A coding named explicitly decides for itself, whatever "*"
			// says, so a refusal here is final.
			named = true
			if !ok {
				return false
			}
		case "*":
			star = ok
		}
	}
	return named || star
}

// weightPositive reads the parameters after a coding (";q=0.5", possibly
// with spaces) and reports whether its weight is above zero. No q parameter
// is weight 1.
func weightPositive(params string) bool {
	for p := range strings.SplitSeq(params, ";") {
		k, v, found := strings.Cut(p, "=")
		if !found || !strings.EqualFold(strings.TrimSpace(k), "q") {
			continue
		}
		q, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return err == nil && q > 0
	}
	return true
}
