package qr

import (
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// The field and the code
// ---------------------------------------------------------------------------

// The multiplication is only right if it is right in the field QR Codes use —
// GF(2⁸) modulo 0x11D, which is not AES's 0x11B and not any of the other
// primitive polynomials of that degree. A wrong polynomial still gives a
// closed, plausible-looking operation, so the property that pins it is that 2
// generates the whole multiplicative group: its powers must cycle with period
// exactly 255, visiting every non-zero byte once.
func TestGaloisFieldIsTheQRFieldOfOrder256(t *testing.T) {
	seen := map[byte]int{}
	x := byte(1)
	for i := 0; i < 255; i++ {
		if prev, dup := seen[x]; dup {
			t.Fatalf("α^%d = α^%d = %#02x: 2 does not generate the group, so the polynomial is wrong", i, prev, x)
		}
		seen[x] = i
		x = gfMul(x, 2)
	}
	if x != 1 {
		t.Errorf("α^255 = %#02x, want 1", x)
	}
	if len(seen) != 255 {
		t.Errorf("visited %d of the 255 non-zero bytes", len(seen))
	}

	if got := gfMul(0x80, 0x02); got != 0x1D {
		t.Errorf("α^7 · α = %#02x, want 0x1D: x⁸ must reduce to x⁴+x³+x²+1", got)
	}
	for _, v := range []byte{0, 1, 2, 0x53, 0xFF} {
		if got := gfMul(v, 1); got != v {
			t.Errorf("%#02x · 1 = %#02x, want itself", v, got)
		}
		if got := gfMul(v, 0); got != 0 {
			t.Errorf("%#02x · 0 = %#02x, want 0", v, got)
		}
		if a, b := gfMul(v, 0x35), gfMul(0x35, v); a != b {
			t.Errorf("%#02x·0x35 = %#02x but 0x35·%#02x = %#02x: not commutative", v, a, v, b)
		}
	}
}

// evalAt evaluates the polynomial whose coefficients are p (highest power
// first) at the field element x, by Horner's rule.
func evalAt(p []byte, x byte) byte {
	var acc byte
	for _, c := range p {
		acc = gfMul(acc, x) ^ c
	}
	return acc
}

// A Reed-Solomon codeword is *defined* as a polynomial divisible by the
// generator, so every root of the generator — α⁰ through α^(n−1) — must be a
// root of the codeword too. That is the check that is independent of the
// division code being tested: it uses only multiplication.
func TestReedSolomonCodewordsVanishAtTheGeneratorsRoots(t *testing.T) {
	message := []byte("The quick brown fox jumps over the lazy dog, 0123456789.")
	for _, ecLen := range []int{7, 10, 13, 16, 17, 18, 20, 22, 24, 26, 28, 30} {
		gen := rsGenerator(ecLen)
		codeword := append(append([]byte{}, message...), rsRemainder(message, gen)...)

		root := byte(1)
		for i := 0; i < ecLen; i++ {
			if got := evalAt(codeword, root); got != 0 {
				t.Errorf("ec=%d: codeword(α^%d) = %#02x, want 0", ecLen, i, got)
			}
			root = gfMul(root, 2)
		}
	}
}

// The generator for two check codewords is (x−α⁰)(x−α¹) = x² + 3x + 2, small
// enough to multiply out by hand, which pins the coefficient order (highest
// power first, leading 1 implicit) that everything else assumes.
func TestGeneratorPolynomialOrderIsHighestPowerFirst(t *testing.T) {
	got := rsGenerator(2)
	if len(got) != 2 || got[0] != 3 || got[1] != 2 {
		t.Errorf("rsGenerator(2) = %v, want [3 2] for x² + 3x + 2", got)
	}
	if got := rsGenerator(1); len(got) != 1 || got[0] != 1 {
		t.Errorf("rsGenerator(1) = %v, want [1] for x + 1", got)
	}
}

// ---------------------------------------------------------------------------
// The tables
// ---------------------------------------------------------------------------

// Byte-mode character capacities, from ISO/IEC 18004 table 7. These are not
// stored anywhere in the package — they fall out of rawDataModules and the two
// block tables — so agreeing with the published figures at both ends of the
// range and at the version-10 header widening is what says the derivation and
// the tables are right together.
func TestByteCapacitiesMatchTheStandard(t *testing.T) {
	cases := []struct {
		version    int
		l, m, q, h int
	}{
		{1, 17, 14, 11, 7},
		{2, 32, 26, 20, 14},
		{3, 53, 42, 32, 24},
		{7, 154, 122, 86, 64},
		{8, 192, 152, 108, 84},
		{9, 230, 180, 130, 98},
		{10, 271, 213, 151, 119},
		{27, 1465, 1125, 805, 625},
		{40, 2953, 2331, 1663, 1273},
	}
	for _, c := range cases {
		want := map[Level]int{Low: c.l, Medium: c.m, Quartile: c.q, High: c.h}
		for level, n := range want {
			// The capacity in characters is what is left of the data
			// codewords once the mode nibble and the count field are paid for.
			got := (dataCodewords(c.version, level)*8 - 4 - charCountBits(c.version)) / 8
			if got != n {
				t.Errorf("version %d level %s: %d bytes, want %d", c.version, level, got, n)
			}
			// And the version chosen for exactly that many bytes must be this
			// one, while one byte more must not fit.
			if v, err := smallestVersion(n, level); err != nil || v > c.version {
				t.Errorf("smallestVersion(%d, %s) = %d, %v; want %d", n, level, v, err, c.version)
			}
			if v, err := smallestVersion(n+1, level); err == nil && v <= c.version {
				t.Errorf("%d bytes fit in version %d at %s, but the standard says %d is the limit", n+1, v, level, n)
			}
		}
	}
}

// Alignment-pattern coordinates, from ISO/IEC 18004 annex E. Version 32 is the
// one the spacing formula gets wrong and the code special-cases, so it is here
// along with its neighbours.
func TestAlignmentPositionsMatchTheStandard(t *testing.T) {
	want := map[int][]int{
		1:  nil,
		2:  {6, 18},
		7:  {6, 22, 38},
		14: {6, 26, 46, 66},
		21: {6, 28, 50, 72, 94},
		31: {6, 30, 56, 82, 108, 134},
		32: {6, 34, 60, 86, 112, 138},
		33: {6, 30, 58, 86, 114, 142},
		40: {6, 30, 58, 86, 114, 142, 170},
	}
	for version, exp := range want {
		got := alignmentPositions(version)
		if len(got) != len(exp) {
			t.Errorf("version %d: %v, want %v", version, got, exp)
			continue
		}
		for i := range exp {
			if got[i] != exp[i] {
				t.Errorf("version %d: %v, want %v", version, got, exp)
				break
			}
		}
	}
}

// ---------------------------------------------------------------------------
// The symbol
// ---------------------------------------------------------------------------

func TestEncodePlacesTheFunctionPatterns(t *testing.T) {
	c, err := Encode("https://example.com/pair?t=abc123", Medium)
	if err != nil {
		t.Fatal(err)
	}
	if c.Size != 4*c.Version+17 {
		t.Errorf("size %d does not match version %d", c.Size, c.Version)
	}

	// A finder is a 7×7 ring pattern at three corners, and the fourth corner
	// is what tells a reader the symbol's rotation, so it must not have one.
	for _, corner := range []struct {
		name   string
		cx, cy int
	}{
		{"top-left", 3, 3},
		{"top-right", c.Size - 4, 3},
		{"bottom-left", 3, c.Size - 4},
	} {
		for dy := -4; dy <= 4; dy++ {
			for dx := -4; dx <= 4; dx++ {
				x, y := corner.cx+dx, corner.cy+dy
				if x < 0 || y < 0 || x >= c.Size || y >= c.Size {
					continue
				}
				ring := max(abs(dx), abs(dy))
				if want := ring != 2 && ring != 4; c.Dark(x, y) != want {
					t.Fatalf("%s finder wrong at (%d,%d): ring %d is %v", corner.name, x, y, ring, c.Dark(x, y))
				}
			}
		}
	}

	// The timing patterns: alternating, starting dark, along row and column 6.
	for i := 8; i < c.Size-8; i++ {
		if want := i%2 == 0; c.Dark(i, 6) != want || c.Dark(6, i) != want {
			t.Fatalf("timing pattern breaks at %d", i)
		}
	}

	// The dark module, which is fixed and is the one module a reader can rely
	// on being set whatever the data.
	if !c.Dark(8, c.Size-8) {
		t.Error("the dark module at (8, size−8) is light")
	}
}

func TestEncodeRejectsWhatCannotFit(t *testing.T) {
	if _, err := Encode(strings.Repeat("x", 2954), Low); err != ErrTooLong {
		t.Errorf("2954 bytes at L: err = %v, want ErrTooLong (the limit is 2953)", err)
	}
	if _, err := Encode(strings.Repeat("x", 2953), Low); err != nil {
		t.Errorf("2953 bytes at L should fit: %v", err)
	}
	if _, err := Encode("hi", Level(9)); err == nil {
		t.Error("an unknown level should be refused, not clamped to a level the caller did not ask for")
	}
}

// An empty string is a legal, if useless, symbol: the smallest one, all
// padding. It is here because a widget's zero value will hit it.
func TestEncodeEmptyStringIsTheSmallestSymbol(t *testing.T) {
	c, err := Encode("", Medium)
	if err != nil {
		t.Fatal(err)
	}
	if c.Version != 1 || c.Size != 21 {
		t.Errorf("version %d size %d, want version 1 at 21 modules", c.Version, c.Size)
	}
	if got, err := decode(c); err != nil || got != "" {
		t.Errorf("round trip = %q, %v; want the empty string back", got, err)
	}
}

func TestDarkIsLightOutsideTheGrid(t *testing.T) {
	c, err := Encode("x", Low)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {c.Size, 0}, {0, c.Size}, {-5, -5}} {
		if c.Dark(p[0], p[1]) {
			t.Errorf("(%d,%d) outside the grid is dark; the quiet zone must read light", p[0], p[1])
		}
	}
	var nilCode *Code
	if nilCode.Dark(0, 0) {
		t.Error("a nil Code should read light rather than panic")
	}
}

