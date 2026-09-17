// Package qr encodes a string as a QR Code symbol: a square grid of dark and
// light modules, per ISO/IEC 18004.
//
// It exists so that comps.QRCode can draw one without the module taking a
// third-party dependency. grmob's go.mod holds no runtime dependency outside
// this author's own packages and the gomobile toolchain, and a QR encoder is a
// closed, fully specified algorithm with published test vectors — the kind of
// thing that is cheaper to own than to track. (github.com/skip2/go-qrcode was
// the alternative; it is MIT and correct, but it pulls in image/png and a
// bitmap model this package does not need, and it would be the first
// third-party dependency the framework's own widgets require.)
//
// # What it does and does not do
//
// Byte mode only, in one segment. The other three modes (numeric,
// alphanumeric, kanji) pack denser for their alphabets, but the caller here is
// a widget encoding URLs and deep links — "cats://pair?t=..." is lowercase, so
// alphanumeric mode (uppercase, digits and nine punctuation marks) could not
// take it anyway. Byte mode encodes any byte string, which makes the API total:
// there is no input this package rejects for its characters, only for length.
//
// Versions 1 through 40, error-correction levels L/M/Q/H, and the eight data
// masks with the standard penalty scoring. Structured append (one message
// split across several symbols) and ECI (a declared character set) are out;
// both are for readers this package's caller does not target, and both would
// change the API from "a string" to "a message".
//
// # Shape of the algorithm
//
//	Encode(data, level)
//	  ├─ smallestVersion   pick the smallest symbol the data fits at that level
//	  ├─ bitstream         mode nibble, character count, the bytes, terminator,
//	  │                    pad to a byte, then 0xEC/0x11 alternating to capacity
//	  ├─ interleave        split into blocks, Reed-Solomon each, interleave
//	  ├─ drawFunctionPatterns   finders, timing, alignment, the dark module
//	  ├─ drawCodewords     the zigzag, two columns at a time, right to left
//	  └─ for each of 8 masks: apply, score, keep the best
package qr

import (
	"errors"
	"fmt"
)

// Level is the error-correction level: how much of the symbol is redundancy,
// and so how much of it may be smudged, torn or covered and still read.
//
// Higher levels cost data capacity at a given version, which is why the
// default in comps.QRCode is Medium rather than High: a code on a screen is
// rarely damaged, and a smaller symbol has larger modules at the same drawn
// size, which is what actually makes a phone camera's job easy.
type Level int

const (
	// Low recovers about 7% of the codewords.
	Low Level = iota
	// Medium recovers about 15%. The usual choice.
	Medium
	// Quartile recovers about 25%.
	Quartile
	// High recovers about 30%. Use it when a logo will be laid over the
	// middle, or when the code is printed and will be handled.
	High
)

// String names the level as the specification does.
func (l Level) String() string {
	switch l {
	case Low:
		return "L"
	case Medium:
		return "M"
	case Quartile:
		return "Q"
	case High:
		return "H"
	}
	return fmt.Sprintf("Level(%d)", int(l))
}

// valid reports whether l is one of the four defined levels. An out-of-range
// Level is a programming error, not a value to silently clamp: clamping would
// encode at a level the caller did not ask for and no reader could tell.
func (l Level) valid() bool { return l >= Low && l <= High }

// formatBits is the level's two-bit field in the format information, which is
// *not* in the same order as the Level constants: the specification numbers
// the levels L=01, M=00, Q=11, H=10. Level's own order is by strength, because
// that is the order a caller thinks in.
func (l Level) formatBits() int {
	return [...]int{1, 0, 3, 2}[l]
}

// ErrTooLong is returned when the data does not fit in a version-40 symbol at
// the requested level. It is the only input that cannot be encoded; every byte
// string shorter than that can.
var ErrTooLong = errors.New("qr: data too long for a QR Code at this error-correction level")

