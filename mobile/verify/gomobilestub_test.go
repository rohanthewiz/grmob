package verify

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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
//
// Short, but no longer *open*. What was outside this set used to be decided by
// hand, and the standard was "does this bridge use it" — which is the wrong
// question, because the cost of leaving a type out is paid by a bridge function
// that grows one later, and paid in silence. Every type gobind carries is now
// classified: spelled here, or refused with a reason in gobindCarriesUnused,
// and the set itself is read out of gobind's own isSupported by
// TestEveryTypeGobindCarriesIsClassified. A type in neither table fails that
// test, and a bridge function using a refused one fails by name — so the
// position `error` sat in is a position nothing can occupy again.
var bindableGoTypes = map[string]bool{
	"string": true,
	"bool":   true,
	"int":    true,
	// error is here because gobind binds it, not because the bridge uses it.
	// That distinction matters: this set decides which functions the stub is
	// *required* to declare, so a type gobind carries and this set omits is a
	// bridge function with no stub declaration and no type-check — the exact
	// hole TestTheGomobileStubMatchesTheBoundGoSurface exists to close.
	//
	// It was omitted, and the omission was invisible because nothing in
	// `mobile` returns an error. `func F() error` binds as
	// `BOOL F(NSError**)`, and `func F(e error)` as `void F(NSError*)`; both
	// were verified against a real `gomobile bind` rather than reasoned about.
	// See swiftResults for what a signature carrying one turns into.
	//
	// It is also the reason gobindCarriesUnused exists rather than a fourth,
	// fifth and sixth row here. `error` was findable because somebody thought
	// to look; the fixed-width numerics and []byte are in exactly its old
	// position, and the answer is not to guess their spellings but to make a
	// function that uses one fail loudly until somebody runs a bind.
	"error": true,

	// The fixed-width numerics, and rune, which is int32 under another name.
	//
	// They were in gobindCarriesUnused until their Swift spellings could be
	// read, which is the only thing that ever kept them out. The refusal's own
	// text said so: their PARAMETER spellings were legible off
	// bind/testdata/basictypes.objc.h.golden and it was the out-pointer a
	// (value, error) result moves into that nobody had read — so a bridge
	// function taking an int8 and returning nothing was refused for a reason
	// that did not apply to it, and told to go and run `gomobile bind` on a Mac
	// before it could exist.
	//
	// ios/verify/importer.swift is that reading, and it is a reading rather
	// than a derivation: gobind's C is declared as gobind spells it and the
	// Swift compiler says what it imports as. Its control is that it reproduces
	// the two spellings that WERE read off a real bind, including
	// UnsafeMutablePointer<ObjCBool>, which is the answer no rule stated here
	// would have produced. See TestTheSwiftSpellingsAreReadOffTheImporter.
	//
	// A narrow bridge surface is still the intent, and it is not what this set
	// enforces — this set decides which functions the stub is *required* to
	// declare, so a carried type omitted from it is a bridge function with no
	// stub and no type-check. Refusing a type whose spelling is legible was
	// enforcing the design by accident, and the sentence it printed while doing
	// so ("bind a package with this shape and add the rows from what it
	// produced") was asking for the one thing the file forbids.
	"int8":    true,
	"int16":   true,
	"int32":   true,
	"int64":   true,
	"float32": true,
	"float64": true,
	"rune":    true,

	// []byte, which was the last row in gobindCarriesUnused whose refusal had
	// an answer.
	//
	// Its stated reason was never about the spelling: gobind's own golden has
	// `NSData* _Nullable BasictypesByteArrays(NSData* _Nullable x)`, so the C
	// was legible in both positions. What was missing was the two-result split
	// — "a nullable first result stays the return where a scalar moves into an
	// out-pointer, and no golden exercises ([]byte, error)" — and the
	// instruction it printed was to go and run a bind.
	//
	// Both halves are readings now, and neither needs one. NSData* is nullable
	// in gobind's golden and isNullableType is what funcSummary splits on, so
	// the value stays the return; ios/verify/importer.swift asks the compiler
	// what NSData* is called in Swift, and what the error convention does to a
	// method that returns a nullable object. That last one had been prose since
	// somebody ran a bind once — it is the arm `(Iface, error)` already used —
	// and it is settled locally now, on a machine with no gomobile.
	//
	// The refusal that remains is uint8 and its alias, and it is a refusal
	// about the C rather than about the Swift: gobind spells a bare `byte` that
	// nothing it emits declares, so there is no C to hand the importer.
	goByteSlice: true,
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
	src := codeIn(t, gomobileStub)

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
			ok, refuse := bindableSignature(fn.Type)
			if refuse != "" {
				t.Errorf("mobile.%s carries a type `gomobile bind` binds and this file "+
					"cannot spell, so it is about to be dropped out of the stub check "+
					"without a word — the shape `error` was in.\n\n%s",
					fn.Name.Name, refuse)
				continue
			}
			if ok {
				out = append(out, fn.Name.Name)
			}
		}
	}
	sort.Strings(out)
	return out
}

// bindableSignature reports whether every parameter and result is a type
// gobind can carry AND this file can spell: one of bindableGoTypes, or an
// interface declared in this package (which binds as a protocol).
//
// The second return is the reason a signature gobind *does* carry cannot be
// checked here, and it is the whole difference between this and the two-valued
// version it replaces. See gobindCarriesUnused: a type in that table is bound
// by `gomobile bind`, so the symbol exists and the stub owes a declaration for
// it — and answering "not bindable" for one would drop the function out of both
// halves of this file without a word. That is the shape `error` was in.
func bindableSignature(sig *ast.FuncType) (bool, string) {
	ok := true
	refuse := ""
	fields := append([]*ast.Field{}, sig.Params.List...)
	if sig.Results != nil {
		fields = append(fields, sig.Results.List...)
	}
	for _, f := range fields {
		if spelled, why := carriedButUnspelled(f.Type); spelled {
			ok = false
			if refuse == "" {
				refuse = why
			}
			continue
		}
		name, keyed := goTypeKey(f.Type)
		if !keyed {
			// A pointer, a func, a map, a qualified type from another
			// package — all of them stop the bind.
			ok = false
			continue
		}
		if !bindableGoTypes[name] && !ast.IsExported(name) {
			ok = false
		}
	}
	return ok, refuse
}

// carriedButUnspelled reports whether a parameter or result type is one
// `gomobile bind` carries and this file declines to spell, and why.
//
// Two shapes, because gobind carries two: a named basic type (uint8 today) and
// []byte, which is the one slice its generators handle.
func carriedButUnspelled(expr ast.Expr) (bool, string) {
	switch t := expr.(type) {
	case *ast.Ident:
		why, listed := gobindCarriesUnused[t.Name]
		return listed, why
	case *ast.ArrayType:
		if t.Len != nil {
			return false, "" // an array, which gobind does not carry
		}
		elem, ok := t.Elt.(*ast.Ident)
		if !ok {
			return false, ""
		}
		if elem.Name != "byte" && elem.Name != "uint8" {
			return false, ""
		}
		// Looked up rather than assumed refused. []byte is spelled now, so this
		// arm answers "no" — and it keeps the lookup so that moving the row back
		// into gobindCarriesUnused is one edit rather than two.
		why, listed := gobindCarriesUnused[goByteSlice]
		return listed, why
	}
	return false, ""
}

// goByteSlice is how the one slice type gobind carries is keyed in
// gobindCarriesUnused. A key rather than a special case so the totality test
// below can hold it to gen.go's own Slice arm like every other row.
const goByteSlice = "[]byte"

// gobindCarriesUnused names every type `gomobile bind` carries that this file
// does not spell, and why not.
//
// # Why this table has to exist
//
// bindableGoTypes is deliberately short: this bridge is written to a narrow
// surface, and a parameter type outside that set is meant to be the signal that
// a new kind of value is crossing the FFI. What "outside that set" used to mean
// was "bindableSignature says false", which is the same answer it gives to a
// *core.Context — and those are not the same situation at all:
//
//	gobind cannot carry it     the generated header says "skipped function ...
//	                           with unsupported parameter or return types".
//	                           There is no symbol. A stub declaring one would be
//	                           offering the shell a call that cannot exist.
//
//	gobind carries it and      `gomobile bind` produces the symbol. The stub is
//	this file cannot spell it  never asked to declare it, GomobileBridge.swift
//	                           silently drops out of the type-check for it, and
//	                           nothing in this repository says so.
//
// The second is exactly what happened to `error`, which was in neither table on
// the reasonable-looking grounds that no bridge function returns one. The
// difference now is that "reasonable-looking grounds" is no longer how the
// question is settled: TestEveryTypeGobindCarriesIsClassified derives the
// carried set from gobind's own isSupported and requires every member to be in
// one table or the other, so the next type in `error`'s position is a failure
// on the day gen.go is read rather than on the day somebody notices.
//
// # Why these are unspelled rather than spelled
//
// Every reading in this file was taken off gobind's own source or its golden
// output, never derived (see gobindSwiftTypes). The rows below are the types
// where that source does not settle the answer, and inventing one is how a stub
// check starts agreeing with itself instead of with the toolchain.
//
// A bridge function using one of these does not go quietly: bindableSignature
// returns the reason, and the tests refuse the function by name and say what to
// do about it.
var gobindCarriesUnused = map[string]string{
	"uint8": "bind/genobjc.go's objcType maps Go uint8 to a bare `byte`, and " +
		"nothing gobind emits declares that type — it is in no golden header in " +
		"bind/testdata and Universe.objc.h does not define it. What Swift makes of " +
		"the result is therefore not readable from the generator, and every other " +
		"spelling in this file was",
	// The one alias left here, classified beside the kind it names because
	// bindableSignature reads the identifier a signature actually writes: a
	// parameter spelled `byte` is a uint8, and a table that knew only the
	// canonical spelling would let it through as "not carried". (`rune` was
	// the other, and it is in bindableGoTypes now — int32's spellings are read,
	// so its alias's are too.)
	"byte": "an alias for uint8, which gobind spells as a bare `byte` nothing " +
		"it emits declares — see the uint8 row",
}

