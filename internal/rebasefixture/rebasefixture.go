// Package rebasefixture is the reference for how a native text field replays
// in-flight typing onto a rewrite from Go (core/text_edit.go, "The protocol"),
// and the case table both native harnesses check their copy against.
//
// # Where this sits
//
// Rebasing is the host's job and runs on no Go code path: the two natives each
// carry a transliteration (rebaseEdit and rebaseCaret in GrMobTextEdits.kt and
// GrMobTextEdits.swift). A rule written out twice by hand agrees only as well
// as the person copying it, so this package is the one statement of it, and
// android/verify and ios/verify execute both copies against Cases. The same
// arrangement as menufixture and canvasfixture.
//
// # The rule: a three-way merge of two single-span edits
//
// Three texts meet when a rewrite arrives at a focused field:
//
//	basis    what Go read before it rewrote (the host's record of the last
//	         edit Go applied)
//	local    what the field shows now: basis plus typing Go has not seen
//	rewrite  what Go replaced basis with
//
// Each of basis→local and basis→rewrite is read as ONE contiguous change, by
// stripping the longest common prefix and then the longest common suffix:
//
//	basis    H E L L O a W O R L D
//	local    H E L L O a b W O R L D      the user's span: [6,6) → "b"
//	rewrite  H E L L O A W O R L D        Go's span:       [5,6) → "A"
//
// The user's span is then carried across Go's change by mapping its two ends
// from basis offsets to rewrite offsets (mapOffset), and the user's text is
// spliced into the rewrite there: "HELLOA" + "b" + "WORLD".
//
// An offset maps exactly when it is outside Go's span, on either side. Inside
// it, an offset maps only when Go's change kept the length (UPPERCASE, a
// character-for-character normaliser), where offset i is still offset i.
// Anywhere else there is no telling where the user's edit belongs in text Go
// has replaced, and Go's text wins outright, which is what every rewrite did
// before replay existed.
//
// "Kept the length" is a heuristic, and its known miss is stated here: Go
// replacing "cat" with "dog" also keeps the length, and a key typed inside
// "cat" lands at the same offset in "dog". A rewrite of *other* text while the
// user types inside it is rare next to a transform of the same text, and the
// alternative (Go wins) loses the key in the common case to be right in the
// rare one.
//
// # What the old rule did, and the case it lost
//
// The first version replayed only an insertion at either end of basis. That
// covered what outruns a round trip most often (a run of keys at the end of a
// field) and nothing else: two keys typed quickly into the middle of a field
// under an UPPERCASE transform lost the second, because the typing was neither
// a prefix nor a suffix of basis. The Android emulator showed it with
// `adb shell input text` at a caret placed mid-text. Every case the old rule
// replayed is replayed identically here; Cases carries them first.
//
// # Where both inserted at one point
//
// When the user's edit and Go's change both sit at one offset (Go inserted
// there and the user typed there) the user's text goes AFTER Go's. That is the
// old rule's order ("typed at the end" was tested before "typed at the
// start"), and comps.PINInput depends on it: a cell Go filled with "1" under a
// "4" the user had typed must come back "14", which PINInput reads as a paste
// at that cell (core/text_edit.go, "Every field, once the host is sequenced").
//
// # Units
//
// Offsets are UTF-16 code units, because both hosts count them: a Kotlin
// String is UTF-16, and the Swift copy works over `utf16`, which is also what
// UITextView's selectedRange counts. A common prefix or suffix is never allowed
// to end between the two halves of a surrogate pair, so a replayed span is
// always whole characters of the string it came from.
package rebasefixture

import "unicode/utf16"

// Case is one rewrite arriving at a focused field.
//
// The JSON names are what ios/verify's Swift decoder reads; android/verify
// takes the table as Kotlin literals instead (see its gen.go for why).
type Case struct {
	Name    string `json:"name"`
	Basis   string `json:"basis"`
	Local   string `json:"local"`
	Rewrite string `json:"rewrite"`
	// Caret is the caret in Local, in UTF-16 units: where the user was when
	// the rewrite arrived.
	Caret int `json:"caret"`

	// Want and WantCaret are Rebase's and Caret's answers, filled by Cases.
	Want      string `json:"want"`
	WantCaret int    `json:"wantCaret"`
}

