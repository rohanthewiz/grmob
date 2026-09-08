/*
 * The C declarations `gomobile bind` emits for the Go types mobile/verify's
 * tables spell, in gobind's own spelling — so the Swift importer can be asked
 * what it makes of them.
 *
 * # Why this file exists
 *
 * mobile/verify/gomobilestub_test.go maps each bound Go type onto the Swift
 * type the shell will see, and the rule the file holds itself to is that every
 * such reading comes off gobind's source or its golden output, never off
 * reasoning. That rule had a gap it could not close: gobind's output is an
 * *Objective-C header*, and what the shell writes against is what the Swift
 * importer makes of that header. Two of the rows — the out-pointers a
 * (value, error) result turns into — were therefore marked "read off a real
 * bind", and every other type that would need such a row was refused with an
 * instruction to go and run `gomobile bind` on a Mac first.
 *
 * That instruction was the whole problem. It blocked the next bridge function
 * of an ordinary shape on somebody having Xcode and a spare afternoon, and its
 * own text warned that writing the row on a guess was the one thing that must
 * not happen. So the question is asked directly instead: these are the C
 * spellings, importer.swift states what Swift is expected to import them as,
 * and swiftc settles it.
 *
 * # Why this answers the same question a real bind does
 *
 * Two halves, and each is read rather than assumed:
 *
 *   the C spelling      gobind's own output. `BasictypesInts(int8_t x,
 *                       int16_t y, int32_t z, int64_t t, long u)` and
 *                       `BasictypesErrorPair(long* _Nullable ret0_, NSError*
 *                       _Nullable* _Nullable error)` are lines of
 *                       bind/testdata/basictypes.objc.h.golden; the float arms
 *                       are genobjc.go's objcType (Float32 -> "float",
 *                       Float64 -> "double"). mobile/verify pins the file this
 *                       is transcribed from, at the version go.mod holds.
 *   the Swift name      this pass. A C function declared here imports exactly
 *                       as the same declaration in a generated header does —
 *                       it is the same importer reading the same C.
 *
 * The control for that second claim is in the file: BOOL* and long* are the two
 * spellings that WERE read off a real bind, and one of them
 * (UnsafeMutablePointer<ObjCBool>) is the answer nothing in this repository
 * would have predicted. They are declared here alongside the rest, so if this
 * arrangement did not reproduce the real bind's answers it would say so on the
 * two cases where the real bind's answers are already known.
 *
 * # What is deliberately not here
 *
 * NSString*, whose nullability gobind chooses per position — and the choice,
 * not the import, is the fact worth reading; genobjc.go's objcParamType is
 * where it is read, and it is read there. This file is for the types whose C
 * spelling is unambiguous and whose *Swift* name was the gap.
 *
 * # The protocol at the end, which is a different question
 *
 * Everything above asks "what is this C type called in Swift". The protocol
 * asks something else: what Clang's ERROR CONVENTION does to an Objective-C
 * method that takes a trailing NSError**. mobile/verify's swiftResults decides
 * that for every bound method it emits, in three arms, and all three were read
 * off one real `gomobile bind` and were prose ever since.
 *
 * The nullable-object arm came first, because it is the one a `([]byte, error)`
 * result lands in. `NSData*` is _Nullable in gobind's own golden
 * (bind/testdata/basictypes.objc.h.golden, line 24) in both positions, so
 * funcSummary leaves it as the return rather than moving it into an
 * out-pointer — and a bound interface method returning it is a nullable object
 * return, which is that arm's shape. The other two arms were left as prose for
 * no reason except that nobody had written them down: they are the same
 * question, asked of the same importer, and declarable in the same protocol.
 * The methods below are all three. Both halves of each are readings; the second
 * half is the compiler's now.
 */

#include <stdint.h>
#import <Foundation/Foundation.h>

/* Parameters, as `void MobileF(T x)`: the plain scalar position. */
int8_t GrMobImportInt8(int8_t x);
int16_t GrMobImportInt16(int16_t x);
int32_t GrMobImportInt32(int32_t x);
int64_t GrMobImportInt64(int64_t x);
long GrMobImportInt(long x);
float GrMobImportFloat32(float x);
double GrMobImportFloat64(double x);
BOOL GrMobImportBool(BOOL x);

/*
 * Out-pointers, as `BOOL MobileF(T* ret0_, NSError** error)`: the shape
 * funcSummary produces when a (value, error) result's first value is not
 * nullable in Objective-C. The error parameter is left off — swiftResults reads
 * that half off gobind and the importer's treatment of NSError** is the error
 * convention, which is a separate subject — so what these ask is one thing: the
 * pointer's Swift spelling.
 */
BOOL GrMobImportOutInt8(int8_t* ret0_);
BOOL GrMobImportOutInt16(int16_t* ret0_);
BOOL GrMobImportOutInt32(int32_t* ret0_);
BOOL GrMobImportOutInt64(int64_t* ret0_);
BOOL GrMobImportOutInt(long* ret0_);
BOOL GrMobImportOutFloat32(float* ret0_);
BOOL GrMobImportOutFloat64(double* ret0_);
BOOL GrMobImportOutBool(BOOL* ret0_);

/*
 * []byte, in the two positions it can appear in. gobind's own golden:
 *
 *   FOUNDATION_EXPORT NSData* _Nullable BasictypesByteArrays(NSData* _Nullable x);
 *
 * Nullable in both, unlike NSString*, which is why a ([]byte, error) result
 * stays the return where an int moves into an out-pointer.
 */
NSData* _Nullable GrMobImportData(NSData* _Nullable x);

/*
 * And the error convention. Declared as a protocol because the convention
 * applies to Objective-C methods and not to C functions — which is itself one
 * of mobile/verify's readings, and the reason a package-level func does not
 * throw.
 *
 * # Three methods, because swiftResults has three arms
 *
 * A bound METHOD's (value, error) result lands in one of three shapes, and
 * which one it lands in is decided by gobind's nullability annotation on the
 * return rather than by the Go type's kind (see nullableObjectResult). The
 * convention reads that annotation and does three different things:
 *
 *   nullable object    it throws, and the import DROPS the optional: nil is
 *                      what signals the error, so it can no longer also be a
 *                      value. `[]byte` and every bound interface land here.
 *   _Nonnull object    it declines to rewrite the method at all — there is no
 *                      way to signal failure through a return annotated
 *                      non-null — so the error parameter stays and nothing
 *                      throws. `string` lands here, because objcType
 *                      annotates a returned NSString* non-null.
 *   BOOL               the textbook shape: the return IS the signal, so it
 *                      disappears and the method becomes a plain `throws`.
 *                      Every scalar result lands here, because a scalar is
 *                      moved into an out-pointer and what is left is a BOOL.
 *
 * All three were read off one real `gomobile bind` and have been prose since.
 * Only the first was settled by a compiler, and it was settled first because it
 * is the arm `([]byte, error)` reaches. The other two are the same question
 * asked of the same importer, and they are declarable in the same header — so
 * there was no reason for them to stay prose except that nobody had written
 * them down.
 */
@protocol GrMobImportErrorConvention
- (nullable NSData *)dataOrError:(NSError * _Nullable * _Nullable)error;
- (nonnull NSString *)nameOrError:(NSError * _Nullable * _Nullable)error;
- (BOOL)storeOrError:(NSError * _Nullable * _Nullable)error;
@end
