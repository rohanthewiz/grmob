package core

import "github.com/rohanthewiz/grmob/internal/testreset"

// init hands core's process-wide resets to internal/testreset, so
// grmobtest.Reset can return a test process to a fresh one's state without
// core exporting a test seam. See that package's comment for why it is a
// registry rather than exported functions.
//
// Each of these owns a record or a table that outlives any one Context: the
// device records (window, lifecycle, audio, heading, location), whose last
// report a later test's first render would otherwise read as its own, and
// the pending host requests (clipboard reads, shares, notification sweeps),
// which a later test's host answer could otherwise land on. The concern
// collector is not here: it is public (ClearConcerns), and grmobtest calls it
// directly.
//
// The names are the functions' own, prefixed with the package, because the
// census in internal/testreset finds every reset…ForTest function in the
// module's source and holds it to a registration under exactly that name.
func init() {
	testreset.Register("core.resetWindowForTest", resetWindowForTest)
	testreset.Register("core.resetLifecycleForTest", resetLifecycleForTest)
	testreset.Register("core.resetAudioForTest", resetAudioForTest)
	testreset.Register("core.resetHeadingForTest", resetHeadingForTest)
	testreset.Register("core.resetLocationForTest", resetLocationForTest)
	testreset.Register("core.resetClipboardForTest", resetClipboardForTest)
	testreset.Register("core.resetShareForTest", resetShareForTest)
	testreset.Register("core.resetSweepsForTest", resetSweepsForTest)
}
