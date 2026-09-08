// What the Swift importer makes of gobind's C spellings — the readings
// mobile/verify's type tables are built on, stated as type annotations so the
// compiler is the one that agrees or does not.
//
// Each line is one row of one of two tables in
// mobile/verify/gomobilestub_test.go, and the pairing is held both ways by
// TestTheSwiftSpellingsAreReadOffTheImporter over there: a row added to a table
// with no line here fails, and a line here naming a Swift type the table does
// not is the same failure from the other side. See importer.h for why asking
// the compiler is a reading rather than a derivation.
//
// The annotations are what carry the claim. `let f: (Int8) -> Int8 =
// GrMobImportInt8` type-checks only if the importer really did give that
// function that signature; anything else is an error at this line naming both
// types. Nothing is called and nothing is linked — importer.h declares these
// symbols and no translation unit defines them, which is exactly right for a
// question about declarations.
//
// The two control rows are marked. They are the spellings that were read off a
// real `gomobile bind`, and they are here so that a Swift toolchain which did
// NOT reproduce a real bind's answers would fail on the cases where the real
// bind's answers are already written down, rather than quietly supplying its own
// for the cases where they are not.

// MARK: - parameters and plain results

let grMobImportInt8: (Int8) -> Int8 = GrMobImportInt8
let grMobImportInt16: (Int16) -> Int16 = GrMobImportInt16
let grMobImportInt32: (Int32) -> Int32 = GrMobImportInt32
let grMobImportInt64: (Int64) -> Int64 = GrMobImportInt64
let grMobImportFloat32: (Float) -> Float = GrMobImportFloat32
let grMobImportFloat64: (Double) -> Double = GrMobImportFloat64

// CONTROL: Go `int` is gobind's `long`, and Go `bool` its `BOOL`. Both already
// have rows read off a real bind, so these two lines are the check on the
// arrangement rather than on the types.
let grMobImportInt: (Int) -> Int = GrMobImportInt
let grMobImportBool: (Bool) -> Bool = GrMobImportBool

// MARK: - the out-pointer a (value, error) result moves into

let grMobImportOutInt8: (UnsafeMutablePointer<Int8>?) -> Bool = GrMobImportOutInt8
let grMobImportOutInt16: (UnsafeMutablePointer<Int16>?) -> Bool = GrMobImportOutInt16
let grMobImportOutInt32: (UnsafeMutablePointer<Int32>?) -> Bool = GrMobImportOutInt32
let grMobImportOutInt64: (UnsafeMutablePointer<Int64>?) -> Bool = GrMobImportOutInt64
let grMobImportOutFloat32: (UnsafeMutablePointer<Float>?) -> Bool = GrMobImportOutFloat32
let grMobImportOutFloat64: (UnsafeMutablePointer<Double>?) -> Bool = GrMobImportOutFloat64

// CONTROL: the two rows gobindErrorOutPointer already held, and the reason this
// file is trustworthy. `BOOL*` does NOT import as UnsafeMutablePointer<Bool> —
// ObjCBool is the C ABI's one-byte spelling and it surfaces only through a
// pointer, which is precisely the kind of answer no rule stated in Go would have
// produced. If this arrangement were answering a different question from the one
// a real bind answers, this is the line that would say so.
let grMobImportOutInt: (UnsafeMutablePointer<Int>?) -> Bool = GrMobImportOutInt
let grMobImportOutBool: (UnsafeMutablePointer<ObjCBool>?) -> Bool = GrMobImportOutBool

// MARK: - the OBJECT results, and what the error convention does to them

// []byte, in both positions. `NSData* _Nullable` either way — unlike NSString*,
// whose nullability gobind chooses per position — so there is no asymmetry to
// record and one line settles both columns.
let grMobImportData: (Data?) -> Data? = GrMobImportData

// The three arms of swiftResults' method branch, which had been prose since
// somebody ran a bind once. Each is what Clang's error convention does to a
// bound method carrying a trailing NSError**, and which one applies is decided
// by gobind's nullability annotation on the return — see nullableObjectResult
// over in Go, and importer.h for the C these read.
//
// Nothing runs any of them. This file is type-checked and never linked.

// 1. A NULLABLE object return: the method throws, and the import LOSES the
// optional. nil is what signals the error, so it can no longer also be a value.
// This is the arm a `([]byte, error)` result lands in, and the arm
// `(Iface, error)` already used.
//
// Spelled as an annotated VALUE rather than an annotated return, because that
// is the form the compiler discriminates on: `try p.dataOrError()` is accepted
// where a `Data?` is asked for, since Swift widens on the way out — so
// annotating this function's own return would prove nothing. `let value: Data`
// fails if the import kept the optional.
func grMobImportErrorConventionDropsTheOptional(
    _ p: any GrMobImportErrorConvention) throws {
    let value: Data = try p.dataOrError()
    _ = value
}

// 2. A _Nonnull object return: the convention DECLINES to rewrite the method.
// There is no way to signal failure through a return annotated non-null, so the
// NSError** stays an ordinary parameter and nothing throws. This is the arm
// `(string, error)` lands in, because genobjc.go's objcType annotates a
// returned NSString* non-null.
//
// Spelled as an annotated METHOD REFERENCE rather than a call, because the
// claim is about the method's SHAPE and half of that shape is the parameter
// that survived. `(NSErrorPointer) -> String` is the arm applying to nothing;
// the alternative — the convention applying after all — would be
// `() throws -> String`, which does not convert to it in either direction.
func grMobImportErrorConventionKeepsTheErrorParameter(
    _ p: any GrMobImportErrorConvention) {
    let method: (NSErrorPointer) -> String = p.nameOrError
    _ = method
}

// 3. A BOOL return: the convention's textbook shape. The return IS the signal,
// so it disappears entirely and the method becomes a plain `throws` with no
// result at all. This is the arm every SCALAR (value, error) result reaches —
// the value has already moved into an out-pointer by then, and what is left for
// the convention to read is the BOOL.
//
// A method reference again, and the annotation discriminates in both of the
// ways that matter: a return of `Bool` would not convert to `Void`, and a
// method the convention had declined to rewrite would be
// `(NSErrorPointer) -> Bool`, which is not a throwing nullary function.
func grMobImportErrorConventionDropsABoolReturn(
    _ p: any GrMobImportErrorConvention) throws {
    let method: () throws -> Void = p.storeOrError
    try method()
}