// Code is one encoded symbol: a Size × Size grid of modules, where a dark
// module is drawn and a light one is left as background.
//
// The grid includes the function patterns and excludes the quiet zone — the
// blank margin a reader needs around the symbol is the drawing's business, not
// the encoder's, because the encoder has no units to express it in.
type Code struct {
	// Version is 1 to 40. Size is 4*Version + 17.
	Version int

	// Level is the error-correction level the symbol was built at.
	Level Level

	// Mask is the data mask pattern chosen, 0 to 7.
	Mask int

	// Size is the side of the grid in modules.
	Size int

	// modules is row-major, Size*Size long. Kept unexported so a Code cannot
	// be mutated into something that is no longer a valid symbol.
	modules []bool

	// isFunction marks the modules belonging to the finder, timing, alignment
	// and format/version patterns. Masking skips them, so it must outlive the
	// draw pass that set them.
	isFunction []bool

	// scratch is one line's worth of buffer, reused by the mask scoring.
	// Kept on the Code so the eight candidate passes allocate once between
	// them rather than twice per line.
	scratch []bool
}

// Dark reports whether the module at column x, row y is dark. Coordinates
// outside the grid are light, so a caller may walk a padded area without
// bounds-checking every access.
func (c *Code) Dark(x, y int) bool {
	if c == nil || x < 0 || y < 0 || x >= c.Size || y >= c.Size {
		return false
	}
	return c.modules[y*c.Size+x]
}

// Encode builds the smallest symbol that holds data at the given level.
//
// The version is chosen rather than taken as a parameter because a caller who
// names a version has to handle "it did not fit" anyway, and a caller who does
// not care — every caller here — wants the smallest symbol, which has the
// largest modules at a fixed drawn size.
func Encode(data string, level Level) (*Code, error) {
	if !level.valid() {
		return nil, fmt.Errorf("qr: unknown error-correction level %d", int(level))
	}

	version, err := smallestVersion(len(data), level)
	if err != nil {
		return nil, err
	}

	c := &Code{
		Version: version,
		Level:   level,
		Size:    4*version + 17,
		Mask:    -1,
	}
	c.modules = make([]bool, c.Size*c.Size)
	c.isFunction = make([]bool, c.Size*c.Size)

	c.drawFunctionPatterns()
	c.drawCodewords(interleave(bitstream(data, version, level), version, level))

	// The eight masks exist because an unmasked symbol can come out with large
	// blank areas or accidental finder-like runs that confuse a reader. Which
	// one is best depends on the data, so all eight are tried and scored; the
	// lowest penalty wins, ties to the lower mask number.
	best, bestPenalty := 0, -1
	for mask := 0; mask < 8; mask++ {
		c.applyMask(mask)
		c.drawFormatBits(mask)
		p := c.penalty()
		if bestPenalty < 0 || p < bestPenalty {
			best, bestPenalty = mask, p
		}
		// XOR is its own inverse, so re-applying the same mask undoes it and
		// leaves the grid ready for the next candidate.
		c.applyMask(mask)
	}
	c.applyMask(best)
	c.drawFormatBits(best)
	c.Mask = best

	// The function-pattern map and the scoring buffer have done their job;
	// dropping them keeps a finished Code to one grid, which matters when a
	// widget holds one per render.
	c.isFunction, c.scratch = nil, nil
	return c, nil
}

// ---------------------------------------------------------------------------
// Capacity
// ---------------------------------------------------------------------------

