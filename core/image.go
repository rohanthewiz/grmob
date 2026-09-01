package core

// ContentMode says how an image's intrinsic aspect ratio is reconciled with
// the box the layout gave it. It is the one image property that is genuinely
// not styling: every renderer expresses it through the image view's own API
// (SwiftUI's content mode, Compose's ContentScale, CSS object-fit), not
// through the box modifiers, so it travels as a node prop.
//
// The four values are the intersection all three targets can express exactly:
//
//	         | fits inside | fills box | ratio kept
//	---------+-------------+-----------+-----------
//	Fit      | yes         | no        | yes
//	Fill     | no (crops)  | yes       | yes
//	Stretch  | no          | yes       | no
//	Center   | no          | no        | yes (1:1 pixels)
//
// Fit is the default — it is what core.Image has always rendered as, so an
// existing call site keeps its layout — and is also the safe default: it is
// the only mode that never crops and never distorts.
type ContentMode string

const (
	// ContentModeFit scales the image down until it fits entirely inside the
	// box, preserving the aspect ratio and leaving empty space on the axis
	// that ran out first. CSS `object-fit: contain`.
	ContentModeFit ContentMode = "fit"

	// ContentModeFill scales the image up until it covers the box, preserving
	// the aspect ratio and cropping the overflow on the longer axis. The mode
	// for avatars, hero images and thumbnails — anything where empty space
	// would be worse than losing an edge. CSS `object-fit: cover`.
	//
	// The crop is real on every target: CSS object-fit clips, Compose's
	// ContentScale.Crop clips, and the SwiftUI path adds an explicit
	// .clipped() — an unclipped image would paint over its siblings.
	ContentModeFill ContentMode = "fill"

	// ContentModeStretch distorts the image to exactly the box's dimensions,
	// ignoring the aspect ratio. CSS `object-fit: fill`. Rarely what a design
	// wants; included because the platforms offer it and because a Stretch
	// spelled out beats an app pre-scaling its assets.
	ContentModeStretch ContentMode = "stretch"

	// ContentModeCenter draws the image at its intrinsic size, centered, with
	// no scaling in either direction — larger than the box means it is
	// cropped, smaller means it is surrounded by space. CSS `object-fit:
	// none`. For pixel-exact assets (icons, QR codes) that scaling would blur.
	ContentModeCenter ContentMode = "center"
)

// ContentModes returns every declared ContentMode, in declaration order.
//
// Go cannot enumerate the constants of a named string type at run time, so
// the set has to be written out a second time — and a second copy of a list
// is exactly the thing that goes stale. This one is pinned to the const block
// above by TestContentModesMatchTheDeclaredConstants, which reads them out of
// this file's syntax tree, so adding a constant without adding it here fails
// `go test ./...` rather than silently shrinking the set.
//
// It exists because four renderers each map these modes onto their own
// vocabulary — CSS object-fit in htmlout and the WASM runtime, SwiftUI
// scaling in Renderer.swift, Compose's ContentScale in Renderer.kt — and
// none of them can be asked "did you cover every mode?" without a list to
// check against. All four are now held to it:
//
//	htmlout.ObjectFits          htmlout/objectfit_test.go
//	the WASM runtime's copy     wasm/verify/objectfit_test.go (via htmlout)
//	Renderer.swift              mobile/verify/contentmode_test.go
//	Renderer.kt                 mobile/verify/contentmode_test.go
//
// The first two are table comparisons — both sides map a mode onto the same
// CSS keyword, so the values can be compared as well as the keys. The natives
// map onto SwiftUI and Compose vocabularies that share nothing with CSS or
// with each other, so only the key set is comparable; those two checks read
// the arms out of the native source and check coverage alone.
//
// A fresh slice per call rather than a package-level var: a var of slice type
// is writable by any importer, and four elements are cheaper to build than to
// defend.
func ContentModes() []ContentMode {
	return []ContentMode{
		ContentModeFit,
		ContentModeFill,
		ContentModeStretch,
		ContentModeCenter,
	}
}

// Image renders a remote or bundled image at the default content mode
// (ContentModeFit). Use ImageWithMode to choose another.
func Image(src string, styleProps ...StyleProp) View {
	return imageNode(src, "", styleProps)
}

// ImageWithMode is Image plus an explicit ContentMode.
//
// A separate builder rather than a variadic change to Image, matching
// InputWithSubmit: every existing Image call site keeps compiling and keeps
// its current rendering, and the mode stays a required, visible argument at
// the sites that care rather than an option buried in a style list.
func ImageWithMode(src string, mode ContentMode, styleProps ...StyleProp) View {
	return imageNode(src, mode, styleProps)
}

// imageNode is the shared body. An empty mode is omitted from Props entirely
// rather than written as "fit": the renderers each default to fit on a
// missing prop, and leaving it out keeps the prop maps — and therefore the
// reconciler's update-props diffs and every existing snapshot test — byte
// identical to what Image produced before ContentMode existed.
func imageNode(src string, mode ContentMode, styleProps []StyleProp) View {
	return ComponentFunc(func(ctx *Context) *Node {
		base := ctx.Theme().Components.Camera
		style := &base
		for _, sp := range styleProps {
			sp.Apply(style)
		}

		props := map[string]any{
			"src": src,
		}
		if mode != "" {
			props["contentMode"] = string(mode)
		}

		return &Node{
			Type:  "Image",
			Props: props,
			Style: style,
		}
	})
}
