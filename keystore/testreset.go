package keystore

import "github.com/rohanthewiz/grmob/internal/testreset"

// init hands this package's reset to internal/testreset, so grmobtest.Reset
// drops a test's unanswered calls before the next test's host can answer
// them. The reset stays unexported; see internal/testreset for why it is
// reached through a registry.
func init() {
	testreset.Register("keystore.resetForTest", resetForTest)
}
