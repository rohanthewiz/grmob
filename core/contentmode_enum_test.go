package core

import "testing"

// ContentModes is a hand-written list of constants declared a few lines above
// it in image.go, pinned here to the declaration it restates.
//
// This was the first of these pins and for a while the only one; the parse it
// used to carry now lives in enum_pin_test.go, shared with the three alignment
// enumerations, which have the same problem for the same reason.
//
// Every renderer's ContentMode coverage check ultimately rests on this list —
// see htmlout.ObjectFits, wasm/verify/objectfit_test.go and
// mobile/verify/contentmode_test.go — so it is the one link in that chain that
// has to be pinned to something other than another list.
func TestContentModesMatchTheDeclaredConstants(t *testing.T) {
	requireExactEnum(t, "image.go", "ContentMode", "ContentModes()", ContentModes())
}