// The fixed-width numerics used to be six more rows above, and this is what
// happened to them.
//
// Their refusal read: the parameter spellings are legible —
// bind/testdata/basictypes.objc.h.golden has `BasictypesInts(int8_t x, int16_t
// y, int32_t z, int64_t t, long u)` — and the RESULT spelling is the gap,
// because a C scalar is not nullable, so funcSummary moves a (value, error)
// pair's first result into an out-parameter and makes the return BOOL, and the
// Swift spelling of that pointer is what gobindErrorOutPointer holds. Both rows
// in it were read off a real bind, and one is UnsafeMutablePointer<ObjCBool>,
// which no rule stated here would have predicted. Deriving the other six from
// the two would have been assuming the case that already surprised us once.
//
// All of that was right, and the conclusion drawn from it was wrong. The
// refusal was type-level where the gap was position-level, so a bridge function
// taking an int8 and returning nothing was refused for a reason that did not
// apply to it — and the instruction it printed, "bind a package with this shape
// and add the rows from what it produced", made the next such function wait on
// somebody owning a Mac.
//
// What was missing was not a bind. It was a way to ask the Swift importer
// anything at all: gobind emits Objective-C, the shell writes Swift, and
// nothing here could see the step between them. ios/verify/importer.swift is
// that step, declared in gobind's own C spelling and settled by the compiler,
// with the two already-known rows as its control. So the six are spelled now,
// in both positions. []byte followed them, one session later and for the same
// reason turned up in the same place: its refusal named the two-result split as
// the gap, and the split is isNullableType — one line of the generator, which
// makes a slice nullable because nil is assignable to it. So the value stays the
// return, and what its Swift name is, and what the error convention does to a
// method returning one, are both questions importer.swift asks a compiler.
//
// The refusal that remains is uint8 and its alias, and it is a refusal about the
// C rather than about the Swift: gobind spells a bare `byte` that nothing it
// emits declares, so there is no C spelling to hand the importer at all. That is
// a gap this arrangement cannot close and does not pretend to.

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
// split by position because gobind's nullability is not symmetric.
//
// # Where the facts come from
//
// Read off gobind's own generator, not off a header somebody once produced.
// `golang.org/x/mobile` is in this module — held there by go.mod's `tool`
// block, which pins the gobind that ios/verify's stub is written against — so
// the mapping is a file in the module cache rather than an artefact of a build
// nobody can rerun without Xcode:
//
//	bind/genobjc.go, objcParamType   a parameter's ObjC type
//	bind/genobjc.go, objcType        every other position, results included
//	bind/genobjc.go, funcSummary     how a result clause is shaped
//
// That provenance is what closed two refusals this file used to carry. Both
// said, in effect, "read it off Headers/Mobile.objc.h and add the row" — which
// made the next bridge function of either shape blocked on somebody having run
// a `gomobile bind` at least once, on a Mac with Xcode, for a fact that was
// sitting in the module cache the whole time. See swiftType and swiftResults.
//
// gobindVersion below pins the version those readings were made against.
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
//
// The asymmetry is `string` and nothing else, which is worth stating because
// two columns imply otherwise. objcParamType special-cases exactly one Go type
// — String — and falls through to objcType for everything else, so every other
// type in the language is spelled identically in both positions. That is what
// makes the returned-interface row below answerable rather than a gap.
var gobindSwiftTypes = map[string]struct{ param, result string }{
	"string": {"String?", "String"},
	"bool":   {"Bool", "Bool"},
	"int":    {"Int", "Int"},
	// error has no result spelling, and the empty string is the statement of
	// that rather than an oversight. A Go error never appears as a Swift result
	// type: it becomes a trailing NSError** the importer either turns into
	// `throws` or leaves as a pointer parameter, which is swiftResults' whole
	// subject. swiftType refuses an empty spelling rather than returning it, so
	// this cell cannot be reached by accident.
	//
	// As a *parameter* it is an ordinary type — `NSError* _Nullable`, which
	// Swift imports as `(any Error)?`.
	"error": {"(any Error)?", ""},

	// The fixed-width numerics. One column's worth of answer twice over, for
	// the reason stated above: objcParamType special-cases String and nothing
	// else, so every other type is spelled identically in both positions.
	//
	// Every one of these is a line of ios/verify/importer.swift, which is where
	// the Swift names come from. `rune` is int32 under another name and gets
	// int32's spelling — gobind agrees, and says so in its own golden:
	// `FOUNDATION_EXPORT const int32_t BasictypesARune`.
	"int8":    {"Int8", "Int8"},
	"int16":   {"Int16", "Int16"},
	"int32":   {"Int32", "Int32"},
	"int64":   {"Int64", "Int64"},
	"float32": {"Float", "Float"},
	"float64": {"Double", "Double"},
	"rune":    {"Int32", "Int32"},

	// []byte. One column's answer twice over like the numerics, and for a
	// stronger reason than theirs: NSString* is the only type objcParamType
	// special-cases, and gobind's golden shows NSData* annotated _Nullable in
	// both positions on the same line.
	goByteSlice: {"Data?", "Data?"},
}

// gobindErrorOutPointer is the other half of gobind's two-result split, and the
// split itself: a first result listed here is *not* nullable in Objective-C, so
// funcSummary moves it into an out-parameter and makes the return BOOL. A first
// result that is not listed — a string, a bound interface, a []byte — stays the
// return.
//
// The value is the Swift spelling of that out-parameter. Both were read off a
// real bind rather than derived: `long*` imports as UnsafeMutablePointer<Int>?
// and `BOOL*` as UnsafeMutablePointer<ObjCBool>?, which is not the same as the
// `Bool` a plain BOOL parameter gives — ObjCBool is the C ABI's one-byte
// spelling, and it only surfaces through a pointer.
//
// Membership rather than a boolean column on gobindSwiftTypes, because the two
// facts are one: a type is in the non-nullable arm exactly when it has a
// pointer spelling to be moved into.
var gobindErrorOutPointer = map[string]string{
	"bool": "UnsafeMutablePointer<ObjCBool>?",
	"int":  "UnsafeMutablePointer<Int>?",

	// The rest of the non-nullable scalars, read the same way the two above
	// were — off the importer rather than off a rule — by
	// ios/verify/importer.swift. They are here because membership in this map
	// IS the claim that a type moves out of the return: a C scalar is not
	// nullable, so funcSummary has nowhere to put a nil, and every one of these
	// is a C scalar in gobind's own golden.
	//
	// The reason this map could not simply be completed before is the reason
	// the ObjCBool row is worth staring at: it is not what a plain BOOL
	// parameter imports as, and a map filled in by pattern from the two known
	// rows would have written UnsafeMutablePointer<Bool> and type-checked
	// against nothing.
	"int8":    "UnsafeMutablePointer<Int8>?",
	"int16":   "UnsafeMutablePointer<Int16>?",
	"int32":   "UnsafeMutablePointer<Int32>?",
	"int64":   "UnsafeMutablePointer<Int64>?",
	"float32": "UnsafeMutablePointer<Float>?",
	"float64": "UnsafeMutablePointer<Double>?",
	"rune":    "UnsafeMutablePointer<Int32>?",
}

// goErrorType is the Go spelling gobind treats as the bridge's error channel.
const goErrorType = "error"

func isGoError(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == goErrorType
}

// gobindVersion is the golang.org/x/mobile the mapping above and the two result
// rules in swiftResults were read out of.
//
// Pinned because the readings are of a *generator*, and a generator is a thing
// that changes. The pin is not a claim that a newer gobind is wrong — it is the
// moment at which somebody has to look, and the only such moment there is:
// nothing else in this repository would notice that a version bump had changed
// how a result clause is shaped, because the two shapes this file describes are
// both refusals and no bound function has either.
//
// The version lives in go.mod, held there by the `tool` block rather than by an
// import, which is also what keeps `go mod tidy` from dropping it.
const gobindVersion = "v0.0.0-20251021151156-188f512ec823"

// The two type tables answer for each other.
//
// bindableGoTypes decides which functions the stub is *required* to declare;
// gobindSwiftTypes decides how each of their types is spelled. They are two
// halves of one claim about what this bridge can carry, and they can disagree
// in both directions, silently:
//
//	in bindableGoTypes only    a function is required to have a declaration and
//	                           swiftType cannot build one, so every stub check
//	                           for it fails with a refusal rather than a diff
//	in gobindSwiftTypes only   the checker knows the spelling and does not ask
//	                           for the declaration — `gomobile bind` produces
//	                           the symbol, the stub omits it, and the app layer
//	                           silently drops out of the type-check for it
//
// The second is the one that has actually happened. `error` was in neither
// table, on the reasonable-looking grounds that no bridge function returns one
// — but gobind binds it (`func F() error` becomes `BOOL F(NSError**)`), so a
// bridge function that grew an error would have been judged unbindable here,
// gone undeclared in the stub, and taken GomobileBridge.swift's type-check with
// it. Nothing in the repository would have said so.
//
// Interfaces are deliberately outside this: bindableSignature admits any
// exported identifier, because a bound interface's spelling is derived from its
// name rather than looked up.
func TestTheBindableTypesAndTheirSpellingsAreTheSameSet(t *testing.T) {
	for name := range bindableGoTypes {
		if _, ok := gobindSwiftTypes[name]; !ok {
			t.Errorf("bindableGoTypes has %q and gobindSwiftTypes does not: a function "+
				"using it is required to have a stub declaration that swiftType cannot "+
				"build, so the check fails with a refusal instead of a difference", name)
		}
	}
	for name := range gobindSwiftTypes {
		if !bindableGoTypes[name] {
			t.Errorf("gobindSwiftTypes can spell %q and bindableGoTypes does not admit "+
				"it: `gomobile bind` produces the symbol, bindableFuncs skips it, the "+
				"stub is never asked to declare it, and the app layer stops being "+
				"type-checked for it without anything failing", name)
		}
	}
}

// The pinned gobind is the one in go.mod.
//
// A one-line test for a one-line fact, and the fact is the whole warrant for
// three of this file's tables. Reading go.mod rather than the module cache
// deliberately: the cache may hold several versions, and the question is which
// one this module builds against.
func TestTheGobindReadingsArePinnedToTheModulesOwnVersion(t *testing.T) {
	// nativeFile's own reach-up-two-levels, which lands on the module root
	// because this package is two directories down from it.
	raw, err := os.ReadFile(nativeFile("go.mod"))
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}
	if !strings.Contains(string(raw), "golang.org/x/mobile "+gobindVersion) {
		t.Errorf("go.mod no longer requires golang.org/x/mobile %s.\n\n"+
			"gobindSwiftTypes, swiftType's interface row and swiftResults' arms "+
			"rules are all readings of that version's bind/genobjc.go — objcParamType, "+
			"objcType and funcSummary. Re-read them against the new one and move this "+
			"constant, or the stub is being checked against a mapping the toolchain no "+
			"longer produces.", gobindVersion)
	}
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
	src := codeIn(t, gomobileStub)
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
	shape, err := swiftResults(sig, ifaces, false)
	if err != nil {
		return "", err
	}
	return "public func " + name + "(" + joinParams(params, shape.outParams) + ")" +
		shape.clause(), nil
}