// Rebase replays the typing from basis to local onto rewrite.
func Rebase(basis, local, rewrite string) string {
	b, l, r := units(basis), units(local), units(rewrite)
	m, ok := merge(b, l, r)
	if !ok {
		return rewrite
	}
	return string(utf16.Decode(m.text))
}

// Caret is where the caret goes after Rebase, for a host that owns its
// selection (the code editors). caret is the caret in local.
//
// It follows the typing, since that is where the user was: a caret before the
// user's span stays with the text before it, one inside the replayed text
// keeps its place in it, and one after it keeps its place in the text after.
// Where Rebase gave Go's text outright, or the caret sat inside text Go
// replaced, it goes to the end of the result, which is where every rewrite put
// it before there was a replay.
func Caret(basis, local, rewrite string, caret int) int {
	b, l, r := units(basis), units(local), units(rewrite)
	m, ok := merge(b, l, r)
	if !ok {
		return len(r)
	}
	at := -1
	switch {
	case caret <= m.userStart:
		// In the text before the user's span, which is basis's own text:
		// map it through Go's change like any basis offset.
		at = mapOffset(caret, len(b), m.goStart, m.goEnd, len(r))
	case caret < m.userStart+len(m.replayed):
		// Inside the replayed text, which lands whole at spliceStart.
		at = m.spliceStart + (caret - m.userStart)
	default:
		// In the text after the user's span: back to a basis offset, through
		// Go's change, then shifted by how much the splice changed the length
		// of the stretch it replaced.
		basisAt := caret - len(l) + len(b)
		mapped := mapOffset(basisAt, len(b), m.goStart, m.goEnd, len(r))
		if mapped >= 0 {
			at = mapped - m.spliceEnd + m.spliceStart + len(m.replayed)
		}
	}
	if at < 0 || at > len(m.text) || splits(m.text, at) {
		return len(m.text)
	}
	return at
}

// splits reports whether offset i of u falls between the two halves of a
// surrogate pair.
func splits(u []uint16, i int) bool {
	return i > 0 && i < len(u) && isHigh(u[i-1]) && isLow(u[i])
}

// merged is one successful merge and the offsets Caret needs from it.
type merged struct {
	text []uint16
	// userStart is where the user's span starts, in basis and local alike
	// (it is the length of their common prefix).
	userStart int
	// replayed is the user's text: local's side of the user's span.
	replayed []uint16
	// goStart and goEnd are Go's span in basis.
	goStart, goEnd int
	// spliceStart and spliceEnd are where the user's span landed in rewrite.
	spliceStart, spliceEnd int
}

// merge is Rebase's arithmetic, returning false where Go's text wins.
func merge(b, l, r []uint16) (merged, bool) {
	if equal(b, l) {
		// Nothing typed since the edit Go read: there is nothing to replay.
		return merged{}, false
	}
	up, us := span(b, l)
	gp, gs := span(b, r)
	m := merged{
		userStart: up,
		replayed:  l[up : len(l)-us],
		goStart:   gp,
		goEnd:     len(b) - gs,
	}
	m.spliceStart = mapOffset(up, len(b), m.goStart, m.goEnd, len(r))
	m.spliceEnd = mapOffset(len(b)-us, len(b), m.goStart, m.goEnd, len(r))
	// A splice point between the halves of a surrogate pair is no telling
	// either: the length-kept arm maps offsets one unit at a time, and Go's
	// text can hold a pair where basis held two single units.
	if m.spliceStart < 0 || m.spliceEnd < 0 || m.spliceStart > m.spliceEnd ||
		splits(r, m.spliceStart) || splits(r, m.spliceEnd) {
		return merged{}, false
	}
	m.text = make([]uint16, 0, m.spliceStart+len(m.replayed)+len(r)-m.spliceEnd)
	m.text = append(m.text, r[:m.spliceStart]...)
	m.text = append(m.text, m.replayed...)
	m.text = append(m.text, r[m.spliceEnd:]...)
	return m, true
}

// mapOffset carries basis offset i across Go's change of basis[goStart:goEnd],
// which made basis (basisLen units) into a rewrite of rewriteLen units.
// Returns -1 where there is no telling.
//
// "After Go's span" is tested first, which is the tie-break the package doc
// describes: at an offset that is both (Go inserted there), the user's text
// goes after Go's.
func mapOffset(i, basisLen, goStart, goEnd, rewriteLen int) int {
	switch {
	case i >= goEnd:
		return i + rewriteLen - basisLen
	case i <= goStart:
		return i
	case rewriteLen == basisLen:
		// Inside a change that kept the length: a character-for-character
		// transform, so offset i is still offset i.
		return i
	}
	return -1
}