// ecCodewordsPerBlock[level][version] is the number of error-correction
// codewords in each block, and numBlocks[level][version] how many blocks the
// data is split into. Index 0 is unused padding so that a version indexes
// directly. These two tables plus rawDataModules replace the more commonly
// printed 160-row capacity table: every other figure is derived from them.
var ecCodewordsPerBlock = [4][41]int{
	// 0   1   2   3   4   5   6   7   8   9  10  11  12  13  14  15  16  17  18  19  20  21  22  23  24  25  26  27  28  29  30  31  32  33  34  35  36  37  38  39  40
	{0, 7, 10, 15, 20, 26, 18, 20, 24, 30, 18, 20, 24, 26, 30, 22, 24, 28, 30, 28, 28, 28, 28, 30, 30, 26, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},  // L
	{0, 10, 16, 26, 18, 24, 16, 18, 22, 22, 26, 30, 22, 22, 24, 24, 28, 28, 26, 26, 26, 26, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28}, // M
	{0, 13, 22, 18, 26, 18, 24, 18, 22, 20, 24, 28, 26, 24, 20, 30, 24, 28, 28, 26, 30, 28, 30, 30, 30, 30, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30}, // Q
	{0, 17, 28, 22, 16, 22, 28, 26, 26, 24, 28, 24, 28, 22, 24, 24, 30, 28, 28, 26, 28, 30, 24, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30}, // H
}

var numBlocks = [4][41]int{
	// 0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40
	{0, 1, 1, 1, 1, 1, 2, 2, 2, 2, 4, 4, 4, 4, 4, 6, 6, 6, 6, 7, 8, 8, 9, 9, 10, 12, 12, 12, 13, 14, 15, 16, 17, 18, 19, 19, 20, 21, 22, 24, 25},              // L
	{0, 1, 1, 1, 2, 2, 4, 4, 4, 5, 5, 5, 8, 9, 9, 10, 10, 11, 13, 14, 16, 17, 17, 18, 20, 21, 23, 25, 26, 28, 29, 31, 33, 35, 37, 38, 40, 43, 45, 47, 49},     // M
	{0, 1, 1, 2, 2, 4, 4, 6, 6, 8, 8, 8, 10, 12, 16, 12, 17, 16, 18, 21, 20, 23, 23, 25, 27, 29, 34, 34, 35, 38, 40, 43, 45, 48, 51, 53, 56, 59, 62, 65, 68},  // Q
	{0, 1, 1, 2, 4, 4, 4, 5, 6, 8, 8, 11, 11, 16, 16, 18, 16, 19, 21, 25, 25, 25, 34, 30, 32, 35, 37, 40, 42, 45, 48, 51, 54, 57, 60, 63, 66, 70, 74, 77, 81}, // H
}

// rawDataModules is the number of modules a version's symbol has left for data
// and error correction once every function pattern is taken out — that is, the
// total area less the three finders with their separators, the two timing
// lines, the alignment patterns, the format information, and (from version 7)
// the version information.
//
// It is computed rather than tabulated because the geometry is regular:
//
//	whole symbol      (4v+17)²                                = 16v² + 136v + 289
//	less finders,     three 8×8 corners and the two 15-bit     − 8·8·3 − 31
//	format, dark        format fields and the dark module
//	less timing       the two lines, minus where they cross    − (4v+1)·2 + ... = − (8v+2) + 6·2
//	                    the finder corners already counted
//	  giving                                                   16v² + 128v + 64
//	less alignment    n = v/7 + 2 patterns per side, 5×5 each, − (25n − 10)n + 55
//	                    less the three that sit on finders and
//	                    the 5 modules each shares with timing
//	less version      two 3×6 blocks, from version 7           − 36
func rawDataModules(version int) int {
	result := (16*version+128)*version + 64
	if version >= 2 {
		n := version/7 + 2
		result -= (25*n-10)*n - 55
		if version >= 7 {
			result -= 36
		}
	}
	return result
}

// dataCodewords is how many 8-bit codewords of payload (message plus padding,
// before error correction) a version holds at a level.
func dataCodewords(version int, level Level) int {
	return rawDataModules(version)/8 - ecCodewordsPerBlock[level][version]*numBlocks[level][version]
}

// charCountBits is the width of the character-count field, which widens twice
// as the symbol grows. In byte mode it is 8 bits up to version 9 and 16 bits
// from version 10 — the standard's third bracket (versions 27+) applies to
// other modes, not to byte.
func charCountBits(version int) int {
	if version <= 9 {
		return 8
	}
	return 16
}

