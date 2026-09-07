package verify

import (
	"fmt"
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
// # What is checked
//
// Two questions, in two tests.
//
// The names: every bindable exported function and every exported interface in
// package `mobile` has a declaration in the stub, under gobind's naming, and
// nothing in the stub claims a Go symbol that is not there.
//
// The signatures: every one of those declarations is character-for-character
// what gobind would emit for the Go signature behind it — parameter types and
// order, argument labels, nullability and the return — for the package
// functions and for the interfaces' methods alike.
//
// # Why the signatures were once left out, and why that changed
//
// This file used to check only the names, on the argument that copying
// gobind's type mapping into a test would be reimplementing gobind to check a
// file that exists to avoid running gobind — and that the payoff was small,
// because a wrong signature fails the Swift type-check in ios/verify the
// moment the shell calls it.
//
// The first half was right about gobind's *general* mapping and wrong about
// this bridge's. Package `mobile` is written to a narrow surface on purpose,
// so the mapping the stub can possibly need is three rows and a protocol rule
// (see gobindSwiftTypes), and a type outside that set already stops the bind
// at bindableSignature. Stating three rows is not reimplementing gobind.
//
// The second half quietly assumed every declaration has a call site. Three do
// not — MobileDataDir, MobileRenderAgain and MobileReportHostEvent are all
// reachable from a shell that never touches them — and for those the
// type-check proves nothing, so a drifted declaration was checked by nothing
// at all. That is the gap the signature test closes.
//
// # The two failure shapes the name check still owns
//
// A *missing* declaration is the case that fails silently in ios/verify (the
// file simply does not compile, but only once someone runs it against a real
// bind), and a *surplus* one reads as support for a bridge function that does
// not exist.

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

// --- The signatures --------------------------------------------------------

// gobindSwiftTypes is the Go -> Swift mapping this bridge's surface uses,
// transcribed from the generated header (Headers/Mobile.objc.h) and split by
// position because gobind's nullability is not symmetric.
//
// # Why a table and not a translator
//
// The header comment above used to say the types were deliberately unchecked,
// on the grounds that copying gobind's type mapping into a test would be
// reimplementing gobind. That argument was right about the general mapping
// and wrong about this one. Package `mobile` is written to a narrow surface
// on purpose (see docs/platforms/native.md and bindableGoTypes), so the whole
// mapping the stub can possibly need is the three rows below plus the
// protocol rule — and stating three rows is not reimplementing anything, it
// is the same move aria/verify/testdata/aria.json makes for role semantics:
// the fact goes in one place where a test can hold the code to it.
//
// The gap it closes is the one bindableGoTypes' own argument left open. A
// wrong signature does fail the Swift type-check in ios/verify — but only for
// a declaration the shell actually calls. MobileDataDir, MobileRenderAgain
// and MobileReportHostEvent are reachable from a shell that never touches
// them, and a drifted declaration for one of those was checked by nothing at
// all: not by this file, which only counted names, and not by the type-check,
// which needs a call site.
//
// # The nullability asymmetry is the reason for two columns
//
// gobind annotates every NSString* parameter _Nullable and every NSString*
// return _Nonnull, so a Go `string` argument arrives in Swift as `String?`
// and a Go `string` result as `String`. Bool and int are C scalars and carry
// no nullability at all. Copying that asymmetry is the point: a stub taking
// `String` everywhere would accept shell code the real framework rejects,
// which is precisely the drift this file exists to catch.
var gobindSwiftTypes = map[string]struct{ param, result string }{
	"string": {"String?", "String"},
	"bool":   {"Bool", "Bool"},
	"int":    {"Int", "Int"},
}

// swiftDecl matches any declaration line the stub can carry — a public
// function or a protocol method — capturing the name and everything after it
// up to the body.
//
// One regexp for both because the difference between them is the prefix, and
// the prefix is what the caller is already scanning for. The body is cut
// separately (see stubDeclHeader): a parameter type cannot contain a brace,
// so the first one on the line always opens the body.
//
// The indent is [ \t]* and not \s*, which is not a style choice: \s matches a
// newline, so the shorter spelling lets a match begin at the *previous* line's
// start and swallow the break, after which the match index no longer points
// into the declaration's own line and the header cut comes back empty.
var swiftDecl = regexp.MustCompile(`(?m)^[ \t]*(?:public )?func ([A-Za-z0-9_]+)\(`)

// Every bindable Go signature must be transcribed exactly by the stub.
//
// Exactly, character for character, rather than checked type by type: the
// declaration is one string, and comparing it whole catches a reordered
// parameter pair, a dropped argument label and a missing `?` with the same
// assertion. Both `mobile`'s package functions and its interfaces' methods
// are covered — the protocols are how a callback crosses the FFI, so a
// drifted method there is a shell conforming to something Go will not call.
func TestTheGomobileStubTranscribesEveryBoundSignature(t *testing.T) {
	src := readNative(t, gomobileStub)
	ifaces := interfaceSet(t)

	for _, fn := range bindableFuncDecls(t) {
		name := gobindPrefix + fn.Name.Name
		got, ok := stubDeclHeader(src, name)
		if !ok {
			// The name check above already reports this; a second failure
			// for the same cause would only add noise.
			continue
		}
		want, err := swiftFuncSignature(name, fn.Type, ifaces)
		if err != nil {
			t.Errorf("mobile.%s: %v", fn.Name.Name, err)
			continue
		}
		if got != want {
			t.Errorf("%s: declares\n\t%s\nbut mobile.%s binds as\n\t%s",
				gomobileStub, got, fn.Name.Name, want)
		}
	}

	for _, iface := range boundInterfaceDecls(t) {
		proto := gobindPrefix + iface.name + "Protocol"
		body, ok := protocolBody(src, proto)
		if !ok {
			continue // reported by the name check
		}
		declared := map[string]string{}
		for _, m := range swiftDecl.FindAllStringSubmatchIndex(body, -1) {
			mName := body[m[2]:m[3]]
			declared[mName] = strings.TrimSpace(cutBody(body[m[0]:lineEnd(body, m[0])]))
		}
		for _, method := range iface.methods {
			goName := methodName(method)
			if goName == "" {
				t.Errorf("mobile.%s embeds an interface; gobind flattens an embedding into "+
					"the protocol, which nothing here transcribes", iface.name)
				continue
			}
			swiftName := lowerFirst(goName)
			got, ok := declared[swiftName]
			if !ok {
				t.Errorf("%s: protocol %s declares no %s — mobile.%s.%s cannot be called "+
					"through it", gomobileStub, proto, swiftName, iface.name, goName)
				continue
			}
			delete(declared, swiftName)
			want, err := swiftMethodSignature(swiftName, methodType(method), ifaces)
			if err != nil {
				t.Errorf("mobile.%s.%s: %v", iface.name, goName, err)
				continue
			}
			if got != want {
				t.Errorf("%s: protocol %s declares\n\t%s\nbut mobile.%s.%s binds as\n\t%s",
					gomobileStub, proto, got, iface.name, goName, want)
			}
		}
		for _, extra := range sortedNamesOf(declared) {
			t.Errorf("%s: protocol %s declares %s, which mobile.%s does not have — a shell "+
				"conforming to it implements a method Go will never call",
				gomobileStub, proto, extra, iface.name)
		}
	}
}

// swiftFuncSignature builds the declaration gobind produces for a package
// function.
//
// Every parameter is unlabeled. A bound package function becomes a C
// function in the generated header — MobileReportHostEvent(NSString*,
// NSString*) — and Swift imports a C function with no argument labels at all.
// That is the one place the two shapes in this file differ: an ObjC *method*
// carries selector pieces, which Swift turns into labels, and
// swiftMethodSignature encodes that.
func swiftFuncSignature(name string, sig *ast.FuncType, ifaces map[string]bool) (string, error) {
	params, err := swiftParams(sig, ifaces, func(int, string) string { return "_ " })
	if err != nil {
		return "", err
	}
	ret, err := swiftResult(sig, ifaces)
	if err != nil {
		return "", err
	}
	return "public func " + name + "(" + params + ")" + ret, nil
}

// swiftMethodSignature builds the declaration gobind produces for one method
// of a bound interface.
//
// The labels are the selector. `OnSystemEvent(name, payload string)` becomes
// the ObjC selector `onSystemEvent:payload:`, which Swift imports as
// `onSystemEvent(_ name:, payload:)` — the first piece is folded into the
// method name and so takes no label, and every piece after it is the Go
// parameter's own name.
func swiftMethodSignature(name string, sig *ast.FuncType, ifaces map[string]bool) (string, error) {
	params, err := swiftParams(sig, ifaces, func(i int, goName string) string {
		if i == 0 {
			return "_ "
		}
		return ""
	})
	if err != nil {
		return "", err
	}
	ret, err := swiftResult(sig, ifaces)
	if err != nil {
		return "", err
	}
	return "func " + name + "(" + params + ")" + ret, nil
}

// swiftParams renders a parameter list. label decides each parameter's
// argument label, which is the only thing that differs between a bound
// package function and a bound interface method.
func swiftParams(sig *ast.FuncType, ifaces map[string]bool, label func(int, string) string) (string, error) {
	var out []string
	for _, p := range flattenParams(sig.Params) {
		if p.name == "" {
			return "", fmt.Errorf("parameter %d is unnamed; gobind names it p%d and the stub "+
				"cannot be derived from the Go source. Name it", len(out), len(out))
		}
		typ, err := swiftType(p.expr, ifaces, false)
		if err != nil {
			return "", err
		}
		out = append(out, label(len(out), p.name)+p.name+": "+typ)
	}
	return strings.Join(out, ", "), nil
}

// swiftResult renders the return clause, which is empty for a void function.
//
// More than one result is rejected rather than mapped: gobind turns a
// (T, error) pair into a throwing function and anything wider into an
// out-parameter shape, and neither has ever crossed this bridge. A second
// result is the signal that a new kind of value is being returned and that
// the mapping needs a human — the same stance bindableGoTypes takes on
// parameter types.
func swiftResult(sig *ast.FuncType, ifaces map[string]bool) (string, error) {
	if sig.Results == nil || len(sig.Results.List) == 0 {
		return "", nil
	}
	results := flattenParams(sig.Results)
	if len(results) != 1 {
		return "", fmt.Errorf("%d results; gobind maps a multi-result signature onto a "+
			"throwing function or an out-parameter, neither of which this bridge has "+
			"ever used. Add the mapping deliberately", len(results))
	}
	typ, err := swiftType(results[0].expr, ifaces, true)
	if err != nil {
		return "", err
	}
	return " -> " + typ, nil
}

// swiftType maps one Go type onto its Swift spelling, in the given position.
func swiftType(expr ast.Expr, ifaces map[string]bool, isResult bool) (string, error) {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return "", fmt.Errorf("type is not a plain identifier; gobind cannot bind it, so "+
			"bindableSignature should already have excluded this function (%T)", expr)
	}
	if m, ok := gobindSwiftTypes[ident.Name]; ok {
		if isResult {
			return m.result, nil
		}
		return m.param, nil
	}
	if ifaces[ident.Name] {
		if isResult {
			// Never yet emitted. Refused rather than guessed: the nullability
			// of a returned protocol is a fact about the generated header,
			// and this table is only allowed to hold facts read off one.
			return "", fmt.Errorf("returns the bound interface %s; no bridge function has "+
				"done that, so the return's nullability is not transcribed anywhere. "+
				"Read it off Headers/Mobile.objc.h and add the row", ident.Name)
		}
		return gobindPrefix + ident.Name + "Protocol?", nil
	}
	return "", fmt.Errorf("type %s has no row in gobindSwiftTypes and is not a bound "+
		"interface — a new kind of value is crossing the bridge", ident.Name)
}