// span returns the lengths of the common prefix and the common suffix of a
// and b, with the suffix limited so the two never overlap, and neither ending
// inside a surrogate pair.
func span(a, b []uint16) (prefix, suffix int) {
	n := min(len(a), len(b))
	for prefix < n && a[prefix] == b[prefix] {
		prefix++
	}
	// A prefix whose last unit is a high surrogate stops before that
	// surrogate's low half, whatever follows in either string.
	if prefix > 0 && isHigh(a[prefix-1]) {
		prefix--
	}
	limit := n - prefix
	for suffix < limit && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	// A suffix whose first unit is a low surrogate starts after that
	// surrogate's high half.
	if suffix > 0 && isLow(a[len(a)-suffix]) {
		suffix--
	}
	return prefix, suffix
}

func isHigh(u uint16) bool { return u >= 0xD800 && u < 0xDC00 }
func isLow(u uint16) bool  { return u >= 0xDC00 && u < 0xE000 }

func units(s string) []uint16 { return utf16.Encode([]rune(s)) }

func equal(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Cases is the table both native harnesses run, with Want and WantCaret
// filled in by Rebase and Caret. The first group is every case the old
// ends-only rule replayed, which must come out as it always did.
func Cases() []Case {
	cs := []Case{
		// The old rule's cases, unchanged.
		{Name: "typed at the end, draft cleared (TagInput)", Basis: "beta,", Local: "beta,gam", Rewrite: "", Caret: 8},
		{Name: "typed at the start", Basis: "ab", Local: "xab", Rewrite: "AB", Caret: 1},
		{Name: "nothing typed since", Basis: "hello", Local: "hello", Rewrite: "HELLO", Caret: 5},
		{Name: "a PIN cell Go filled under a typed key", Basis: "", Local: "4", Rewrite: "1", Caret: 1},
		{Name: "typed at the end, rewrite appended", Basis: "ab", Local: "abc", Rewrite: "ab!", Caret: 3},

		// Mid-text typing, which the old rule gave to Go.
		{Name: "mid-text under UPPERCASE", Basis: "HELLOaWORLD", Local: "HELLOabWORLD", Rewrite: "HELLOAWORLD", Caret: 7},
		{Name: "mid-text, a run of keys under UPPERCASE", Basis: "HELLOaWORLD", Local: "HELLOabcdWORLD", Rewrite: "HELLOAWORLD", Caret: 9},
		{Name: "mid-text before Go's change", Basis: "one two", Local: "onex two", Rewrite: "one TWO", Caret: 4},
		{Name: "mid-text after Go's change", Basis: "one two", Local: "one twox", Rewrite: "ONE two", Caret: 8},
		{Name: "a deletion replayed", Basis: "hello", Local: "hell", Rewrite: "Hello", Caret: 4},
		{Name: "a replacement replayed", Basis: "cat sat", Local: "cat mat", Rewrite: "Cat sat", Caret: 5},

		// Where Go's text still wins.
		{Name: "typing inside text Go replaced with other text", Basis: "beta,", Local: "bet", Rewrite: "", Caret: 3},
		{Name: "typing inside a rewrite of other length", Basis: "hello world", Local: "hello, world", Rewrite: "bye", Caret: 6},

		// Beside a span Go deleted: the user's text lands at the edge of it.
		{Name: "typing beside a space Go collapsed", Basis: "a  b", Local: "a x b", Rewrite: "a b", Caret: 3},

		// Units: a surrogate pair is never split.
		{Name: "an emoji typed after an emoji", Basis: "😀", Local: "😀😁", Rewrite: "😀!", Caret: 4},
		{Name: "an emoji replaced by one sharing its high half", Basis: "x😀", Local: "x😁", Rewrite: "X😀", Caret: 3},
	}
	for i := range cs {
		cs[i].Want = Rebase(cs[i].Basis, cs[i].Local, cs[i].Rewrite)
		cs[i].WantCaret = Caret(cs[i].Basis, cs[i].Local, cs[i].Rewrite, cs[i].Caret)
	}
	return cs
}