// smallestVersion finds the least version whose capacity at level holds a
// byte-mode segment of n bytes, header included.
//
// The header's own width depends on the version, so the loop cannot be
// replaced by a division: crossing from version 9 to 10 costs eight more bits
// of header at the same time as it gains capacity.
func smallestVersion(n int, level Level) (int, error) {
	for version := 1; version <= 40; version++ {
		// 4 bits of mode indicator, then the count field, then the bytes.
		need := 4 + charCountBits(version) + n*8
		if need <= dataCodewords(version, level)*8 {
			return version, nil
		}
	}
	return 0, ErrTooLong
}

// ---------------------------------------------------------------------------
// The bit stream
// ---------------------------------------------------------------------------

// bits is a big-endian bit buffer: append(bit) writes towards the low end of
// the stream, which is the order the symbol is filled in.
type bits struct {
	b []bool
}

func (s *bits) append(value, n int) {
	for i := n - 1; i >= 0; i-- {
		s.b = append(s.b, (value>>uint(i))&1 != 0)
	}
}

func (s *bits) len() int { return len(s.b) }

// bitstream builds the full payload for one symbol: the segment, the
// terminator, the byte alignment and the pad codewords, which together always
// come to exactly the version's data capacity.
//
//	0100 │ count │ the bytes │ 0000 │ 0… │ EC 11 EC 11 …
//	mode   8/16              terminator  to a byte   to capacity
func bitstream(data string, version int, level Level) []byte {
	capacityBytes := dataCodewords(version, level)
	capacityBits := capacityBytes * 8

	var s bits
	s.append(0b0100, 4) // byte mode
	s.append(len(data), charCountBits(version))
	for i := 0; i < len(data); i++ {
		s.append(int(data[i]), 8)
	}

	// The terminator is four zero bits, truncated when the symbol is nearly
	// full: a reader stops at the end of the symbol as readily as at a
	// terminator, so there is no need to reserve room for a whole one.
	s.append(0, min(4, capacityBits-s.len()))
	// Then zero bits up to the next byte boundary, because everything after
	// this point is counted in codewords.
	s.append(0, (8-s.len()%8)%8)

	out := make([]byte, 0, capacityBytes)
	for i := 0; i < s.len(); i += 8 {
		var v byte
		for j := 0; j < 8; j++ {
			if s.b[i+j] {
				v |= 1 << uint(7-j)
			}
		}
		out = append(out, v)
	}

	// Then the two pad codewords alternating to capacity. They are fixed by
	// the specification (11101100, 00010001) rather than left as zeros so that
	// a decoder reading past the message sees recognisable filler instead of
	// plausible data.
	for i := 0; len(out) < capacityBytes; i++ {
		if i%2 == 0 {
			out = append(out, 0xEC)
		} else {
			out = append(out, 0x11)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Reed-Solomon
// ---------------------------------------------------------------------------

// interleave splits the payload into blocks, computes each block's
// error-correction codewords, and lays the whole lot out in the order the
// symbol is drawn in.
//
// The interleaving is the point of the exercise: a scratch across the symbol
// damages consecutive drawn codewords, and spreading each block's codewords
// across the symbol turns one long burst into a short burst in each block,
// which is what a Reed-Solomon code corrects well.
//
//	blocks   [A0 A1 A2 …][B0 B1 B2 …]      data, then per-block ECC
//	drawn     A0 B0 A1 B1 A2 B2 …           column-wise, short blocks first
func interleave(data []byte, version int, level Level) []byte {
	blockCount := numBlocks[level][version]
	ecLen := ecCodewordsPerBlock[level][version]
	rawCodewords := rawDataModules(version) / 8

	// The blocks are as equal as they can be: the first (blockCount − r) are
	// one codeword shorter than the last r, where r is the remainder. The
	// specification calls these group 1 and group 2.
	shortLen := rawCodewords/blockCount - ecLen
	numShort := blockCount - rawCodewords%blockCount

	gen := rsGenerator(ecLen)
	blocks := make([][]byte, blockCount)
	offset := 0
	for i := range blocks {
		n := shortLen
		if i >= numShort {
			n = shortLen + 1
		}
		dat := data[offset : offset+n]
		offset += n
		// A short block is padded with a zero at the *end of its data* when
		// interleaved, so that column i of every block exists; the padding is
		// not stored, it is skipped at draw time (see the loop below).
		blocks[i] = append(append([]byte{}, dat...), rsRemainder(dat, gen)...)
	}

	out := make([]byte, 0, rawCodewords)
	// Data columns. Short blocks have no codeword in the last data column, so
	// that column is taken from the long blocks only.
	for i := 0; i < shortLen+1; i++ {
		for j, block := range blocks {
			if i < shortLen || j >= numShort {
				out = append(out, block[i])
			}
		}
	}
	// Error-correction columns. Every block has the same number of these, so
	// this half is a plain transpose.
	for i := 0; i < ecLen; i++ {
		for _, block := range blocks {
			out = append(out, block[len(block)-ecLen+i])
		}
	}
	return out
}

// rsGenerator returns the generator polynomial for an n-codeword
// Reed-Solomon code: (x − α⁰)(x − α¹)…(x − α^(n−1)) over GF(256), with the
// leading 1 term dropped because it is implicit.
//
// Coefficients are big-endian (highest power first), which is the order the
// remainder loop consumes them in.
func rsGenerator(n int) []byte {
	result := make([]byte, n)
	result[n-1] = 1 // the polynomial "1", i.e. x⁰

	root := byte(1)
	for i := 0; i < n; i++ {
		// Multiply the accumulated polynomial by (x − root). In GF(256)
		// subtraction is XOR, so (x − root) is (x + root).
		for j := 0; j < n; j++ {
			result[j] = gfMul(result[j], root)
			if j+1 < n {
				result[j] ^= result[j+1]
			}
		}
		root = gfMul(root, 0x02) // next power of α, where α = 2
	}
	return result
}

// rsRemainder divides the message by the generator polynomial and returns the
// remainder: the error-correction codewords, most significant first.
//
// This is polynomial long division done in place over a sliding window the
// width of the remainder, which is all that is needed — the quotient is
// discarded.
func rsRemainder(data, generator []byte) []byte {
	result := make([]byte, len(generator))
	for _, b := range data {
		factor := b ^ result[0]
		copy(result, result[1:])
		result[len(result)-1] = 0
		for i, g := range generator {
			result[i] ^= gfMul(g, factor)
		}
	}
	return result
}

// gfMul multiplies in GF(2⁸) modulo the QR Code's primitive polynomial
// x⁸ + x⁴ + x³ + x² + 1 (0x11D).
//
// Russian-peasant multiplication rather than log/antilog tables: it is
// branch-simple, allocates nothing, and a symbol's worth of multiplications is
// in the tens of thousands — far below where a 512-byte table would earn its
// keep, and without the table's zero special case.
func gfMul(x, y byte) byte {
	var z byte
	for i := 7; i >= 0; i-- {
		// Double z in the field: shift, and fold in the primitive polynomial
		// whenever the shift pushed a bit out of the byte.
		z = (z << 1) ^ ((z >> 7) * 0x1D)
		z ^= ((y >> uint(i)) & 1) * x
	}
	return z
}

// ---------------------------------------------------------------------------
// Drawing
// ---------------------------------------------------------------------------

// set writes one module and marks whether it belongs to a function pattern.
func (c *Code) set(x, y int, dark, function bool) {
	if x < 0 || y < 0 || x >= c.Size || y >= c.Size {
		return
	}
	c.modules[y*c.Size+x] = dark
	if function {
		c.isFunction[y*c.Size+x] = true
	}
}

// drawFunctionPatterns lays down everything that is fixed by the version: the
// three finders, the timing lines, the alignment patterns and the dark module,
// plus placeholder format and version fields so their modules are marked as
// function modules before the data is placed around them.
func (c *Code) drawFunctionPatterns() {
	// Timing patterns: alternating modules along row 6 and column 6, starting
	// dark. They give a reader a ruler for the module pitch.
	for i := 0; i < c.Size; i++ {
		dark := i%2 == 0
		c.set(6, i, dark, true)
		c.set(i, 6, dark, true)
	}

	// The three finder patterns, at the corners that are not bottom-right.
	// Each is drawn with its separator, so the calls place a 9×9 area whose
	// outer ring is light.
	c.drawFinder(3, 3)
	c.drawFinder(c.Size-4, 3)
	c.drawFinder(3, c.Size-4)

	// Alignment patterns at every intersection of the version's coordinates,
	// except the three that would land on a finder.
	pos := alignmentPositions(c.Version)
	for i, x := range pos {
		for j, y := range pos {
			last := len(pos) - 1
			if (i == 0 && j == 0) || (i == 0 && j == last) || (i == last && j == 0) {
				continue
			}
			c.drawAlignment(x, y)
		}
	}

	// Format and version information. The bits depend on the mask, which is
	// not chosen yet; drawing them with mask 0 now reserves their modules so
	// the data placement skips them, and Encode redraws them per candidate.
	c.drawFormatBits(0)
	c.drawVersionBits()
}

// drawFinder draws the 7×7 concentric finder centred at (cx, cy) together with
// its one-module light separator, as a 9×9 block of concentric rings:
//
//	█████████   ring 4 is the separator, outside the symbol's 7×7 finder
//	█ ┌─────┐   ring 3 dark, ring 2 light, rings 0–1 dark
//	█ │ ███ │
//	█ │ ███ │
//	█ └─────┘
//
// Chebyshev distance from the centre gives the ring number, so the whole
// pattern is one predicate rather than a literal bitmap.
func (c *Code) drawFinder(cx, cy int) {
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			ring := max(abs(dx), abs(dy))
			c.set(cx+dx, cy+dy, ring != 2 && ring != 4, true)
		}
	}
}

// drawAlignment draws the 5×5 alignment pattern centred at (cx, cy): a dark
// centre, a light ring, a dark ring.
func (c *Code) drawAlignment(cx, cy int) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			c.set(cx+dx, cy+dy, max(abs(dx), abs(dy)) != 1, true)
		}
	}
}

