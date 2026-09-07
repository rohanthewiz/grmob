// A stand-in for the gomobile-generated GrMob framework, so that the app
// layer can be type-checked without a `gomobile bind`.
//
// # Why this file exists
//
// ios/GrMob/App holds six files. Three of them — AudioPlayer, HeadingSensor,
// SystemEvents — reach only Apple frameworks and have been type-checked by
// run.sh since the compass landed. The other three could not be, because
// GomobileBridge.swift begins `import GrMob`: the module produced by
// `gomobile bind` (ios/build.sh), which needs full Xcode, a Go toolchain
// wired to gomobile, and about a minute. Requiring that would have made the
// harness unrunnable for most of the people who need it, so the three files
// were left out of every check the repository has — including the @main entry
// point.
//
// A stand-in module closes that. This file is compiled as `GrMob`, so
// `import GrMob` resolves and the three files type-check like any others. It
// declares only what the shell calls; nothing here runs, and nothing here is
// linked into a shipped app — ios/build.sh still produces the real framework
// and the Xcode project still links that.
//
// # What it is checked against
//
// A hand-written stand-in for generated code is a copy that can drift, and a
// drifted copy is worse than no check: the shell would keep type-checking
// against a bridge Go no longer has. So the declarations below are pinned to
// the Go source they stand for by mobile/verify/gomobilestub_test.go, which
// reads `mobile`'s exported functions and interfaces and requires two things
// of each: a declaration here under gobind's naming rules, and a signature
// that matches character for character what gobind would emit for it —
// parameter types and order, argument labels, nullability and the return.
// Adding a bindable function to mobile/bridge.go without adding it here fails
// `go test ./...`, and so does renaming one of its parameters.
//
// The signature half is why the nullability rule below is load-bearing rather
// than advisory: the test derives `String?` for a `string` parameter and
// `String` for a `string` result, so a declaration that took `String`
// everywhere is now a failure rather than a latitude.
//
// # The naming rules being imitated
//
// gobind emits an Objective-C API and Swift imports that, so a Go symbol
// arrives here through two renamings rather than one:
//
//	Go                              ObjC                     Swift
//	mobile.RenderInitial()          MobileRenderInitial()    MobileRenderInitial()
//	mobile.PatchListener            @protocol MobilePatchListener
//	                                + @interface MobilePatchListener
//	                                                         MobilePatchListenerProtocol
//
// The package name, capitalized, prefixes every symbol — which is why the
// bound package being called `mobile` puts `Mobile` on the front of a
// framework whose module is named GrMob. A Go interface becomes both an ObjC
// protocol and a class of the same name, and Swift breaks that collision by
// suffixing the protocol with `Protocol`; that suffixed spelling is the one
// GomobileBridge.swift conforms to, so it is the one declared here.
//
// Nullability is not symmetric: gobind annotates every `NSString*` parameter
// `_Nullable` and every `NSString*` return `_Nonnull`, so a Go `string`
// argument arrives as `String?` and a Go `string` result as `String`. Keeping
// that asymmetry is the point of copying it — a stub that took `String`
// everywhere would accept shell code the real framework rejects.
//
// # The rules, in a form that cannot go stale
//
// Everything above is prose, and prose about a generator is a copy like any
// other. It agreed with the checker on the day it was written because it was
// written from it, and nothing would have said so on the day it stopped —
// which is the same failure this whole file exists to prevent, one level up.
//
// So the load-bearing half is stated again below as rows, and
// mobile/verify/gomobilestub_test.go reads them out of this comment and holds
// each one to the thing it describes: the version to go.mod, the prefix and
// suffix to the names the checker builds, each type row to gobindSwiftTypes
// and swiftType, and each result row to what swiftResult actually does with a
// signature of that shape. A reworded paragraph up there is a style change; a
// wrong row down here is a test failure.
//
// The rows are readings of `bind/genobjc.go` — objcParamType for a parameter,
// objcType for every other position, funcSummary for a result clause — in the
// gobind the first row pins. That version is the moment at which somebody has
// to look, and the only one there is.
//
//	--- checked against mobile/verify/gomobilestub_test.go ---
//	gobind   v0.0.0-20251021151156-188f512ec823
//	prefix   Mobile
//	suffix   Protocol
//	type     string        String?                 String
//	type     bool          Bool                    Bool
//	type     int           Int                     Int
//	type     <interface>   Mobile<Name>Protocol?   Mobile<Name>Protocol?
//	results  0   bound, with no return clause
//	results  1   bound, as the result's own spelling
//	results  2   refused: gobind keeps the first as the return when
//	results  3   refused: refuses more than two outright
//	--- end ---
import Foundation

/// Go's mobile.PatchListener. Single-method by necessity: gobind cannot bind
/// a Go func parameter, so a callback crosses the FFI as an interface or not
/// at all.
public protocol MobilePatchListenerProtocol: AnyObject {
    func applyPatches(_ patches: String?)
}

/// Go's mobile.SystemEventListener — the app→host sink for toasts, external
/// URLs, audio commands and sensor start/stop.
public protocol MobileSystemEventListenerProtocol: AnyObject {
    func onSystemEvent(_ name: String?, payload: String?)
}

public func MobileDataDir() -> String { "" }
public func MobileRenderInitial() -> String { "" }
public func MobileRenderAgain() -> String { "" }
public func MobileReportHostEvent(_ name: String?, _ payload: String?) -> String { "" }
public func MobileSetDataDir(_ path: String?) {}
public func MobileSetListener(_ l: MobilePatchListenerProtocol?) {}
public func MobileSetSystemEventListener(_ l: MobileSystemEventListenerProtocol?) {}
public func MobileTriggerCallback(_ id: String?) -> String { "" }
public func MobileTriggerTextCallback(_ id: String?, _ value: String?) -> String { "" }
public func MobileTriggerBoolCallback(_ id: String?, _ value: Bool) -> String { "" }
public func MobileTriggerIntCallback(_ id: String?, _ value: Int) -> String { "" }
