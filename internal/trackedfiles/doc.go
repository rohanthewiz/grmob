// Package trackedfiles holds checks about what this repository commits, as
// opposed to what its code does. It has no code of its own; its test reads
// the files git tracks.
//
// # Why a package of its own
//
// wasm/verify is where most repository-wide censuses live, but it registers
// every walk of the whole tree in a table against a budget (repowalks_test.go)
// and its shared walk skips files over 4 MiB and anything with a NUL byte in
// it. The check here is about exactly those files, so it is not a question
// that walk can be asked, and its cost (one `git ls-files` and four bytes of
// each file) does not belong in that package's accounting.
package trackedfiles