// alignmentPositions is the list of row/column centres for a version's
// alignment patterns. Version 1 has none.
//
// The first is always 6 (aligned with the timing line) and the last always
// size−7; the ones between are spaced by a step derived from how many are
// needed, working backwards from the last so the *even* spacing is at the far
// end and any slack falls in the first gap. Version 32 is the one version
// whose derived step disagrees with the published table, and is special-cased.
func alignmentPositions(version int) []int {
	if version == 1 {
		return nil
	}
	n := version/7 + 2
	step := 26
	if version != 32 {
		step = (version*4 + n*2 + 1) / (n*2 - 2) * 2
	}
	out := make([]int, n)
	out[0] = 6
	for i, pos := n-1, version*4+10; i >= 1; i, pos = i-1, pos-step {
		out[i] = pos
	}
	return out
}

// drawFormatBits writes the 15-bit format information — the error-correction
// level and the mask number, protected by a BCH(15,5) code — into its two
// copies: an L around the top-left finder, and a split copy beside the other
// two finders so that a symbol with one damaged corner still reads.
func (c *Code) drawFormatBits(mask int) {
	data := c.Level.formatBits()<<3 | mask
	// BCH(15,5): ten check bits, the remainder of data·x¹⁰ modulo the
	// generator x¹⁰+x⁸+x⁵+x⁴+x²+x+1.
	rem := data
	for i := 0; i < 10; i++ {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	// The mask 0x5412 stops an all-zero format field, which would otherwise
	// be indistinguishable from blank symbol area.
	value := (data<<10 | rem) ^ 0x5412

	// First copy: down the left of the top-left finder, then right along its
	// bottom, skipping the timing line.
	for i := 0; i <= 5; i++ {
		c.set(8, i, bit(value, i), true)
	}
	c.set(8, 7, bit(value, 6), true)
	c.set(8, 8, bit(value, 7), true)
	c.set(7, 8, bit(value, 8), true)
	for i := 9; i < 15; i++ {
		c.set(14-i, 8, bit(value, i), true)
	}

	// Second copy: along the bottom of the top-right finder and up the side of
	// the bottom-left one.
	for i := 0; i < 8; i++ {
		c.set(c.Size-1-i, 8, bit(value, i), true)
	}
	for i := 8; i < 15; i++ {
		c.set(8, c.Size-15+i, bit(value, i), true)
	}
	// The dark module: always dark, always just above the bottom-left format
	// field. It is not information, it is a fixed reference.
	c.set(8, c.Size-8, true, true)
}

// drawVersionBits writes the 18-bit version information (6 data bits and a
// BCH(18,6) remainder) in two 3×6 blocks, above the bottom-left finder and
// left of the top-right one. Versions below 7 carry none: a reader counts the
// modules instead.
func (c *Code) drawVersionBits() {
	if c.Version < 7 {
		return
	}
	rem := c.Version
	for i := 0; i < 12; i++ {
		rem = (rem << 1) ^ ((rem >> 11) * 0x1F25)
	}
	value := c.Version<<12 | rem

	for i := 0; i < 18; i++ {
		dark := bit(value, i)
		a, b := c.Size-11+i%3, i/3
		c.set(a, b, dark, true) // top-right block
		c.set(b, a, dark, true) // bottom-left block, transposed
	}
}

// drawCodewords places the interleaved codewords in the symbol's zigzag: two
// columns at a time from the right edge leftwards, alternating upwards and
// downwards, most significant bit first, skipping function modules.
//
//	┌──────────┐   ↑ ↑   the pair of columns is walked as one two-wide
//	│          │   │ │   ribbon, right module then left, row by row
//	│          │   ↓ ↓
//
// Column 6 is the vertical timing line; it is skipped whole rather than
// module-by-module so that the two-wide ribbon stays two data columns wide on
// both sides of it.
func (c *Code) drawCodewords(data []byte) {
	i := 0 // bit index into data
	for right := c.Size - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for vert := 0; vert < c.Size; vert++ {
			for j := 0; j < 2; j++ {
				x := right - j
				// Columns alternate direction so the ribbon snakes rather than
				// jumping back to the top; ((right+1)&2) is 0 for every other
				// pair of columns.
				upward := ((right + 1) & 2) == 0
				y := vert
				if upward {
					y = c.Size - 1 - vert
				}
				if c.isFunction[y*c.Size+x] || i >= len(data)*8 {
					continue
				}
				c.modules[y*c.Size+x] = bit(int(data[i>>3]), 7-i&7)
				i++
			}
		}
	}
}