// The end-to-end check: take the symbol apart again — read its format
// information, undo the mask, walk the zigzag, de-interleave the blocks and
// unpack the segment — and get the original string back. It covers every step
// between the bytes and the grid at once, which is the part with no published
// vector to check against piecemeal.
func TestEncodeRoundTripsThroughTheSymbol(t *testing.T) {
	inputs := []string{
		"",
		"A",
		"cats://pair?t=9f2c1a&host=studio",
		"https://example.com/" + strings.Repeat("path/", 30),
		strings.Repeat("The quick brown fox. ", 60), // past the version-10 header widening
		"\x00\x01\xff\xfe binary and ünïcödé ☕",
	}
	for _, level := range []Level{Low, Medium, Quartile, High} {
		for _, in := range inputs {
			c, err := Encode(in, level)
			if err != nil {
				t.Fatalf("%s %q: %v", level, truncate(in), err)
			}
			if c.Level != level {
				t.Errorf("%s %q: symbol says level %s", level, truncate(in), c.Level)
			}
			if c.Mask < 0 || c.Mask > 7 {
				t.Errorf("%s %q: mask %d is not one of the eight", level, truncate(in), c.Mask)
			}
			got, err := decode(c)
			if err != nil {
				t.Errorf("%s %q: decoding: %v", level, truncate(in), err)
				continue
			}
			if got != in {
				t.Errorf("%s: round trip gave %q, want %q", level, truncate(got), truncate(in))
			}
		}
	}
}

