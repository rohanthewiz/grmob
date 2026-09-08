// Package gateharness is the shape both verify passes' gate tests are.
//
// It holds no Go code. harness.sh carries the counting, the prefix match and
// the FAIL/OK footer that ios/verify/gate_test.sh and android/verify/gate_test.sh
// used to hold a copy apiece of; each of those keeps its own table, which is
// the only part that is about an SDK or about a JVM.
//
// The Go file exists so the directory is a package, and the package exists so
// that harness_test.sh — the shell script that reaches harness.sh's own arms —
// has somewhere to be run from by `go test ./...`. See harness_go_test.go for
// why that matters.
package gateharness