// joinParams puts gobind's own out-parameters after the ones the Go signature
// declared, which is where the generated header puts them: the error pointer is
// always last, and a moved-out first result sits immediately before it.
func joinParams(declared string, added []string) string {
	all := append([]string{}, added...)
	if declared != "" {
		all = append([]string{declared}, added...)
	}
	return strings.Join(all, ", ")
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
	shape, err := swiftResults(sig, ifaces, true)
	if err != nil {
		return "", err
	}
	return "func " + name + "(" + joinParams(params, shape.outParams) + ")" +
		shape.clause(), nil
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

// resultShape is what gobind and the Swift importer between them make of a
// signature's results: extra parameters, a `throws`, and a return clause.
//
// Three fields because a Go result does not always become a Swift result. An
// error becomes a trailing NSError** out-parameter in the generated header, and
// what Swift then does with that depends on where the symbol sits — see
// swiftResults.
type resultShape struct {
	// outParams are the Swift parameters gobind adds after the declared ones,
	// already carrying their argument labels.
	outParams []string
	// throws replaces both the error parameter and (for a BOOL-returning
	// method) the return, which is what Clang's error convention does.
	throws bool
	// ret is " -> T", or empty for a Swift function returning nothing.
	ret string
}

// clause is everything after the closing parenthesis of a declaration.
func (r resultShape) clause() string {
	if r.throws {
		return " throws" + r.ret
	}
	return r.ret
}

// swiftResults renders what a signature's results do to a Swift declaration.
//
// isMethod says which of the two positions the symbol is in, and it changes the
// answer — see the error convention section below.
//
// # This used to be two refusals, and is not any more
//
// The previous version of this function refused every two-result signature and
// told the reader to run `gomobile bind` once and write the row from what it
// produced. That instruction was right and it was also the whole problem: the
// next bridge function of that shape was blocked on somebody having a Mac with
// Xcode, and the refusal's own text warned that writing the row on a guess was
// the one thing that must not happen.
//
// Somebody ran it. Every arm below was read off a real `gomobile bind -target=ios`
// of a package written to have one function of each shape, and then off what
// `swiftc` says the importer makes of the resulting module — not off the
// generator, and not off reasoning about Clang. The generator readings in
// TestTheResultArmsAreReadOffThePinnedGobind still stand and still matter, but
// they answer a different question: they say gobind still emits the header this
// was transcribed from.
//
// # The error convention, which is the whole of why isMethod exists
//
// The old refusal predicted that a package-level func would *not* import as
// `throws`, and that prediction was correct. gobind emits every package-level
// func as a plain C function (genFuncH: FOUNDATION_EXPORT ... s.asFunc(g)), and
// Clang's error convention — the rewrite of a trailing NSError** into a Swift
// `throws` — applies to Objective-C *methods*. A C function gets none of it
// without an explicit swift_error attribute, and gobind emits none:
//
//	func F() (string, error)
//	  ->  FOUNDATION_EXPORT NSString* _Nonnull MobileF(NSError* _Nullable* _Nullable error);
//	  ->  public func MobileF(_ error: NSErrorPointer) -> String
//
// A bound *interface* method is the other position, and there the convention
// does apply — with one exception that no amount of reading would have
// produced. The convention needs a return it can use to signal failure: BOOL,
// or a nullable object. A `(string, error)` method returns `NSString* _Nonnull`,
// which is neither, so it keeps its explicit error parameter and does not
// throw, while `(Iface, error)` returns a nullable object, throws, *and loses
// the optional* — nil is the error signal, so the imported return is
// non-optional.
//
// That last row is about the ANNOTATION and not about the Go type, which is why
// the arm asks nullableObjectResult rather than "is this an interface". A
// `([]byte, error)` result is `NSData* _Nullable` and takes the same arm, and
// ios/verify/importer.swift settles it with a compiler rather than with this
// paragraph: a protocol method of that exact shape, and a `let value: Data`
// that fails if the import kept the optional:
//
//	                        package func                       interface method
//	() error                (_ error:) -> Bool                 throws
//	(string, error)         (_ error:) -> String               (error:) -> String
//	(Iface, error)          (_ error:) -> MobileXProtocol?     throws -> MobileXProtocol
//	([]byte, error)         (_ error:) -> Data?                throws -> Data
//	(int, error)            (_ ret0:, _ error:) -> Bool        (ret0_:) throws
//	(bool, error)           (_ ret0:, _ error:) -> Bool        (ret0_:) throws
//
// The labels differ for the reason swiftMethodSignature gives: a C function
// imports with no argument labels at all, and a method's labels are its
// selector pieces, which gobind spells `error:` and `ret0_:`.
//
// # What is still refused
//
// Three or more results, and a two-result signature whose second result is not
// an error. Both are refused *by gobind itself* — verified, not read: it stops
// with "too many result values" and "second result value must be of type
// error" respectively, and builds nothing. So neither is a mapping this table
// is missing; the fix is to change the Go signature.
func swiftResults(sig *ast.FuncType, ifaces map[string]bool, isMethod bool) (resultShape, error) {
	if sig.Results == nil || len(sig.Results.List) == 0 {
		return resultShape{}, nil
	}
	results := flattenParams(sig.Results)
	if len(results) > 2 {
		return resultShape{}, fmt.Errorf("%d results; gobind refuses more than two "+
			"outright (bind/genobjc.go, funcSummary: \"too many result values\"), so this "+
			"is not a mapping this table is missing — `gomobile bind` will not build the "+
			"function at all. Change the Go signature", len(results))
	}
	if len(results) == 2 && !isGoError(results[1].expr) {
		return resultShape{}, fmt.Errorf("two results and the second is not an error; " +
			"gobind refuses that shape outright (\"second result value must be of type " +
			"error\") and builds nothing. A bound symbol returns zero or one values, and " +
			"optionally an error. Change the Go signature")
	}
	if isGoError(results[0].expr) && len(results) == 2 {
		return resultShape{}, fmt.Errorf("both results are errors; nothing here has " +
			"asked a real `gomobile bind` what it does with that, and every other arm " +
			"in this function was read off one. Change the Go signature, or bind a " +
			"package with this shape in it and add the arm from what it produced")
	}

	// The ordinary single result: no error anywhere, so the whole answer is
	// the type's own result spelling.
	if len(results) == 1 && !isGoError(results[0].expr) {
		typ, err := swiftType(results[0].expr, ifaces, true)
		if err != nil {
			return resultShape{}, err
		}
		return resultShape{ret: " -> " + typ}, nil
	}

	// Everything below carries an error, which is a parameter in the header
	// whatever else happens. The label is the one thing the two positions
	// always disagree about.
	errParam := "_ error: NSErrorPointer"
	retParam := "_ ret0: "
	if isMethod {
		errParam = "error: NSErrorPointer"
		retParam = "ret0_: "
	}

	if len(results) == 1 { // an error and nothing else
		if isMethod {
			// BOOL errOnly:error: — the convention's textbook shape.
			return resultShape{throws: true}, nil
		}
		return resultShape{outParams: []string{errParam}, ret: " -> Bool"}, nil
	}

	typ, err := swiftType(results[0].expr, ifaces, true)
	if err != nil {
		return resultShape{}, err
	}
	first, _ := goTypeKey(results[0].expr)
	if ptr, movedOut := gobindErrorOutPointer[first]; movedOut {
		// Not nullable in ObjC, so the value leaves through a pointer and the
		// return becomes BOOL — which is a return the error convention can
		// use, so a method throws and has no return at all.
		// grMobImportErrorConventionDropsABoolReturn is that arm, compiled.
		if isMethod {
			return resultShape{outParams: []string{retParam + ptr}, throws: true}, nil
		}
		return resultShape{
			outParams: []string{retParam + ptr, errParam},
			ret:       " -> Bool",
		}, nil
	}
	// Nullable in ObjC: the value stays the return.
	if isMethod && nullableObjectResult(first, ifaces) {
		// A nullable object return is the convention's other usable shape, and
		// the import drops the optional: nil is what signals the error, so it
		// can no longer also be a value.
		// grMobImportErrorConventionDropsTheOptional is that arm, compiled.
		return resultShape{throws: true, ret: " -> " + strings.TrimSuffix(typ, "?")}, nil
	}
	// A method returning NSString* _Nonnull is the one arm that keeps its error
	// parameter: the convention has no way to signal failure through a return
	// annotated non-null, so it declines to rewrite the method at all.
	// grMobImportErrorConventionKeepsTheErrorParameter is that arm, compiled.
	return resultShape{outParams: []string{errParam}, ret: " -> " + typ}, nil
}

// nullableObjectResult reports whether a first result is spelled as a nullable
// Objective-C object POINTER, which is what decides the interface-method arm.
//
// # Why this is not `ifaces[name]`
//
// It used to be, and the two were the same set while []byte was refused. They
// are not the same claim. The error convention needs a return it can signal
// failure through, and gobind's returns come in three kinds: a C scalar (moved
// into an out-pointer, so the return is BOOL — the convention's textbook
// shape), an object pointer annotated _Nonnull, and one annotated _Nullable.
// Only the last two stay the return, and only the last one gives the convention
// something to use.
//
//	string     NSString* _Nonnull  — objcType annotates a returned NSString*
//	                                 non-null, so a method keeps its explicit
//	                                 error parameter and does not throw
//	Iface      id<Proto> _Nullable — throws, and the import drops the optional
//	[]byte     NSData*   _Nullable — the same, and the reason this predicate
//	                                 exists rather than a membership test
//
// The distinction is gobind's annotation, not the Go type's kind, which is the
// same position-versus-type confusion the fixed-width numerics' refusal was.
//
// ios/verify/importer.swift settles all three consequences with a compiler, on
// one protocol carrying one method per arm: a `nullable NSData *` return
// assigned to a non-optional Data, a `nonnull NSString *` return whose method
// reference still takes an NSErrorPointer and does not throw, and a BOOL return
// whose method reference is a throwing nullary function with no result at all.
// Two of those three had been prose since somebody ran a bind once, and they
// are the two arms this predicate exists to send a method to.
func nullableObjectResult(name string, ifaces map[string]bool) bool {
	return ifaces[name] || name == goByteSlice
}

// goTypeKey names a type expression the way the tables above key it: an
// identifier by its own name, and the one slice gobind carries as "[]byte".
//
// A function rather than a type switch at each site, because three places ask
// the same question — is this a type the tables know — and two of them used to
// answer it with `expr.(*ast.Ident)`, which is "no" for every slice. That was
// right while []byte was refused and became a latent nil dereference the moment
// it was not: swiftResults reads `first.Name` off the assertion's result.
func goTypeKey(expr ast.Expr) (string, bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, true
	case *ast.ArrayType:
		if t.Len != nil {
			return "", false // a fixed-size array, which gobind does not carry
		}
		elem, ok := t.Elt.(*ast.Ident)
		if !ok || (elem.Name != "byte" && elem.Name != "uint8") {
			return "", false
		}
		return goByteSlice, true
	}
	return "", false
}

