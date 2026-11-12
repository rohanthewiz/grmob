package permission

import "github.com/rohanthewiz/grmob/internal/testreset"

// init hands this package's reset to internal/testreset, so grmobtest.Reset
// clears the status record, its subscriptions and the foreground watches
// between tests. resetForTest already releases the foreground watches first
// (resetForegroundForTest), in the order its comment explains, so that one
// function is the only registration. See internal/testreset for why it is
// reached through a registry.
func init() {
	testreset.Register("permission.resetForTest", resetForTest)
}