// maskAt reports whether the given mask pattern covers module (x, y). The
// eight formulas are the specification's, in its numbering.
func maskAt(mask, x, y int) bool {
	switch mask {
	case 0:
		return (x+y)%2 == 0
	case 1:
		return y%2 == 0
	case 2:
		return x%3 == 0
	case 3:
		return (x+y)%3 == 0
	case 4:
		return (x/3+y/2)%2 == 0
	case 5:
		return x*y%2+x*y%3 == 0
	case 6:
		return (x*y%2+x*y%3)%2 == 0
	case 7:
		return ((x+y)%2+x*y%3)%2 == 0
	}
	return false
}

// applyMask XORs a mask pattern over every data module, leaving function
// modules alone. It is an involution, so calling it twice with the same mask
// restores the grid.
func (c *Code) applyMask(mask int) {
	for y := 0; y < c.Size; y++ {
		for x := 0; x < c.Size; x++ {
			idx := y*c.Size + x
			if !c.isFunction[idx] && maskAt(mask, x, y) {
				c.modules[idx] = !c.modules[idx]
			}
		}
	}
}

// penalty scores the masked symbol by the specification's four rules. A lower
// score is a symbol a reader is less likely to misread, so Encode keeps the
// mask with the lowest.
func (c *Code) penalty() int {
	const (
		n1 = 3  // per run of five, plus one per module beyond
		n2 = 3  // per 2×2 block of one colour
		n3 = 40 // per finder-lookalike
		n4 = 10 // per 5% that the dark share strays from half
	)
	total := 0

	// Rules 1 and 3, along rows and then along columns. Both are run-length
	// questions, so one pass over each line answers both.
	//
	// The line is copied into a scratch buffer first. This is the inner loop of
	// the whole encoder — two directions × every line × eight candidate masks —
	// and reading a column straight out of the row-major grid strides the cache
	// by a whole row per module, while branching on the direction inside the
	// loop costs a test per module on top.
	line := c.scratch
	if len(line) < c.Size {
		line = make([]bool, c.Size)
		c.scratch = line
	}
	for i := 0; i < c.Size; i++ {
		copy(line, c.modules[i*c.Size:(i+1)*c.Size])
		total += lineScore(line, n1, n3)
		for j := 0; j < c.Size; j++ {
			line[j] = c.modules[j*c.Size+i]
		}
		total += lineScore(line, n1, n3)
	}

	// Rule 2: every 2×2 window that is one colour. Overlapping windows are
	// each counted, which is what makes a large blank area expensive.
	dark := 0
	for y := 0; y < c.Size; y++ {
		row := c.modules[y*c.Size : (y+1)*c.Size]
		for x, a := range row {
			if a {
				dark++
			}
			if x+1 < c.Size && y+1 < c.Size {
				next := c.modules[(y+1)*c.Size+x : (y+1)*c.Size+x+2]
				if a == row[x+1] && a == next[0] && a == next[1] {
					total += n2
				}
			}
		}
	}

	// Rule 4: how far the dark share is from half.
	//
	//	k = ceil(|dark share − 50%| / 5%) − 1
	//
	// so a symbol between 45% and 55% dark scores nothing, one between 40%
	// and 60% scores 10, and so on. Written over integers — |20·dark − 10·total|
	// divided by total is |share − 50%| in units of 5% — so the score is exact
	// and identical on every platform.
	totalModules := c.Size * c.Size
	k := (abs(dark*20-totalModules*10)+totalModules-1)/totalModules - 1
	total += k * n4

	return total
}