// Every version and both header widths, at one level, so that the block
// splitting — which changes shape 40 times — is exercised whole rather than at
// the handful of sizes the strings above happen to reach.
func TestEveryVersionRoundTrips(t *testing.T) {
	for version := 1; version <= 40; version++ {
		level := Level(version % 4)
		n := (dataCodewords(version, level)*8 - 4 - charCountBits(version)) / 8
		in := strings.Repeat("qr", n/2+n%2)[:n]

		c, err := Encode(in, level)
		if err != nil {
			t.Fatalf("version %d at %s: %v", version, level, err)
		}
		if c.Version != version {
			t.Fatalf("filling version %d at %s produced version %d", version, level, c.Version)
		}
		got, err := decode(c)
		if err != nil {
			t.Errorf("version %d: %v", version, err)
			continue
		}
		if got != in {
			t.Errorf("version %d: round trip lost the payload (%d of %d bytes match)", version, commonPrefix(got, in), len(in))
		}
	}
}

// The mask is chosen by score, so it is not fixed; what must hold is that the
// choice is the lowest-scoring one, and that it is stable — the same input
// must always give the same symbol, or a cached drawing would be wrong.
func TestMaskIsTheLowestScoringAndTheChoiceIsStable(t *testing.T) {
	const data = "cats://pair?t=9f2c1a"
	c, err := Encode(data, Quartile)
	if err != nil {
		t.Fatal(err)
	}

	// Rebuild the symbol under each mask and score it the way Encode does.
	scores := make([]int, 8)
	for mask := 0; mask < 8; mask++ {
		probe := &Code{Version: c.Version, Level: c.Level, Size: c.Size}
		probe.modules = make([]bool, probe.Size*probe.Size)
		probe.isFunction = make([]bool, probe.Size*probe.Size)
		probe.drawFunctionPatterns()
		probe.drawCodewords(interleave(bitstream(data, probe.Version, probe.Level), probe.Version, probe.Level))
		probe.applyMask(mask)
		probe.drawFormatBits(mask)
		scores[mask] = probe.penalty()
	}
	for mask, score := range scores {
		if score < scores[c.Mask] {
			t.Errorf("chose mask %d scoring %d, but mask %d scores %d", c.Mask, scores[c.Mask], mask, score)
		}
	}

	again, err := Encode(data, Quartile)
	if err != nil {
		t.Fatal(err)
	}
	if again.Mask != c.Mask {
		t.Fatalf("encoding twice chose masks %d and %d", c.Mask, again.Mask)
	}
	for y := 0; y < c.Size; y++ {
		for x := 0; x < c.Size; x++ {
			if c.Dark(x, y) != again.Dark(x, y) {
				t.Fatalf("encoding is not deterministic: (%d,%d) differs", x, y)
			}
		}
	}
}