// swiftType maps one Go type onto its Swift spelling, in the given position.
//
// The position matters for one type. See gobindSwiftTypes for why, and for
// where the readings come from.
func swiftType(expr ast.Expr, ifaces map[string]bool, isResult bool) (string, error) {
	name, ok := goTypeKey(expr)
	if !ok {
		return "", fmt.Errorf("type is not one the tables key; gobind cannot bind it, so "+
			"bindableSignature should already have excluded this function (%T)", expr)
	}
	ident := &ast.Ident{Name: name}
	if m, ok := gobindSwiftTypes[ident.Name]; ok {
		if isResult {
			if m.result == "" {
				// error is the only row with an empty result cell, and this is
				// what makes the empty string a statement rather than a hole:
				// a Go error never becomes a Swift result type, and a caller
				// asking for one has skipped swiftResults' error arms.
				return "", fmt.Errorf("%s has no Swift result spelling: it is not "+
					"returned, it becomes a trailing NSError** that the importer "+
					"either rewrites into `throws` or leaves as a pointer parameter. "+
					"swiftResults is what handles it", ident.Name)
			}
			return m.result, nil
		}
		return m.param, nil
	}
	if ifaces[ident.Name] {
		// The same spelling in both positions, which used to be a refusal.
		//
		// It was refused on the grounds that the nullability of a returned
		// protocol is a fact about the generated header and this table may only
		// hold facts read off one — a good rule that turned out to be pointing
		// at the wrong document. The generator is in this module (see
		// gobindVersion), and it settles the question in two lines:
		//
		//	objcParamType   special-cases exactly one Go type, String, and
		//	                falls through to objcType for everything else
		//	objcType        an implementable bound interface is
		//	                `id<PrefixName> _Nullable`, in every position
		//
		// So the asymmetry this table's two columns exist for is `string` and
		// nothing else, and a returned protocol is `_Nullable` — Swift
		// `GrMobXProtocol?` — exactly as a parameter is.
		//
		// Still nothing returns one, and that is now a fact about this bridge
		// rather than a hole in the table: the next function that does gets a
		// checked declaration instead of a refusal telling it to go and run
		// `gomobile bind` on a Mac first.
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
			// The refusal is reported by bindableFuncs, which every test
			// calling this one also calls; reporting it twice would name the
			// same function in two failures.
			if ok, _ := bindableSignature(fn.Type); ok {
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

// The shapes no bridge function has, exercised directly.
//
// None of them reaches the checks above, because `mobile` has no function of
// any of these shapes — which is exactly why they need a test of their own. A
// transcribed arm with no caller is indistinguishable from an untranscribed one
// until something asks it the question, and the whole point of transcribing
// them was that the *next* bridge function of one of these shapes should get a
// checked declaration rather than an instruction to go and run `gomobile bind`
// on a Mac.
//
// The declarations below were read off a real bind of a package written to have
// one function and one interface method of each shape; see swiftResults for the
// two positions and for why they differ.
func TestTheTableDescribesTheShapesNoBridgeFunctionHas(t *testing.T) {
	ifaces := interfaceSet(t)
	if len(ifaces) == 0 {
		t.Fatal("mobile declares no bound interfaces; this test has nothing to ask about")
	}
	// Any one of them: the spelling is a function of the name, and which name
	// is not the subject.
	iface := sortedNamesOf(ifaces)[0]

	// A returned bound interface. objcParamType special-cases String alone, so
	// every other type — a protocol included — is spelled by objcType in both
	// positions, and objcType makes an implementable interface _Nullable.
	want := gobindPrefix + iface + "Protocol?"
	for _, isResult := range []bool{false, true} {
		got, err := swiftType(&ast.Ident{Name: iface}, ifaces, isResult)
		if err != nil {
			t.Errorf("swiftType(%s, isResult=%v) refused it: %v\n\n"+
				"gobind spells a bound interface the same way in both positions; a "+
				"refusal here is the old one that sent readers to a header",
				iface, isResult, err)
			continue
		}
		if got != want {
			t.Errorf("swiftType(%s, isResult=%v) = %q, want %q", iface, isResult, got, want)
		}
	}

	// Every result shape, in both positions, as a whole declaration. Comparing
	// declarations rather than clauses is deliberate: an error result changes
	// the *parameter list* as well as the return, and a check that only looked
	// at the tail would pass a signature missing its NSErrorPointer.
	for _, tc := range []struct {
		sig      string
		wantFunc string // the package-level declaration, or "" if refused
		wantMeth string // the protocol method's, or "" if refused
		mustSay  string // for a refusal, a phrase the message has to carry
	}{
		{sig: "func()",
			wantFunc: "public func F()",
			wantMeth: "func m()"},
		{sig: "func(s string) string",
			wantFunc: "public func F(_ s: String?) -> String",
			wantMeth: "func m(_ s: String?) -> String"},
		// An error and nothing else. The method throws; the C function cannot,
		// so the pointer stays and BOOL is the return.
		{sig: "func() error",
			wantFunc: "public func F(_ error: NSErrorPointer) -> Bool",
			wantMeth: "func m() throws"},
		// The pair the old refusal was about. Neither position throws: the
		// function because it is a C function, the method because a _Nonnull
		// object return gives the error convention nothing to signal with.
		{sig: "func() (string, error)",
			wantFunc: "public func F(_ error: NSErrorPointer) -> String",
			wantMeth: "func m(error: NSErrorPointer) -> String"},
		// Not nullable, so the value leaves through a pointer and the return
		// becomes BOOL — which the method then spends on `throws`.
		{sig: "func() (int, error)",
			wantFunc: "public func F(_ ret0: UnsafeMutablePointer<Int>?, _ error: NSErrorPointer) -> Bool",
			wantMeth: "func m(ret0_: UnsafeMutablePointer<Int>?) throws"},
		{sig: "func() (bool, error)",
			wantFunc: "public func F(_ ret0: UnsafeMutablePointer<ObjCBool>?, _ error: NSErrorPointer) -> Bool",
			wantMeth: "func m(ret0_: UnsafeMutablePointer<ObjCBool>?) throws"},
		// An error as an ordinary argument, which gobind carries like any
		// other reference type.
		{sig: "func(e error)",
			wantFunc: "public func F(_ e: (any Error)?)",
			wantMeth: "func m(_ e: (any Error)?)"},
		// The slice, in both positions. NSData* is _Nullable either way, so the
		// value stays the return like a string's — and unlike a string's, the
		// method arm throws, because a nullable object return is something the
		// error convention can signal through and a _Nonnull one is not.
		// ios/verify/importer.swift is where both halves are settled.
		{sig: "func(b []byte) []byte",
			wantFunc: "public func F(_ b: Data?) -> Data?",
			wantMeth: "func m(_ b: Data?) -> Data?"},
		{sig: "func() ([]byte, error)",
			wantFunc: "public func F(_ error: NSErrorPointer) -> Data?",
			wantMeth: "func m() throws -> Data"},
		{sig: "func() (string, int, error)", mustSay: "refuses more than two"},
		{sig: "func() (string, string)", mustSay: "second is not an error"},
	} {
		expr, err := parser.ParseExpr(tc.sig)
		if err != nil {
			t.Fatalf("parsing %q: %v", tc.sig, err)
		}
		for _, pos := range []struct {
			what  string
			build func() (string, error)
			want  string
		}{
			{"a package function", func() (string, error) {
				return swiftFuncSignature("F", expr.(*ast.FuncType), ifaces)
			}, tc.wantFunc},
			{"an interface method", func() (string, error) {
				return swiftMethodSignature("m", expr.(*ast.FuncType), ifaces)
			}, tc.wantMeth},
		} {
			got, err := pos.build()
			if tc.mustSay == "" {
				if err != nil {
					t.Errorf("%s as %s: refused it: %v", tc.sig, pos.what, err)
				} else if got != pos.want {
					t.Errorf("%s as %s:\n\tgot  %s\n\twant %s", tc.sig, pos.what, got, pos.want)
				}
				continue
			}
			if err == nil {
				t.Errorf("%s as %s = %q and should have refused: gobind will not build "+
					"the symbol at all", tc.sig, pos.what, got)
				continue
			}
			if !strings.Contains(err.Error(), tc.mustSay) {
				t.Errorf("%s as %s: refused with %q, which does not mention %q — the "+
					"refusal has to say which shape gobind is rejecting, or the reader "+
					"is back to guessing", tc.sig, pos.what, err, tc.mustSay)
			}
		}
	}
}

// The three result arms, held to the generator they were read out of.
//
// # Why a source check and not prose
//
// swiftResults' two refusals, and the header shape its arms were transcribed
// from, are readings of one file
// — bind/genobjc.go in the gobind gobindVersion pins — and that file is in the
// module cache on any machine that has run `go mod download`. Until this test
// they were prose about it: a sentence naming funcSummary and isNullableType,
// with nothing anywhere able to notice that the version bump which moved
// gobindVersion had also moved what those functions do.
//
// gobindVersion's own doc says as much — "the moment at which somebody has to
// look, and the only such moment there is". This is what looking consists of,
// and it is cheap because the readings are all one line each.
//
// # What it can and cannot settle
//
// It settles that the *generator* still emits the header the result arms were
// transcribed from. It cannot settle what Swift makes of that header — whether
// a trailing NSError** on a C function becomes a `throws` — because that is a
// fact about Clang's importer and not about anything in the module cache.
//
// That question used to be the reason the two-result arms were refused. It is
// answered now, and it was answered the only way it could be: by running
// `gomobile bind` and asking `swiftc` what the module looks like (see
// swiftResults, and the rows in gomobile_stub.swift). What this test protects
// is the link between the two — a gobind that started emitting package
// functions as Objective-C *methods* would make every one of those readings
// wrong at once, and the first row below is what notices.
//
// # When it does not run
//
// It skips when the module cache has no copy, which is the stance ios/verify
// takes toward a missing iPhoneOS SDK: `go test ./...` must not fail on a
// machine that has the repository and not the download. The pin on go.mod
// (TestTheGobindPinIsTheOneInGoMod) is what still runs there.
func TestTheResultArmsAreReadOffThePinnedGobind(t *testing.T) {
	src, ok := gobindSource(t, "bind", "genobjc.go")
	if !ok {
		return
	}
	types, _ := gobindSource(t, "bind", "types.go")

	for _, c := range []struct {
		what, want, why string
	}{
		{
			what: "a package-level func is emitted as a C function",
			want: `g.Printf("FOUNDATION_EXPORT %s;\n", s.asFunc(g))`,
			why: "this is the whole reason the two-result arm cannot simply be " +
				"transcribed as a Swift `throws` function: Clang's error convention is " +
				"the Objective-C *method* one, and every symbol this bridge exports is " +
				"a package-level func. If gobind has started emitting these as methods, " +
				"the throwing spelling is suddenly the right one and swiftResults' " +
				"refusal is describing a generator that no longer exists",
		},
		{
			what: "the two-result split is on nullability",
			want: "if isNullableType(typ) {",
			why: "swiftResults says the first result stays the return when it is " +
				"nullable and becomes an out-parameter when it is not. That split is " +
				"this line",
		},
		{
			what: "a non-nullable first result returns BOOL",
			want: `s.ret = "BOOL" // Return is not nullable`,
			why:  "the other half of the same split, and the half that adds a parameter",
		},
		{
			what: "three or more results are refused outright",
			want: `g.errorf("too many result values: %s", f)`,
			why: "swiftResults tells a reader to change the Go signature rather than to " +
				"go and find a mapping, on the strength of gobind refusing the shape. " +
				"A gobind that had relaxed this (the TODO beside it says it might) " +
				"would make that advice wrong",
		},
	} {
		if !strings.Contains(src, c.want) {
			t.Errorf("bind/genobjc.go at gobind %s no longer contains %s:\n\t%s\n\n%s",
				gobindVersion, c.what, c.want, c.why)
		}
	}

	// isNullableType is in a different file, and it is the predicate the split
	// above turns on. Both of its clauses matter, and they cover the two
	// nullable things this bridge's surface can carry:
	//
	//	the string clause   what puts a (string, error) pair in the first arm
	//	                    rather than the second
	//	the nil clause      types.AssignableTo(UntypedNil, t), which is what
	//	                    makes a SLICE nullable — and therefore what says a
	//	                    ([]byte, error) result stays the return instead of
	//	                    moving into an out-pointer. That was the whole of
	//	                    []byte's refusal, and it is one line of the
	//	                    generator rather than a bind somebody has to run.
	for _, c := range []struct{ want, why string }{
		{`t.String() == "string"`,
			"a Go string is nullable because NSString* is, and that clause is what " +
				"decides which of gobind's two arms a (string, error) pair lands in — " +
				"swiftResults' two arms are named for it"},
		{"types.AssignableTo(types.Typ[types.UntypedNil].Underlying(), t)",
			"nil is assignable to a slice, so []byte is nullable and a ([]byte, error) " +
				"result keeps the value as the return. gobindErrorOutPointer's " +
				"membership is exactly the complement of this predicate, and []byte's " +
				"absence from it is this line"},
	} {
		if types != "" && !strings.Contains(types, c.want) {
			t.Errorf("bind/types.go at gobind %s no longer contains %s.\n\n%s",
				gobindVersion, c.want, c.why)
		}
	}
}

// gobindSource reads one file out of the pinned golang.org/x/mobile in the
// module cache.
//
// Returns ok=false after skipping, rather than failing, when the cache has no
// copy. A checkout is not broken because a module nothing imports has not been
// downloaded — golang.org/x/mobile is held in go.mod by the tool block and by
// nothing else, so `go build ./...` never fetches it.
func gobindSource(t *testing.T, parts ...string) (string, bool) {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Skipf("cannot read GOMODCACHE, so the pinned gobind is unreachable: %v", err)
		return "", false
	}
	path := filepath.Join(append([]string{
		strings.TrimSpace(string(out)), "golang.org", "x", "mobile@" + gobindVersion,
	}, parts...)...)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("golang.org/x/mobile@%s is not in the module cache, so the readings "+
			"in this file cannot be held to it here. `go mod download "+
			"golang.org/x/mobile` fetches it.", gobindVersion)
		return "", false
	}
	return string(raw), true
}

// --- The stub's own doc comment --------------------------------------------

// stubRules cuts the checked block out of gomobile_stub.swift's header
// comment: the lines between the two markers, with the comment prefix and the
// leading tab stripped.
//
// Anchored on both markers rather than on the first alone, because the failure
// of an unanchored cut is a block that silently grows: a marker somebody moved
// would take the rest of the file with it, and every row after the real end
// would be reported as malformed rather than as missing.
var stubRulesBlock = regexp.MustCompile(
	`(?s)//\t--- checked against mobile/verify/gomobilestub_test\.go ---\n(.*?)//\t--- end ---`)

// stubRule is one row of that block: a tag and the fields after it.
type stubRule struct {
	tag    string
	fields []string
	line   string // as written, for a failure to quote
}

func stubRules(t *testing.T, src string) []stubRule {
	t.Helper()

	m := stubRulesBlock.FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("%s: the header comment has no checked block. It is the half of that "+
			"comment that is held to this file rather than believed; if it was "+
			"deliberately removed, delete the tests below with it rather than "+
			"leaving them matching nothing", gomobileStub)
	}
	var out []stubRule
	for _, raw := range strings.Split(m[1], "\n") {
		line := strings.TrimPrefix(raw, "//")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		out = append(out, stubRule{tag: fields[0], fields: fields[1:], line: line})
	}
	return out
}