// param is one named parameter or result, with its group flattened out.
type param struct {
	name string
	expr ast.Expr
}

// flattenParams expands Go's grouped declarations — `func f(a, b string)` is
// one *ast.Field with two names — into the one-per-parameter list the Swift
// spelling needs.
func flattenParams(list *ast.FieldList) []param {
	if list == nil {
		return nil
	}
	var out []param
	for _, f := range list.List {
		if len(f.Names) == 0 {
			out = append(out, param{expr: f.Type})
			continue
		}
		for _, n := range f.Names {
			out = append(out, param{name: n.Name, expr: f.Type})
		}
	}
	return out
}

// bindableFuncDecls is bindableFuncs with the declarations kept, for the
// checks that need the signature rather than only the name. The two are
// deliberately separate: the name check must keep working even if a
// signature stops being derivable, since a missing declaration is the worse
// failure of the two.
func bindableFuncDecls(t *testing.T) []*ast.FuncDecl {
	t.Helper()

	var out []*ast.FuncDecl
	for _, file := range parseMobilePackage(t) {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() {
				continue
			}
			if bindableSignature(fn.Type) {
				out = append(out, fn)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name.Name < out[j].Name.Name })
	return out
}

// boundInterface is one exported interface and the methods a shell must
// implement to conform to its protocol.
type boundInterface struct {
	name    string
	methods []*ast.Field
}

