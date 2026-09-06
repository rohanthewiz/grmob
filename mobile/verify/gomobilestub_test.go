package verify

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ios/verify/gomobile_stub.swift, held against the Go package it stands for.
//
// # What the stub is and why it needs pinning
//
// Three files in ios/GrMob/App were checked by nothing at all, the @main
// entry point among them, because GomobileBridge.swift imports `GrMob` — the
// module a `gomobile bind` produces. A harness that required the bind would
// need full Xcode *and* a gomobile toolchain, which is most of this
// repository's contributors excluded from running it, so ios/verify compiles
// a stand-in module of that name instead and type-checks the app layer
// against it.
//
// That trade buys a real check and creates a new way to be wrong: a
// hand-written stand-in for generated code is a copy, and a copy drifts. The
// failure is quiet and unusually bad — the shell keeps type-checking green
// against a bridge Go no longer has, which is the exact opposite of what the
// pass was added for. So the stub is not trusted; it is held to the Go source
// here, in the package whose subject is "a rule the native shells must obey
// that no native toolchain can see".
//
// # What is checked, and what deliberately is not
//
// Checked: every bindable exported function and every exported interface in
// package `mobile` has a declaration in the stub, under gobind's naming; and
// nothing in the stub claims a Go symbol that is not there.
//
// Not checked: the parameter and result *types*. Copying gobind's full type
// mapping into a test would be reimplementing gobind to check a file that
// exists to avoid running gobind, and the payoff is small — a wrong signature
// in the stub fails the Swift type-check in ios/verify the moment the shell
// calls it, because the shell is compiled against exactly these declarations.
// A *missing* declaration is the case that fails silently there (the file
// simply does not compile, but only once someone runs it against a bind), and
// a *surplus* one is the case that reads as support for a bridge function
// that does not exist. Those two are what this file catches.

// gobind's naming, which the stub imitates. See the header comment in
// gomobile_stub.swift for the two-step renaming these encode.
const gobindPrefix = "Mobile" // the bound package name, capitalized

// bindableGoTypes are the Go types gobind can carry across the FFI that this
// bridge actually uses. The list is deliberately short rather than complete:
// mobile/ is written to a narrow surface on purpose (see docs/platforms/
// native.md), so a parameter type outside this set is the signal that a new
// kind of value is crossing the bridge and that the mapping needs a human,
// not that this list wants extending on reflex.
var bindableGoTypes = map[string]bool{
	"string": true,
	"bool":   true,
	"int":    true,
}

// swiftFuncDecl matches a stub function declaration, capturing its name.
var swiftFuncDecl = regexp.MustCompile(`(?m)^public func ([A-Za-z0-9_]+)\(`)

// swiftProtocolDecl matches a stub protocol declaration, capturing its name.
var swiftProtocolDecl = regexp.MustCompile(`(?m)^public protocol ([A-Za-z0-9_]+)`)

var gomobileStub = nativeFile("ios", "verify", "gomobile_stub.swift")

// Every bindable symbol in `mobile` must have a stub declaration, and every
// stub declaration must name a bindable symbol.
//
// Both directions matter, for the reasons requireRoleCoverage gives one file
// over. A missing declaration means the next person to add a bridge function
// finds the app layer silently dropping out of the type-check for it. A
// surplus one is a declaration the real framework does not have: the shell
// can call it, the harness stays green, and the failure surfaces at
// `gomobile bind` time or later.
func TestTheGomobileStubMatchesTheBoundGoSurface(t *testing.T) {
	src := readNative(t, gomobileStub)

	want := map[string]string{} // Swift name -> what it stands for
	for _, fn := range bindableFuncs(t) {
		want[gobindPrefix+fn] = "mobile." + fn
	}
	for _, iface := range boundInterfaces(t) {
		// A Go interface becomes both an ObjC protocol and a class of that
		// name; Swift resolves the collision by suffixing the protocol, and
		// the suffixed spelling is the one the shell conforms to.
		want[gobindPrefix+iface+"Protocol"] = "mobile." + iface
	}

	got := map[string]bool{}
	for _, m := range swiftFuncDecl.FindAllStringSubmatch(src, -1) {
		got[m[1]] = true
	}
	for _, m := range swiftProtocolDecl.FindAllStringSubmatch(src, -1) {
		got[m[1]] = true
	}

	missing := map[string]string{}
	for name, origin := range want {
		if !got[name] {
			missing[name] = origin
		}
		delete(got, name)
	}

	for _, name := range sortedNamesOf(missing) {
		t.Errorf("%s: no declaration for %s (%s) — the iOS app layer stops being "+
			"type-checked against it", gomobileStub, name, missing[name])
	}
	for _, name := range sortedNamesOf(got) {
		t.Errorf("%s: declares %s, which no exported bindable symbol in package mobile "+
			"produces — the shell may call a bridge function that will not exist "+
			"after a gomobile bind", gomobileStub, name)
	}
}

// bindableFuncs returns the exported package-level functions in `mobile`
// whose whole signature gobind can carry.
//
// Register is the one this filter exists for: it takes a *core.Context and a
// func, neither of which gobind can bind, so the generated header carries
// "skipped function Register with unsupported parameter or return types"
// where the declaration would be. A stub that declared it would be offering
// the shell a call that cannot exist.
func bindableFuncs(t *testing.T) []string {
	t.Helper()

	var out []string
	for _, file := range parseMobilePackage(t) {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() {
				continue
			}
			if bindableSignature(fn.Type) {
				out = append(out, fn.Name.Name)
			}
		}
	}
	sort.Strings(out)
	return out
}

// bindableSignature reports whether every parameter and result is a type
// gobind can carry: one of bindableGoTypes, or an interface declared in this
// package (which binds as a protocol).
func bindableSignature(sig *ast.FuncType) bool {
	ok := true
	fields := append([]*ast.Field{}, sig.Params.List...)
	if sig.Results != nil {
		fields = append(fields, sig.Results.List...)
	}
	for _, f := range fields {
		name, isIdent := f.Type.(*ast.Ident)
		if !isIdent {
			// A pointer, a func, a map, a qualified type from another
			// package — all of them stop the bind.
			ok = false
			continue
		}
		if !bindableGoTypes[name.Name] && !ast.IsExported(name.Name) {
			ok = false
		}
	}
	return ok
}

// boundInterfaces returns the exported interface types in `mobile`. Each
// becomes a protocol the shell can conform to — the only way a callback
// crosses the FFI, since gobind cannot bind a func parameter.
func boundInterfaces(t *testing.T) []string {
	t.Helper()

	var out []string
	for _, file := range parseMobilePackage(t) {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				if _, isIface := ts.Type.(*ast.InterfaceType); isIface {
					out = append(out, ts.Name.Name)
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

// parseMobilePackage reads the bridge package's non-test sources.
//
// The syntax tree rather than the text, unlike everything else in this
// package: the subject here is Go's own source, where a parser is available
// and free, and the question being asked (which functions have a signature
// gobind can carry) is one no substring match can answer.
func parseMobilePackage(t *testing.T) []*ast.File {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, nativeFile("mobile"), func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parsing package mobile: %v", err)
	}
	pkg, ok := pkgs["mobile"]
	if !ok {
		t.Fatalf("no package mobile under %s", nativeFile("mobile"))
	}

	files := make([]*ast.File, 0, len(pkg.Files))
	for _, f := range pkg.Files {
		files = append(files, f)
	}
	return files
}

// sortedNamesOf gives the failures a stable order, for the reason sortedRoles
// exists in role_test.go: a run reporting several drifted symbols should
// report them in the same order twice.
func sortedNamesOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for name := range m {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
