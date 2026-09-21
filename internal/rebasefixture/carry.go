package rebasefixture

// Carry is where a focused field's caret goes when the host writes new text
// into it: before is what the field showed, after is what it shows now, and
// caret is the caret in before. All three are UTF-16 units.
//
// # Why a second rule, beside Caret
//
// Caret answers "where was the user typing?", from three texts, and only when
// there was typing in flight to replay. With nothing in flight (local equals
// basis, the ordinary case at human speed) it sends the caret to the end. A
// plain field cannot take that answer: under an UPPERCASE onChange every key
// is a rewrite, and a caret sent to the end each time would put the second
// letter typed mid-text at the end of the field.
//
// So a plain field places its caret from two texts, the one it had and the one
// it is given, whatever produced the new one (Go's rewrite as it stands, or
// Rebase's merge of it with the typing). The browser's writeFieldValue and the
// iOS field's write have always done this. The Android field used to keep the
// caret's raw offset instead, clamped, and a formatting onChange showed what
// that costs:
//
//	mask "(###) ###-####", keys 1 2 3 4 5 6, a second apart
//	"1"    → Go: "(1"    caret kept at 1:  "(|1"
//	"(21"  → Go: "(21"   an echo           "(2|1"
//	…                                      read back: "(234) 651"
//
// Go inserted a "(" before the caret and the caret did not move with its
// text, so every later key went in before the first one.
//
// # The rule
//
// before→after is read as one differing span, as Rebase reads its two edits
// (span), and the caret is mapped across it by mapOffset, the same function
// that carries the ends of the user's span:
//
//	caret at or after the span's end    shifted by the change in length
//	caret at or before its start        left where it is
//	caret inside a span that kept its   left where it is: a character-for-
//	length                              character transform
//	caret inside a span that changed    the end of the new span: the nearest
//	length                              place that is still after the text
//	                                    the caret was after
//
// A result that would split a surrogate pair goes to the end of the text, as
// Caret's does.
func Carry(before, after string, caret int) int {
	b, a := units(before), units(after)
	prefix, suffix := span(b, a)
	at := mapOffset(caret, len(b), prefix, len(b)-suffix, len(a))
	if at < 0 {
		at = len(a) - suffix
	}
	if at > len(a) || splits(a, at) {
		return len(a)
	}
	return at
}

// CarryCase is one write into a focused field.
type CarryCase struct {
	Name   string `json:"name"`
	Before string `json:"before"`
	After  string `json:"after"`
	Caret  int    `json:"caret"`

	// Want is Carry's answer, filled by CarryCases.
	Want int `json:"want"`
}

// CarryCases is the table the hosts' caret-carrying code is run against.
func CarryCases() []CarryCase {
	cs := []CarryCase{
		// A formatting onChange (comps.MaskedInput): Go inserts literals
		// before the caret, and the caret must stay after the key just typed.
		{Name: "a mask's leading literal arrives with the first key", Before: "1", After: "(1", Caret: 1},
		{Name: "a mask's group break arrives with the fourth key", Before: "(5556", After: "(555) 6", Caret: 5},
		{Name: "a mask drops a key it refuses", Before: "(55x", After: "(55", Caret: 4},
		{Name: "a key typed mid-mask reflows the text after it", Before: "(5955) 666", After: "(595) 566-6", Caret: 3},

		// A character-for-character transform keeps the caret where it is.
		{Name: "UPPERCASE mid-text", Before: "HELLOa WORLD", After: "HELLOA WORLD", Caret: 6},
		{Name: "UPPERCASE at the end", Before: "HELLo", After: "HELLO", Caret: 5},

		// A rewrite that shortens the text.
		{Name: "a committed TagInput draft", Before: "beta,", After: "", Caret: 5},
		{Name: "the caret before a change stays", Before: "one two", After: "one TWO!", Caret: 2},
		{Name: "the caret inside text replaced at another length", Before: "hello world", After: "help world", Caret: 4},

		// Units.
		{Name: "an emoji before the caret", Before: "😀a", After: "😀A", Caret: 3},
		{Name: "text inserted before an emoji and the caret", Before: "😀", After: "(😀", Caret: 2},
	}
	for i := range cs {
		cs[i].Want = Carry(cs[i].Before, cs[i].After, cs[i].Caret)
	}
	return cs
}