// The stub's header comment states the checker's own rules, and every row of
// it is held to the code that implements them.
//
// # Why the comment is a test subject at all
//
// The declarations in gomobile_stub.swift are pinned character-for-character
// by the two tests above. The comment over them was not: it is a hand-written
// description of gobind's naming, its nullability asymmetry and its three
// result arms, and it agreed with gobindSwiftTypes, swiftType and swiftResults
// on the day it was written because it was written from them.
//
// That is exactly the shape this whole file exists to refuse one level up. A
// copy of generated behaviour drifts, the drift is silent, and the reader who
// most needs the comment — someone adding a bridge function of a shape nothing
// has used yet — is the one who cannot tell that it has stopped being true.
// The signature checker would still fail them, eventually, in a message about
// a declaration rather than about the rule they had just read.
//
// So the load-bearing half of the comment is written as rows, and this is what
// reads them. Every row is compared against the thing it describes rather than
// against a second copy here: the version against go.mod's, the names against
// the constants the checker builds names from, the type rows against swiftType
// itself, and the result rows against what the checker does when handed a
// signature of that shape.
//
// The prose around the rows is not checked and is not meant to be. A paragraph
// that explains *why* nullability is asymmetric cannot be wrong in the way a
// row saying `string String? String` can.
func TestTheStubsHeaderStatesTheRulesTheCheckerUses(t *testing.T) {
	src := proseIn(t, gomobileStub)
	ifaces := interfaceSet(t)

	// Every tag the block may carry, and how many rows of it must be present.
	// Counting is what catches a row that was *deleted*: each check below
	// verifies the rows it finds, and a rule quietly dropped from the comment
	// would otherwise be verified by nothing.
	seen := map[string]int{}

	for _, r := range stubRules(t, src) {
		seen[r.tag]++
		switch r.tag {
		case "gobind":
			if len(r.fields) != 1 || r.fields[0] != gobindVersion {
				t.Errorf("%s: the header pins gobind %q, but the readings in this file "+
					"were made against %s. Re-read bind/genobjc.go against the new "+
					"version, or the comment is describing a generator nobody used",
					gomobileStub, r.line, gobindVersion)
			}
		case "prefix":
			if len(r.fields) != 1 || r.fields[0] != gobindPrefix {
				t.Errorf("%s: the header says the symbol prefix is %q; the checker "+
					"builds every name with %q", gomobileStub, r.line, gobindPrefix)
			}
		case "suffix":
			// Derived from a real interface rather than from a constant,
			// because there is no constant: the suffix is spelled inline
			// where the protocol names are built, and reading it back out of
			// one is what holds the comment to that spelling.
			name := sortedNamesOf(ifaces)[0]
			spelled, err := swiftType(&ast.Ident{Name: name}, ifaces, false)
			if err != nil {
				t.Fatalf("swiftType refused a bound interface: %v", err)
			}
			want := strings.TrimSuffix(strings.TrimPrefix(spelled, gobindPrefix+name), "?")
			if len(r.fields) != 1 || r.fields[0] != want {
				t.Errorf("%s: the header says a bound interface is suffixed %q; the "+
					"checker spells one %s, which is a %q suffix",
					gomobileStub, r.line, spelled, want)
			}
		case "type":
			checkStubTypeRule(t, r, ifaces)
		case "results":
			checkStubResultRule(t, r, ifaces)
		default:
			t.Errorf("%s: the checked block carries a row this test does not know how "+
				"to hold to anything: %q. An unrecognised tag is a claim nothing "+
				"verifies, which is what the block exists to stop",
				gomobileStub, r.line)
		}
	}

	// One row per scalar fact, one type row per row of gobindSwiftTypes plus
	// the interface rule, and one result row per arm swiftResults distinguishes
	// in each of the two positions, plus the two shapes gobind refuses.
	//
	// The result count is written out rather than derived, and there is nothing
	// to derive it from: the arms are branches in a function, not entries in a
	// table, which is itself the reason the rows exist. A branch added without
	// a row is a shape the comment does not describe, and this number is what
	// notices.
	for tag, want := range map[string]int{
		"gobind":  1,
		"prefix":  1,
		"suffix":  1,
		"type":    len(gobindSwiftTypes) + 1,
		"results": 14,
	} {
		if seen[tag] != want {
			t.Errorf("%s: the checked block has %d %q rows, want %d — a rule dropped "+
				"from the comment is verified by nothing, which is how the comment "+
				"drifted the first time", gomobileStub, seen[tag], tag, want)
		}
	}
}