// boundInterfaceDecls is boundInterfaces with the method lists kept.
func boundInterfaceDecls(t *testing.T) []boundInterface {
	t.Helper()

	var out []boundInterface
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
				iface, isIface := ts.Type.(*ast.InterfaceType)
				if !isIface {
					continue
				}
				out = append(out, boundInterface{ts.Name.Name, iface.Methods.List})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// interfaceSet is the bound interface names, for swiftType's lookup.
func interfaceSet(t *testing.T) map[string]bool {
	t.Helper()

	set := map[string]bool{}
	for _, name := range boundInterfaces(t) {
		set[name] = true
	}
	return set
}

// methodName is an interface method's Go name, or "" for an embedded
// interface — which is a *ast.Field with a type and no names, the same shape
// an anonymous struct field has.
func methodName(f *ast.Field) string {
	if len(f.Names) == 0 {
		return ""
	}
	return f.Names[0].Name
}

// methodType is the signature of an interface method field.
func methodType(f *ast.Field) *ast.FuncType {
	if ft, ok := f.Type.(*ast.FuncType); ok {
		return ft
	}
	// Unreachable for a named method: a method field's type is always a
	// FuncType. Kept total rather than asserted so a malformed tree fails as
	// a signature mismatch rather than as a panic in a test helper.
	return &ast.FuncType{Params: &ast.FieldList{}}
}

// stubDeclHeader returns the stub's declaration for name, with any body cut.
func stubDeclHeader(src, name string) (string, bool) {
	for _, m := range swiftDecl.FindAllStringSubmatchIndex(src, -1) {
		if src[m[2]:m[3]] != name {
			continue
		}
		return strings.TrimSpace(cutBody(src[m[0]:lineEnd(src, m[0])])), true
	}
	return "", false
}

// protocolBody returns the text between a protocol's braces.
//
// Brace counting rather than a regexp: a protocol body holds declarations,
// and the first `}` at column zero would be the right answer only for as long
// as nobody indents one. The stub is small enough that either would work
// today, which is exactly when it is cheap to pick the one that keeps working.
func protocolBody(src, name string) (string, bool) {
	at := strings.Index(src, "public protocol "+name)
	if at < 0 {
		return "", false
	}
	open := strings.Index(src[at:], "{")
	if open < 0 {
		return "", false
	}
	depth := 0
	for i := at + open; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[at+open+1 : i], true
			}
		}
	}
	return "", false
}

// cutBody drops a Swift declaration's body, which starts at the first brace:
// no parameter type in this bridge's surface contains one.
func cutBody(line string) string {
	if i := strings.IndexByte(line, '{'); i >= 0 {
		return line[:i]
	}
	return line
}

// lineEnd is the index just past the line containing from.
func lineEnd(s string, from int) int {
	if i := strings.IndexByte(s[from:], '\n'); i >= 0 {
		return from + i
	}
	return len(s)
}

// lowerFirst lowercases the first character, which is gobind's Go-method to
// ObjC-selector rule.
//
// The first *byte*, not the first word: gobind lowercases one character, so
// an initialism keeps its remaining capitals (URLChanged -> uRLChanged, which
// is ugly and is what it emits). Every method on this bridge starts with an
// ASCII capital, and a non-ASCII one would not be an exported Go method name.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	if c := s[0]; c >= 'A' && c <= 'Z' {
		return string(c+'a'-'A') + s[1:]
	}
	return s
}
