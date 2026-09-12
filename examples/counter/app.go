// Package counter is the first app in this repository's documentation: a
// screen with a number on it and two buttons that change the number.
//
// # Why it is a package and not a fenced block
//
// It was a fenced block, twice. README.md opened with it and
// docs/getting-started.md opened with a near-identical one, and neither
// corresponded to anything that compiled — the two had already drifted
// apart on core.Padding, which is the kind of difference a reader only
// finds by typing both in.
//
// The whole argument this repository makes about numbers in prose applies
// to code in prose: a snippet is a copy of something, written while it was
// true, with nothing that reads it. So the snippet moved here and the two
// documents quote it. TestTheDocumentedCounterIsThisPackage is what holds
// them to it, on the same rule examples/todoapp's tutorial excerpts are
// held to: every non-elided line of the quotation appears in this source,
// in order.
//
// # What it is for besides being quoted
//
// It is the smallest thing ./build.sh can mount, which is what makes
// docs/images/counter.png reproducible — wasm/shots mounts this package by
// name and drives it to three taps. Before it existed the README's first
// screenshot was of a file that lived in a scratch directory for the
// length of one session.
package counter

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/mobile"
)

// init registers the app with the mobile bridge. This is the whole
// integration contract: the native shells and the browser host both reach
// an app through the bridge's singleton manager, and an app package earns
// its place there by running this at link time.
func init() {
	mobile.Register(core.NewContext(), App)
}

// AppName gives gomobile a bindable symbol so the package — and therefore
// the init above — links into the native library. A func-typed App is not
// bindable itself, so without an exported function of a bindable shape the
// linker would drop the package and the registration with it.
func AppName() string { return "Counter" }

// App is the root view: a function from a context to a view, called again
// on every render pass.
//
// Three things carry most of the framework and all three are visible here.
// Views are values, so composing a screen is calling functions. App reads
// state and returns a tree rather than mutating the screen. And count.Set
// is the entire update path — it marks the tree dirty, and the diff of the
// next pass against the last is what reaches whichever renderer is
// attached.
func App(ctx *core.Context) core.View {
	count := core.NewState(ctx, 0)

	return core.SafeArea(
		core.Column(
			core.Gap(12),
			core.Padding(24),
			core.Text("Counter", core.FontSize(28), core.FontWeight(core.Bold)),
			core.Text(fmt.Sprintf("Count: %d", count.Get())),
			core.Row(
				core.Gap(8),
				core.Button("−", func() { count.Set(count.Get() - 1) }),
				core.Button("+", func() { count.Set(count.Get() + 1) }),
			),
		),
	)
}
