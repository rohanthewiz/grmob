// Command claims prints, as JSON, the strings each screenshot in docs/images
// claims to show, keyed by file name. wasm/shots/shoot.sh hands the output to
// shot.mjs, which refuses to write a picture in which a claimed string is
// clipped.
//
// # Why the camera checks this and the claim tests cannot
//
// The claim tests (see internal/shotclaims) read a rendered node tree, which
// knows what text a screen holds and nothing about where it lands. A line of
// code wider than a phone is in the tree whole and in the picture cut at the
// frame's edge, and tutorial-lesson.png once claimed exactly such a line. Only
// the browser that takes the picture has the layout, so the check is made
// there, at the shutter, against the same manifest the Go tests read.
//
// # Why a program rather than a copy of the strings in JavaScript
//
// internal/shotclaims is the manifest; a second list in shot.mjs would be a
// copy free to drift from it, which is the failure that package exists to
// prevent. Printing the manifest keeps one source.
//
//	internal/shotclaims.Claims ──▶ go run ./claims ──▶ claims.json
//	                                                        │
//	shoot.sh ── node shot.mjs --claims claims.json ◀────────┘
//	                 └── every Shows string whole inside the frame, or no PNG
package main

import (
	"encoding/json"
	"os"

	"github.com/rohanthewiz/grmob/internal/shotclaims"
)

func main() {
	out := map[string][]string{}
	for _, c := range shotclaims.Claims {
		// A composite has no text of its own to check: it is a picture of
		// its parts, and its parts' strings were checked when they were taken.
		if c.Composite() {
			continue
		}
		out[c.File] = c.Shows
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		os.Stderr.WriteString("claims: " + err.Error() + "\n")
		os.Exit(1)
	}
}
