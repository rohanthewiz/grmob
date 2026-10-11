package verify

import (
	"strings"
	"testing"
)

// An Image's natural size on iOS (N-021).
//
// A browser floors an `<img>` flex item at min(declared width, natural width
// carried through its declared height), so a squeezed row may narrow an image
// box down to the bitmap's own proportions. iOS used to floor every Image with
// a src at its declared width, because GrMobImage was an AsyncImage and no
// natural size ever reached the tree walk. Three pieces now carry it, and
// each is a silent no-op if it goes missing:
//
//  1. GrMobImage decodes the bitmap itself and records its size in
//     GrMobImageSizes. Without the record the floor never leaves the
//     declared width.
//  2. FlexChildren marks a Row's image child squeezable, and grMobBox hands
//     that to grMobDimension. Without it the Width stays a rigid frame that
//     reports the declared width whatever the Row proposes, so a lowered
//     floor would only make the image draw over its neighbour.
//  3. grMobDimension's flexible arm takes the squeezable case.
//
// The floor's arithmetic (imageFloor: Chrome's 110 × 40 measurements, the
// unsized and percentage heights, a zero side, margins, an image still
// loading) runs on macOS in ios/verify's mincontent.swift. These checks hold
// the wiring, which only a simulator would otherwise exercise.
func TestIOSImageFloorReadsTheNaturalSize(t *testing.T) {
	renderer := valuesIn(t, swiftRenderer)
	style := valuesIn(t, swiftStyle)
	minContent := valuesIn(t, maxWidthMinContent)
	for _, pin := range []struct{ file, src, code, why string }{
		{swiftRenderer, renderer, "GrMobImageSizes.shared.record(image.size, for: src)",
			"GrMobImage must record the decoded bitmap's natural size, or the floor never leaves the declared width"},
		{swiftRenderer, renderer, "squeezable: axis == .horizontal\n                                                   && GrMobMinContent.isReplacedImage(child)",
			"a Row must mark its image children squeezable, the same images the floor treats as replaced"},
		{swiftStyle, style, "squeezable: grow.squeezesWidth)",
			"grMobBox must hand the squeezable flag to the Width frame"},
		{swiftStyle, style, "case .horizontal where relativeCap || squeezable:",
			"a squeezable Width must be the flexible frame, or it ignores the slot the Row narrowed"},
		{maxWidthMinContent, minContent, "imageFloor(declared: w, height: s.height,\n                                  natural: GrMobImageSizes.shared.size(for: node.stringProp(\"src\")))",
			"the width walk must read the recorded natural size for an Image with a src"},
		{maxWidthMinContent, minContent, "@Observable\nfinal class GrMobImageSizes",
			"the store must be observable, or a Row whose floor was read before its image arrived keeps the stale floor"},
	} {
		if !strings.Contains(pin.src, pin.code) {
			t.Errorf("%s: lacks %q — %s", pin.file, pin.code, pin.why)
		}
	}
	// AsyncImage hands its content a SwiftUI Image and no size, which is the
	// shape this replaced. Back on it, the record above would have nothing to
	// record.
	if strings.Contains(renderer, "AsyncImage(") {
		t.Errorf("%s: GrMobImage is an AsyncImage again; it has no natural size to give the floor", swiftRenderer)
	}
}
