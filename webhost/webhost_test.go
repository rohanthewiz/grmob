package webhost

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

// The two hand-written browser hosts in this repository. Each must install
// exactly Bindings: a binding added to the library and not to a host (or the
// other way round) is a page contract that differs between that host and every
// app built on webhost.
//
// wasm/shots/host used to be allowed a subset, on the grounds that the
// screenshot driver never hot-reloads it and so it needed no Shutdown. That
// held only while shoot.sh was the one way to load it; under serve -dev a host
// with no Shutdown turns every rebuild into a page reload. It installs the
// whole set now, and the allowance went with the reason for it.
var hosts = []string{
	filepath.Join("..", "wasm", "main.go"),
	filepath.Join("..", "wasm", "shots", "host", "main.go"),
}

func TestHandWrittenHostsInstallTheLibrarysBindings(t *testing.T) {
	for _, path := range hosts {
		got := grmobWASMKeys(t, path)
		if len(got) == 0 {
			t.Fatalf("%s: found no js.Global().Set(\"GrMobWASM\", map[string]any{...}); "+
				"the host changed shape and this test is no longer reading it", path)
		}
		for _, name := range got {
			if !slices.Contains(Bindings, name) {
				t.Errorf("%s installs GrMobWASM.%s, which webhost.Bindings does not name. "+
					"Add it to webhost (bindings.go and host.go) so apps built on the library get it too.",
					path, name)
			}
		}
		for _, name := range Bindings {
			if !slices.Contains(got, name) {
				t.Errorf("%s does not install GrMobWASM.%s, which webhost.Bindings names", path, name)
			}
		}
	}
}

// grmobWASMKeys returns the keys of the map literal passed as the value of
// js.Global().Set("GrMobWASM", ...) in the file at path.
func grmobWASMKeys(t *testing.T, path string) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	var keys []string
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Set" {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if s, _ := strconv.Unquote(lit.Value); s != "GrMobWASM" {
			return true
		}
		m, ok := call.Args[1].(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, elt := range m.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if k, ok := kv.Key.(*ast.BasicLit); ok {
				if s, err := strconv.Unquote(k.Value); err == nil {
					keys = append(keys, s)
				}
			}
		}
		return false
	})
	return keys
}