// One `type` row: a Go spelling and its parameter and result spellings, held
// to swiftType.
//
// The interface row is written with placeholders — `<interface>` standing for
// any bound interface and `<Name>` for its name — and is checked by asking
// swiftType about an interface literally called `<Name>`. That is not a trick:
// the spelling is a pure function of the name, so substituting the placeholder
// *is* the general case, and a row written with a real interface's name would
// go stale the day that interface was renamed.
func checkStubTypeRule(t *testing.T, r stubRule, ifaces map[string]bool) {
	t.Helper()

	// Bar-separated, like the result rows and for the same reason: a Swift
	// spelling can contain a space — `(any Error)?` does — so counting fields
	// would split one cell into two and report the row as malformed.
	cells := strings.Split(r.line, "|")
	if len(cells) != 3 {
		t.Errorf("%s: type row %q has %d bar-separated cells, want 3 "+
			"(`type <go type> | <parameter> | <result>`)",
			gomobileStub, r.line, len(cells))
		return
	}
	goType := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(cells[0]), "type"))
	wantParam, wantResult := strings.TrimSpace(cells[1]), strings.TrimSpace(cells[2])

	set := ifaces
	name := goType
	if goType == "<interface>" {
		name = "<Name>"
		set = map[string]bool{name: true}
	} else if _, ok := gobindSwiftTypes[goType]; !ok {
		t.Errorf("%s: type row %q names %q, which has no row in gobindSwiftTypes and "+
			"is not the interface rule — the comment describes a mapping the "+
			"checker does not have", gomobileStub, r.line, goType)
		return
	}

	for _, c := range []struct {
		isResult bool
		want     string
	}{{false, wantParam}, {true, wantResult}} {
		got, err := swiftType(&ast.Ident{Name: name}, set, c.isResult)
		// "-" is the row's way of saying there is no spelling in that
		// position, which is a claim as much as a type is: a Go error is not
		// returned, it becomes the trailing pointer swiftResults handles. A
		// refusal is what must happen, so a checker that started producing one
		// is the failure.
		if c.want == "-" {
			if err == nil {
				t.Errorf("%s: the header says a Go %s has no result spelling, and the "+
					"checker produced %q. If gobind has started returning one, the "+
					"result rows below are describing a different generator",
					gomobileStub, goType, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: type row %q: swiftType refused %s: %v",
				gomobileStub, r.line, goType, err)
			continue
		}
		if got != c.want {
			position := "parameter"
			if c.isResult {
				position = "result"
			}
			t.Errorf("%s: the header says a Go %s is %q in %s position; the checker "+
				"spells it %q. The stub's declarations follow the checker, so the "+
				"comment is describing a file that no longer looks like this",
				gomobileStub, goType, c.want, position, got)
		}
	}
}

// One `results` row: a position, a Go signature, and the declaration gobind and
// the Swift importer produce for it.
//
// # Why a whole declaration and not a return clause
//
// The row used to be a result *count* and a phrase, which was the right shape
// while every multi-result signature was refused: there was nothing to state
// but which refusal. Now that the arms are transcribed there is, and an error
// result changes the parameter list as well as the return — a row that named
// only the tail would be silent about the NSErrorPointer, which is the half a
// shell author actually has to type.
//
// So the row is the declaration, and it is built by the same two functions that
// build the stub's own: swiftFuncSignature and swiftMethodSignature. `F` and
// `m` are stand-in names, and the position column is which of the two to ask —
// they disagree, and the disagreement is the reason the rows exist.
//
// A `refused` row names a phrase the refusal must carry, on the old reasoning:
// comparing against the message rather than against a second copy of the rule
// keeps the comment pointing a reader at the shape gobind actually rejects.
func checkStubResultRule(t *testing.T, r stubRule, ifaces map[string]bool) {
	t.Helper()

	// Cut on the bar rather than on whitespace. A Go signature and a Swift
	// declaration both contain spaces, so field counting cannot separate them —
	// and the failure of a heuristic that tried would be a row silently checked
	// against the wrong half of itself.
	left, want, found := strings.Cut(r.line, "|")
	if !found {
		t.Errorf("%s: results row %q has no `|` — a row is "+
			"`results <position> <go signature> | <what it binds as>`",
			gomobileStub, r.line)
		return
	}
	want = strings.TrimSpace(want)
	fields := strings.Fields(left)
	if len(fields) < 3 {
		t.Errorf("%s: results row %q wants a position and a Go signature before the `|`",
			gomobileStub, r.line)
		return
	}
	position, sig := fields[1], strings.Join(fields[2:], " ")

	expr, err := parser.ParseExpr(sig)
	if err != nil {
		t.Errorf("%s: results row %q names %q, which does not parse as a Go signature: %v",
			gomobileStub, r.line, sig, err)
		return
	}
	fn, ok := expr.(*ast.FuncType)
	if !ok {
		t.Errorf("%s: results row %q names %q, which is not a func type",
			gomobileStub, r.line, sig)
		return
	}

	var got string
	switch position {
	case "func":
		got, err = swiftFuncSignature("F", fn, ifaces)
	case "method":
		got, err = swiftMethodSignature("m", fn, ifaces)
	case "refused":
		// Both positions, because a refusal is gobind's and gobind builds
		// neither: a shape refused for a function and quietly bound for a
		// method would be a row that is only half true.
		for _, pos := range []struct {
			what  string
			build func() (string, error)
		}{
			{"a package function", func() (string, error) { return swiftFuncSignature("F", fn, ifaces) }},
			{"an interface method", func() (string, error) { return swiftMethodSignature("m", fn, ifaces) }},
		} {
			bound, ferr := pos.build()
			if ferr == nil {
				t.Errorf("%s: the header says %q is refused, and the checker binds it as "+
					"%s for %s", gomobileStub, sig, bound, pos.what)
				continue
			}
			if !strings.Contains(ferr.Error(), want) {
				t.Errorf("%s: the header says %q is refused because %q; for %s the "+
					"checker says %q. The comment sends a reader to the wrong shape",
					gomobileStub, sig, want, pos.what, ferr)
			}
		}
		return
	default:
		t.Errorf("%s: results row %q names position %q; the two positions a bound "+
			"symbol can be in are `func` and `method`, and `refused` is the third row "+
			"kind", gomobileStub, r.line, position)
		return
	}
	if err != nil {
		t.Errorf("%s: the header says %q binds as %q, and the checker refuses it: %v",
			gomobileStub, sig, want, err)
		return
	}
	if got != want {
		t.Errorf("%s: the header says %q binds as\n\t%s\nand the checker produces\n\t%s",
			gomobileStub, sig, want, got)
	}
}

// --- What gobind carries, read out of gobind ---------------------------------

// gobindBasicKinds maps a go/types Basic kind, as gobind's own isSupported
// spells it, onto the Go type names a signature can write for it.
//
// Two things make this a table rather than a lowercasing:
//
//	the aliases      byte is uint8 and rune is int32, and bindableSignature
//	                 reads the identifier a signature actually spells. gobind's
//	                 switch names the kinds (with `// types.Byte` and
//	                 `// types.Rune` in the comments); a Go author writes either.
//
//	the untyped ones isSupported admits UntypedBool, UntypedInt, UntypedRune,
//	                 UntypedFloat and UntypedString, and none of them can appear
//	                 in a declared signature — they are what a constant
//	                 expression has before assignment, which is why gobind lists
//	                 them (it binds exported constants too). They map to no type
//	                 name, and that is a statement rather than an omission: a
//	                 kind missing from this table is a hard failure below.
var gobindBasicKinds = map[string][]string{
	"Bool":    {"bool"},
	"Int":     {"int"},
	"Int8":    {"int8"},
	"Uint8":   {"uint8", "byte"},
	"Int16":   {"int16"},
	"Int32":   {"int32", "rune"},
	"Int64":   {"int64"},
	"Float32": {"float32"},
	"Float64": {"float64"},
	"String":  {"string"},

	"UntypedBool":   nil,
	"UntypedInt":    nil,
	"UntypedRune":   nil,
	"UntypedFloat":  nil,
	"UntypedString": nil,
}

// Every type gobind carries is in exactly one of the two tables.
//
// # The hole this closes
//
// bindableGoTypes decides which functions the stub is *required* to declare.
// Until here, what it did NOT contain was decided by hand, and the standard was
// "does this bridge use it" — which is the wrong question, because the cost of
// omitting a type is paid by a bridge function that grows one later. `error`
// was left out on exactly that reasoning and the omission was invisible: gobind
// binds it, so a function returning one would have produced a symbol, gone
// undeclared in the stub, and taken GomobileBridge.swift's type-check with it,
// with nothing in the repository saying so.
//
// The fix is not a longer hand-written list, because a longer hand-written list
// has the same failure mode one type further along. It is to stop deciding: the
// carried set is read out of `bind/gen.go`'s own isSupported, and every member
// of it has to be classified — spelled (bindableGoTypes) or refused with a
// reason (gobindCarriesUnused). A type in neither fails here.
//
// # Why isSupported and not objcType
//
// They disagree, and isSupported is the gate. genobjc.go's objcType can spell
// uint16, uint32 and uint64; isSupported does not admit them, so a function
// carrying one is skipped before any spelling is asked for. Deriving the set
// from the speller rather than the gate would demand classifications for three
// types `gomobile bind` will not bind.
//
// # What is not derived here
//
// The two non-basic carriers that are not slices: `error`, which isSupported
// admits through isErrorType before the switch, and exported named interfaces
// and pointers, which it admits through validPkg. Both are in bindableSignature
// already — error by name, the named types by ast.IsExported — and neither is a
// set that can grow a member without somebody writing it in this package.
func TestEveryTypeGobindCarriesIsClassified(t *testing.T) {
	src, ok := gobindSource(t, "bind", "gen.go")
	if !ok {
		return
	}

	basic, slices := gobindSupportedKinds(t, src)

	for _, kind := range basic {
		names, known := gobindBasicKinds[kind]
		if !known {
			t.Errorf("bind/gen.go's isSupported admits types.%s at gobind %s and "+
				"gobindBasicKinds cannot name it. gobind has grown a Go type it can "+
				"carry that nothing here has classified, which is `error`'s position "+
				"exactly: a bridge function using it would bind, go undeclared in the "+
				"stub, and stop being type-checked without a word.", kind, gobindVersion)
			continue
		}
		for _, name := range names {
			_, spelled := bindableGoTypes[name]
			_, refused := gobindCarriesUnused[name]
			switch {
			case spelled && refused:
				t.Errorf("%q is in bindableGoTypes and in gobindCarriesUnused. The two "+
					"say opposite things — one requires a stub declaration and the "+
					"other refuses to check one — and swiftType would build a spelling "+
					"for a type this file has said it cannot spell", name)
			case !spelled && !refused:
				t.Errorf("`gomobile bind` carries Go %s (bind/gen.go isSupported, "+
					"types.%s) and neither bindableGoTypes nor gobindCarriesUnused "+
					"mentions it.\n\n"+
					"A bridge function using it binds to a real symbol, is never asked "+
					"for a stub declaration, and takes GomobileBridge.swift's "+
					"type-check with it. That is what happened to `error`.\n\n"+
					"Add it to bindableGoTypes with its spellings in gobindSwiftTypes "+
					"(both read off a real bind, never derived), or to "+
					"gobindCarriesUnused with the reason its spelling is not readable "+
					"from the generator.", name, kind)
			}
		}
	}

	// The one slice gobind carries. Pinned as a set rather than looked for,
	// because the failure that matters is gobind growing a second one: a
	// []string parameter that started binding would be carried, unclassified
	// and — since bindableSignature reads an *ast.ArrayType only for a byte
	// element — silently skipped.
	if len(slices) != 1 || slices[0] != "Uint8" {
		t.Errorf("bind/gen.go's isSupported admits slices of %v at gobind %s, and this "+
			"file is written for []byte alone: carriedButUnspelled recognises a byte "+
			"element and nothing else, so any other slice type is carried and silently "+
			"skipped. Classify it and teach carriedButUnspelled to see it.",
			slices, gobindVersion)
	}
	if _, refused := gobindCarriesUnused[goByteSlice]; !refused {
		if _, spelled := bindableGoTypes[goByteSlice]; !spelled {
			t.Errorf("gobind carries []byte and neither table mentions it")
		}
	}
}