func TestLevelNames(t *testing.T) {
	for level, want := range map[Level]string{Low: "L", Medium: "M", Quartile: "Q", High: "H"} {
		if got := level.String(); got != want {
			t.Errorf("Level(%d) = %q, want %q", int(level), got, want)
		}
	}
	if got := Level(9).String(); got != "Level(9)" {
		t.Errorf("an out-of-range level prints %q", got)
	}
}

// ---------------------------------------------------------------------------
// A decoder, for the tests only
// ---------------------------------------------------------------------------

// decode reverses Encode as far as the message: it reads the format
// information out of the symbol (rather than trusting the Code's own fields),
// undoes the mask, reads the zigzag, de-interleaves the blocks, and unpacks the
// byte-mode segment. The error-correction codewords are checked for the
// Reed-Solomon property and then discarded — correcting errors is a decoder's
// job and there are none here.
func decode(c *Code) (string, error) {
	// The function-pattern map is dropped once a symbol is built, so rebuild
	// it from the version alone, which is all it depends on.
	skeleton := &Code{Version: c.Version, Size: c.Size}
	skeleton.modules = make([]bool, c.Size*c.Size)
	skeleton.isFunction = make([]bool, c.Size*c.Size)
	skeleton.drawFunctionPatterns()

	level, mask, err := readFormat(c)
	if err != nil {
		return "", err
	}
	if level != c.Level || mask != c.Mask {
		return "", fmt.Errorf("format information says %s/mask %d, the symbol says %s/mask %d", level, mask, c.Level, c.Mask)
	}

	// Undo the mask over the data modules.
	plain := &Code{Version: c.Version, Level: level, Size: c.Size}
	plain.modules = append([]bool{}, c.modules...)
	plain.isFunction = skeleton.isFunction
	plain.applyMask(mask)

	// Walk the same zigzag Encode wrote, collecting bits into codewords.
	rawCodewords := rawDataModules(c.Version) / 8
	raw := make([]byte, 0, rawCodewords)
	var cur byte
	bitsRead := 0
	for right := c.Size - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for vert := 0; vert < c.Size; vert++ {
			for j := 0; j < 2; j++ {
				x := right - j
				y := vert
				if ((right + 1) & 2) == 0 {
					y = c.Size - 1 - vert
				}
				if plain.isFunction[y*c.Size+x] || len(raw) == rawCodewords {
					continue
				}
				cur <<= 1
				if plain.modules[y*c.Size+x] {
					cur |= 1
				}
				if bitsRead++; bitsRead == 8 {
					raw, cur, bitsRead = append(raw, cur), 0, 0
				}
			}
		}
	}

	data, err := deinterleave(raw, c.Version, level)
	if err != nil {
		return "", err
	}

	// Unpack the segment: mode nibble, count, then the bytes.
	if len(data) < 2 {
		return "", fmt.Errorf("only %d data codewords", len(data))
	}
	if mode := data[0] >> 4; mode != 0b0100 {
		return "", fmt.Errorf("mode indicator %04b, want byte mode 0100", mode)
	}
	countBits := charCountBits(c.Version)
	// The count starts on a nibble boundary, so read it bit by bit rather than
	// pretending it is byte-aligned.
	read := func(offset, n int) int {
		v := 0
		for i := 0; i < n; i++ {
			p := offset + i
			v <<= 1
			if data[p/8]&(1<<uint(7-p%8)) != 0 {
				v |= 1
			}
		}
		return v
	}
	count := read(4, countBits)
	start := 4 + countBits
	if (start+count*8+7)/8 > len(data) {
		return "", fmt.Errorf("segment claims %d bytes, which overruns %d data codewords", count, len(data))
	}
	out := make([]byte, count)
	for i := range out {
		out[i] = byte(read(start+i*8, 8))
	}
	return string(out), nil
}

