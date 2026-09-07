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