// gobindSupportedKinds reads isSupported's two switches: the go/types Basic
// kinds it admits, and the slice element kinds.
//
// Parsed rather than grepped. The kinds are spelled `types.Bool` in a case
// clause and `types.Uint8` in a comparison, and a regexp over the file would
// also collect the kinds objcType spells — which are a different, larger set
// (see the test above), so the derivation would demand classifications for
// types gobind will not bind.
func gobindSupportedKinds(t *testing.T, src string) (basic, slices []string) {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "gen.go", src, 0)
	if err != nil {
		t.Fatalf("parsing the pinned gobind's bind/gen.go: %v", err)
	}

	var body *ast.BlockStmt
	for _, decl := range file.Decls {
		fn, isFunc := decl.(*ast.FuncDecl)
		if isFunc && fn.Name.Name == "isSupported" && fn.Recv != nil {
			body = fn.Body
		}
	}
	if body == nil {
		t.Fatalf("bind/gen.go at gobind %s has no (*Generator).isSupported. It is the "+
			"gate that decides which Go types `gomobile bind` will carry, and the whole "+
			"of what makes bindableGoTypes' completeness checkable rather than "+
			"believed. Find where the decision moved to and re-point this.", gobindVersion)
	}

	// The type switch inside it has one case per shape gobind handles. The
	// Basic case holds a second switch over kinds; the Slice case compares an
	// element's kind. Both are "every types.X mentioned under this case", which
	// is one walk with the case as the boundary.
	ast.Inspect(body, func(n ast.Node) bool {
		clause, isClause := n.(*ast.CaseClause)
		if !isClause || len(clause.List) != 1 {
			return true
		}
		star, isStar := clause.List[0].(*ast.StarExpr)
		if !isStar {
			return true
		}
		sel, isSel := star.X.(*ast.SelectorExpr)
		if !isSel {
			return true
		}
		switch sel.Sel.Name {
		case "Basic":
			// The kinds are the inner switch's own case lists:
			//
			//	case types.Bool, types.UntypedBool,
			//		types.Int,
			//		types.Int8, types.Uint8, // types.Byte
			basic = append(basic, caseListKinds(clause.Body)...)
		case "Slice":
			// The element kind is a comparison instead:
			//
			//	case *types.Basic:
			//		return e.Kind() == types.Uint8
			//
			// Read from the comparison rather than from every types.X under
			// the clause, which would also collect the `*types.Basic` the
			// element is switched on and report it as a carried kind.
			slices = append(slices, comparedKinds(clause.Body)...)
		}
		return true
	})

	if len(basic) == 0 {
		t.Fatalf("bind/gen.go at gobind %s: isSupported's *types.Basic case names no "+
			"kinds, so the carried set this test derives is empty and every "+
			"classification below would pass over nothing", gobindVersion)
	}
	sort.Strings(basic)
	sort.Strings(slices)
	return basic, slices
}

// caseListKinds collects the `types.X` a switch's case clauses select on.
func caseListKinds(body []ast.Stmt) []string {
	c := kindCollector{seen: map[string]bool{}}
	inspectStmts(body, func(n ast.Node) bool {
		clause, isClause := n.(*ast.CaseClause)
		if !isClause {
			return true
		}
		for _, expr := range clause.List {
			c.add(expr)
		}
		return true
	})
	return c.out
}

// comparedKinds collects the `types.X` a node compares something against with
// ==, which is how gobind spells the one element kind its Slice arm admits.
func comparedKinds(body []ast.Stmt) []string {
	c := kindCollector{seen: map[string]bool{}}
	inspectStmts(body, func(n ast.Node) bool {
		bin, isBin := n.(*ast.BinaryExpr)
		if !isBin || bin.Op != token.EQL {
			return true
		}
		c.add(bin.X)
		c.add(bin.Y)
		return true
	})
	return c.out
}

// inspectStmts is ast.Inspect over a case clause's body, which the AST gives as
// a bare statement slice rather than as a node.
func inspectStmts(body []ast.Stmt, f func(ast.Node) bool) {
	for _, stmt := range body {
		ast.Inspect(stmt, f)
	}
}

// kindCollector gathers `types.X` selector names, deduplicated and in order.
type kindCollector struct {
	seen map[string]bool
	out  []string
}

func (c *kindCollector) add(expr ast.Expr) {
	sel, isSel := expr.(*ast.SelectorExpr)
	if !isSel {
		return
	}
	pkg, isIdent := sel.X.(*ast.Ident)
	if !isIdent || pkg.Name != "types" || c.seen[sel.Sel.Name] {
		return
	}
	c.seen[sel.Sel.Name] = true
	c.out = append(c.out, sel.Sel.Name)
}

// The Swift spellings this file states must be the ones the importer gives.
//
// # The step nothing could see
//
// Every reading here is meant to come off gobind's source or its golden output,
// and one step of the chain is neither. gobind emits an Objective-C header; the
// shell writes Swift; what the shell sees is what the *importer* makes of that
// header, and no document in this module says what that is. So two rows of
// gobindErrorOutPointer were marked "read off a real bind" and every type that
// would have needed a third was refused with an instruction to go and produce
// one — which made the next bridge function of an ordinary shape wait on
// somebody owning a Mac, and which the refusal's own text admitted was asking
// for a guess if nobody did.
//
// ios/verify/importer.swift asks the compiler instead. It declares gobind's C
// as gobind spells it and annotates each symbol with the Swift type this file
// says it imports as, so a disagreement is a type error at the line that names
// both. This is the Go end of that pairing: it holds the two tables to the
// annotations, in both directions.
//
// # Why both directions
//
// A row with no reading is the old problem back again — a Swift spelling
// somebody wrote down. A reading with no row is a line that type-checks and
// answers a question nothing asked, which is the shape a stale control takes:
// the file would go on passing while the table it exists for had moved.
//
// # What it cannot do
//
// Settle the reading. This test holds the two tables to what importer.swift
// SAYS; what holds that file to the truth is a compiler, and Clang's
// Objective-C importer is the compiler in question.
//
// That division used to be stated here and left there, which made it the silent
// half: on a machine with no Swift toolchain the pairing was verified, the
// reading behind it was not, and nothing said so — the rows were as good as the
// last machine somebody had run ios/verify/run.sh on, and there was no way to
// tell from a green `go test ./...` whether that had ever happened.
// TestTheImporterReadingIsSettledByACompiler is the other half: it runs the
// typecheck where it can, names the gap where it cannot, and GRMOB_IMPORTER
// turns the gap into a failure for a machine that is supposed to have the
// toolchain. Same shape as the Compose census's source half one file over, and
// for the same reason.
var importerReading = nativeFile("ios", "verify", "importer.swift")

// importerHeader is the C the reading is about, and importerDir is where both
// files live — the typecheck is run from there so the `-import-objc-header`
// path is the one the file's own header names.
var importerHeader = nativeFile("ios", "verify", "importer.h")

// The macOS target the typecheck uses.
//
// The runtime targets iOS 17, whose observation and layout APIs correspond to
// macOS 14; nothing in importer.swift touches either, and the target is here so
// that this and ios/verify/run.sh ask the same question. It is macOS rather
// than iOS because an iOS target needs the iPhoneOS SDK, which is Xcode rather
// than the Command Line Tools — and the importer's answer does not depend on
// which of the two it is reading for.
const importerTarget = "arm64-apple-macos14.0"

// GRMOB_IMPORTER, and the three states the reading can be in.
//
// The same arrangement GRMOB_COMPOSE_SOURCES has, for a gap of the same shape:
// a check whose subject needs something the machine may not have, whose absence
// is the normal case, and whose silence is indistinguishable from a pass.
const importerEnv = "GRMOB_IMPORTER"

// importerRequired is the one value the variable takes. Spelled out rather than
// "any non-empty value is truthy" so that a typo is a failure that names itself
// instead of a setting that silently did nothing.
const importerRequired = "required"

// importerVerdict decides whether the typecheck runs, and what to say when it
// does not.
//
// darwin is whether this is a machine whose Clang has the Objective-C importer;
// swiftc is whether there is a compiler to drive it with; setting is
// GRMOB_IMPORTER as the environment spells it.
//
// A function of values, so its arms are reachable without owning a machine in
// each state — which is the whole reason the gap was worth naming rather than
// merely stating.
func importerVerdict(darwin, swiftc bool, setting string) (run, fail bool, why string) {
	switch setting {
	case "", importerRequired:
	default:
		return false, true, fmt.Sprintf("%s is set to %q, which is not a value it takes. "+
			"The only one is %q; unset it to let a machine without a Swift toolchain "+
			"skip.", importerEnv, setting, importerRequired)
	}

	switch {
	case darwin && swiftc:
		return true, false, "type-checked against the Objective-C importer on this machine"
	case !darwin:
		why = "the Swift spellings in gobindSwiftTypes and gobindErrorOutPointer are " +
			"what Clang's Objective-C importer makes of gobind's C, and that importer " +
			"is an Xcode toolchain. This is not a Mac, so the two tables are held to " +
			"what ios/verify/importer.swift says and nothing here holds that file to a " +
			"compiler."
	default:
		why = "there is no swiftc on PATH to run the reading through. The Xcode " +
			"Command Line Tools supply one:\n\n    xcode-select --install\n"
	}
	if setting == importerRequired {
		return false, true, why + "\n" + importerEnv + "=" + importerRequired +
			" says this machine is one that should settle it."
	}
	return false, false, why + "\nSet " + importerEnv + "=" + importerRequired +
		" on a machine that is supposed to, to hear about this rather than be excused."
}