// readFormat recovers the error-correction level and the mask from the
// symbol's first format copy, undoing the 0x5412 mask.
func readFormat(c *Code) (Level, int, error) {
	value := 0
	get := func(x, y int, i int) {
		if c.Dark(x, y) {
			value |= 1 << uint(i)
		}
	}
	for i := 0; i <= 5; i++ {
		get(8, i, i)
	}
	get(8, 7, 6)
	get(8, 8, 7)
	get(7, 8, 8)
	for i := 9; i < 15; i++ {
		get(14-i, 8, i)
	}
	value ^= 0x5412

	data := value >> 10
	// Check the BCH remainder, which is the point of carrying ten check bits.
	rem := data
	for i := 0; i < 10; i++ {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	if data<<10|rem != value {
		return 0, 0, fmt.Errorf("format information fails its BCH check")
	}
	levels := map[int]Level{1: Low, 0: Medium, 3: Quartile, 2: High}
	return levels[data>>3], data & 7, nil
}

// deinterleave undoes interleave: it rebuilds the blocks, checks each one's
// Reed-Solomon syndromes, and returns the data codewords in message order.
func deinterleave(raw []byte, version int, level Level) ([]byte, error) {
	blockCount := numBlocks[level][version]
	ecLen := ecCodewordsPerBlock[level][version]
	rawCodewords := rawDataModules(version) / 8
	shortLen := rawCodewords/blockCount - ecLen
	numShort := blockCount - rawCodewords%blockCount

	blocks := make([][]byte, blockCount)
	for i := range blocks {
		n := shortLen + ecLen
		if i >= numShort {
			n++
		}
		blocks[i] = make([]byte, 0, n)
	}
	k := 0
	for i := 0; i < shortLen+1; i++ {
		for j := range blocks {
			if i < shortLen || j >= numShort {
				blocks[j] = append(blocks[j], raw[k])
				k++
			}
		}
	}
	for i := 0; i < ecLen; i++ {
		for j := range blocks {
			blocks[j] = append(blocks[j], raw[k])
			k++
		}
	}

	out := make([]byte, 0, rawCodewords-ecLen*blockCount)
	for i, block := range blocks {
		root := byte(1)
		for s := 0; s < ecLen; s++ {
			if evalAt(block, root) != 0 {
				return nil, fmt.Errorf("block %d has a non-zero syndrome at α^%d", i, s)
			}
			root = gfMul(root, 2)
		}
		out = append(out, block[:len(block)-ecLen]...)
	}
	return out, nil
}

func truncate(s string) string {
	if len(s) <= 40 {
		return s
	}
	return s[:40] + "…"
}

func commonPrefix(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

// ---------------------------------------------------------------------------
// The two BCH codes, against the standard's own printed strings
// ---------------------------------------------------------------------------
//
// These are the one place where the package can be checked against published
// values rather than against itself: ISO/IEC 18004 prints all 32 format
// strings and all 34 version strings in full. Getting these right also pins
// the level-to-bits mapping (L=01, M=00, Q=11, H=10), which is not the order
// the Level constants are in and is easy to get backwards.

// formatString reads the 15 bits back out of a symbol's first format copy, in
// the specification's order (most significant bit first).
func formatString(c *Code) string {
	var b strings.Builder
	read := func(x, y int) {
		if c.Dark(x, y) {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	// Bit 14 down to bit 0, which is the reverse of the placement order.
	for i := 14; i >= 9; i-- {
		read(14-i, 8)
	}
	read(7, 8)
	read(8, 8)
	read(8, 7)
	for i := 5; i >= 0; i-- {
		read(8, i)
	}
	return b.String()
}

func TestFormatBitsMatchTheStandardsPrintedStrings(t *testing.T) {
	cases := []struct {
		level Level
		mask  int
		want  string
	}{
		{Low, 0, "111011111000100"},
		{Medium, 0, "101010000010010"},
		{High, 0, "001011010001001"},
	}
	for _, c := range cases {
		// Build a bare symbol and draw only the format field: the rest of the
		// grid is irrelevant to what is being checked.
		sym := &Code{Version: 1, Level: c.level, Size: 21}
		sym.modules = make([]bool, 21*21)
		sym.isFunction = make([]bool, 21*21)
		sym.drawFormatBits(c.mask)
		if got := formatString(sym); got != c.want {
			t.Errorf("%s mask %d: %s, want %s", c.level, c.mask, got, c.want)
		}
	}

	// The code is BCH(15,5), whose minimum distance is 7: any two of the 32
	// format strings differ in at least seven bits, which is what lets a
	// reader recover the level and mask from a scuffed corner. A wrong
	// generator or a wrong XOR mask would collapse that distance.
	var all []string
	for level := Low; level <= High; level++ {
		for mask := 0; mask < 8; mask++ {
			sym := &Code{Version: 1, Level: level, Size: 21}
			sym.modules = make([]bool, 21*21)
			sym.isFunction = make([]bool, 21*21)
			sym.drawFormatBits(mask)
			all = append(all, formatString(sym))
		}
	}
	for i := range all {
		for j := i + 1; j < len(all); j++ {
			if d := hamming(all[i], all[j]); d < 7 {
				t.Errorf("format strings %d and %d differ in only %d bits, want 7 or more", i, j, d)
			}
		}
	}
}

// versionString reads the 18 bits of the top-right version block, most
// significant first.
func versionString(c *Code) string {
	out := make([]byte, 18)
	for i := 0; i < 18; i++ {
		ch := byte('0')
		if c.Dark(c.Size-11+i%3, i/3) {
			ch = '1'
		}
		out[17-i] = ch
	}
	return string(out)
}

func TestVersionBitsMatchTheStandardsPrintedStrings(t *testing.T) {
	cases := map[int]string{
		7:  "000111110010010100",
		40: "101000110001101001",
	}
	for version, want := range cases {
		size := 4*version + 17
		sym := &Code{Version: version, Size: size}
		sym.modules = make([]bool, size*size)
		sym.isFunction = make([]bool, size*size)
		sym.drawVersionBits()
		if got := versionString(sym); got != want {
			t.Errorf("version %d: %s, want %s", version, got, want)
		}
	}

	// Versions below 7 carry no version block; a reader counts modules.
	sym := &Code{Version: 6, Size: 41}
	sym.modules = make([]bool, 41*41)
	sym.isFunction = make([]bool, 41*41)
	sym.drawVersionBits()
	for _, dark := range sym.modules {
		if dark {
			t.Fatal("version 6 drew a version block; only versions 7 and up carry one")
		}
	}

	// BCH(18,6) has minimum distance 8 over the 34 versions that carry it.
	var all []string
	for version := 7; version <= 40; version++ {
		size := 4*version + 17
		s := &Code{Version: version, Size: size}
		s.modules = make([]bool, size*size)
		s.isFunction = make([]bool, size*size)
		s.drawVersionBits()
		all = append(all, versionString(s))
	}
	for i := range all {
		for j := i + 1; j < len(all); j++ {
			if d := hamming(all[i], all[j]); d < 8 {
				t.Errorf("version strings for %d and %d differ in only %d bits, want 8 or more", i+7, j+7, d)
			}
		}
	}
}

func hamming(a, b string) int {
	n := 0
	for i := range a {
		if a[i] != b[i] {
			n++
		}
	}
	return n
}
