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

// The two hand-written browser hosts in this repository, and how each is held
// to the library.
//
// wasm/main.go is the full host the tutorial site ships, so it must install
// exactly Bindings: a binding added to the library and not to it (or the other
// way round) is a page contract that differs between the site and every app
// built on webhost.
//
// wasm/shots/host is a local screenshot harness that never hot-reloads, so it
// has no Shutdown and is allowed a subset — but not a name the library does not
// know, which would be a contract only the harness speaks.
var hosts = []struct {
	path   string
	subset bool
}{
	{filepath.Join("..", "wasm", "main.go"), false},
	{filepath.Join("..", "wasm", "shots", "host", "main.go"), true},
}

func TestHandWrittenHostsInstallTheLibrarysBindings(t *testing.T) {
	for _, h := range hosts {
		got := grmobWASMKeys(t, h.path)
		if len(got) == 0 {
			t.Fatalf("%s: found no js.Global().Set(\"GrMobWASM\", map[string]any{...}); "+
				"the host changed shape and this test is no longer reading it", h.path)
		}
		for _, name := range got {
			if !slices.Contains(Bindings, name) {
				t.Errorf("%s installs GrMobWASM.%s, which webhost.Bindings does not name. "+
					"Add it to webhost (bindings.go and host.go) so apps built on the library get it too.",
					h.path, name)
			}
		}
		if h.subset {
			continue
		}
		for _, name := range Bindings {
			if !slices.Contains(got, name) {
				t.Errorf("%s does not install GrMobWASM.%s, which webhost.Bindings names", h.path, name)
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