// lineScore returns one row's or column's rule-1 and rule-3 penalties.
//
// Rule 3 looks for the 1:1:3:1:1 finder proportion with four light modules on
// one side — the eleven-module patterns 10111010000 and 00001011101 — because
// a reader hunting for finder patterns can lock onto one of those in the data
// and mislocate the symbol. The four light modules may be supplied by the
// quiet zone, so the window is slid over the line with four light modules
// imagined at each end.
//
// The window is carried as an eleven-bit register and shifted one module at a
// time, so each module is read once rather than eleven times.
func lineScore(line []bool, n1, n3 int) int {
	const (
		pattern = 0b10111010000
		mirror  = 0b00001011101
		width   = 11
		keep    = 1<<width - 1
	)

	total := 0
	window := 0
	runColor, runLen := line[0], 1

	// i walks the line with four imagined light modules at each end, so the
	// module at i is line[i-4] when that index exists and light otherwise.
	for i := 0; i < len(line)+8; i++ {
		var dark bool
		if p := i - 4; p >= 0 && p < len(line) {
			dark = line[p]

			// Rule 1, over the real line only.
			if p > 0 {
				if dark == runColor {
					runLen++
				} else {
					if runLen >= 5 {
						total += n1 + (runLen - 5)
					}
					runColor, runLen = dark, 1
				}
			}
		}

		window = (window << 1) & keep
		if dark {
			window |= 1
		}
		if i >= width-1 && (window == pattern || window == mirror) {
			total += n3
		}
	}
	if runLen >= 5 {
		total += n1 + (runLen - 5)
	}

	return total
}

// bit reports whether bit i (counting from the least significant) of value is
// set.
func bit(value, i int) bool { return (value>>uint(i))&1 != 0 }

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
