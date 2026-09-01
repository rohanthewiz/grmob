// Package verify holds the checks that span both native shells.
//
// iOS and Android each have their own harness — ios/verify replays a bridge
// transcript through the real Swift runtime, and the Android build has its own
// gradle path — but neither is the right home for a rule that has to hold in
// *both* renderers at once, and neither of them runs under `go test ./...`.
// This package is where such a rule lives: it sits under mobile/ because
// mobile is the bridge surface both native shells are written against, so
// "true of every native renderer" is a statement about this package's
// consumers.
//
// Everything here reads native source as text. That is a deliberate limit, not
// a shortcut — see contentmode_test.go for why the alternative (asking the
// Swift and Kotlin compilers) cannot see the thing being checked, and why a
// check that needed Xcode or gradle would stop running for most of the people
// who need it.
//
// The package is test-only; this file exists so the directory is a package
// that `go build ./...` and `go vet ./...` will name in their output rather
// than skip.
package verify