// The tables' Swift spellings, settled by the compiler that produces them.
//
// # What this adds to the pairing above
//
// TestTheSwiftSpellingsAreReadOffTheImporter holds gobindSwiftTypes and
// gobindErrorOutPointer to importer.swift's annotations, in both directions.
// That is a comparison of two documents. The annotation is only worth anything
// because `swiftc` accepts it — `let f: (Int8) -> Int8 = GrMobImportInt8`
// type-checks exactly when the importer really gave that function that
// signature — and until this test the only thing that ever ran that compiler
// was ios/verify/run.sh, which needs a Mac.
//
// So on every other machine the arrangement was: two tables held to a file, and
// the file held to nothing, with no way to tell. Running it here means a Mac
// with a Go toolchain settles it in `go test ./...` without running the iOS
// pass at all, and every other machine says which half it did.
//
// # Why the same command as ios/verify/run.sh
//
// It is not the same command twice: run.sh calls this test now, so there is one
// spelling of the typecheck and it is here. That is the arrangement
// android/verify/run.sh already has with the Compose census.
func TestTheImporterReadingIsSettledByACompiler(t *testing.T) {
	_, err := exec.LookPath("swiftc")
	run, fail, why := importerVerdict(runtime.GOOS == "darwin", err == nil,
		os.Getenv(importerEnv))
	if !run {
		if fail {
			t.Fatalf("the importer reading: %s", why)
		}
		t.Skipf("the importer reading: %s", why)
	}

	cmd := exec.Command("swiftc", "-typecheck", "-target", importerTarget,
		"-import-objc-header", filepath.Base(importerHeader),
		filepath.Base(importerReading))
	cmd.Dir = filepath.Dir(importerReading)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s does not type-check against %s (%v).\n\n"+
			"Every annotation in that file is one row of gobindSwiftTypes or "+
			"gobindErrorOutPointer, stated as a type so the compiler is what agrees or "+
			"does not. A failure here is the importer disagreeing with a Swift spelling "+
			"this package hands the stub generator — which is a declaration the shell "+
			"cannot call.\n\n%s", importerReading, importerHeader, err, out)
	}
}

// And every state that verdict can be in, none of which this machine can be put
// into by running the tests.
func TestTheImporterVerdictNamesEveryState(t *testing.T) {
	for _, c := range []struct {
		what           string
		darwin, swiftc bool
		setting        string
		run, fail      bool
		mentions       string
	}{
		{what: "a Mac with a compiler", darwin: true, swiftc: true, run: true,
			mentions: "Objective-C importer"},
		{what: "a Mac with a compiler, required", darwin: true, swiftc: true,
			setting: importerRequired, run: true, mentions: "Objective-C importer"},
		{what: "not a Mac", mentions: "not a Mac"},
		{what: "not a Mac, required", setting: importerRequired, fail: true,
			mentions: "not a Mac"},
		{what: "a Mac with no swiftc", darwin: true, mentions: "xcode-select"},
		{what: "a Mac with no swiftc, required", darwin: true, setting: importerRequired,
			fail: true, mentions: "xcode-select"},
		{what: "a spelling the variable does not take", darwin: true, swiftc: true,
			setting: "1", fail: true, mentions: importerRequired},
	} {
		run, fail, why := importerVerdict(c.darwin, c.swiftc, c.setting)
		if run != c.run || fail != c.fail {
			t.Errorf("%s: run=%v fail=%v, want run=%v fail=%v", c.what, run, fail, c.run, c.fail)
		}
		if !strings.Contains(why, c.mentions) {
			t.Errorf("%s: the verdict does not mention %q, so a reader is told the state "+
				"without being told what to do about it:\n%s", c.what, c.mentions, why)
		}
		if run && fail {
			t.Errorf("%s: the verdict says to run the check AND to fail", c.what)
		}
		// A skip that does not name the switch is the state this whole
		// arrangement was moved out of: an absence indistinguishable from a
		// pass, with nothing telling the reader it can be made loud.
		if !run && !fail && !strings.Contains(why, importerEnv) {
			t.Errorf("%s: the skip does not name %s, so a machine that is supposed to "+
				"settle the reading has no way to learn it can say so:\n%s",
				c.what, importerEnv, why)
		}
	}
}

// One annotated symbol: `let x: (Int8) -> Int8 = GrMobImportInt8`.
//
// A regexp rather than a Swift parse, for the reason the rest of this package
// reads native source with one: the file is written to be read this way — one
// declaration per line, the annotation before the symbol — and that constraint
// is stated in its own header.
var importerLine = regexp.MustCompile(
	`(?m)^let \w+: \((.+)\) -> (\S+) = (GrMobImport(?:Out)?\w+)$`)

// The Go types importer.h declares, keyed by the suffix its symbols carry.
//
// Written out rather than derived from the tables, because it is the third
// statement of the pairing and a derived one would agree with whichever side it
// was derived from. `byte` and `uint8` are absent on purpose: they are still
// refused, and for a reason about the C rather than about the Swift (see
// gobindCarriesUnused) — gobind spells a bare `byte` that nothing it emits
// declares, so there is no C to hand the importer.
var importerSymbols = map[string]string{
	"Int8": "int8", "Int16": "int16", "Int32": "int32", "Int64": "int64",
	"Int": "int", "Float32": "float32", "Float64": "float64", "Bool": "bool",
	// The slice, keyed by the Swift name its C spelling imports as rather than
	// by a Go word: importer.h declares NSData* and the symbol is named for it.
	"Data": goByteSlice,
}

// gobindAliases are the Go spellings that are another type under a different
// name. They carry that type's rows and share its reading rather than having
// one of their own, which is what makes them aliases rather than entries.
var gobindAliases = map[string]string{"rune": "int32"}

func TestTheSwiftSpellingsAreReadOffTheImporter(t *testing.T) {
	src := codeIn(t, importerReading)

	// What the importer said, keyed by Go type: the plain spelling and the
	// out-pointer a (value, error) result moves into.
	type reading struct{ plain, out string }
	got := map[string]*reading{}
	for _, m := range importerLine.FindAllStringSubmatch(src, -1) {
		param, result, symbol := m[1], m[2], m[3]
		out := strings.HasPrefix(symbol, "GrMobImportOut")
		suffix := strings.TrimPrefix(strings.TrimPrefix(symbol, "GrMobImport"), "Out")
		goType, ok := importerSymbols[suffix]
		if !ok {
			t.Errorf("%s reads %s, and importerSymbols says no Go type is spelled that "+
				"way. A reading with no row is a line that type-checks and answers a "+
				"question nothing asked.", importerReading, symbol)
			continue
		}
		if got[goType] == nil {
			got[goType] = &reading{}
		}
		if out {
			if result != "Bool" {
				t.Errorf("%s: %s returns %s. The out-pointer shape is `BOOL F(T* ret0_)` "+
					"— the return is what signals the error — so a reading that is not "+
					"about a Bool is not about that shape.", importerReading, symbol, result)
			}
			got[goType].out = param
			continue
		}
		if param != result {
			t.Errorf("%s: %s imports as (%s) -> %s. objcParamType special-cases String "+
				"and nothing else, so every other type is spelled the same in both "+
				"positions — two different answers here mean that argument has stopped "+
				"holding and gobindSwiftTypes needs two real columns for this type.",
				importerReading, symbol, param, result)
		}
		got[goType].plain = param
	}
	if len(got) == 0 {
		t.Fatalf("%s yielded no readings (%s). A parse that finds nothing must not read "+
			"as a pass — it would agree with every row in both tables.",
			importerReading, importerLine)
	}

	// Every spelled scalar has a reading, and the reading is what the row says.
	for goType, spelling := range gobindSwiftTypes {
		canonical := goType
		if alias, isAlias := gobindAliases[goType]; isAlias {
			canonical = alias
		}
		r := got[canonical]
		if r == nil {
			// string, error and the bound interfaces are read off genobjc.go's
			// own nullability choices rather than off the importer, and
			// importer.h says why. Anything else with no reading is a spelling
			// somebody wrote down.
			if goType == "string" || goType == goErrorType {
				continue
			}
			t.Errorf("gobindSwiftTypes spells %s and %s reads nothing for it. A row with "+
				"no reading is the position the fixed-width numerics were in: a Swift "+
				"name in a table, and nothing between it and the header gobind emits.",
				goType, importerReading)
			continue
		}
		if r.plain != spelling.param || r.plain != spelling.result {
			t.Errorf("gobindSwiftTypes spells %s as %q / %q and the importer gives %q "+
				"(%s). The table is what the stub is generated against, so a spelling "+
				"that is not the importer's is a declaration the shell cannot call.",
				goType, spelling.param, spelling.result, r.plain, importerReading)
		}
	}

	// And every out-pointer row.
	for goType, ptr := range gobindErrorOutPointer {
		canonical := goType
		if alias, isAlias := gobindAliases[goType]; isAlias {
			canonical = alias
		}
		r := got[canonical]
		if r == nil || r.out == "" {
			t.Errorf("gobindErrorOutPointer says a %s result moves into %s and %s reads "+
				"no out-pointer for it. Membership in that map IS the claim that the "+
				"value leaves through a pointer, and the pointer's spelling is the one "+
				"thing about it no rule here predicts — UnsafeMutablePointer<ObjCBool> "+
				"is the standing proof of that.", goType, ptr, importerReading)
			continue
		}
		if r.out != ptr {
			t.Errorf("gobindErrorOutPointer spells a %s out-pointer %q and the importer "+
				"gives %q (%s)", goType, ptr, r.out, importerReading)
		}
	}

	// The other direction: a reading with no row.
	for goType, r := range got {
		if _, spelled := gobindSwiftTypes[goType]; !spelled {
			t.Errorf("%s reads %s as %q and gobindSwiftTypes has no row for it — the "+
				"reading is answering a question nothing asks", importerReading, goType, r.plain)
		}
		if r.out == "" {
			continue
		}
		if _, moved := gobindErrorOutPointer[goType]; !moved {
			t.Errorf("%s reads an out-pointer for %s and gobindErrorOutPointer has no "+
				"row for it. That map's membership decides whether swiftResults moves "+
				"the value out at all, so a reading with no row is a shape the stub "+
				"generator will never produce.", importerReading, goType)
		}
	}
}
