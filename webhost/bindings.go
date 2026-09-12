package webhost

// Bindings names every function Run installs on the GrMobWASM global, in the
// order the page's lifecycle reaches them.
//
// It lives in a file with no build constraint, apart from host.go (js/wasm
// only), so that an ordinary `go test ./...` on a development machine can
// read it and compare it with the hand-written hosts in this repository; see
// webhost_test.go.
var Bindings = []string{"RenderInitial", "RenderAgain", "ReceiveEvent", "IsDirty", "HostEvent", "Shutdown"}
